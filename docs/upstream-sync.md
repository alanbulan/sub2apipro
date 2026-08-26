# Daily Upstream Sync

The checker fetches the upstream remote, writes `.codex-upstream-sync/report.md`, and returns the pending commit count.

```bash
git remote add upstream https://github.com/Wei-Shaw/sub2api.git
./scripts/check-upstream.sh
```

Run a manual Codex analysis:

```bash
codex exec "Use the upstream-sync skill. Read .codex-upstream-sync/report.md and analyze/merge the pending upstream updates without modifying protected paths."
```

Install a daily 09:00 server-time job:

```cron
0 9 * * * /sub2api/deploy/cron/upstream-sync
```

Required configuration:

- `upstream` remote must point to the source repository;
- `custom/protected-paths.txt` lists files and directories that must not be overwritten;
- `codex` must be available on `PATH` for scheduled execution;
- logs are written under `.codex-upstream-sync/logs/`.
