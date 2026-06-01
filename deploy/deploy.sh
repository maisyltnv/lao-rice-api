#!/bin/bash
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
SERVICE="lao-rice-api"
GITHUB_KEY="${GITHUB_KEY:-$HOME/.ssh/github_lao_rice}"

cd "$APP_DIR"

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
dc down 2>/dev/null || true
docker rm -f lao-rice-db 2>/dev/null || true
if ! dc up -d db; then
  echo "Retrying after Docker restart..."
  systemctl restart docker || true
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
/usr/local/go/bin/go build -o /usr/local/bin/lao-rice-api ./cmd/server

echo "==> Restart service"
systemctl restart "$SERVICE"
sleep 2
systemctl is-active --quiet "$SERVICE"

echo "==> Health check"
curl -sf "http://127.0.0.1:${PORT:-8081}/health" >/dev/null
echo "Deploy OK"
