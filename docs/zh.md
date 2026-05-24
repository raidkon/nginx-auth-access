# nginx-auth-access 文档

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

如果你在 nginx 后面用不同子域名运行各种应用——Home Assistant、Nextcloud、管理面板、仪表盘——又不希望互联网上的陌生人随意访问，**nginx-auth-access** 会在它们前面加一道共用的**带通行证的闸门**。

用户先在专用站点登录（用户名、密码、身份验证器应用中的代码），获得在指定时长内有效的 cookie——之后 nginx 才允许访问目标服务。未登录时，nginx 会重定向到登录表单或返回 **401**。校验通过 nginx 内置的 **`auth_request`** 完成：一个独立的轻量端口只检查「是否有通行证」，没有额外 UI。用户和所有设置都在同一个 **`config.toml`** 中。

单个二进制文件（Angular 21 + Go **`go:embed`**）或 Docker 镜像。

## 目录

1. [安装（systemd）](#安装linux-systemd)
2. [Docker](#docker)
3. [从源码构建](#从源码构建)
4. [端口](#端口)
5. [主要参数](#主要参数)
6. [身份验证](#身份验证)
7. [Nginx 集成](#nginx-集成)
8. [其他语言文档](#其他语言文档)

## 安装（Linux，systemd）

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

该脚本从 [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) 下载发布包（二进制文件、systemd 单元、配置示例），创建用户 `nginx-auth-access`、目录并启用服务。

安装后：

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

选项：`--version v1.0.0`、`--no-start`。卸载：[uninstall.sh](../uninstall.sh)（`--purge` 会删除配置和数据）。

默认配置：**`/etc/nginx-auth-access/config.toml`**（`-config` 标志或 **`ACCESS_CONFIG_PATH`**）。

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

容器内配置为 **`/data/config.toml`**（镜像中已设置 `ACCESS_CONFIG_PATH`）。示例：[`config.example.toml`](../config.example.toml)。

使用 GHCR 已发布镜像的 Compose：

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

参见 [`docker-compose.example.yml`](../docker-compose.example.yml) 中的注释。

## 从源码构建

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

未指定 `-config` 时使用 **`/etc/nginx-auth-access/config.toml`**。

前端开发：

```bash
cd frontend && npm install && npm run start
```

## 端口

| 端口 | 用途 |
|------|------|
| **80** | **登录页面**——用户在浏览器中看到的内容：登录表单、TOTP、会话时长；成功登录后在此设置「通行证」cookie。nginx 从外部将 **`panel_hostname`** 对应的子域名代理到这里（例如 `access.example.com`）。 |
| **81** | 仅 **`GET /internal/verify`**——供 nginx `auth_request` 使用。无 UI。 |

## 主要参数

示例：[`config.example.toml`](../config.example.toml)。

| 参数 | 说明 |
|------|------|
| `signing_key` | cookie 的 HMAC-JWT 密钥（**≥ 16 个字符**）。 |
| `panel_hostname` | 登录面板的公开主机名（nginx `server_name`），例如 `access.example.com`。 |
| `cookie_domain` | 可选的父级 cookie 域，例如 `.example.com`。 |
| `cookie_secure` | 当所有登录仅通过 HTTPS 时设为 `true`。 |
| `net_safe_access` | CIDR/IP：匹配时 verify 无 cookie 也返回 **204**。 |
| `[listen].public` / `.verify` | HTTP 监听地址（默认 `:80` / `:81`）。 |
| `[[users]]` | 用户（bcrypt + TOTP）。 |

编辑 `config.toml` 后请重启服务或容器。

## 身份验证

用户列表为空时的引导：用户名/密码 **`noauth`**，TOTP **`000000`**。

正常登录需要：**用户名**、**密码**、**TOTP**、**有效期**（`30m`、`1h`、`3h`、`8h`、`24h`）。第一个创建的用户获得 `admin = true`。一旦存在任何 `[[users]]` 条目，**`noauth`** 登录即被禁用。

## Nginx 集成

1. 将 **`panel_hostname`** 代理到 access 服务的 **80** 端口。
2. 在受保护的 `location` 块中：

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

3. 将未认证用户重定向到 `https://$panel_hostname/`。

## 其他语言文档

| 语言 | 文件 |
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

## 许可证

[MIT](../LICENSE)
