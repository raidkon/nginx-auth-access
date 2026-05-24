# nginx-auth-access دستاویز

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

اگر آپ nginx کے پیچھے الگ الگ سب ڈومینز پر مختلف ایپس چلاتے ہیں — Home Assistant، Nextcloud، ایڈمن پینل، ڈیش بورڈ — اور انٹرنیٹ کے بے ترتیب صارفین تک رسائی نہیں چاہتے، **nginx-auth-access** ان کے سامنے مشترکہ **پاس والا گیٹ** لگاتا ہے۔

صارف پہلے مخصوص سائٹ پر لاگ ان کرتا ہے (username، password، authenticator ایپ سے کوڈ)، منتخب مدت کے لیے cookie حاصل کرتا ہے — اور تب ہی nginx ہدف سروس تک رسائی دیتا ہے۔ لاگ ان کے بغیر nginx فارم پر redirect یا **401** دیتا ہے۔ تصدیق nginx میں **`auth_request`** کے ذریعے built-in ہے: الگ ہلکا پورٹ صرف «کیا پاس ہے» چیک کرتا ہے، اضافی UI نہیں۔ صارفین اور تمام settings ایک **`config.toml`** میں ہیں۔

ایک binary (Angular 21 + Go **`go:embed`**) یا Docker image۔

## فہرست مضامین

1. [انسٹالیشن (systemd)](#انسٹالیشن-linux-systemd)
2. [Docker](#docker)
3. [سورس سے build](#سورس-سے-build)
4. [پورٹس](#پورٹس)
5. [اہم parameters](#اہم-parameters)
6. [Authentication](#authentication)
7. [Nginx integration](#nginx-integration)
8. [دیگر زبانوں میں دستاویز](#دیگر-زبانوں-میں-دستاویز)

## انسٹالیشن (Linux، systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

اسکرپٹ [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) سے release tarball ڈاؤنلوڈ کرتا ہے (binary، systemd unit، config example)، `nginx-auth-access` user، directories بناتا ہے اور service enable کرتا ہے۔

انسٹال کے بعد:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Options: `--version v1.0.0`، `--no-start`۔ Uninstall: [uninstall.sh](../uninstall.sh) (`--purge` config اور data ہٹاتا ہے)۔

Default config: **`/etc/nginx-auth-access/config.toml`** (`-config` flag یا **`ACCESS_CONFIG_PATH`**)۔

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

Container میں config **`/data/config.toml`** ہے (`ACCESS_CONFIG_PATH` image میں set)۔ Example: [`config.example.toml`](../config.example.toml)۔

Published GHCR image کے ساتھ Compose:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

[`docker-compose.example.yml`](../docker-compose.example.yml) میں comments دیکھیں۔

## سورس سے build

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

`-config` کے بغیر **`/etc/nginx-auth-access/config.toml`** استعمال ہوتا ہے۔

Frontend development:

```bash
cd frontend && npm install && npm run start
```

## پورٹس

| پورٹ | مقصد |
|------|------|
| **80** | **Login page** — browser میں صارف جو دیکھتا ہے: login form، TOTP، session duration؛ کامیاب login کے بعد یہاں «pass» cookie set ہوتا ہے۔ nginx باہر سے **`panel_hostname`** کا subdomain یہاں proxy کرتا ہے (مثلاً `access.example.com`)۔ |
| **81** | صرف **`GET /internal/verify`** — nginx `auth_request` کے لیے۔ UI نہیں۔ |

## اہم parameters

Example: [`config.example.toml`](../config.example.toml)۔

| Parameter | Description |
|-----------|-------------|
| `signing_key` | cookie کے لیے HMAC-JWT secret (**≥ 16 characters**)۔ |
| `panel_hostname` | login panel کا public hostname (nginx `server_name`)، مثلاً `access.example.com`۔ |
| `cookie_domain` | اختیاری parent cookie domain، مثلاً `.example.com`۔ |
| `cookie_secure` | جب تمام logins صرف HTTPS ہوں تو `true`۔ |
| `net_safe_access` | CIDR/IP: match → cookie کے بغیر verify پر **204**۔ |
| `[listen].public` / `.verify` | HTTP listen addresses (default `:80` / `:81`)۔ |
| `[[users]]` | Users (bcrypt + TOTP)۔ |

`config.toml` edit کرنے کے بعد service یا container restart کریں۔

## Authentication

خالی user list پر bootstrap: username/password **`noauth`**، TOTP **`000000`**۔

Normal login کے لیے درکار: **username**، **password**، **TOTP**، **period** (`30m`، `1h`، `3h`، `8h`، `24h`)۔ پہلا بنایا user `admin = true` پاتا ہے۔ کوئی بھی `[[users]]` entry موجود ہو تو **`noauth`** login disable ہو جاتا ہے۔

## Nginx integration

1. **`panel_hostname`** کو access service کے **80** port پر proxy کریں۔
2. Protected `location` blocks میں:

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

3. Unauthenticated users کو `https://$panel_hostname/` پر redirect کریں۔

## دیگر زبانوں میں دستاویز

| زبان | فائل |
|------|------|
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
