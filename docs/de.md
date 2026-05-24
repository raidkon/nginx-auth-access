# nginx-auth-access Dokumentation

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

Wenn Sie hinter nginx verschiedene Anwendungen auf getrennten Subdomains betreiben — Home Assistant, Nextcloud, Admin-Panels, Dashboards — und nicht möchten, dass beliebige Internetnutzer darauf zugreifen, setzt **nginx-auth-access** eine gemeinsame **Zugangskontrolle mit Pass** davor.

Der Benutzer meldet sich zuerst auf einer dedizierten Seite an (Benutzername, Passwort, Code aus einer Authenticator-App), erhält ein Cookie für die gewählte Dauer — und erst dann erlaubt nginx den Zugriff auf den Zieldienst. Ohne Anmeldung liefert nginx eine Weiterleitung zum Formular oder **401**. Die Prüfung ist in nginx über **`auth_request`** integriert: ein separater leichter Port prüft nur „gibt es einen Pass“, ohne zusätzliche UI. Benutzer und alle Einstellungen liegen in einer **`config.toml`**.

Einzelnes Binary (Angular 21 + Go **`go:embed`**) oder Docker-Image.

## Inhaltsverzeichnis

1. [Installation (systemd)](#installation-linux-systemd)
2. [Docker](#docker)
3. [Aus Quellcode bauen](#aus-quellcode-bauen)
4. [Ports](#ports)
5. [Hauptparameter](#hauptparameter)
6. [Authentifizierung](#authentifizierung)
7. [Nginx-Integration](#nginx-integration)
8. [Dokumentation in anderen Sprachen](#dokumentation-in-anderen-sprachen)

## Installation (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

Das Skript lädt ein Release-Tarball von [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) herunter (Binary, systemd-Unit, Config-Beispiel), legt den Benutzer `nginx-auth-access`, Verzeichnisse an und aktiviert den Dienst.

Nach der Installation:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Optionen: `--version v1.0.0`, `--no-start`. Deinstallation: [uninstall.sh](../uninstall.sh) (`--purge` entfernt Config und Daten).

Standard-Config: **`/etc/nginx-auth-access/config.toml`** (`-config`-Flag oder **`ACCESS_CONFIG_PATH`**).

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

Im Container ist die Config **`/data/config.toml`** (`ACCESS_CONFIG_PATH` ist im Image gesetzt). Beispiel: [`config.example.toml`](../config.example.toml).

Compose mit veröffentlichtem GHCR-Image:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

Siehe Kommentare in [`docker-compose.example.yml`](../docker-compose.example.yml).

## Aus Quellcode bauen

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Ohne `-config` wird **`/etc/nginx-auth-access/config.toml`** verwendet.

Frontend-Entwicklung:

```bash
cd frontend && npm install && npm run start
```

## Ports

| Port | Zweck |
|------|-------|
| **80** | **Anmeldeseite** — was der Benutzer im Browser sieht: Login-Formular, TOTP, Sitzungsdauer; nach erfolgreicher Anmeldung wird hier das „Pass“-Cookie gesetzt. nginx proxyt die Subdomain von **`panel_hostname`** von außen hierher (z. B. `access.example.com`). |
| **81** | Nur **`GET /internal/verify`** — für nginx `auth_request`. Keine UI. |

## Hauptparameter

Beispiel: [`config.example.toml`](../config.example.toml).

| Parameter | Beschreibung |
|-----------|--------------|
| `signing_key` | HMAC-JWT-Geheimnis für Cookie (**≥ 16 Zeichen**). |
| `panel_hostname` | Öffentlicher Hostname des Login-Panels (nginx `server_name`), z. B. `access.example.com`. |
| `cookie_domain` | Optionaler übergeordneter Cookie-Domain, z. B. `.example.com`. |
| `cookie_secure` | `true`, wenn alle Anmeldungen nur über HTTPS erfolgen. |
| `net_safe_access` | CIDR/IP: Treffer → **204** bei verify ohne Cookie. |
| `[listen].public` / `.verify` | HTTP-Listen-Adressen (Standard `:80` / `:81`). |
| `[[users]]` | Benutzer (bcrypt + TOTP). |

Dienst oder Container nach Bearbeitung von `config.toml` neu starten.

## Authentifizierung

Bootstrap mit leerer Benutzerliste: Benutzername/Passwort **`noauth`**, TOTP **`000000`**.

Normaler Login erfordert: **Benutzername**, **Passwort**, **TOTP**, **Zeitraum** (`30m`, `1h`, `3h`, `8h`, `24h`). Der erste angelegte Benutzer erhält `admin = true`. Sobald ein `[[users]]`-Eintrag existiert, ist der Login **`noauth`** deaktiviert.

## Nginx-Integration

1. **`panel_hostname`** auf Port **80** des access-Dienstes proxyen.
2. In geschützten `location`-Blöcken:

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

3. Nicht authentifizierte Benutzer nach `https://$panel_hostname/` weiterleiten.

## Dokumentation in anderen Sprachen

| Sprache | Datei |
|---------|-------|
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

## Lizenz

[MIT](../LICENSE)
