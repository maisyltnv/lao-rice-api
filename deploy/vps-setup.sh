#!/bin/bash
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
REPO="git@github.com:maisyltnv/lao-rice-api.git"
if [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
  _DEPLOY_HOME="$(getent passwd "$SUDO_USER" | cut -d: -f6)"
else
  _DEPLOY_HOME="${HOME}"
fi
GITHUB_KEY="${GITHUB_KEY:-${_DEPLOY_HOME}/.ssh/github_lao_rice}"
DB_PASSWORD="${DB_PASSWORD:-$(openssl rand -hex 16)}"
JWT_SECRET="${JWT_SECRET:-$(openssl rand -hex 32)}"
API_PORT="${PORT:-8081}"

setup_git_ssh() {
  if [ ! -f "$GITHUB_KEY" ]; then
    echo ""
    echo "ERROR: Deploy key not found: $GITHUB_KEY"
    echo ""
    echo "Run on VPS (as deploy user):"
    echo "  ssh-keygen -t ed25519 -f $GITHUB_KEY -N \"\""
    echo "  cat ${GITHUB_KEY}.pub"
    echo ""
    echo "Add the public key at:"
    echo "  https://github.com/maisyltnv/lao-rice-api/settings/keys"
    echo ""
    exit 1
  fi
  export GIT_SSH_COMMAND="ssh -i $GITHUB_KEY -o StrictHostKeyChecking=accept-new"
}

echo "==> Install packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y git curl ufw
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

if ! command -v go >/dev/null 2>&1 || ! go version | grep -q 'go1.2'; then
  echo "==> Install Go"
  GO_VER="1.23.4"
  curl -fsSL "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -o /tmp/go.tgz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tgz
  echo 'export PATH=$PATH:/usr/local/go/bin' >/etc/profile.d/golang.sh
fi
export PATH="$PATH:/usr/local/go/bin"

echo "==> Clone or update repo"
setup_git_ssh
mkdir -p /opt
if [ ! -d "$APP_DIR/.git" ]; then
  git clone -b main "$REPO" "$APP_DIR"
else
  cd "$APP_DIR"
  git fetch origin main
  git reset --hard origin/main
fi
cd "$APP_DIR"

DB_PASS_FILE="$APP_DIR/.db_password"
if [ -f "$DB_PASS_FILE" ]; then
  DB_PASSWORD="$(cat "$DB_PASS_FILE")"
elif [ -f "$APP_DIR/.env" ]; then
  DB_PASSWORD="$(grep '^DATABASE_URL=' "$APP_DIR/.env" | sed -n 's|.*postgres:\([^@]*\)@.*|\1|p')"
fi
printf '%s' "$DB_PASSWORD" > "$DB_PASS_FILE"
chmod 600 "$DB_PASS_FILE"

echo "==> Configure PostgreSQL password in docker-compose"
sed -i "s/POSTGRES_PASSWORD: .*/POSTGRES_PASSWORD: ${DB_PASSWORD}/" docker-compose.yml

echo "==> Create .env"
cat > .env <<EOF
PORT=${API_PORT}
DATABASE_URL=postgres://postgres:${DB_PASSWORD}@localhost:5433/lao_rice?sslmode=disable
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRY_HOURS=72
SHIPPING_FEE_LAK=30000
FREE_SHIPPING_MIN_SUBTOTAL_LAK=500000
UPLOAD_DIR=${APP_DIR}/uploads
UPLOAD_URL_PREFIX=/uploads
EOF

mkdir -p uploads/payment-receipts
chmod +x deploy/deploy.sh

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

echo "==> Start database only"
dc down 2>/dev/null || true
docker rm -f lao-rice-db 2>/dev/null || true
dc up -d db
for i in $(seq 1 30); do
  if dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "==> Verify database login"
if ! dc exec -T -e PGPASSWORD="$DB_PASSWORD" db psql -U postgres -d lao_rice -c 'SELECT 1' >/dev/null 2>&1; then
  echo "Database password mismatch. Recreating database volume..."
  dc down -v
  dc up -d db
  for i in $(seq 1 30); do
    if dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
fi

echo "==> Seed sample data (first run only)"
if ! dc exec -T -e PGPASSWORD="$DB_PASSWORD" db psql -U postgres -d lao_rice -tAc "SELECT 1 FROM categories LIMIT 1" 2>/dev/null | grep -q 1; then
  go mod download
  set -a
  # shellcheck disable=SC1091
  source "$APP_DIR/.env"
  set +a
  go run ./cmd/seed
fi

mkdir -p "$APP_DIR/bin"
echo "==> Build binary"
go mod download
go build -o "$APP_DIR/bin/lao-rice-api" ./cmd/server

DEPLOY_USER="${SUDO_USER:-deploy}"
if [ "$DEPLOY_USER" = "root" ]; then DEPLOY_USER="deploy"; fi
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$APP_DIR"

echo "==> Install systemd service"
cp deploy/lao-rice-api.service /etc/systemd/system/lao-rice-api.service
systemctl daemon-reload
systemctl enable lao-rice-api
systemctl restart lao-rice-api

echo "==> Firewall"
ufw allow OpenSSH || true
ufw allow "${API_PORT}/tcp" || true
ufw --force enable || true

echo "==> Health check"
sleep 2
curl -sf "http://127.0.0.1:${API_PORT}/health"
echo ""
PUBLIC_IP="$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')"
echo "Setup complete."
echo "API: http://${PUBLIC_IP}:${API_PORT}"
echo "Health: http://${PUBLIC_IP}:${API_PORT}/health"
echo ""
echo "Create admin (once):"
echo "  curl -X POST http://127.0.0.1:${API_PORT}/auth/admin/register \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"username\":\"admin\",\"password\":\"YOUR_STRONG_PASSWORD\"}'"
