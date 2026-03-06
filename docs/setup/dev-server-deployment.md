# ConoHa VPS — GIVErS DEV環境デプロイ手順

**サーバー**: ConoHa VPS v3 (Ubuntu 24)
**ドメイン**: dev.givers.work
**IP**: 163.44.124.145
**最終更新**: 2026-03-05

---

## 1. サーバー初期設定

### SSH 接続

```bash
ssh givers-conoha-dev
```

- SSH鍵認証: `~/.ssh/givers_conoha_dev_key` (ed25519)
- パスワード認証: 無効化済み (`/etc/ssh/sshd_config.d/50-cloud-init.conf`)

### ConoHa セキュリティグループ

VPSに以下のセキュリティグループを割り当て:
- `default`
- `IPv4v6-SSH` — ポート22
- `IPv4v6-Web` — ポート80/443

### UFW（OS ファイアウォール）

```bash
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

### DNS

ConoHa DNS で設定:

| タイプ | 名称 | TTL  | 値               |
|--------|------|------|------------------|
| A      | dev  | 3600 | 163.44.124.145   |

---

## 2. サーバーに Docker をインストール

```bash
ssh givers-conoha-dev
curl -fsSL https://get.docker.com | sh
systemctl enable docker
```

---

## 3. デプロイ方式

本番と同じ **ローカルビルド + SCP 方式**。

```
ローカル PC                          ConoHa VPS (DEV)
─────────────                       ──────────────
1. git pull (対象ブランチ)
2. /tmp にパッケージ作成
   (node_modules/.git 除外)
3. rsync/scp で転送 ──────────────→ /opt/givers-dev/
                                    4. docker compose build
                                    5. docker compose up -d
```

### デプロイスクリプト

```bash
# デフォルト (main ブランチ)
./scripts/deploy-dev.sh

# ブランチ指定
./scripts/deploy-dev.sh feature/xxx
```

スクリプトの処理:
1. ローカルで指定ブランチを pull
2. `/tmp/givers-deploy-dev` に必要ファイルをコピー
3. `rsync` でサーバーの `/opt/givers-dev` に転送（`.env` は除外）
4. サーバーで `docker compose -f docker-compose.dev.yml up -d --build`
5. DBマイグレーション実行

オプション:
- `--init` — ファイル転送のみ（初回セットアップ用、ビルド・起動しない）

---

## 4. 初回セットアップ

### 4.1 初回デプロイ（ファイル転送のみ）

```bash
# ローカルから実行
./scripts/deploy-dev.sh --init
```

### 4.2 環境変数の設定

サーバー上で `.env` を作成:

```bash
ssh givers-conoha-dev
cd /opt/givers-dev
cp .env.dev.example .env
nano .env    # 各値をDEV用に編集
chmod 600 .env
```

**必須の変更項目**:
- `POSTGRES_PASSWORD` — `openssl rand -base64 24` で生成
- `DATABASE_URL` — 上記パスワードと一致させる
- `SESSION_SECRET` — `openssl rand -base64 32` で生成
- `STRIPE_SECRET_KEY` — **サンドボックス (テストモード) のキー** (`sk_test_...`)
- `STRIPE_WEBHOOK_SECRET` — DEV用Webhookの署名シークレット (`whsec_...`)
- `CERTBOT_EMAIL` — SSL 証明書の通知先メール
- OAuth の各 Client ID/Secret（後述）

### 4.3 SSL 証明書の初回取得

```bash
ssh givers-conoha-dev
cd /opt/givers-dev
./scripts/init-ssl-dev.sh
```

処理内容:
1. HTTP-only の nginx 設定で起動
2. certbot で Let's Encrypt 証明書を取得（`dev.givers.work` のみ、`www` なし）
3. HTTPS 対応の nginx 設定に切り替え
4. 全サービスを起動

---

## 5. 通常のデプロイ更新

ローカルから実行するだけ:

```bash
./scripts/deploy-dev.sh
```

---

## 6. サーバー上での操作

```bash
ssh givers-conoha-dev
cd /opt/givers-dev

# 起動
docker compose -f docker-compose.dev.yml up -d

# 停止
docker compose -f docker-compose.dev.yml down

