#!/bin/sh
# Общие функции для install.sh / uninstall.sh (запуск из клонированного репозитория).
# install.sh, скачанный через curl, самодостаточен и этот файл не использует.

REPO="${NGINX_AUTH_ACCESS_REPO:-raidkon/nginx-auth-access}"
INSTALL_BIN="${NGINX_AUTH_ACCESS_INSTALL_BIN:-/usr/local/bin/nginx-auth-access}"
CONFIG_DIR="${NGINX_AUTH_ACCESS_CONFIG_DIR:-/etc/nginx-auth-access}"
DATA_DIR="${NGINX_AUTH_ACCESS_DATA_DIR:-/var/lib/nginx-auth-access}"
SERVICE_NAME="${NGINX_AUTH_ACCESS_SERVICE:-nginx-auth-access}"
USER_NAME="${NGINX_AUTH_ACCESS_USER:-nginx-auth-access}"
GROUP_NAME="${NGINX_AUTH_ACCESS_GROUP:-nginx-auth-access}"

log() {
  printf '%s\n' "$*"
}

log_dry() {
  printf '[DRY-RUN] %s\n' "$*"
}

run() {
  if [ "${DRY_RUN:-0}" -eq 1 ]; then
    log_dry "$*"
  else
    "$@"
  fi
}

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    log "Ошибка: нужны права root (sudo)." >&2
    exit 1
  fi
}

detect_arch() {
  machine="$(uname -m)"
  case "$machine" in
    x86_64 | amd64) echo amd64 ;;
    aarch64 | arm64) echo arm64 ;;
    *)
      log "Неподдерживаемая архитектура: $machine (нужен amd64 или arm64)." >&2
      exit 1
      ;;
  esac
}

resolve_tag() {
  version="$1"
  if [ "$version" = "latest" ]; then
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
      | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
      | head -n 1
  else
    printf '%s' "$version"
  fi
}

tag_to_version() {
  printf '%s' "$1" | sed 's/^v//'
}
