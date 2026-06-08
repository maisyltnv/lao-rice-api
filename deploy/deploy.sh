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

# Migrate legacy .env (DATABASE_URL / .db_password) → Docker format (DB_PASSWORD)
_db_pass=""
if [ -f "$APP_DIR/.db_password" ]; then
  _db_pass="$(tr -d '\n' < "$APP_DIR/.db_password")"
fi
if [ -z "$_db_pass" ] && grep -q '^DATABASE_URL=' "$APP_DIR/.env"; then
  _db_pass="$(grep '^DATABASE_URL=' "$APP_DIR/.env" | sed -n 's|.*postgres://postgres:\([^@]*\)@.*|\1|p')"
fi
if [ -z "$_db_pass" ] && [ -s "$ENV_BACKUP" ] && grep -q '^DATABASE_URL=' "$ENV_BACKUP"; then
  _db_pass="$(grep '^DATABASE_URL=' "$ENV_BACKUP" | sed -n 's|.*postgres://postgres:\([^@]*\)@.*|\1|p')"
fi
if [ -z "$_db_pass" ]; then
  echo "ERROR: cannot find DB password in .env, .db_password, or backup." >&2
  echo "Run: bash deploy/reset-docker-production.sh" >&2
  exit 1
fi

_jwt="$(grep -E '^JWT_SECRET=' "$APP_DIR/.env" | head -1 | cut -d= -f2- || true)"
[ -n "$_jwt" ] || _jwt="$(grep -E '^JWT_SECRET=' "$ENV_BACKUP" 2>/dev/null | head -1 | cut -d= -f2- || true)"
[ -n "$_jwt" ] || _jwt="change-me-in-production-use-long-random-secret"
_api_port="$(grep -E '^API_PORT=' "$APP_DIR/.env" | head -1 | cut -d= -f2- || true)"
[ -n "$_api_port" ] || _api_port="$(grep -E '^PORT=' "$APP_DIR/.env" | head -1 | cut -d= -f2- || true)"
[ -n "$_api_port" ] || _api_port="8081"

cat > "$APP_DIR/.env" <<EOF
DB_PASSWORD=${_db_pass}
JWT_SECRET=${_jwt}
API_PORT=${_api_port}
JWT_EXPIRY_HOURS=72
SHIPPING_FEE_LAK=30000
FREE_SHIPPING_MIN_SUBTOTAL_LAK=500000
EOF
printf '%s' "$_db_pass" > "$APP_DIR/.db_password"
chmod 600 "$APP_DIR/.db_password" "$APP_DIR/.env"
echo "==> .env migrated to Docker format (DB_PASSWORD)"

rm -f "$ENV_BACKUP"

echo "==> Disable old systemd API (Docker mode)"
sudo -n systemctl stop lao-rice-api lao-rice-guard.timer 2>/dev/null || true
sudo -n systemctl disable lao-rice-api lao-rice-guard.timer 2>/dev/null || true

echo "==> Backup database (before deploy)"
mkdir -p backups/postgres
if dc ps --status running --services 2>/dev/null | grep -qx db; then
  # sync password in existing volume to match .env (keeps data)
  _escaped="$(printf '%s' "$_db_pass" | sed "s/'/''/g")"
  dc exec -T db psql -U postgres -d lao_rice -c "ALTER USER postgres PASSWORD '${_escaped}';" 2>/dev/null || true
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
