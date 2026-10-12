#!/usr/bin/env bash
# Executed by Pro's Build and Deploy Action, never builds on the server.
set -Eeuo pipefail
umask 077

deploy_path=${1:?deployment directory required}
compose_name=${2:?compose filename required}
staged_compose=${3:?staged compose required}
app_image=${4:?immutable image required}
test "$(realpath "$deploy_path")" = /opt/sub2api
case "$compose_name" in
  docker-compose.local.yml|docker-compose.yml|docker-compose.standalone.yml) ;;
  *) exit 1 ;;
esac
[[ "$app_image" =~ ^ghcr\.io/alanbulan/sub2apipro:sha-[0-9a-f]{40}$ ]] || exit 1
test "$(realpath "$staged_compose")" = "$deploy_path/.deploy/${app_image##*:sha-}/$compose_name"
cd "$deploy_path"
test -s .env
test -s "$compose_name"
test -s "$staged_compose"
exec 9>.action-deploy.lock
flock -n 9 || { echo 'Another deployment is active.' >&2; exit 1; }
project=$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' sub2api)
previous_image=$(docker inspect --format '{{.Config.Image}}' sub2api)
test -n "$project"
# Keep only small configuration values in memory for failure recovery.
# The sentinel preserves trailing newlines through command substitution.
original_env=$(cat .env; printf '.')
original_env=${original_env%.}
original_compose=$(cat "$compose_name"; printf '.')
original_compose=${original_compose%.}
stopped=0
completed=0
persisting=0

compose_new() {
  APP_IMAGE="$app_image" docker compose --project-directory "$deploy_path" -p "$project" -f "$staged_compose" "$@"
}

recover() {
  status=$?
  trap - EXIT
  if [ "$completed" = 0 ] && [ "$stopped" = 1 ]; then
    echo 'Deployment failed; restarting the previous application with the existing database.' >&2
    if [ "$persisting" = 1 ]; then
      printf '%s' "$original_compose" > "$compose_name"
      printf '%s' "$original_env" > .env
    fi
    APP_IMAGE="$previous_image" docker compose --project-directory "$deploy_path" -p "$project" -f "$compose_name" up -d --no-build --no-deps sub2api || true
  fi
  rm -f .compose.action-next .env.action-next
  exit "$status"
}
trap recover EXIT

sql() {
  docker exec sub2api-postgres sh -c 'psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Atc "$1"' sh "$1"
}
counts_sql="SELECT 'users', count(*) FROM users UNION ALL SELECT 'accounts', count(*) FROM accounts UNION ALL SELECT 'api_keys', count(*) FROM api_keys UNION ALL SELECT 'groups', count(*) FROM groups ORDER BY 1;"

check_space() {
  local path=$1 required_kib=$2 available_kib available_inodes
  available_kib=$(df -Pk "$path" | awk 'NR == 2 {print $4}')
  available_inodes=$(df -Pi "$path" | awk 'NR == 2 {print $4}')
  if ! [[ "$available_kib" =~ ^[0-9]+$ && "$available_inodes" =~ ^[0-9]+$ ]]; then
    echo "Unable to determine free disk space at $path; leaving the application running." >&2
    return 1
  fi
  if (( available_kib < required_kib || available_inodes < 1024 )); then
    echo "Insufficient disk space at $path: ${available_kib} KiB, ${available_inodes} inodes available; require ${required_kib} KiB and 1024 inodes. Application has not been stopped." >&2
    return 1
  fi
}

# Validate and download before interrupting traffic. Dependencies remain running.
compose_new config --quiet
docker_root=$(docker info --format '{{.DockerRootDir}}')
test -n "$docker_root"
# Reserve 2 GiB before downloading, and 1 GiB for runtime/migrations afterwards.
check_space "$deploy_path" 2097152
check_space "$docker_root" 2097152
compose_new pull sub2api
check_space "$deploy_path" 1048576
check_space "$docker_root" 1048576

echo 'Image ready; draining application requests. No upgrade snapshots are created.'
stopped=1
docker stop --time 90 sub2api
counts_before=$(sql "$counts_sql")

echo 'Starting the new application against the existing data.'
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
counts_after=$(sql "$counts_sql")
[ "$counts_before" = "$counts_after" ] || { echo 'Core record counts changed during deployment.' >&2; exit 1; }

# Promote the validated manifest and persist the exact image for future restarts.
cp "$staged_compose" .compose.action-next
chmod --reference="$compose_name" .compose.action-next
chown --reference="$compose_name" .compose.action-next
persisting=1
mv .compose.action-next "$compose_name"
awk -v image="$app_image" '!/^APP_IMAGE=/ {print} END {print "APP_IMAGE=" image}' .env > .env.action-next
chmod --reference=.env .env.action-next
chown --reference=.env .env.action-next
mv .env.action-next .env
completed=1
echo 'Deployment healthy; core records retained. No upgrade snapshots were created.'

# Remove only obsolete Pro image tags. Docker refuses removal of images in use.
# Keep the previous image until all validation and configuration updates succeed.
if image_tags=$(docker image ls ghcr.io/alanbulan/sub2apipro --format '{{.Repository}}:{{.Tag}}'); then
  while IFS= read -r image_tag; do
    case "$image_tag" in
      ghcr.io/alanbulan/sub2apipro:sha-*)
        if [ "$image_tag" != "$app_image" ]; then
          docker image rm "$image_tag" || echo "Could not remove obsolete image $image_tag; continuing." >&2
        fi
        ;;
    esac
  done <<< "$image_tags"
else
  echo 'Could not list obsolete application images; continuing.' >&2
fi
