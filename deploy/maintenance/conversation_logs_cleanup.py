#!/usr/bin/env python3
"""Keep the existing 5 GiB / 3 GiB conversation-log policy bounded.

Run on the production host as maintenance, never as a build/test command.
Only completed conversation log records are eligible; core data stays intact.
"""

import datetime
import fcntl
import json
import os
from pathlib import Path
import subprocess
import sys
import time

GIB = 1024**3
MAX_BYTES = 5 * GIB
TARGET_BYTES = 3 * GIB
BODY_BUDGET_BYTES = int(2.5 * GIB)
MIN_FREE_BYTES = 3 * GIB
REWRITE_RESERVE_BYTES = GIB
BATCH_SIZE = 200
MAX_BATCHES = 100
DATA_DIR = "/opt/sub2api/postgres_data"
LOG_FILE = Path("/var/log/clean_conversation_logs.log")
APPLICATION_NAME = "sub2api-conversation-cleanup"
IDLE_SECONDS = 60


def log(event, **values):
    entry = {"time_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
             "event": event, **values}
    line = json.dumps(entry, ensure_ascii=False)
    print(line, flush=True)
    with LOG_FILE.open("a") as destination:
        destination.write(line + "\n")


def free_bytes():
    stat = os.statvfs(DATA_DIR)
    return stat.f_bavail * stat.f_frsize


def sql_command(statement, *, maintenance=False, fast=False):
    timeout = "500ms" if fast else "5min" if maintenance else "30s"
    setup = (f"SET application_name='{APPLICATION_NAME}'; "
             "SET lock_timeout='2s'; "
             f"SET statement_timeout='{timeout}';")
    # Separate -c arguments are required: VACUUM cannot run inside the
    # implicit transaction created by one multi-statement -c argument.
    return ["docker", "exec", "sub2api-postgres", "sh", "-c",
            'psql -X -qAt -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" '
            '-d "$POSTGRES_DB" -c "$1" -c "$2"', "sh", setup, statement]


def sql(statement, *, maintenance=False, fast=False):
    result = subprocess.run(sql_command(statement, maintenance=maintenance,
                                        fast=fast),
                            capture_output=True, text=True,
                            timeout=2 if fast else 315 if maintenance else 40)
    if result.returncode:
        diagnostic = result.stderr.lower()
        kind = ("lock_timeout" if "lock timeout" in diagnostic else
                "statement_timeout" if "statement timeout" in diagnostic else
                "disk_full" if "no space left" in diagnostic else
                "database_error")
        raise RuntimeError(kind)
    return None if maintenance else json.loads(result.stdout)


def relation():
    return sql("""
        SELECT json_build_object(
          'rows', count(*),
          'physical_bytes', pg_total_relation_size('conversation_logs'),
          'stored_bytes', coalesce(sum(pg_column_size(request_body)::bigint
                         + pg_column_size(response_body)::bigint + 1024), 0),
          'pending_rows', count(*) FILTER (
                         WHERE status='pending' OR completed_at IS NULL))
        FROM conversation_logs;
    """)


def rewrite_activity():
    # Values and SQL text are evaluated in PostgreSQL; only counts leave it.
    # A new log query blocked by our rewrite must not reach the application's
    # five-second log-write timeout.
    return sql(f"""
        SELECT json_build_object(
          'active_log_queries', (
            SELECT count(*) FROM pg_stat_activity
            WHERE pid <> pg_backend_pid() AND state='active'
              AND application_name <> '{APPLICATION_NAME}'
              AND query ILIKE '%conversation_logs%'),
          'blocked_by_cleanup', (
            SELECT count(*) FROM pg_stat_activity AS waiting
            WHERE waiting.pid <> pg_backend_pid()
              AND waiting.application_name <> '{APPLICATION_NAME}'
              AND EXISTS (
                SELECT 1 FROM pg_stat_activity AS cleaner
                WHERE cleaner.application_name='{APPLICATION_NAME}'
                  AND cleaner.pid=ANY(pg_blocking_pids(waiting.pid)))))
    """, fast=True)


def cancel_rewrite():
    sql(f"""
        SELECT json_build_object('cancelled', count(*)) FROM (
          SELECT pg_cancel_backend(pid) FROM pg_stat_activity
          WHERE pid <> pg_backend_pid()
            AND application_name='{APPLICATION_NAME}'
            AND state='active' AND query ILIKE 'VACUUM%'
        ) AS cancelled
    """, fast=True)


