# Dokumentasi nginx-auth-access

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

Jika Anda menjalankan berbagai aplikasi di belakang nginx pada subdomain terpisah — Home Assistant, Nextcloud, panel admin, dashboard — dan tidak ingin pengguna internet sembarangan mengaksesnya, **nginx-auth-access** menempatkan **gerbang dengan pass** bersama di depannya.

Pengguna login dulu di situs khusus (username, password, kode dari aplikasi autentikator), menerima cookie untuk durasi yang dipilih — baru kemudian nginx mengizinkan akses ke layanan target. Tanpa login, nginx mengembalikan redirect ke formulir atau **401**. Verifikasi terintegrasi di nginx via **`auth_request`**: port ringan terpisah hanya memeriksa «apakah ada pass», tanpa UI tambahan. Pengguna dan semua pengaturan ada dalam satu **`config.toml`**.

Satu binary (Angular 21 + Go **`go:embed`**) atau image Docker.

## Daftar isi

1. [Instalasi (systemd)](#instalasi-linux-systemd)
2. [Docker](#docker)
3. [Build dari sumber](#build-dari-sumber)
4. [Port](#port)
5. [Parameter utama](#parameter-utama)
6. [Autentikasi](#autentikasi)
7. [Integrasi Nginx](#integrasi-nginx)
8. [Dokumentasi bahasa lain](#dokumentasi-bahasa-lain)

## Instalasi (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

Skrip mengunduh tarball release dari [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (binary, unit systemd, contoh config), membuat user `nginx-auth-access`, direktori, dan mengaktifkan layanan.

Setelah instalasi:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Opsi: `--version v1.0.0`, `--no-start`. Uninstall: [uninstall.sh](../uninstall.sh) (`--purge` menghapus config dan data).

Config default: **`/etc/nginx-auth-access/config.toml`** (flag `-config` atau **`ACCESS_CONFIG_PATH`**).

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

Di container, config ada di **`/data/config.toml`** (`ACCESS_CONFIG_PATH` diset di image). Contoh: [`config.example.toml`](../config.example.toml).

Compose dengan image GHCR yang dipublikasikan:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

Lihat komentar di [`docker-compose.example.yml`](../docker-compose.example.yml).

## Build dari sumber

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Tanpa `-config`, **`/etc/nginx-auth-access/config.toml`** digunakan.

Pengembangan frontend:

```bash
cd frontend && npm install && npm run start
```

## Port

| Port | Fungsi |
|------|--------|
| **80** | **Halaman login** — yang dilihat pengguna di browser: formulir login, TOTP, durasi sesi; setelah login berhasil cookie «pass» diset di sini. nginx mem-proxy subdomain dari **`panel_hostname`** ke sini dari luar (mis. `access.example.com`). |
| **81** | Hanya **`GET /internal/verify`** — untuk nginx `auth_request`. Tanpa UI. |

## Parameter utama

Contoh: [`config.example.toml`](../config.example.toml).

| Parameter | Deskripsi |
|-----------|-----------|
| `signing_key` | Rahasia HMAC-JWT untuk cookie (**≥ 16 karakter**). |
| `panel_hostname` | Hostname publik panel login (nginx `server_name`), mis. `access.example.com`. |
| `cookie_domain` | Domain cookie induk opsional, mis. `.example.com`. |
| `cookie_secure` | `true` bila semua login hanya via HTTPS. |
| `net_safe_access` | CIDR/IP: cocok → **204** pada verify tanpa cookie. |
| `[listen].public` / `.verify` | Alamat listen HTTP (default `:80` / `:81`). |
| `[[users]]` | Pengguna (bcrypt + TOTP). |

Restart layanan atau container setelah mengedit `config.toml`.

## Autentikasi

Bootstrap dengan daftar pengguna kosong: username/password **`noauth`**, TOTP **`000000`**.

Login normal memerlukan: **username**, **password**, **TOTP**, **periode** (`30m`, `1h`, `3h`, `8h`, `24h`). Pengguna pertama yang dibuat mendapat `admin = true`. Setelah ada entri `[[users]]`, login **`noauth`** dinonaktifkan.

## Integrasi Nginx

1. Proxy **`panel_hostname`** ke port **80** layanan access.
2. Di blok `location` yang dilindungi:

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

3. Arahkan pengguna tidak terautentikasi ke `https://$panel_hostname/`.

## Dokumentasi bahasa lain

| Bahasa | File |
|--------|------|
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

## Lisensi

[MIT](../LICENSE)
