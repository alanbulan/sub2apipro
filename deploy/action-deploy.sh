#!/usr/bin/env bash
# Executed by Pro's Build and Deploy Action, never builds on the server.
set -Eeuo pipefail
umask 077

deploy_path=${1:?deployment directory required}
compose_name=${2:?compose filename required}
staged_compose=${3:?staged compose required}
app_image=${4:?immutable image required}
case "$compose_name" in
  docker-compose.local.yml|docker-compose.yml|docker-compose.standalone.yml) ;;
  *) exit 1 ;;
esac
case "$app_image" in ghcr.io/alanbulan/sub2apipro:sha-*) ;; *) exit 1 ;; esac
cd "$deploy_path"
test -s .env
test -s "$compose_name"
test -s "$staged_compose"
exec 9>.action-deploy.lock
flock -n 9 || { echo 'Another deployment is active.' >&2; exit 1; }
project=$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' sub2api)
previous_image=$(docker inspect --format '{{.Config.Image}}' sub2api)
test -n "$project"
backup_dir="$deploy_path/backups/action-$(date -u +%Y%m%dT%H%M%SZ)-${app_image##*:}"
mkdir -p "$backup_dir"
cp -p .env "$backup_dir/env"
cp -p "$compose_name" "$backup_dir/compose.yml"
printf '%s\n' "$previous_image" > "$backup_dir/previous-image"
printf '%s\n' "$app_image" > "$backup_dir/new-image"
stopped=0
completed=0

compose_new() {
  APP_IMAGE="$app_image" docker compose --project-directory "$deploy_path" -p "$project" -f "$staged_compose" "$@"
}

recover() {
  status=$?
  trap - EXIT
  if [ "$completed" = 0 ] && [ "$stopped" = 1 ]; then
    echo 'Deployment failed; restarting the previous application with the existing database.' >&2
    # These upgrades are additive. Never restore a database automatically:
    # doing so could discard transactions committed after the backup.
    cp -p "$backup_dir/compose.yml" "$compose_name"
    cp -p "$backup_dir/env" .env
    APP_IMAGE="$previous_image" docker compose --project-directory "$deploy_path" -p "$project" -f "$compose_name" up -d --no-build --no-deps sub2api || true
    echo "Recovery snapshot retained at $backup_dir" >&2
  fi
  exit "$status"
}
trap recover EXIT

sql() {
  docker exec sub2api-postgres sh -c 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "$1"' sh "$1"
}
counts_sql="SELECT 'users', count(*) FROM users UNION ALL SELECT 'accounts', count(*) FROM accounts UNION ALL SELECT 'api_keys', count(*) FROM api_keys UNION ALL SELECT 'groups', count(*) FROM groups ORDER BY 1;"

# Validate and download before interrupting traffic. Dependencies remain running.
compose_new config --quiet
compose_new pull sub2api
db_bytes=$(sql 'SELECT pg_database_size(current_database());')
data_kib=$(du -sk data | awk '{print $1}')
available_kib=$(df -Pk "$deploy_path" | awk 'NR == 2 {print $4}')
required_kib=$((db_bytes / 1024 + data_kib + 524288))
if [ "$available_kib" -lt "$required_kib" ]; then
  echo 'Insufficient space for a complete pre-upgrade backup.' >&2
  exit 1
fi

echo 'Image ready; draining application requests before the consistent backup.'
stopped=1
docker stop --time 90 sub2api
sql "$counts_sql" > "$backup_dir/core-counts-before.tsv"
sql 'SELECT filename, checksum FROM schema_migrations ORDER BY filename;' > "$backup_dir/migrations-before.tsv"
echo 'Saving full PostgreSQL backup, application files, configuration, and Redis snapshot.'
docker exec sub2api-postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc -Z1' > "$backup_dir/database.dump.partial"
test -s "$backup_dir/database.dump.partial"
docker exec -i sub2api-postgres pg_restore --list < "$backup_dir/database.dump.partial" > "$backup_dir/database-contents.txt"
mv "$backup_dir/database.dump.partial" "$backup_dir/database.dump"
tar -czf "$backup_dir/app-data.tar.gz" data
docker exec sub2api-redis sh -c 'REDISCLI_AUTH="${REDIS_PASSWORD:-}" redis-cli SAVE'
docker cp sub2api-redis:/data/dump.rdb "$backup_dir/redis.rdb"
sha256sum "$backup_dir/database.dump" "$backup_dir/app-data.tar.gz" "$backup_dir/redis.rdb" > "$backup_dir/SHA256SUMS"
printf '%s\n' 'Backup complete; database and Redis services were not restarted.' > "$backup_dir/backup-complete"

echo 'Backup complete; starting the new application against the existing data.'
compose_new up -d --no-build --no-deps sub2api
healthy=0
for attempt in $(seq 1 60); do
  health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' sub2api)
  if [ "$health" = healthy ]; then healthy=1; break; fi
  if [ "$health" = unhealthy ]; then break; fi
  sleep 5
done
[ "$healthy" = 1 ] || { echo 'New application did not become healthy.' >&2; exit 1; }
[ "$(docker inspect --format '{{.Config.Image}}' sub2api)" = "$app_image" ]
sql "$counts_sql" > "$backup_dir/core-counts-after.tsv"
diff -u "$backup_dir/core-counts-before.tsv" "$backup_dir/core-counts-after.tsv"
sql 'SELECT filename, checksum FROM schema_migrations ORDER BY filename;' > "$backup_dir/migrations-after.tsv"

# Promote the validated manifest and persist the exact image for future restarts.
cp -p "$staged_compose" "$compose_name"
awk -v image="$app_image" '!/^APP_IMAGE=/ {print} END {print "APP_IMAGE=" image}' .env > .env.action-next
chmod --reference=.env .env.action-next
mv .env.action-next .env
completed=1
echo "Deployment healthy; core records retained. Recovery snapshot: $backup_dir"
