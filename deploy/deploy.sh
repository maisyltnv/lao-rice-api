#!/bin/bash
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
SERVICE="lao-rice-api"
BIN="$APP_DIR/bin/lao-rice-api"
GITHUB_KEY="${GITHUB_KEY:-$HOME/.ssh/github_lao_rice}"

cd "$APP_DIR"
chmod +x deploy/deploy.sh 2>/dev/null || true
mkdir -p "$APP_DIR/bin"

if [ -f "$GITHUB_KEY" ]; then
  export GIT_SSH_COMMAND="ssh -i $GITHUB_KEY -o StrictHostKeyChecking=accept-new"
fi

echo "==> Pull latest main"
git fetch origin main
git reset --hard origin/main

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

echo "==> Start PostgreSQL"
if ! dc up -d db; then
  echo "Retrying after Docker restart..."
  sudo -n systemctl restart docker || true
  sleep 8
  dc up -d db
fi

for i in $(seq 1 30); do
  if dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "==> Build API"
/usr/local/go/bin/go mod download
/usr/local/go/bin/go build -o "$BIN" ./cmd/server

echo "==> Install systemd unit (if ExecStart path changed)"
sudo -n cp deploy/lao-rice-api.service /etc/systemd/system/lao-rice-api.service
sudo -n systemctl daemon-reload

echo "==> Restart service"
sudo -n systemctl restart "$SERVICE"
sleep 2
sudo -n systemctl is-active --quiet "$SERVICE"

echo "==> Health check"
curl -sf "http://127.0.0.1:${PORT:-8081}/health" >/dev/null
echo "Deploy OK"
