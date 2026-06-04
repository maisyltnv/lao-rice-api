# Shared VPS helpers (source from deploy/*.sh — do not run directly).
# shellcheck shell=bash

vps_systemctl() {
  if command -v systemctl >/dev/null 2>&1; then
    command systemctl "$@"
  elif [ -x /usr/bin/systemctl ]; then
    /usr/bin/systemctl "$@"
  else
    echo "ERROR: systemctl not found (try: ls -la /usr/bin/systemctl)"
    return 127
  fi
}

vps_sudo_systemctl() {
  local sc
  sc="$(command -v systemctl 2>/dev/null || true)"
  [ -n "$sc" ] || sc="/usr/bin/systemctl"
  sudo -n "$sc" "$@"
}

vps_ensure_docker() {
  if docker info >/dev/null 2>&1; then
    return 0
  fi
  echo "==> Docker daemon not running"
  vps_sudo_systemctl reset-failed docker.service 2>/dev/null || true
  if [ ! -x /usr/bin/dockerd ]; then
    echo "==> dockerd missing — installing Docker CE (not docker.io)"
    curl -fsSL https://get.docker.com | sudo sh
    vps_sudo_systemctl enable docker containerd 2>/dev/null || true
  fi
  vps_sudo_systemctl start docker || true
  sleep 5
  if ! docker info >/dev/null 2>&1; then
    echo "ERROR: Docker still not running. On VPS run:"
    echo "  sudo bash /opt/lao-edu-api/deploy/vps-recover-all.sh"
    return 1
  fi
}

vps_dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}
