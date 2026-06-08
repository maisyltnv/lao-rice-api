#!/bin/bash
# Run ONCE on VPS: bash /opt/lao-rice-api/deploy/fix-vps-once.sh
# Fixes DB password drift + installs auto-guard (every 5 min + on boot).
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
cd "$APP_DIR"

echo "=== Lao Rice API — one-time permanent fix ==="

if [ ! -f .db_password ]; then
  echo "ERROR: $APP_DIR/.db_password missing." >&2
  echo "If API worked before, recover password from .env:" >&2
  echo "  grep DATABASE_URL .env" >&2
  echo "Then: echo -n 'PASSWORD' > .db_password && chmod 600 .db_password" >&2
  exit 1
fi

PASS="$(tr -d '\n' < .db_password)"
[ -n "$PASS" ] || { echo "ERROR: .db_password is empty" >&2; exit 1; }

mkdir -p backups/postgres deploy

# --- ensure-db-login.sh ---
cat > deploy/ensure-db-login.sh << 'SCRIPT'
#!/bin/bash
set -euo pipefail
APP_DIR="/opt/lao-rice-api"
dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$APP_DIR/docker-compose.yml" "$@"
  else
    docker-compose -f "$APP_DIR/docker-compose.yml" "$@"
  fi
}
cd "$APP_DIR"
[ -f .db_password ] || exit 0
DB_PASSWORD="$(tr -d '\n' < .db_password)"
[ -n "$DB_PASSWORD" ] || exit 0
if ! dc ps --status running --services 2>/dev/null | grep -qx db; then
  dc up -d db
fi
for i in $(seq 1 30); do
  dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1 && break
  [ "$i" -eq 30 ] && exit 1
  sleep 1
done
verify() {
  dc exec -T -e PGPASSWORD="$DB_PASSWORD" db psql -h 127.0.0.1 -U postgres -d lao_rice -c 'SELECT 1' >/dev/null 2>&1
}
if ! verify; then
  esc="$(printf '%s' "$DB_PASSWORD" | sed "s/'/''/g")"
  dc exec -T db psql -U postgres -d lao_rice -c "ALTER USER postgres PASSWORD '${esc}';"
  verify || exit 1
fi
# sync .env
if [ -f .env ]; then
  sed -i "s|^DATABASE_URL=.*|DATABASE_URL=postgres://postgres:${DB_PASSWORD}@localhost:5433/lao_rice?sslmode=disable|" .env
fi
PORT=8081
[ -f .env ] && source .env
curl -sf "http://127.0.0.1:${PORT:-8081}/health" >/dev/null && exit 0
sudo systemctl restart lao-rice-api
sleep 3
curl -sf "http://127.0.0.1:${PORT:-8081}/health" >/dev/null || exit 1
SCRIPT
chmod +x deploy/ensure-db-login.sh

# --- cron-backup.sh ---
cat > deploy/cron-backup.sh << 'SCRIPT'
#!/bin/bash
set -euo pipefail
cd /opt/lao-rice-api
mkdir -p backups/postgres
docker compose exec -T db pg_dump -U postgres --no-owner --no-acl lao_rice \
  | gzip > "backups/postgres/lao_rice_$(date +%Y%m%d_%H%M%S).sql.gz"
SCRIPT
chmod +x deploy/cron-backup.sh

# --- systemd: API service (no ExecStartPre) ---
sudo tee /etc/systemd/system/lao-rice-api.service >/dev/null << EOF
[Unit]
Description=Lao Rice API
After=network-online.target docker.service
Wants=network-online.target
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=$APP_DIR
EnvironmentFile=$APP_DIR/.env
ExecStart=$APP_DIR/bin/lao-rice-api
Restart=always
RestartSec=5
StartLimitIntervalSec=0

[Install]
WantedBy=multi-user.target
EOF

# --- systemd: guard timer every 5 min ---
sudo tee /etc/systemd/system/lao-rice-guard.service >/dev/null << EOF
[Unit]
Description=Lao Rice API guard
After=docker.service

[Service]
Type=oneshot
WorkingDirectory=$APP_DIR
ExecStart=/bin/bash $APP_DIR/deploy/ensure-db-login.sh
EOF

sudo tee /etc/systemd/system/lao-rice-guard.timer >/dev/null << 'EOF'
[Unit]
Description=Run Lao Rice API guard every 5 minutes

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min
AccuracySec=1min

[Install]
WantedBy=timers.target
EOF

# --- sync .env now ---
cat > .env << EOF
PORT=8081
DATABASE_URL=postgres://postgres:${PASS}@localhost:5433/lao_rice?sslmode=disable
JWT_SECRET=$(grep '^JWT_SECRET=' .env 2>/dev/null | cut -d= -f2- || echo 'change-me-in-production-use-long-random-secret')
JWT_EXPIRY_HOURS=72
SHIPPING_FEE_LAK=30000
FREE_SHIPPING_MIN_SUBTOTAL_LAK=500000
UPLOAD_DIR=${APP_DIR}/uploads
UPLOAD_URL_PREFIX=/uploads
IMAGES_DIR=${APP_DIR}/images
EOF

echo "==> Start PostgreSQL"
docker compose up -d db
sleep 3

echo "==> Sync DB password"
docker compose exec -T db psql -U postgres -d lao_rice -c "ALTER USER postgres PASSWORD '$PASS';"

echo "==> Enable services"
sudo systemctl daemon-reload
sudo systemctl enable docker lao-rice-api lao-rice-guard.timer
sudo systemctl start lao-rice-guard.timer
sudo systemctl restart lao-rice-api
sleep 3

echo "==> Nightly backup cron"
(crontab -l 2>/dev/null | grep -v 'cron-backup.sh' | grep -v 'cron-health.sh' || true
 echo "0 2 * * * $APP_DIR/deploy/cron-backup.sh >> $APP_DIR/backups/backup.log 2>&1") | crontab -

echo "==> Verify"
bash deploy/ensure-db-login.sh
curl -s "http://127.0.0.1:8081/health"
echo ""
systemctl is-active lao-rice-api
systemctl is-active lao-rice-guard.timer

echo ""
echo "=== DONE ==="
echo "Guard runs every 5 minutes automatically."
echo "You do NOT need to reinstall VPS or redeploy the project."
echo "Keep $APP_DIR/.db_password — never delete it."
