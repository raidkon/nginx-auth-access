# Documentation nginx-auth-access

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

Si vous faites tourner différentes applications derrière nginx sur des sous-domaines distincts — Home Assistant, Nextcloud, panneaux d’administration, tableaux de bord — et que vous ne voulez pas que n’importe qui sur Internet y accède, **nginx-auth-access** place devant elles un **contrôle d’accès avec laissez-passer** partagé.

L’utilisateur se connecte d’abord sur un site dédié (identifiant, mot de passe, code d’une application d’authentification), reçoit un cookie pour la durée choisie — et seulement alors nginx autorise l’accès au service cible. Sans connexion, nginx renvoie une redirection vers le formulaire ou **401**. La vérification est intégrée à nginx via **`auth_request`** : un port léger séparé ne fait que vérifier « y a-t-il un laissez-passer », sans UI supplémentaire. Utilisateurs et tous les paramètres sont dans un seul **`config.toml`**.

Un seul binaire (Angular 21 + Go **`go:embed`**) ou image Docker.

## Table des matières

1. [Installation (systemd)](#installation-linux-systemd)
2. [Docker](#docker)
3. [Compilation depuis les sources](#compilation-depuis-les-sources)
4. [Ports](#ports)
5. [Paramètres principaux](#paramètres-principaux)
6. [Authentification](#authentification)
7. [Intégration Nginx](#intégration-nginx)
8. [Documentation dans d’autres langues](#documentation-dans-dautres-langues)

## Installation (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

Le script télécharge une archive de release depuis [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) (binaire, unité systemd, exemple de config), crée l’utilisateur `nginx-auth-access`, les répertoires et active le service.

Après l’installation :

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

Options : `--version v1.0.0`, `--no-start`. Désinstallation : [uninstall.sh](../uninstall.sh) (`--purge` supprime config et données).

Config par défaut : **`/etc/nginx-auth-access/config.toml`** (flag `-config` ou **`ACCESS_CONFIG_PATH`**).

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

Dans le conteneur, la config est **`/data/config.toml`** (`ACCESS_CONFIG_PATH` est défini dans l’image). Exemple : [`config.example.toml`](../config.example.toml).

Compose avec l’image publiée sur GHCR :

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

Voir les commentaires dans [`docker-compose.example.yml`](../docker-compose.example.yml).

## Compilation depuis les sources

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

Sans `-config`, **`/etc/nginx-auth-access/config.toml`** est utilisé.

Développement frontend :

```bash
cd frontend && npm install && npm run start
```

## Ports

| Port | Rôle |
|------|------|
| **80** | **Page de connexion** — ce que l’utilisateur voit dans le navigateur : formulaire de login, TOTP, durée de session ; après une connexion réussie, le cookie « laissez-passer » est défini ici. nginx fait proxy du sous-domaine de **`panel_hostname`** ici depuis l’extérieur (p. ex. `access.example.com`). |
| **81** | Uniquement **`GET /internal/verify`** — pour nginx `auth_request`. Pas d’UI. |

## Paramètres principaux

Exemple : [`config.example.toml`](../config.example.toml).

| Paramètre | Description |
|-----------|-------------|
| `signing_key` | Secret HMAC-JWT pour le cookie (**≥ 16 caractères**). |
| `panel_hostname` | Nom d’hôte public du panneau de connexion (nginx `server_name`), p. ex. `access.example.com`. |
| `cookie_domain` | Domaine parent optionnel du cookie, p. ex. `.example.com`. |
| `cookie_secure` | `true` lorsque toutes les connexions passent uniquement par HTTPS. |
| `net_safe_access` | CIDR/IP : correspondance → **204** sur verify sans cookie. |
| `[listen].public` / `.verify` | Adresses d’écoute HTTP (par défaut `:80` / `:81`). |
| `[[users]]` | Utilisateurs (bcrypt + TOTP). |

Redémarrez le service ou le conteneur après modification de `config.toml`.

## Authentification

Bootstrap avec une liste d’utilisateurs vide : identifiant/mot de passe **`noauth`**, TOTP **`000000`**.

La connexion normale exige : **identifiant**, **mot de passe**, **TOTP**, **période** (`30m`, `1h`, `3h`, `8h`, `24h`). Le premier utilisateur créé obtient `admin = true`. Dès qu’une entrée `[[users]]` existe, la connexion **`noauth`** est désactivée.

## Intégration Nginx

1. Proxifiez **`panel_hostname`** vers le port **80** du service access.
2. Dans les blocs `location` protégés :

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

3. Redirigez les utilisateurs non authentifiés vers `https://$panel_hostname/`.

## Documentation dans d’autres langues

| Langue | Fichier |
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

## Licence

[MIT](../LICENSE)
