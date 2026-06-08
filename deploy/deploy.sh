#!/bin/bash
# Production deploy: API + DB run in Docker Compose (no systemd binary).
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
GITHUB_KEY="${GITHUB_KEY:-$HOME/.ssh/github_lao_rice}"

cd "$APP_DIR"
mkdir -p "$APP_DIR/uploads/payment-receipts" "$APP_DIR/uploads/product-images" "$APP_DIR/images/rice"

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

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

chmod +x deploy/deploy.sh deploy/backup-db.sh deploy/restore-db.sh deploy/cron-backup.sh 2>/dev/null || true

_cksum_after="$(cksum "$DEPLOY_SCRIPT" | awk '{print $1}')"
if [ -n "$_cksum_before" ] && [ "$_cksum_before" != "$_cksum_after" ] && [ -z "${DEPLOY_REEXEC:-}" ]; then
  echo "==> deploy.sh updated — re-run with new script"
  export DEPLOY_REEXEC=1
  exec env ENV_BACKUP="$ENV_BACKUP" DEPLOY_REEXEC=1 bash "$DEPLOY_SCRIPT"
fi

echo "==> Restore production .env"
if [ ! -f "$APP_DIR/.env" ] && [ -s "$ENV_BACKUP" ]; then
  cp "$ENV_BACKUP" "$APP_DIR/.env"
fi
if [ ! -f "$APP_DIR/.env" ]; then
  echo "ERROR: $APP_DIR/.env missing. Run deploy/reset-docker-production.sh first." >&2
  exit 1
fi
if ! grep -q '^DB_PASSWORD=' "$APP_DIR/.env"; then
  echo "ERROR: .env must contain DB_PASSWORD=. Run deploy/reset-docker-production.sh" >&2
  exit 1
fi
# Keep JWT_SECRET from backup if git reset overwrote .env keys we care about
if [ -s "$ENV_BACKUP" ]; then
  _jwt="$(grep -E '^JWT_SECRET=' "$ENV_BACKUP" | head -1 | cut -d= -f2- || true)"
  if [ -n "$_jwt" ] && grep -q '^JWT_SECRET=' "$APP_DIR/.env"; then
    sed -i "s|^JWT_SECRET=.*|JWT_SECRET=${_jwt}|" "$APP_DIR/.env"
  fi
fi
rm -f "$ENV_BACKUP"
chmod 600 "$APP_DIR/.env"

echo "==> Disable old systemd API (Docker mode)"
sudo -n systemctl stop lao-rice-api lao-rice-guard.timer 2>/dev/null || true
sudo -n systemctl disable lao-rice-api lao-rice-guard.timer 2>/dev/null || true

echo "==> Backup database (before deploy)"
mkdir -p backups/postgres
if dc ps --status running --services 2>/dev/null | grep -qx db; then
  bash "$APP_DIR/deploy/backup-db.sh" || echo "WARN: backup skipped"
fi

echo "==> Build and restart Docker stack"
if ! dc up -d --build; then
  echo "Retrying after Docker restart..."
  sudo -n systemctl restart docker || true
  sleep 8
  dc up -d --build
fi

echo "==> Health check"
API_PORT="$(grep -E '^API_PORT=' "$APP_DIR/.env" | cut -d= -f2- || echo 8081)"
HEALTH_URL="http://127.0.0.1:${API_PORT}/health"
for i in $(seq 1 30); do
  if curl -sf "$HEALTH_URL" >/dev/null; then
    echo "Deploy OK"
    dc ps
    exit 0
  fi
  sleep 2
done
echo "Health check failed: $HEALTH_URL"
dc logs api --tail 40
exit 1
