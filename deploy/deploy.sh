#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/vps-common.sh
source "$SCRIPT_DIR/lib/vps-common.sh"

APP_DIR="/opt/lao-rice-api"
SERVICE="lao-rice-api"
BIN="$APP_DIR/bin/lao-rice-api"
GITHUB_KEY="${GITHUB_KEY:-$HOME/.ssh/github_lao_rice}"

cd "$APP_DIR"
mkdir -p "$APP_DIR/bin"

if [ -f "$GITHUB_KEY" ]; then
  export GIT_SSH_COMMAND="ssh -i $GITHUB_KEY -o StrictHostKeyChecking=accept-new"
fi

DEPLOY_SCRIPT="$APP_DIR/deploy/deploy.sh"
ENV_BACKUP="${ENV_BACKUP:-$(mktemp)}"
if [ -z "${DEPLOY_REEXEC:-}" ] && [ -f "$APP_DIR/.env" ]; then
  cp "$APP_DIR/.env" "$ENV_BACKUP"
fi

_cksum_before=""
[ -f "$DEPLOY_SCRIPT" ] && _cksum_before="$(cksum "$DEPLOY_SCRIPT" | awk '{print $1}')"

echo "==> Pull latest main"
git fetch origin main
git reset --hard origin/main

chmod +x deploy/deploy.sh deploy/wait-for-db.sh 2>/dev/null || true

_cksum_after="$(cksum "$DEPLOY_SCRIPT" | awk '{print $1}')"
if [ -n "$_cksum_before" ] && [ "$_cksum_before" != "$_cksum_after" ] && [ -z "${DEPLOY_REEXEC:-}" ]; then
  echo "==> deploy.sh updated — re-run with new script"
  export DEPLOY_REEXEC=1
  exec env ENV_BACKUP="$ENV_BACKUP" DEPLOY_REEXEC=1 bash "$DEPLOY_SCRIPT"
fi

echo "==> Restore production .env (never commit .env to git)"
API_PORT="${PORT:-8081}"
JWT_SECRET=""
if [ -s "$ENV_BACKUP" ]; then
  JWT_SECRET="$(grep -E '^JWT_SECRET=' "$ENV_BACKUP" | head -1 | cut -d= -f2- || true)"
fi
if [ -f "$APP_DIR/.db_password" ]; then
  DB_PASSWORD="$(cat "$APP_DIR/.db_password")"
  cat > "$APP_DIR/.env" <<EOF
PORT=${API_PORT}
DATABASE_URL=postgres://postgres:${DB_PASSWORD}@localhost:5433/lao_rice?sslmode=disable
JWT_SECRET=${JWT_SECRET:-change-me-in-production-use-long-random-secret}
JWT_EXPIRY_HOURS=72
SHIPPING_FEE_LAK=30000
FREE_SHIPPING_MIN_SUBTOTAL_LAK=500000
UPLOAD_DIR=${APP_DIR}/uploads
UPLOAD_URL_PREFIX=/uploads
IMAGES_DIR=${APP_DIR}/images
EOF
elif [ -s "$ENV_BACKUP" ]; then
  cp "$ENV_BACKUP" "$APP_DIR/.env"
elif [ -f "$APP_DIR/.env.example" ]; then
  cp "$APP_DIR/.env.example" "$APP_DIR/.env"
  sed -i "s|^UPLOAD_DIR=.*|UPLOAD_DIR=${APP_DIR}/uploads|" "$APP_DIR/.env"
fi
rm -f "$ENV_BACKUP"
mkdir -p "$APP_DIR/uploads/payment-receipts"
mkdir -p "$APP_DIR/uploads/product-images"
mkdir -p "$APP_DIR/images/rice"

echo "==> Start PostgreSQL"
vps_ensure_docker
if ! vps_dc up -d db; then
  vps_sudo_systemctl restart docker || true
  sleep 8
  vps_ensure_docker
  vps_dc up -d db
fi

for i in $(seq 1 30); do
  if vps_dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "==> Build API"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -o "$BIN" ./cmd/server

echo "==> Ensure default admin (seed)"
set -a
# shellcheck disable=SC1091
[ -f "$APP_DIR/.env" ] && source "$APP_DIR/.env"
set +a
/usr/local/go/bin/go run ./cmd/seed -admin-only

echo "==> Install systemd unit (if ExecStart path changed)"
sudo -n cp deploy/lao-rice-api.service /etc/systemd/system/lao-rice-api.service
vps_sudo_systemctl daemon-reload

echo "==> Restart service"
docker rm -f lao-rice-api 2>/dev/null || true
vps_sudo_systemctl restart "$SERVICE"
sleep 2
if ! vps_sudo_systemctl is-active --quiet "$SERVICE"; then
  echo "Service failed to start:"
  vps_sudo_systemctl status "$SERVICE" --no-pager -l || true
  sudo -n journalctl -u "$SERVICE" -n 50 --no-pager || true
  exit 1
fi

echo "==> Health check"
# shellcheck disable=SC1091
[ -f "$APP_DIR/.env" ] && set -a && source "$APP_DIR/.env" && set +a
HEALTH_URL="http://127.0.0.1:${PORT:-8081}/health"
for i in $(seq 1 15); do
  if curl -sf "$HEALTH_URL" >/dev/null; then
    echo "Deploy OK"
    exit 0
  fi
  sleep 2
done
echo "Health check failed: $HEALTH_URL"
exit 1
