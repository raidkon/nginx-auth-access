# توثيق nginx-auth-access

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

إذا كنت تشغّل تطبيقات مختلفة خلف nginx على نطاقات فرعية منفصلة — Home Assistant وNextcloud ولوحات الإدارة ولوحات المعلومات — ولا تريد أن يصل إليها مستخدمون عشوائيون من الإنترنت، فإن **nginx-auth-access** يضع أمامها **بوابة مشتركة بتصريح مرور**.

يسجّل المستخدم الدخول أولاً على موقع مخصّص (اسم المستخدم، كلمة المرور، رمز من تطبيق المصادقة)، ويحصل على cookie للمدة المختارة — وبعدها فقط يسمح nginx بالوصول إلى الخدمة المستهدفة. بدون تسجيل الدخول، يعيد nginx توجيهاً إلى النموذج أو **401**. التحقق مدمج في nginx عبر **`auth_request`**: منفذ خفيف منفصل يتحقق فقط من «هل يوجد تصريح مرور»، دون واجهة إضافية. المستخدمون وجميع الإعدادات في **`config.toml`** واحد.

ملف ثنائي واحد (Angular 21 + Go **`go:embed`**) أو صورة Docker.

## جدول المحتويات

1. [التثبيت (systemd)](#التثبيت-linux-systemd)
2. [Docker](#docker)
3. [البناء من المصدر](#البناء-من-المصدر)
4. [المنافذ](#المنافذ)
5. [المعاملات الرئيسية](#المعاملات-الرئيسية)
6. [المصادقة](#المصادقة)
7. [تكامل Nginx](#تكامل-nginx)
8. [التوثيق بلغات أخرى](#التوثيق-بلغات-أخرى)

## التثبيت (Linux، systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

يحمّل السكربت حزمة إصدار من [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (الملف الثنائي، وحدة systemd، مثال الإعداد)، وينشئ المستخدم `nginx-auth-access` والمجلدات ويُفعّل الخدمة.

بعد التثبيت:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

الخيارات: `--version v1.0.0`، `--no-start`. إلغاء التثبيت: [uninstall.sh](../uninstall.sh) (`--purge` يزيل الإعداد والبيانات).

الإعداد الافتراضي: **`/etc/nginx-auth-access/config.toml`** (علامة `-config` أو **`ACCESS_CONFIG_PATH`**).

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

في الحاوية، الإعداد في **`/data/config.toml`** (`ACCESS_CONFIG_PATH` مضبوط في الصورة). مثال: [`config.example.toml`](../config.example.toml).

Compose مع صورة GHCR المنشورة:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

راجع التعليقات في [`docker-compose.example.yml`](../docker-compose.example.yml).

## البناء من المصدر

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

بدون `-config`، يُستخدم **`/etc/nginx-auth-access/config.toml`**.

تطوير الواجهة الأمامية:

```bash
cd frontend && npm install && npm run start
```

## المنافذ

| المنفذ | الغرض |
|--------|-------|
| **80** | **صفحة تسجيل الدخول** — ما يراه المستخدم في المتصفح: نموذج الدخول، TOTP، مدة الجلسة؛ بعد نجاح تسجيل الدخول يُضبط cookie «تصريح المرور» هنا. يوجّه nginx النطاق الفرعي من **`panel_hostname`** إلى هنا من الخارج (مثل `access.example.com`). |
| **81** | **`GET /internal/verify`** فقط — لـ nginx `auth_request`. بدون واجهة. |

## المعاملات الرئيسية

مثال: [`config.example.toml`](../config.example.toml).

| المعامل | الوصف |
|---------|-------|
| `signing_key` | سر HMAC-JWT للـ cookie (**≥ 16 حرفاً**). |
| `panel_hostname` | اسم المضيف العام للوحة الدخول (nginx `server_name`)، مثل `access.example.com`. |
| `cookie_domain` | نطاق cookie الأب اختياري، مثل `.example.com`. |
| `cookie_secure` | `true` عندما تكون جميع عمليات الدخول عبر HTTPS فقط. |
| `net_safe_access` | CIDR/IP: تطابق → **204** على verify بدون cookie. |
| `[listen].public` / `.verify` | عناوين الاستماع HTTP (افتراضي `:80` / `:81`). |
| `[[users]]` | المستخدمون (bcrypt + TOTP). |

أعد تشغيل الخدمة أو الحاوية بعد تعديل `config.toml`.

## المصادقة

التهيئة الأولية بقائمة مستخدمين فارغة: اسم المستخدم/كلمة المرور **`noauth`**، TOTP **`000000`**.

يتطلب تسجيل الدخول العادي: **اسم المستخدم**، **كلمة المرور**، **TOTP**، **المدة** (`30m`، `1h`، `3h`، `8h`، `24h`). أول مستخدم يُنشأ يحصل على `admin = true`. بمجرد وجود أي إدخال `[[users]]`، يُعطّل دخول **`noauth`**.

## تكامل Nginx

1. وجّه **`panel_hostname`** إلى المنفذ **80** لخدمة access.
2. في كتل `location` المحمية:

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

3. وجّه المستخدمين غير المصادق عليهم إلى `https://$panel_hostname/`.

## التوثيق بلغات أخرى

| اللغة | الملف |
|-------|-------|
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

## الترخيص

[MIT](../LICENSE)
