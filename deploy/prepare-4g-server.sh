#!/usr/bin/env bash
# Prepare a small production host for image-only Compose deployments.
set -Eeuo pipefail

DEPLOY_PATH="${DEPLOY_PATH:-/opt/sub2api}"
SOURCE_DIR="${SOURCE_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.local.yml}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run this script as root." >&2
  exit 1
fi

case "$DEPLOY_PATH" in
  /*) ;;
  *) echo "DEPLOY_PATH must be an absolute path." >&2; exit 1 ;;
esac

case "$COMPOSE_FILE" in
  docker-compose.yml|docker-compose.local.yml|docker-compose.standalone.yml) ;;
  *) echo "Unsupported COMPOSE_FILE: $COMPOSE_FILE" >&2; exit 1 ;;
esac

command -v openssl >/dev/null 2>&1 || {
  echo "openssl is required." >&2
  exit 1
}
command -v docker >/dev/null 2>&1 || {
  echo "Docker is required. Install docker.io first." >&2
  exit 1
}
docker compose version >/dev/null 2>&1 || {
  echo "The Docker Compose plugin is required." >&2
  exit 1
}

SOURCE_COMPOSE="$SOURCE_DIR/deploy/$COMPOSE_FILE"
if [[ ! -f "$SOURCE_COMPOSE" ]]; then
  echo "Compose file not found: $SOURCE_COMPOSE" >&2
  exit 1
fi

install -d -m 0750 "$DEPLOY_PATH"
install -d -m 0750 \
  "$DEPLOY_PATH/data" \
  "$DEPLOY_PATH/postgres_data" \
  "$DEPLOY_PATH/redis_data"
install -m 0644 "$SOURCE_COMPOSE" "$DEPLOY_PATH/$COMPOSE_FILE"

ENV_FILE="$DEPLOY_PATH/.env"
if [[ ! -e "$ENV_FILE" ]]; then
  umask 077
  temp_env="$(mktemp "$DEPLOY_PATH/.env.XXXXXX")"
  cleanup() {
    rm -f -- "$temp_env"
  }
  trap cleanup EXIT

  cat > "$temp_env" <<EOF
# Generated on $(date -Is). Keep this file private.
APP_IMAGE=ghcr.io/alanbulan/sub2apipro:latest
BIND_HOST=127.0.0.1
SERVER_PORT=8080
SERVER_MODE=release
AUTO_SETUP=true

POSTGRES_USER=sub2api
POSTGRES_PASSWORD=$(openssl rand -hex 32)
POSTGRES_DB=sub2api
POSTGRES_MAX_CONNECTIONS=80
POSTGRES_SHARED_BUFFERS=128MB
POSTGRES_EFFECTIVE_CACHE_SIZE=1GB
POSTGRES_MAINTENANCE_WORK_MEM=32MB

DATABASE_MAX_OPEN_CONNS=20
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME_MINUTES=30
DATABASE_CONN_MAX_IDLE_TIME_MINUTES=5

REDIS_PASSWORD=$(openssl rand -hex 32)
REDIS_MAXCLIENTS=1000
REDIS_POOL_SIZE=32
REDIS_MIN_IDLE_CONNS=2

ADMIN_EMAIL=admin@sub2api.local
ADMIN_PASSWORD=$(openssl rand -hex 24)
JWT_SECRET=$(openssl rand -hex 32)
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=$(openssl rand -hex 32)
TZ=Asia/Shanghai
EOF
  chmod 0600 "$temp_env"
  mv -- "$temp_env" "$ENV_FILE"
  trap - EXIT
  echo "Created $ENV_FILE with generated secrets."
else
  chmod 0600 "$ENV_FILE"
  echo "Preserved existing $ENV_FILE."
fi

docker compose --env-file "$ENV_FILE" -f "$DEPLOY_PATH/$COMPOSE_FILE" config --quiet
echo "Prepared image-only deployment at $DEPLOY_PATH."
echo "The host will not build the application; use the GitHub Actions workflow to publish an image."
