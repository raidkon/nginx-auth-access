# nginx-auth-access 문서

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

nginx 뒤에서 Home Assistant, Nextcloud, 관리 패널, 대시보드 등 서로 다른 서브도메인에서 여러 앱을 실행하는데, 인터넷의 임의 사용자가 접근하는 것을 원하지 않는다면 **nginx-auth-access**가 앞에 공통 **통행증 게이트**를 둡니다.

사용자는 먼저 전용 사이트에서 로그인(사용자명, 비밀번호, 인증 앱 코드)하고, 선택한 기간 동안 cookie를 받습니다 — 그 후에야 nginx가 대상 서비스 접근을 허용합니다. 로그인하지 않으면 nginx는 폼으로 리다이렉트하거나 **401**을 반환합니다. 검증은 nginx **`auth_request`**에 내장되어 있습니다: 별도의 가벼운 포트는 «통행증이 있는지»만 확인하며 추가 UI는 없습니다. 사용자와 모든 설정은 하나의 **`config.toml`**에 있습니다.

단일 바이너리(Angular 21 + Go **`go:embed`**) 또는 Docker 이미지.

## 목차

1. [설치 (systemd)](#설치-linux-systemd)
2. [Docker](#docker)
3. [소스에서 빌드](#소스에서-빌드)
4. [포트](#포트)
5. [주요 매개변수](#주요-매개변수)
6. [인증](#인증)
7. [Nginx 연동](#nginx-연동)
8. [다른 언어 문서](#다른-언어-문서)

## 설치 (Linux, systemd)

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

스크립트는 [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases)에서 릴리스 tarball(바이너리, systemd 유닛, 설정 예시)을 다운로드하고, `nginx-auth-access` 사용자와 디렉터리를 만들고 서비스를 활성화합니다.

설치 후:

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

옵션: `--version v1.0.0`, `--no-start`. 제거: [uninstall.sh](../uninstall.sh) (`--purge`는 설정과 데이터 삭제).

기본 설정: **`/etc/nginx-auth-access/config.toml`** (`-config` 플래그 또는 **`ACCESS_CONFIG_PATH`**).

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

컨테이너에서 설정은 **`/data/config.toml`** (이미지에 `ACCESS_CONFIG_PATH` 설정됨). 예: [`config.example.toml`](../config.example.toml).

GHCR 공개 이미지로 Compose:

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

[`docker-compose.example.yml`](../docker-compose.example.yml)의 주석 참고.

## 소스에서 빌드

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

`-config` 없이 **`/etc/nginx-auth-access/config.toml`** 사용.

프론트엔드 개발:

```bash
cd frontend && npm install && npm run start
```

## 포트

| 포트 | 용도 |
|------|------|
| **80** | **로그인 페이지** — 브라우저에서 사용자가 보는 것: 로그인 폼, TOTP, 세션 기간; 로그인 성공 후 여기서 «통행증» cookie가 설정됩니다. nginx는 외부에서 **`panel_hostname`**의 서브도메인을 여기로 프록시합니다(예: `access.example.com`). |
| **81** | **`GET /internal/verify`** 만 — nginx `auth_request`용. UI 없음. |

## 주요 매개변수

예: [`config.example.toml`](../config.example.toml).

| 매개변수 | 설명 |
|----------|------|
| `signing_key` | cookie용 HMAC-JWT 비밀 (**16자 이상**). |
| `panel_hostname` | 로그인 패널의 공개 호스트명(nginx `server_name`), 예: `access.example.com`. |
| `cookie_domain` | 선택적 상위 cookie 도메인, 예: `.example.com`. |
| `cookie_secure` | 모든 로그인이 HTTPS만 사용할 때 `true`. |
| `net_safe_access` | CIDR/IP: 일치 시 cookie 없이 verify에서 **204**. |
| `[listen].public` / `.verify` | HTTP 수신 주소(기본 `:80` / `:81`). |
| `[[users]]` | 사용자(bcrypt + TOTP). |

`config.toml` 수정 후 서비스 또는 컨테이너를 재시작하세요.

## 인증

사용자 목록이 비어 있을 때 부트스트랩: 사용자명/비밀번호 **`noauth`**, TOTP **`000000`**.

일반 로그인에는 **사용자명**, **비밀번호**, **TOTP**, **기간**(`30m`, `1h`, `3h`, `8h`, `24h`)이 필요합니다. 첫 번째 생성된 사용자는 `admin = true`를 받습니다. `[[users]]` 항목이 하나라도 있으면 **`noauth`** 로그인이 비활성화됩니다.

## Nginx 연동

1. **`panel_hostname`**을 access 서비스 **80** 포트로 프록시.
2. 보호된 `location` 블록에서:

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

3. 미인증 사용자를 `https://$panel_hostname/`로 리다이렉트.

## 다른 언어 문서

| 언어 | 파일 |
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

## 라이선스

[MIT](../LICENSE)
