# Документация nginx-auth-access

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

Если у вас за nginx висят поддомены с разным ПО — Home Assistant, Nextcloud, админки, дашборды — и вы не хотите, чтобы к ним заходил кто попало из интернета, **nginx-auth-access** ставит перед ними общую «дверь с пропуском».

Пользователь сначала логинится на отдельном сайте (логин, пароль, код из приложения-аутентификатора), получает cookie на заданное время — и только после этого nginx пускает его к нужному сервису. Без входа nginx отдаёт редирект на форму или 401. Проверка встроена в nginx через **`auth_request`**: отдельный лёгкий порт только для «есть ли пропуск», без лишнего UI. Список пользователей и все настройки — в одном **`config.toml`**.

Один бинарник (Angular 21 + Go **`go:embed`**) или Docker-образ.

## Содержание

1. [Установка (systemd)](#установка-linux-systemd)
2. [Docker](#docker)
3. [Сборка из исходников](#сборка-из-исходников)
4. [Порты](#порты)
5. [Основные параметры](#основные-параметры)
6. [Авторизация](#авторизация)
7. [Интеграция с nginx](#интеграция-с-nginx)
8. [Документация на других языках](#документация-на-других-языках)

## Установка (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

Скрипт скачивает бинарник и файлы из [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (tar.gz с systemd unit и примером конфигурации), создаёт пользователя `nginx-auth-access`, каталоги и сервис.

После установки:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Опции: `--version v1.0.0`, `--no-start`. Удаление: [uninstall.sh](../uninstall.sh) (`--purge` — конфиг и данные).

По умолчанию конфиг: **`/etc/nginx-auth-access/config.toml`** (флаг `-config` или переменная **`ACCESS_CONFIG_PATH`**).

## Docker

```bash
git clone https://github.com/raidkon/nginx-auth-access.git
cd nginx-auth-access
cp config.example.toml store/config.toml
# отредактируйте signing_key, panel_hostname, net_safe_access, cookie_*

docker build -f docker/Dockerfile -t nginx-auth-access:local .
docker run -d --name access \
  -v "$(pwd)/store:/data" \
  --cap-add=NET_BIND_SERVICE \
  nginx-auth-access:local
```

В контейнере конфиг: **`/data/config.toml`** (`ACCESS_CONFIG_PATH` задаётся в образе). Пример: [`config.example.toml`](../config.example.toml).

Compose с опубликованным образом GHCR:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

См. комментарии в [`docker-compose.example.yml`](../docker-compose.example.yml).

## Сборка из исходников

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Без `-config` используется **`/etc/nginx-auth-access/config.toml`**.

Разработка фронтенда:

```bash
cd frontend && npm install && npm run start
```

## Порты

| Порт | Назначение |
|------|------------|
| **80** | **Страница входа** — то, что видит пользователь в браузере: форма логина, TOTP, выбор срока сессии; после успешного входа здесь же выставляется cookie «пропуск». Сюда nginx снаружи проксирует поддомен из **`panel_hostname`** (например `access.example.com`). |
| **81** | Только **`GET /internal/verify`** для nginx `auth_request`. Без UI. |

## Основные параметры

Пример: [`config.example.toml`](../config.example.toml).

| Параметр | Описание |
|----------|----------|
| `signing_key` | Секрет HMAC-JWT для cookie (**≥ 16 символов**). |
| `panel_hostname` | Публичный hostname панели (nginx `server_name`), напр. `access.example.com`. |
| `cookie_domain` | Необязательно: родительский домен cookie, напр. `.example.com`. |
| `cookie_secure` | `true` при входе только по HTTPS. |
| `net_safe_access` | CIDR/IP: совпадение → **204** на verify без cookie. |
| `[listen].public` / `.verify` | Адреса HTTP (по умолчанию `:80` / `:81`). |
| `[[users]]` | Пользователи (bcrypt + TOTP). |

После правки `config.toml` перезапустите сервис или контейнер.

## Авторизация

Bootstrap при пустом списке пользователей: логин/пароль **`noauth`**, TOTP **`000000`**.

Обычный вход требует: **логин**, **пароль**, **TOTP**, **период** (`30m`, `1h`, `3h`, `8h`, `24h`). Первый созданный пользователь получает `admin = true`. После появления `[[users]]` вход **`noauth`** отключается.

## Интеграция с nginx

1. Проксируйте **`panel_hostname`** на порт **80** сервиса access.
2. В защищённых `location`:

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

3. Перенаправляйте неавторизованных на `https://$panel_hostname/`.

## Документация на других языках

| Язык | Файл |
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

## Лицензия

[MIT](../LICENSE)
