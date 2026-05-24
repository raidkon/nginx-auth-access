# nginx-auth-access ドキュメント

[Русский](ru.md) · [English](en.md) · [中文](zh.md) · [हिन्दी](hi.md) · [Español](es.md) · [Français](fr.md) · [العربية](ar.md) · [বাংলা](bn.md) · [Português](pt.md) · [اردو](ur.md) · [Bahasa Indonesia](id.md) · [Deutsch](de.md) · [日本語](ja.md) · [Türkçe](tr.md) · [한국어](ko.md)

nginx の背後で Home Assistant、Nextcloud、管理パネル、ダッシュボードなど、別々のサブドメインでさまざまなアプリを動かしていて、インターネット上の不特定のユーザーにアクセスさせたくない場合、**nginx-auth-access** はそれらの前に共通の**パス付きゲート**を置きます。

ユーザーはまず専用サイトでログイン（ユーザー名、パスワード、認証アプリのコード）し、選択した期間の cookie を受け取ります——その後に nginx が対象サービスへのアクセスを許可します。未ログイン時、nginx はフォームへのリダイレクトまたは **401** を返します。検証は nginx の **`auth_request`** に組み込まれています：別の軽量ポートは「パスがあるか」だけを確認し、追加の UI はありません。ユーザーとすべての設定は 1 つの **`config.toml`** にあります。

単一バイナリ（Angular 21 + Go **`go:embed`**）または Docker イメージ。

## 目次

1. [インストール（systemd）](#インストールlinux-systemd)
2. [Docker](#docker)
3. [ソースからビルド](#ソースからビルド)
4. [ポート](#ポート)
5. [主要パラメータ](#主要パラメータ)
6. [認証](#認証)
7. [Nginx 連携](#nginx-連携)
8. [他言語のドキュメント](#他言語のドキュメント)

## インストール（Linux、systemd）

```bash
curl -fsSL https://github.com/raidkon/nginx-auth-access/raw/master/install.sh -o get-nginx-auth-access.sh
sudo sh ./get-nginx-auth-access.sh --dry-run
sudo sh ./get-nginx-auth-access.sh
```

スクリプトは [GitHub Release](https://github.com/raidkon/nginx-auth-access/releases) からリリース tarball（バイナリ、systemd ユニット、設定例）をダウンロードし、ユーザー `nginx-auth-access`、ディレクトリを作成してサービスを有効化します。

インストール後：

```bash
sudo nano /etc/nginx-auth-access/config.toml   # signing_key, panel_hostname, net_safe_access
sudo systemctl restart nginx-auth-access
sudo journalctl -u nginx-auth-access -f
```

オプション：`--version v1.0.0`、`--no-start`。アンインストール：[uninstall.sh](../uninstall.sh)（`--purge` で設定とデータを削除）。

デフォルト設定：**`/etc/nginx-auth-access/config.toml`**（`-config` フラグまたは **`ACCESS_CONFIG_PATH`**）。

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

コンテナ内の設定は **`/data/config.toml`**（イメージで `ACCESS_CONFIG_PATH` が設定済み）。例：[`config.example.toml`](../config.example.toml)。

GHCR 公開イメージでの Compose：

```bash
cp docker-compose.example.yml docker-compose.yml
cp config.example.toml config.toml
mkdir -p data/logs
docker compose up -d
```

[`docker-compose.example.yml`](../docker-compose.example.yml) のコメントを参照。

## ソースからビルド

```bash
sudo mkdir -p /etc/nginx-auth-access
sudo cp config.example.toml /etc/nginx-auth-access/config.toml

cd frontend && npm ci && npm run build
rsync -a --delete dist/frontend/browser/ ../backend/internal/uiembed/browser/
cd ../backend && go run ./cmd/access -config /etc/nginx-auth-access/config.toml
```

`-config` なしでは **`/etc/nginx-auth-access/config.toml`** を使用。

フロントエンド開発：

```bash
cd frontend && npm install && npm run start
```

## ポート

| ポート | 用途 |
|--------|------|
| **80** | **ログインページ** — ブラウザでユーザーが見るもの：ログインフォーム、TOTP、セッション期間；ログイン成功後、ここで「パス」cookie が設定されます。nginx は外部から **`panel_hostname`** のサブドメインをここにプロキシします（例：`access.example.com`）。 |
| **81** | **`GET /internal/verify`** のみ — nginx `auth_request` 用。UI なし。 |

## 主要パラメータ

例：[`config.example.toml`](../config.example.toml)。

| パラメータ | 説明 |
|------------|------|
| `signing_key` | cookie 用 HMAC-JWT シークレット（**16 文字以上**）。 |
| `panel_hostname` | ログインパネルの公開ホスト名（nginx `server_name`）、例：`access.example.com`。 |
| `cookie_domain` | 任意の親 cookie ドメイン、例：`.example.com`。 |
| `cookie_secure` | すべてのログインが HTTPS のみの場合 `true`。 |
| `net_safe_access` | CIDR/IP：一致時、cookie なしで verify が **204**。 |
| `[listen].public` / `.verify` | HTTP リッスンアドレス（デフォルト `:80` / `:81`）。 |
| `[[users]]` | ユーザー（bcrypt + TOTP）。 |

`config.toml` 編集後はサービスまたはコンテナを再起動してください。

## 認証

ユーザーリストが空のブートストラップ：ユーザー名/パスワード **`noauth`**、TOTP **`000000`**。

通常ログインには **ユーザー名**、**パスワード**、**TOTP**、**期間**（`30m`、`1h`、`3h`、`8h`、`24h`）が必要です。最初に作成されたユーザーは `admin = true` を取得します。`[[users]]` エントリが 1 つでも存在すると、**`noauth`** ログインは無効になります。

## Nginx 連携

1. **`panel_hostname`** を access サービスの **80** ポートにプロキシ。
2. 保護された `location` ブロックで：

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

3. 未認証ユーザーを `https://$panel_hostname/` にリダイレクト。

## 他言語のドキュメント

| 言語 | ファイル |
|------|----------|
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

## ライセンス

[MIT](../LICENSE)