# ログ確認
docker compose -f docker-compose.dev.yml logs -f

# 特定サービスのログ
docker compose -f docker-compose.dev.yml logs -f backend

# コンテナ状態
docker compose -f docker-compose.dev.yml ps
```

---

## 7. SSL 証明書の自動更新

```bash
ssh givers-conoha-dev
crontab -e
```

```
0 3 * * * cd /opt/givers-dev && docker compose -f docker-compose.dev.yml run --rm certbot renew && docker compose -f docker-compose.dev.yml exec nginx nginx -s reload
```

---

## 8. 外部サービス設定

### OAuth プロバイダー

コールバックURLは `BACKEND_URL` 環境変数から自動構築される。
外部サービス側にDEV用のコールバックURLを登録する必要がある。

| プロバイダー | 方針 | コールバックURL |
|-------------|------|----------------|
| Google | 既存クライアントにURL追加 | `https://dev.givers.work/api/auth/google/callback` |
| GitHub | **DEV専用OAuth App新規作成** | `https://dev.givers.work/api/auth/github/callback` |
| Discord | 既存アプリにURL追加 | `https://dev.givers.work/api/auth/discord/callback` |

#### Google OAuth
1. [Google Cloud Console](https://console.cloud.google.com/) → APIとサービス → 認証情報
2. 既存の OAuth 2.0 クライアント ID → 承認済みのリダイレクト URI に追加
3. `.env` には本番と同じ `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` を使用

#### GitHub OAuth
1. [GitHub Developer Settings](https://github.com/settings/developers) → OAuth Apps → New OAuth App
2. Application name: `GIVErS DEV`, Homepage URL: `https://dev.givers.work`
3. Authorization callback URL: `https://dev.givers.work/api/auth/github/callback`
4. `.env` にはDEV専用の `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` を使用

#### Discord OAuth
1. [Discord Developer Portal](https://discord.com/developers/applications) → 既存アプリ → OAuth2
2. Redirects に `https://dev.givers.work/api/auth/discord/callback` を追加
3. `.env` には本番と同じ `DISCORD_CLIENT_ID` / `DISCORD_CLIENT_SECRET` を使用

### Stripe (サンドボックス)

DEV環境では Stripe の**サンドボックス (テストモード)** を使用する。

#### APIキー
1. [Stripe Dashboard](https://dashboard.stripe.com/) → テストモードに切り替え
2. 開発者 → APIキー → シークレットキー (`sk_test_...`) を取得
3. `.env` の `STRIPE_SECRET_KEY` に設定

#### Webhookエンドポイント (DEV専用)
1. Stripe Dashboard → テストモード → 開発者 → Webhook → エンドポイントを追加
2. エンドポイントURL: `https://dev.givers.work/api/webhooks/stripe`
3. リッスンするイベント:
   - `payment_intent.succeeded`
   - `customer.subscription.created`
   - `customer.subscription.deleted`
   - `invoice.payment_succeeded`
4. 署名シークレット (`whsec_...`) を `.env` の `STRIPE_WEBHOOK_SECRET` に設定

#### テスト用カード番号

| カード番号 | 結果 |
|-----------|------|
| `4242 4242 4242 4242` | 成功 |
| `4000 0000 0000 0002` | カード拒否 |
| `4000 0025 0000 3155` | 3Dセキュア認証要求 |

有効期限: 未来の任意の日付、CVC: 任意の3桁

---

## 9. 本番との差分

| 項目 | 本番 | DEV |
|------|------|-----|
| ドメイン | `givers.work` | `dev.givers.work` |
| IP | `163.44.111.156` | `163.44.124.145` |
| SSH ホスト | `givers-conoha-root` | `givers-conoha-dev` |
| デプロイ先 | `/opt/givers` | `/opt/givers-dev` |
| Compose | `docker-compose.prod.yml` | `docker-compose.dev.yml` |
| Frontend Dockerfile | `Dockerfile.prod` | `Dockerfile.dev-server` |
| Nginx 設定 | `default.conf` | `default.conf.dev` |
| 環境変数テンプレート | `.env.prod.example` | `.env.dev.example` |
| デプロイスクリプト | `deploy.sh` | `deploy-dev.sh` |
| SSL スクリプト | `init-ssl.sh` | `init-ssl-dev.sh` |
| Stripe | 本番キー (`sk_live_*`) | サンドボックス (`sk_test_*`) |
| GitHub OAuth | 本番 OAuth App | DEV専用 OAuth App |
| DEVバナー | なし | 青色バナー表示 |
| Google Analytics | 有効 | 無効 |
| ログレベル | `info` | `debug` |

---

## 10. DBダンプ

### 本番DBをローカルにダウンロード

```bash
./scripts/dump-prod-db.sh                    # カレントディレクトリに出力
./scripts/dump-prod-db.sh /path/to/output    # 出力先指定
```

### DEV DBをローカルにダウンロード

```bash
./scripts/dump-dev-db.sh
```

### 本番DBをDEVサーバーに投入

```bash
# 1. まず本番からダンプ
./scripts/dump-prod-db.sh

# 2. DEVサーバーに投入
cat givers_prod_YYYYMMDD_HHMMSS.sql | ssh givers-conoha-dev \
  'cd /opt/givers-dev && docker compose -f docker-compose.dev.yml exec -T db psql -U givers -d givers'
```

---

## 11. ファイル構成

```
docker-compose.dev.yml         # DEV用 Compose
.env.dev.example               # DEV環境変数テンプレート
frontend/.env.dev.example      # DEVフロントエンド環境変数テンプレート
frontend/Dockerfile.dev-server # DEVフロント本番ビルド (.env.dev → .env)
nginx/conf.d/
  default.conf.dev             # HTTPS nginx 設定 (dev.givers.work)
  default.conf.dev.initial     # 初回 SSL 取得用 HTTP 設定
scripts/
  deploy-dev.sh                # DEV用デプロイスクリプト
  init-ssl-dev.sh              # DEV用 SSL 初期設定スクリプト
  dump-prod-db.sh              # 本番DBダンプ
  dump-dev-db.sh               # DEV DBダンプ
```

---

## 12. 運用チェックリスト

### 初回セットアップ
- [ ] ConoHa VPS インスタンス作成
- [ ] SSH鍵登録・パスワード認証無効化
- [ ] ConoHa セキュリティグループ `IPv4v6-Web` を追加
- [ ] UFW で 80/443 を許可
- [ ] Docker インストール
- [ ] DNS: `dev.givers.work` の A レコード追加
- [ ] `./scripts/deploy-dev.sh --init` で初回デプロイ
- [ ] サーバーで `.env` を作成・設定
- [ ] `./scripts/init-ssl-dev.sh` で SSL 証明書取得
- [ ] `https://dev.givers.work` でアクセス確認
- [ ] DEVバナー（青色）が表示されることを確認
- [ ] certbot 自動更新 cron 設定
- [ ] Google OAuth: コールバックURL追加
- [ ] GitHub OAuth: DEV専用 App 作成
- [ ] Discord OAuth: リダイレクトURL追加
- [ ] Stripe: DEV用 Webhook エンドポイント登録
- [ ] 各プロバイダーでログイン動作確認
- [ ] Stripe テストカードで決済動作確認

### 通常デプロイ
- [ ] `./scripts/deploy-dev.sh` を実行
- [ ] 動作確認

---

## 13. トラブルシューティング

### SSH 接続できない
- ConoHa セキュリティグループに `IPv4v6-SSH` が割り当てられているか確認
- VPS コンソール（ブラウザ）からログインして `ufw status` を確認

### HTTPS がつながらない
- ConoHa セキュリティグループに `IPv4v6-Web` が必要
- `ufw allow 80/tcp && ufw allow 443/tcp`
- `docker compose -f docker-compose.dev.yml logs nginx` でエラー確認

### DNS が引けない
- ConoHa DNS で A レコードが設定されているか確認
- `dig dev.givers.work A @a.conoha-dns.com` で直接確認

### パスワード認証が無効にならない
- `/etc/ssh/sshd_config.d/50-cloud-init.conf` に `PasswordAuthentication yes` が残っている場合がある
- `sed -i 's/^PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config.d/50-cloud-init.conf && systemctl restart ssh`
