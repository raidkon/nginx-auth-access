# nginx-auth-access belgeleri

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

nginx arkasında Home Assistant, Nextcloud, yönetim panelleri, panolar gibi farklı alt alan adlarında çeşitli uygulamalar çalıştırıyorsanız ve internetteki rastgele kullanıcıların bunlara erişmesini istemiyorsanız, **nginx-auth-access** önlerine ortak bir **geçiş kartlı kapı** koyar.

Kullanıcı önce ayrı bir sitede oturum açar (kullanıcı adı, parola, kimlik doğrulayıcı uygulamasından kod), seçilen süre için bir cookie alır — ve ancak o zaman nginx hedef servise erişime izin verir. Oturum açılmadan nginx form yönlendirmesi veya **401** döner. Doğrulama nginx içinde **`auth_request`** ile yapılır: ayrı hafif bir port yalnızca «geçiş kartı var mı» diye bakar, ek UI yoktur. Kullanıcılar ve tüm ayarlar tek bir **`config.toml`** dosyasındadır.

Tek ikili (Angular 21 + Go **`go:embed`**) veya Docker imajı.

## İçindekiler

1. [Kurulum (systemd)](#kurulum-linux-systemd)
2. [Docker](#docker)
3. [Kaynaktan derleme](#kaynaktan-derleme)
4. [Portlar](#portlar)
5. [Ana parametreler](#ana-parametreler)
6. [Kimlik doğrulama](#kimlik-doğrulama)
7. [Nginx entegrasyonu](#nginx-entegrasyonu)
8. [Diğer dillerde belgeler](#diğer-dillerde-belgeler)

## Kurulum (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

Betik [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) adresinden bir sürüm tarball'ı indirir (ikili, systemd birimi, yapılandırma örneği), `nginx-auth-access` kullanıcısını, dizinleri oluşturur ve servisi etkinleştirir.

Kurulumdan sonra:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Seçenekler: `--version v1.0.0`, `--no-start`. Kaldırma: [uninstall.sh](../uninstall.sh) (`--purge` yapılandırma ve verileri siler).

Varsayılan yapılandırma: **`/etc/nginx-auth-access/config.toml`** (`-config` bayrağı veya **`ACCESS_CONFIG_PATH`**).

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

Konteynerde yapılandırma **`/data/config.toml`** (`ACCESS_CONFIG_PATH` imajda ayarlı). Örnek: [`config.example.toml`](../config.example.toml).

Yayınlanmış GHCR imajı ile Compose:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

[`docker-compose.example.yml`](../docker-compose.example.yml) içindeki yorumlara bakın.

## Kaynaktan derleme

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

`-config` olmadan **`/etc/nginx-auth-access/config.toml`** kullanılır.

Frontend geliştirme:

```bash
cd frontend && npm install && npm run start
```

## Portlar

| Port | Amaç |
|------|------|
| **80** | **Giriş sayfası** — kullanıcının tarayıcıda gördüğü: giriş formu, TOTP, oturum süresi; başarılı girişten sonra «geçiş kartı» cookie'si burada ayarlanır. nginx dışarıdan **`panel_hostname`** alt alan adını buraya vekil eder (ör. `access.example.com`). |
| **81** | Yalnızca **`GET /internal/verify`** — nginx `auth_request` için. UI yok. |

## Ana parametreler

Örnek: [`config.example.toml`](../config.example.toml).

| Parametre | Açıklama |
|-----------|----------|
| `signing_key` | cookie için HMAC-JWT gizli anahtarı (**≥ 16 karakter**). |
| `panel_hostname` | Giriş panelinin genel ana bilgisayar adı (nginx `server_name`), örn. `access.example.com`. |
| `cookie_domain` | İsteğe bağlı üst cookie etki alanı, örn. `.example.com`. |
| `cookie_secure` | Tüm girişler yalnızca HTTPS ise `true`. |
| `net_safe_access` | CIDR/IP: eşleşme → cookie olmadan verify'de **204**. |
| `[listen].public` / `.verify` | HTTP dinleme adresleri (varsayılan `:80` / `:81`). |
| `[[users]]` | Kullanıcılar (bcrypt + TOTP). |

`config.toml` düzenlendikten sonra servisi veya konteyneri yeniden başlatın.

## Kimlik doğrulama

Boş kullanıcı listesinde bootstrap: kullanıcı adı/parola **`noauth`**, TOTP **`000000`**.

Normal giriş gerektirir: **kullanıcı adı**, **parola**, **TOTP**, **süre** (`30m`, `1h`, `3h`, `8h`, `24h`). İlk oluşturulan kullanıcı `admin = true` alır. Herhangi bir `[[users]]` kaydı varken **`noauth`** girişi devre dışıdır.

## Nginx entegrasyonu

1. **`panel_hostname`**'i access servisinin **80** portuna vekil edin.
2. Korunan `location` bloklarında:

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

3. Kimliği doğrulanmamış kullanıcıları `https://$panel_hostname/` adresine yönlendirin.

## Diğer dillerde belgeler

| Dil | Dosya |
|-----|-------|
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

## Lisans

[MIT](../LICENSE)
