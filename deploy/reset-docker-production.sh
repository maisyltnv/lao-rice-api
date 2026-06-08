#!/bin/bash
# ONE-TIME clean reset: API + DB both in Docker (no systemd password drift).
# WARNING: deletes database volume — all data is wiped.
# Run on VPS: cd /opt/lao-rice-api && bash deploy/reset-docker-production.sh
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
cd "$APP_DIR"

echo "=============================================="
echo " Lao Rice — Docker production reset"
echo " WARNING: This deletes ALL database data."
echo "=============================================="
printf "Type YES to continue: "
read -r confirm
[ "$confirm" = "YES" ] || { echo "Aborted."; exit 1; }

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

JWT_SECRET=""
if [ -f .env ]; then
  JWT_SECRET="$(grep -E '^JWT_SECRET=' .env | head -1 | cut -d= -f2- || true)"
fi
[ -n "$JWT_SECRET" ] || JWT_SECRET="$(openssl rand -hex 32)"
DB_PASSWORD="$(openssl rand -hex 16)"

echo "==> Stop old systemd API (no longer used)"
sudo systemctl stop lao-rice-api lao-rice-guard.timer 2>/dev/null || true
sudo systemctl disable lao-rice-api lao-rice-guard.timer 2>/dev/null || true

echo "==> Remove old containers"
docker rm -f lao-rice-api 2>/dev/null || true
dc down -v

mkdir -p uploads/payment-receipts uploads/product-images uploads/banner-images images/rice

echo "==> Write .env (single source of truth for Docker)"
cat > .env <<EOF
DB_PASSWORD=${DB_PASSWORD}
JWT_SECRET=${JWT_SECRET}
API_PORT=8081
JWT_EXPIRY_HOURS=72
SHIPPING_FEE_LAK=30000
FREE_SHIPPING_MIN_SUBTOTAL_LAK=500000
EOF
chmod 600 .env
rm -f .db_password

echo "==> Build and start API + DB in Docker"
dc up -d --build

echo "==> Wait for API healthy"
for i in $(seq 1 60); do
  if curl -sf "http://127.0.0.1:8081/health" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo "==> Seed sample data"
export DATABASE_URL="postgres://postgres:${DB_PASSWORD}@127.0.0.1:5433/lao_rice?sslmode=disable"
if command -v go >/dev/null 2>&1; then
  /usr/local/go/bin/go run ./cmd/seed 2>/dev/null || go run ./cmd/seed
else
  echo "WARN: Go not found — skip seed. Add products/categories in admin."
fi

echo "==> Nightly backup cron"
cat > deploy/cron-backup.sh << 'SCRIPT'
#!/bin/bash
set -euo pipefail
cd /opt/lao-rice-api
mkdir -p backups/postgres
docker compose exec -T db pg_dump -U postgres --no-owner --no-acl lao_rice \
  | gzip > "backups/postgres/lao_rice_$(date +%Y%m%d_%H%M%S).sql.gz"
SCRIPT
chmod +x deploy/cron-backup.sh
(crontab -l 2>/dev/null | grep -v 'cron-backup.sh' | grep -v 'cron-health.sh' | grep -v 'ensure-db-login' || true
 echo "0 2 * * * $APP_DIR/deploy/cron-backup.sh >> $APP_DIR/backups/backup.log 2>&1") | crontab -

echo ""
echo "=== RESET COMPLETE ==="
curl -s "http://127.0.0.1:8081/health"
echo ""
echo "API:  http://$(curl -s ifconfig.me 2>/dev/null || echo YOUR_IP):8081"
echo "Web:  http://$(curl -s ifconfig.me 2>/dev/null || echo YOUR_IP):3000"
echo ""
echo "Create admin (once):"
echo "  curl -X POST http://127.0.0.1:8081/auth/admin/register \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"YOUR_STRONG_PASSWORD\"}'"
echo ""
echo "Password is in $APP_DIR/.env (DB_PASSWORD=...) — keep this file safe."
