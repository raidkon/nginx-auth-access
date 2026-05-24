# nginx-auth-access ডকুমেন্টেশন

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

যদি আপনি nginx-এর পেছনে আলাদা সাবডোমেনে বিভিন্ন অ্যাপ চালান — Home Assistant, Nextcloud, অ্যাডমিন প্যানেল, ড্যাশবোর্ড — এবং ইন্টারনেটের যেকোনো অজানা ব্যবহারকারী তাদের অ্যাক্সেস করুক তা চান না, **nginx-auth-access** তাদের সামনে একটি ভাগ করা **পাসসহ গেট** বসায়।

ব্যবহারকারী প্রথমে একটি নির্দিষ্ট সাইটে লগইন করে (ব্যবহারকারীর নাম, পাসওয়ার্ড, authenticator অ্যাপ থেকে কোড), নির্বাচিত সময়ের জন্য cookie পায় — এবং তখনই nginx লক্ষ্য সেবায় অ্যাক্সেস দেয়। লগইন ছাড়া nginx ফর্মে রিডাইরেক্ট বা **401** দেয়। যাচাই nginx-এ **`auth_request`** দিয়ে অন্তর্নির্মিত: একটি আলাদা হালকা পোর্ট শুধু «পাস আছে কি না» দেখে, অতিরিক্ত UI নেই। ব্যবহারকারী ও সব সেটিংস একটি **`config.toml`**-এ।

একটি বাইনারি (Angular 21 + Go **`go:embed`**) বা Docker ইমেজ।

## সূচিপত্র

1. [ইনস্টলেশন (systemd)](#ইনস্টলেশন-linux-systemd)
2. [Docker](#docker)
3. [সোর্স থেকে বিল্ড](#সোর্স-থেকে-বিল্ড)
4. [পোর্ট](#পোর্ট)
5. [প্রধান প্যারামিটার](#প্রধান-প্যারামিটার)
6. [অথেন্টিকেশন](#অথেন্টিকেশন)
7. [Nginx ইন্টিগ্রেশন](#nginx-ইন্টিগ্রেশন)
8. [অন্যান্য ভাষায় ডকুমেন্টেশন](#অন্যান্য-ভাষায়-ডকুমেন্টেশন)

## ইনস্টলেশন (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

স্ক্রিপ্ট [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) থেকে রিলিজ tarball ডাউনলোড করে (বাইনারি, systemd unit, config উদাহরণ), `nginx-auth-access` ব্যবহারকারী, ডিরেক্টরি তৈরি করে এবং সেবা সক্রিয় করে।

ইনস্টলের পর:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

অপশন: `--version v1.0.0`, `--no-start`। আনইনস্টল: [uninstall.sh](../uninstall.sh) (`--purge` config ও ডেটা মুছে)।

ডিফল্ট config: **`/etc/nginx-auth-access/config.toml`** (`-config` ফ্ল্যাগ বা **`ACCESS_CONFIG_PATH`**)।

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

কনটেইনারে config **`/data/config.toml`** (`ACCESS_CONFIG_PATH` ইমেজে সেট)। উদাহরণ: [`config.example.toml`](../config.example.toml)।

প্রকাশিত GHCR ইমেজ দিয়ে Compose:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

[`docker-compose.example.yml`](../docker-compose.example.yml)-এ মন্তব্য দেখুন।

## সোর্স থেকে বিল্ড

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

`-config` ছাড়া **`/etc/nginx-auth-access/config.toml`** ব্যবহৃত হয়।

ফ্রন্টএন্ড ডেভেলপমেন্ট:

```bash
cd frontend && npm install && npm run start
```

## পোর্ট

| পোর্ট | উদ্দেশ্য |
|-------|---------|
| **80** | **লগইন পেজ** — ব্রাউজারে ব্যবহারকারী যা দেখে: লগইন ফর্ম, TOTP, সেশনের সময়কাল; সফল লগইনের পর এখানেই «পাস» cookie সেট হয়। nginx বাইরে থেকে **`panel_hostname`**-এর সাবডোমেন এখানে proxy করে (যেমন `access.example.com`)। |
| **81** | শুধু **`GET /internal/verify`** — nginx `auth_request`-এর জন্য। UI নেই। |

## প্রধান প্যারামিটার

উদাহরণ: [`config.example.toml`](../config.example.toml)।

| প্যারামিটার | বর্ণনা |
|-------------|--------|
| `signing_key` | cookie-এর HMAC-JWT secret (**≥ 16 অক্ষর**)। |
| `panel_hostname` | লগইন প্যানেলের পাবলিক hostname (nginx `server_name`), যেমন `access.example.com`। |
| `cookie_domain` | ঐচ্ছিক parent cookie domain, যেমন `.example.com`। |
| `cookie_secure` | সব লগইন শুধু HTTPS হলে `true`। |
| `net_safe_access` | CIDR/IP: মিল → cookie ছাড়া verify-তে **204**। |
| `[listen].public` / `.verify` | HTTP listen ঠিকানা (ডিফল্ট `:80` / `:81`)। |
| `[[users]]` | ব্যবহারকারী (bcrypt + TOTP)। |

`config.toml` সম্পাদনার পর সেবা বা কনটেইনার পুনরায় চালু করুন।

## অথেন্টিকেশন

খালি ব্যবহারকারী তালিকায় bootstrap: username/password **`noauth`**, TOTP **`000000`**।

সাধারণ লগইনের জন্য দরকার: **username**, **password**, **TOTP**, **সময়কাল** (`30m`, `1h`, `3h`, `8h`, `24h`)। প্রথম তৈরি ব্যবহারকারী `admin = true` পায়। যেকোনো `[[users]]` এন্ট্রি থাকলে **`noauth`** লগইন নিষ্ক্রিয় হয়।

## Nginx ইন্টিগ্রেশন

1. **`panel_hostname`** access সেবার **80** পোর্টে proxy করুন।
2. সুরক্ষিত `location` ব্লকে:

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

3. অপ্রমাণিত ব্যবহারকারীদের `https://$panel_hostname/`-এ রিডাইরেক্ট করুন।

## অন্যান্য ভাষায় ডকুমেন্টেশন

| ভাষা | ফাইল |
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

## লাইসেন্স

[MIT](../LICENSE)
