# nginx-auth-access

[Русский](docs/ru.md) · [English](docs/en.md) · [中文](docs/zh.md) · [हिन्दी](docs/hi.md) · [Español](docs/es.md) · [Français](docs/fr.md) · [العربية](docs/ar.md) · [বাংলা](docs/bn.md) · [Português](docs/pt.md) · [اردو](docs/ur.md) · [Bahasa Indonesia](docs/id.md) · [Deutsch](docs/de.md) · [日本語](docs/ja.md) · [Türkçe](docs/tr.md) · [한국어](docs/ko.md)

Если у вас за nginx висят поддомены с разным ПО — Home Assistant, Nextcloud, админки, дашборды — и вы не хотите, чтобы к ним заходил кто попало из интернета, **nginx-auth-access** ставит перед ними общую «дверь с пропуском».

Пользователь сначала логинится на отдельном сайте (логин, пароль, код из приложения-аутентификатора), получает cookie на заданное время — и только после этого nginx пускает его к нужному сервису. Без входа nginx отдаёт редирект на форму или 401. Проверка встроена в nginx через **`auth_request`**: отдельный лёгкий порт только для «есть ли пропуск», без лишнего UI. Список пользователей и все настройки — в одном **`config.toml`**.

Один бинарник (Angular 21 + Go **`go:embed`**) или Docker-образ.

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

Опции: `--version v1.0.0`, `--no-start`. Удаление: [uninstall.sh](uninstall.sh) (`--purge` — конфиг и данные).

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

В контейнере конфиг: **`/data/config.toml`** (`ACCESS_CONFIG_PATH` задаётся в образе). Пример: [`config.example.toml`](config.example.toml).

Compose с опубликованным образом GHCR:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

См. комментарии в [`docker-compose.example.yml`](docker-compose.example.yml).

## Сборка из исходников

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Без `-config` используется **`/etc/nginx-auth-access/config.toml`**.

## Порты

| Порт | Назначение |
|------|------------|
| **80** | **Страница входа** — то, что видит пользователь в браузере: форма логина, TOTP, выбор срока сессии; после успешного входа здесь же выставляется cookie «пропуск». Сюда nginx снаружи проксирует поддомен из **`panel_hostname`** (например `access.example.com`). |
| **81** | Только **`GET /internal/verify`** для nginx `auth_request`. Без UI. |

## Основные параметры

| Параметр | Описание |
|----------|----------|
| `signing_key` | Секрет HMAC-JWT для cookie (**≥ 16 символов**). |
| `panel_hostname` | Публичный hostname панели (nginx `server_name`), напр. `access.example.com`. |
| `cookie_domain` | Необязательно: родительский домен cookie, напр. `.example.com`. |
| `cookie_secure` | `true` при входе только по HTTPS. |
| `net_safe_access` | CIDR/IP: совпадение → **204** на verify без cookie. |
| `[listen].public` / `.verify` | Адреса HTTP (по умолчанию `:80` / `:81`). |
| `[[users]]` | Пользователи (bcrypt + TOTP). |

Bootstrap при пустом списке пользователей: логин/пароль **`noauth`**, TOTP **`000000`**.

## Документация

| Язык | Файл |
|------|------|
| Русский | [docs/ru.md](docs/ru.md) |
| English | [docs/en.md](docs/en.md) |
| 中文 | [docs/zh.md](docs/zh.md) |
| हिन्दी | [docs/hi.md](docs/hi.md) |
| Español | [docs/es.md](docs/es.md) |
| Français | [docs/fr.md](docs/fr.md) |
| العربية | [docs/ar.md](docs/ar.md) |
| বাংলা | [docs/bn.md](docs/bn.md) |
| Português | [docs/pt.md](docs/pt.md) |
| اردو | [docs/ur.md](docs/ur.md) |
| Bahasa Indonesia | [docs/id.md](docs/id.md) |
| Deutsch | [docs/de.md](docs/de.md) |
| 日本語 | [docs/ja.md](docs/ja.md) |
| Türkçe | [docs/tr.md](docs/tr.md) |
| 한국어 | [docs/ko.md](docs/ko.md) |

## Лицензия

[MIT](LICENSE)
