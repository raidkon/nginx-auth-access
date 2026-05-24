# Documentação nginx-auth-access

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

Se você executa diferentes aplicativos atrás do nginx em subdomínios separados — Home Assistant, Nextcloud, painéis de administração, dashboards — e não quer que usuários aleatórios da internet acessem, o **nginx-auth-access** coloca na frente um **controle de acesso com passe** compartilhado.

O usuário faz login primeiro em um site dedicado (nome de usuário, senha, código de um app autenticador), recebe um cookie pela duração escolhida — e só então o nginx permite acesso ao serviço de destino. Sem login, o nginx retorna redirecionamento para o formulário ou **401**. A verificação está integrada ao nginx via **`auth_request`**: uma porta leve separada só verifica «há passe», sem UI extra. Usuários e todas as configurações ficam em um único **`config.toml`**.

Binário único (Angular 21 + Go **`go:embed`**) ou imagem Docker.

## Índice

1. [Instalação (systemd)](#instalação-linux-systemd)
2. [Docker](#docker)
3. [Compilar a partir do código-fonte](#compilar-a-partir-do-código-fonte)
4. [Portas](#portas)
5. [Parâmetros principais](#parâmetros-principais)
6. [Autenticação](#autenticação)
7. [Integração com Nginx](#integração-com-nginx)
8. [Documentação em outros idiomas](#documentação-em-outros-idiomas)

## Instalação (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

O script baixa um tarball de release do [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (binário, unidade systemd, exemplo de config), cria o usuário `nginx-auth-access`, diretórios e habilita o serviço.

Após a instalação:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Opções: `--version v1.0.0`, `--no-start`. Desinstalação: [uninstall.sh](../uninstall.sh) (`--purge` remove config e dados).

Config padrão: **`/etc/nginx-auth-access/config.toml`** (flag `-config` ou **`ACCESS_CONFIG_PATH`**).

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

No container, a config é **`/data/config.toml`** (`ACCESS_CONFIG_PATH` definido na imagem). Exemplo: [`config.example.toml`](../config.example.toml).

Compose com imagem publicada no GHCR:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

Veja comentários em [`docker-compose.example.yml`](../docker-compose.example.yml).

## Compilar a partir do código-fonte

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Sem `-config`, usa **`/etc/nginx-auth-access/config.toml`**.

Desenvolvimento do frontend:

```bash
cd frontend && npm install && npm run start
```

## Portas

| Porta | Finalidade |
|-------|------------|
| **80** | **Página de login** — o que o usuário vê no navegador: formulário de login, TOTP, duração da sessão; após login bem-sucedido o cookie de «passe» é definido aqui. O nginx faz proxy do subdomínio de **`panel_hostname`** para cá externamente (ex.: `access.example.com`). |
| **81** | Apenas **`GET /internal/verify`** — para nginx `auth_request`. Sem UI. |

## Parâmetros principais

Exemplo: [`config.example.toml`](../config.example.toml).

| Parâmetro | Descrição |
|-----------|-----------|
| `signing_key` | Segredo HMAC-JWT para cookie (**≥ 16 caracteres**). |
| `panel_hostname` | Hostname público do painel de login (nginx `server_name`), ex.: `access.example.com`. |
| `cookie_domain` | Domínio pai opcional do cookie, ex.: `.example.com`. |
| `cookie_secure` | `true` quando todos os logins usam apenas HTTPS. |
| `net_safe_access` | CIDR/IP: correspondência → **204** no verify sem cookie. |
| `[listen].public` / `.verify` | Endereços de escuta HTTP (padrão `:80` / `:81`). |
| `[[users]]` | Usuários (bcrypt + TOTP). |

Reinicie o serviço ou container após editar `config.toml`.

## Autenticação

Bootstrap com lista de usuários vazia: usuário/senha **`noauth`**, TOTP **`000000`**.

Login normal exige: **usuário**, **senha**, **TOTP**, **período** (`30m`, `1h`, `3h`, `8h`, `24h`). O primeiro usuário criado recebe `admin = true`. Quando existir qualquer entrada `[[users]]`, o login **`noauth`** é desabilitado.

## Integração com Nginx

1. Faça proxy de **`panel_hostname`** para a porta **80** do serviço access.
2. Em blocos `location` protegidos:

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

3. Redirecione usuários não autenticados para `https://$panel_hostname/`.

## Documentação em outros idiomas

| Idioma | Arquivo |
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

## Licença

[MIT](../LICENSE)
