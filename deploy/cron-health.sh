#!/bin/bash
# Lightweight health check every 5 min (for VPS cron). Fixes password drift + restarts API if needed.
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
cd "$APP_DIR"

bash "$APP_DIR/deploy/ensure-db-login.sh"
