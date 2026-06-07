#!/bin/bash
# Nightly backup + DB password sync + API health check (for VPS cron at 02:00).
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
cd "$APP_DIR"

bash "$APP_DIR/deploy/backup-db.sh"
bash "$APP_DIR/deploy/ensure-db-login.sh"

echo "cron-backup done $(date -Iseconds)"
