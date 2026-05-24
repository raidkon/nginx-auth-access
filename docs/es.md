# Documentación de nginx-auth-access

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

Si ejecutas distintas aplicaciones detrás de nginx en subdominios separados — Home Assistant, Nextcloud, paneles de administración, dashboards — y no quieres que usuarios aleatorios de internet accedan a ellas, **nginx-auth-access** coloca delante un **control de acceso con pase** compartido.

El usuario inicia sesión primero en un sitio dedicado (usuario, contraseña, código de una app autenticadora), recibe una cookie por la duración elegida — y solo entonces nginx permite el acceso al servicio de destino. Sin inicio de sesión, nginx devuelve una redirección al formulario o **401**. La verificación está integrada en nginx mediante **`auth_request`**: un puerto ligero aparte solo comprueba «¿hay pase?», sin UI adicional. Usuarios y toda la configuración viven en un único **`config.toml`**.

Un solo binario (Angular 21 + Go **`go:embed`**) o imagen Docker.

## Tabla de contenidos

1. [Instalación (systemd)](#instalación-linux-systemd)
2. [Docker](#docker)
3. [Compilar desde el código fuente](#compilar-desde-el-código-fuente)
4. [Puertos](#puertos)
5. [Parámetros principales](#parámetros-principales)
6. [Autenticación](#autenticación)
7. [Integración con Nginx](#integración-con-nginx)
8. [Documentación en otros idiomas](#documentación-en-otros-idiomas)

## Instalación (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

El script descarga un tarball de release desde [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (binario, unidad systemd, ejemplo de configuración), crea el usuario `nginx-auth-access`, directorios y habilita el servicio.

Tras la instalación:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Opciones: `--version v1.0.0`, `--no-start`. Desinstalación: [uninstall.sh](../uninstall.sh) (`--purge` elimina config y datos).

Configuración por defecto: **`/etc/nginx-auth-access/config.toml`** (flag `-config` o **`ACCESS_CONFIG_PATH`**).

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

En el contenedor, la config es **`/data/config.toml`** (`ACCESS_CONFIG_PATH` está definido en la imagen). Ejemplo: [`config.example.toml`](../config.example.toml).

Compose con imagen publicada en GHCR:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

Consulta los comentarios en [`docker-compose.example.yml`](../docker-compose.example.yml).

## Compilar desde el código fuente

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Sin `-config`, se usa **`/etc/nginx-auth-access/config.toml`**.

Desarrollo del frontend:

```bash
cd frontend && npm install && npm run start
```

## Puertos

| Puerto | Propósito |
|--------|-----------|
| **80** | **Página de inicio de sesión** — lo que ve el usuario en el navegador: formulario de login, TOTP, duración de sesión; tras un login exitoso aquí se establece la cookie de «pase». nginx hace proxy del subdominio de **`panel_hostname`** aquí desde fuera (p. ej. `access.example.com`). |
| **81** | Solo **`GET /internal/verify`** — para nginx `auth_request`. Sin UI. |

## Parámetros principales

Ejemplo: [`config.example.toml`](../config.example.toml).

| Parámetro | Descripción |
|-----------|-------------|
| `signing_key` | Secreto HMAC-JWT para la cookie (**≥ 16 caracteres**). |
| `panel_hostname` | Hostname público del panel de login (nginx `server_name`), p. ej. `access.example.com`. |
| `cookie_domain` | Dominio padre opcional de la cookie, p. ej. `.example.com`. |
| `cookie_secure` | `true` cuando todos los logins usan solo HTTPS. |
| `net_safe_access` | CIDR/IP: coincidencia → **204** en verify sin cookie. |
| `[listen].public` / `.verify` | Direcciones HTTP de escucha (por defecto `:80` / `:81`). |
| `[[users]]` | Usuarios (bcrypt + TOTP). |

Reinicia el servicio o contenedor tras editar `config.toml`.

## Autenticación

Bootstrap con lista de usuarios vacía: usuario/contraseña **`noauth`**, TOTP **`000000`**.

El login normal requiere: **usuario**, **contraseña**, **TOTP**, **periodo** (`30m`, `1h`, `3h`, `8h`, `24h`). El primer usuario creado obtiene `admin = true`. Una vez existe cualquier entrada `[[users]]`, el login **`noauth`** queda deshabilitado.

## Integración con Nginx

1. Haz proxy de **`panel_hostname`** al puerto **80** del servicio access.
2. En bloques `location` protegidos:

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

3. Redirige usuarios no autenticados a `https://$panel_hostname/`.

## Documentación en otros idiomas

| Idioma | Archivo |
|--------|---------|
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

## Licencia

[MIT](../LICENSE)