def guarded_rewrite():
    activity = rewrite_activity()
    if activity['active_log_queries']:
        log('physical_rewrite_deferred_busy', **activity)
        return False
    process = subprocess.Popen(
        sql_command('VACUUM (FULL, ANALYZE) conversation_logs;', maintenance=True),
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    deadline = time.monotonic() + 300
    cancelled = False
    try:
        while process.poll() is None:
            if time.monotonic() >= deadline:
                cancel_rewrite()
                cancelled = True
                log('physical_rewrite_cancelled_deadline')
                break
            activity = rewrite_activity()
            if activity['blocked_by_cleanup']:
                cancel_rewrite()
                cancelled = True
                log('physical_rewrite_cancelled_for_requests', **activity)
                break
            time.sleep(0.5)
        _, diagnostic = process.communicate(timeout=10)
    except (RuntimeError, subprocess.TimeoutExpired, OSError, ValueError):
        # If monitoring cannot establish that the rewrite is unobstructive,
        # cancel this maintenance query, retaining ordinary space reuse.
        cancel_rewrite()
        process.communicate(timeout=10)
        log('physical_rewrite_cancelled_monitor_unavailable')
        return False
    if cancelled:
        return False
    if process.returncode:
        log('physical_rewrite_deferred',
            reason='lock_timeout' if 'lock timeout' in diagnostic.lower()
            else 'maintenance_query_failed')
        return False
    return True


def run():
    with open("/run/lock/sub2api-conversation-cleanup.lock", "a") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            log("already_running")
            return
        before = relation()
        available = free_bytes()
        log("check", **before, available_bytes=available,
            trigger_bytes=MAX_BYTES, target_bytes=TARGET_BYTES)
        if before["physical_bytes"] < MAX_BYTES and available >= MIN_FREE_BYTES:
            log("below_threshold")
            return
        if before["physical_bytes"] <= TARGET_BYTES:
            log("table_already_within_target", available_bytes=available)
            return

        # Body sizes describe live compressed values. A bloated physical table
        # must not cause repeated deletion of otherwise in-budget live rows.
        budget = BODY_BUDGET_BYTES
        deleted = 0
        if before["stored_bytes"] > budget:
            for _ in range(MAX_BATCHES):
                result = sql(f"""
                    WITH ranked AS (
                      SELECT id, created_at, status, completed_at,
                        sum(pg_column_size(request_body)::bigint
                            + pg_column_size(response_body)::bigint + 1024)
                        OVER (ORDER BY created_at DESC, id DESC) AS kept_bytes
                      FROM conversation_logs
                    ), oldest AS (
                      SELECT id FROM ranked
                      WHERE kept_bytes > {budget}
                        AND status <> 'pending' AND completed_at IS NOT NULL
                      ORDER BY created_at ASC, id ASC LIMIT {BATCH_SIZE}
                    ), removed AS (
                      DELETE FROM conversation_logs AS logs USING oldest
                      WHERE logs.id=oldest.id AND logs.status <> 'pending'
                        AND logs.completed_at IS NOT NULL
                      RETURNING 1
                    ) SELECT json_build_object('deleted',count(*)) FROM removed;
                """)
                count = result["deleted"]
                deleted += count
                if count == 0:
                    break
                log("delete_batch", completed_records_deleted=count,
                    total_deleted=deleted, available_bytes=free_bytes())
            else:
                log("batch_budget_reached", total_deleted=deleted)

        sql("VACUUM (ANALYZE) conversation_logs;", maintenance=True)
        compact = relation()
        available = free_bytes()
        required = int(compact["stored_bytes"] * 2.5) + REWRITE_RESERVE_BYTES
        if compact["physical_bytes"] > TARGET_BYTES:
            quiet = sql(f"""
                SELECT json_build_object('recent_logs', count(*))
                FROM conversation_logs
                WHERE created_at > now() - interval '{IDLE_SECONDS} seconds'
                   OR completed_at > now() - interval '{IDLE_SECONDS} seconds'
                   OR (status='pending'
                       AND created_at > now() - interval '10 minutes')
            """)
            if quiet['recent_logs']:
                log('physical_rewrite_deferred_busy', **quiet,
                    ordinary_vacuum_reuses_space=True)
            elif available >= required:
                log("physical_rewrite_start", live_stored_bytes=compact["stored_bytes"],
                    required_free_bytes=required, available_bytes=available)
                guarded_rewrite()
                compact = relation()
            else:
                log("physical_rewrite_deferred", live_stored_bytes=compact["stored_bytes"],
                    required_free_bytes=required, available_bytes=available)
        log("complete", **compact, completed_records_deleted=deleted,
            available_bytes=free_bytes())


if __name__ == "__main__":
    try:
        run()
    except (RuntimeError, subprocess.TimeoutExpired, OSError, ValueError) as error:
        # Never export SQL stderr, request bodies, keys, or credentials.
        kind = str(error) if isinstance(error, RuntimeError) else type(error).__name__
        log("failed", error_class=kind)
        sys.exit(1)
