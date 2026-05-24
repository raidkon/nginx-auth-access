# nginx-auth-access दस्तावेज़

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

यदि आप nginx के पीछे अलग-अलग सबडोमेन पर विभिन्न ऐप चलाते हैं — Home Assistant, Nextcloud, एडमिन पैनल, डैशबोर्ड — और इंटरनेट के किसी भी अज्ञात उपयोगकर्ता को उन तक पहुँच नहीं चाहते, **nginx-auth-access** उनके सामने एक साझा **पास वाला गेट** लगाता है।

उपयोगकर्ता पहले एक समर्पित साइट पर लॉगिन करता है (उपयोगकर्ता नाम, पासवर्ड, authenticator ऐप से कोड), चुनी गई अवधि के लिए cookie पाता है — और तभी nginx लक्ष्य सेवा तक पहुँच देता है। लॉगिन के बिना nginx फ़ॉर्म पर रीडायरेक्ट या **401** देता है। सत्यापन nginx में **`auth_request`** के ज़रिए अंतर्निहित है: एक अलग हल्का पोर्ट केवल «क्या पास है» जाँचता है, अतिरिक्त UI नहीं। उपयोगकर्ता और सभी सेटिंग्स एक **`config.toml`** में हैं।

एक बाइनरी (Angular 21 + Go **`go:embed`**) या Docker इमेज।

## विषय-सूची

1. [इंस्टॉलेशन (systemd)](#इंस्टॉलेशन-linux-systemd)
2. [Docker](#docker)
3. [स्रोत से बिल्ड](#स्रोत-से-बिल्ड)
4. [पोर्ट](#पोर्ट)
5. [मुख्य पैरामीटर](#मुख्य-पैरामीटर)
6. [प्रमाणीकरण](#प्रमाणीकरण)
7. [Nginx एकीकरण](#nginx-एकीकरण)
8. [अन्य भाषाओं में दस्तावेज़](#अन्य-भाषाओं-में-दस्तावेज़)

## इंस्टॉलेशन (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

स्क्रिप्ट [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) से रिलीज़ tarball डाउनलोड करती है (बाइनरी, systemd unit, config उदाहरण), `nginx-auth-access` उपयोगकर्ता, डायरेक्टरी बनाती है और सेवा सक्षम करती है।

इंस्टॉलेशन के बाद:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

विकल्प: `--version v1.0.0`, `--no-start`. अनइंस्टॉल: [uninstall.sh](../uninstall.sh) (`--purge` config और डेटा हटाता है)।

डिफ़ॉल्ट config: **`/etc/nginx-auth-access/config.toml`** (`-config` फ़्लैग या **`ACCESS_CONFIG_PATH`**)।

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

कंटेनर में config **`/data/config.toml`** है (`ACCESS_CONFIG_PATH` इमेज में सेट)। उदाहरण: [`config.example.toml`](../config.example.toml)।

प्रकाशित GHCR इमेज के साथ Compose:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

[`docker-compose.example.yml`](../docker-compose.example.yml) में टिप्पणियाँ देखें।

## स्रोत से बिल्ड

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

`-config` के बिना **`/etc/nginx-auth-access/config.toml`** उपयोग होता है।

फ़्रंटएंड विकास:

```bash
cd frontend && npm install && npm run start
```

## पोर्ट

| पोर्ट | उद्देश्य |
|-------|---------|
| **80** | **लॉगिन पेज** — ब्राउज़र में उपयोगकर्ता जो देखता है: लॉगिन फ़ॉर्म, TOTP, सत्र अवधि; सफल लॉगिन के बाद यहाँ «पास» cookie सेट होता है। nginx बाहर से **`panel_hostname`** के सबडोमेन को यहाँ proxy करता है (जैसे `access.example.com`)। |
| **81** | केवल **`GET /internal/verify`** — nginx `auth_request` के लिए। UI नहीं। |

## मुख्य पैरामीटर

उदाहरण: [`config.example.toml`](../config.example.toml)।

| पैरामीटर | विवरण |
|----------|--------|
| `signing_key` | cookie के लिए HMAC-JWT secret (**≥ 16 वर्ण**)। |
| `panel_hostname` | लॉगिन पैनल का सार्वजनिक hostname (nginx `server_name`), जैसे `access.example.com`। |
| `cookie_domain` | वैकल्पिक parent cookie domain, जैसे `.example.com`। |
| `cookie_secure` | जब सभी लॉगिन केवल HTTPS से हों तो `true`। |
| `net_safe_access` | CIDR/IP: मिलान → cookie के बिना verify पर **204**। |
| `[listen].public` / `.verify` | HTTP listen पते (डिफ़ॉल्ट `:80` / `:81`)। |
| `[[users]]` | उपयोगकर्ता (bcrypt + TOTP)। |

`config.toml` संपादित करने के बाद सेवा या कंटेनर पुनः आरंभ करें।

## प्रमाणीकरण

खाली उपयोगकर्ता सूची पर bootstrap: username/password **`noauth`**, TOTP **`000000`**।

सामान्य लॉगिन के लिए चाहिए: **username**, **password**, **TOTP**, **अवधि** (`30m`, `1h`, `3h`, `8h`, `24h`)। पहला बनाया उपयोगकर्ता `admin = true` पाता है। कोई भी `[[users]]` प्रविष्टि होने पर **`noauth`** लॉगिन अक्षम हो जाता है।

## Nginx एकीकरण

1. **`panel_hostname`** को access सेवा के **80** पोर्ट पर proxy करें।
2. सुरक्षित `location` ब्लॉकों में:

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

3. अप्रमाणित उपयोगकर्ताओं को `https://$panel_hostname/` पर रीडायरेक्ट करें।

## अन्य भाषाओं में दस्तावेज़

| भाषा | फ़ाइल |
|------|-------|
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

## लाइसेंस

[MIT](../LICENSE)
