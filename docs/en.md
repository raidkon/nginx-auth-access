# nginx-auth-access documentation

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

If you run different apps behind nginx on separate subdomains — Home Assistant, Nextcloud, admin panels, dashboards — and you do not want random internet users reaching them, **nginx-auth-access** puts a shared **gate with a pass** in front of them.

The user first logs in on a dedicated site (username, password, code from an authenticator app), receives a cookie for a chosen duration — and only then nginx allows access to the target service. Without login, nginx returns a redirect to the form or **401**. Verification is built into nginx via **`auth_request`**: a separate lightweight port only checks “is there a pass”, with no extra UI. Users and all settings live in one **`config.toml`**.

Single binary (Angular 21 + Go **`go:embed`**) or Docker image.

## Table of contents

1. [Installation (systemd)](#installation-linux-systemd)
2. [Docker](#docker)
3. [Build from source](#build-from-source)
4. [Ports](#ports)
5. [Main parameters](#main-parameters)
6. [Authentication](#authentication)
7. [Nginx integration](#nginx-integration)
8. [Documentation in other languages](#documentation-in-other-languages)

## Installation (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

The script downloads a release tarball from [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (binary, systemd unit, config example), creates user `nginx-auth-access`, directories, and enables the service.

After install:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Options: `--version v1.0.0`, `--no-start`. Uninstall: [uninstall.sh](../uninstall.sh) (`--purge` removes config and data).

Default config: **`/etc/nginx-auth-access/config.toml`** (`-config` flag or **`ACCESS_CONFIG_PATH`**).

## Docker

```bash
git clone https://github.com/raidkon/nginx-auth-access.git
cd nginx-auth-access
cp config.example.toml store/config.toml
# edit signing_key, panel_hostname, net_safe_access, cookie_*

docker build -f docker/Dockerfile -t nginx-auth-access:local .
docker run -d --name access \
  -v "$(pwd)/store:/data" \
  --cap-add=NET_BIND_SERVICE \
  nginx-auth-access:local
```

In the container, config is **`/data/config.toml`** (`ACCESS_CONFIG_PATH` is set in the image). Example: [`config.example.toml`](../config.example.toml).

Compose with published GHCR image:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

See comments in [`docker-compose.example.yml`](../docker-compose.example.yml).

## Build from source

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Without `-config`, **`/etc/nginx-auth-access/config.toml`** is used.

Frontend development:

```bash
cd frontend && npm install && npm run start
```

## Ports

| Port | Purpose |
|------|---------|
| **80** | **Login page** — what the user sees in the browser: login form, TOTP, session duration; after successful login the “pass” cookie is set here. nginx proxies the subdomain from **`panel_hostname`** here from outside (e.g. `access.example.com`). |
| **81** | **`GET /internal/verify`** only — for nginx `auth_request`. No UI. |

## Main parameters

Example: [`config.example.toml`](../config.example.toml).

| Parameter | Description |
|-----------|-------------|
| `signing_key` | HMAC-JWT secret for cookie (**≥ 16 characters**). |
| `panel_hostname` | Public hostname of the login panel (nginx `server_name`), e.g. `access.example.com`. |
| `cookie_domain` | Optional parent cookie domain, e.g. `.example.com`. |
| `cookie_secure` | `true` when all logins use HTTPS only. |
| `net_safe_access` | CIDR/IP: match → **204** on verify without cookie. |
| `[listen].public` / `.verify` | HTTP listen addresses (default `:80` / `:81`). |
| `[[users]]` | Users (bcrypt + TOTP). |

Restart the service or container after editing `config.toml`.

## Authentication

Bootstrap with an empty user list: username/password **`noauth`**, TOTP **`000000`**.

Normal login requires: **username**, **password**, **TOTP**, **period** (`30m`, `1h`, `3h`, `8h`, `24h`). The first created user gets `admin = true`. Once any `[[users]]` entry exists, **`noauth`** login is disabled.

## Nginx integration

1. Proxy **`panel_hostname`** to port **80** of the access service.
2. In protected `location` blocks:

```nginx
auth_request /internal/verify;
auth_request_set $auth_status $upstream_status;
error_page 401 = @access_login;

location = /internal/verify {
    internal;
    proxy_pass http://nginx-auth-access:81/internal/verify;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Cookie $http_cookie;
}
```

3. Redirect unauthenticated users to `https://$panel_hostname/`.

## Documentation in other languages

| Language | File |
|----------|------|
| Русский | [ru.md](ru.md) |
| English | [en.md](en.md) |
| 中文 | [zh.md](zh.md) |
| हिन्दी | [hi.md](hi.md) |
| Español | [es.md](es.md) |
| Français | [fr.md](fr.md) |
| العربية | [ar.md](ar.md) |
| বাংলা | [bn.md](bn.md) |
| Português | [pt.md](pt.md) |
| اردو | [ur.md](ur.md) |
| Bahasa Indonesia | [id.md](id.md) |
| Deutsch | [de.md](de.md) |
| 日本語 | [ja.md](ja.md) |
| Türkçe | [tr.md](tr.md) |
| 한국어 | [ko.md](ko.md) |

## License

[MIT](../LICENSE)
