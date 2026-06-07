#!/bin/bash
# One-time setup on VPS: auto-guard API + nightly backup cron.
# Run: cd /opt/lao-rice-api && bash deploy/install-guards.sh
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
cd "$APP_DIR"

if [ ! -f .db_password ]; then
  echo "ERROR: $APP_DIR/.db_password not found." >&2
  echo "Create it first: echo -n 'YOUR_PASSWORD' > .db_password && chmod 600 .db_password" >&2
  exit 1
fi

mkdir -p backups/postgres
chmod +x deploy/ensure-db-login.sh deploy/cron-backup.sh deploy/cron-health.sh deploy/backup-db.sh 2>/dev/null || true

PASS="$(tr -d '\n' < .db_password)"
if [ -f .env ]; then
  if grep -q '^DATABASE_URL=' .env; then
    sed -i "s|^DATABASE_URL=.*|DATABASE_URL=postgres://postgres:${PASS}@localhost:5433/lao_rice?sslmode=disable|" .env
  fi
fi

echo "==> Start PostgreSQL"
docker compose up -d db

echo "==> Install systemd units"
sudo cp deploy/lao-rice-api.service /etc/systemd/system/lao-rice-api.service
sudo cp deploy/lao-rice-guard.service /etc/systemd/system/lao-rice-guard.service
sudo cp deploy/lao-rice-guard.timer /etc/systemd/system/lao-rice-guard.timer
sudo systemctl daemon-reload
sudo systemctl enable docker lao-rice-api lao-rice-guard.timer
sudo systemctl start lao-rice-guard.timer

echo "==> Nightly backup cron (02:00)"
(crontab -l 2>/dev/null | grep -v 'deploy/cron-backup.sh' | grep -v 'deploy/cron-health.sh' || true
 echo "0 2 * * * $APP_DIR/deploy/cron-backup.sh >> $APP_DIR/backups/backup.log 2>&1") | crontab -

echo "==> Run guard now"
bash "$APP_DIR/deploy/ensure-db-login.sh"

echo ""
echo "Guards installed."
echo "  - Timer: every 5 min (systemctl status lao-rice-guard.timer)"
echo "  - Backup: nightly 02:00"
echo "  - Boot: docker + API enabled"
systemctl is-active lao-rice-api 2>/dev/null || sudo systemctl is-active lao-rice-api || true
curl -sf "http://127.0.0.1:${PORT:-8081}/health" && echo ""
