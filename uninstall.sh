#!/bin/sh
# Удаление nginx-auth-access, установленного через install.sh.
#
#   curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/uninstall.sh -o uninstall-nginx-auth-access.sh
#   sudo sh ./uninstall-nginx-auth-access.sh --dry-run
#   sudo sh ./uninstall-nginx-auth-access.sh
#
# Опции:
#   --dry-run    только показать действия
#   --purge      удалить /etc/nginx-auth-access и /var/lib/nginx-auth-access
#   -h, --help   справка

set -eu

INSTALL_BIN="/usr/local/bin/nginx-auth-access"
CONFIG_DIR="/etc/nginx-auth-access"
DATA_DIR="/var/lib/nginx-auth-access"
SERVICE_NAME="nginx-auth-access"
USER_NAME="nginx-auth-access"
GROUP_NAME="nginx-auth-access"

DRY_RUN=0
PURGE=0

log() { printf '%s\n' "$*"; }
log_dry() { printf '[DRY-RUN] %s\n' "$*"; }

run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    log_dry "$*"
  else
    "$@"
  fi
}

usage() {
  sed -n '2,11p' "$0"
  exit 0
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY_RUN=1 ;;
    --purge) PURGE=1 ;;
    -h | --help) usage ;;
    *)
      log "Неизвестный аргумент: $1" >&2
      exit 2
      ;;
  esac
  shift
done

if [ "$(id -u)" -ne 0 ]; then
  log "Ошибка: нужны права root (sudo)." >&2
  exit 1
fi

if command -v systemctl >/dev/null 2>&1; then
  run systemctl stop "$SERVICE_NAME.service" 2>/dev/null || true
  run systemctl disable "$SERVICE_NAME.service" 2>/dev/null || true
  run rm -f "/etc/systemd/system/${SERVICE_NAME}.service" "/lib/systemd/system/${SERVICE_NAME}.service"
  run systemctl daemon-reload
fi

run rm -f "$INSTALL_BIN"

if [ "$PURGE" -eq 1 ]; then
  run rm -rf "$CONFIG_DIR" "$DATA_DIR"
  if getent passwd "$USER_NAME" >/dev/null 2>&1; then
    run userdel --system "$USER_NAME" 2>/dev/null || run userdel "$USER_NAME" 2>/dev/null || true
  fi
  if getent group "$GROUP_NAME" >/dev/null 2>&1; then
    run groupdel "$GROUP_NAME" 2>/dev/null || true
  fi
  log "Удалены конфиг, данные, пользователь $USER_NAME"
else
  log "Конфиг ($CONFIG_DIR) и данные ($DATA_DIR) сохранены. Полное удаление: --purge"
fi

log "nginx-auth-access удалён"
