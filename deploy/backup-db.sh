#!/bin/bash
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
# shellcheck disable=SC1091
source "$APP_DIR/deploy/db-tools.sh"

cd "$APP_DIR"
ensure_db_password
wait_for_postgres 30
ensure_db_login
backup_database

echo "Backup complete."
