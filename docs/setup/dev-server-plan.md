# DEV環境サーバー構築 — 実装計画書

## 概要

本番環境 (`givers.work`) と同じ構成で、開発・ステージング用サーバーを `dev.givers.work` に構築する。
DEV環境には「開発用」であることを示すバナーを表示し、本番GIVErSへのリンクを併記する。
本番DBダンプのローカルダウンロードスクリプトも用意する。

## 現状の把握

| 項目 | 現状 |
|------|------|
| 本番サーバー | ConoHa VPS v3 (Ubuntu 24, 1GB/2Core), IP: `163.44.111.156` |
| SSH | `givers-conoha-root` |
| デプロイ先 | `/opt/givers` |
| 構成 | Docker Compose (nginx + certbot + frontend + backend + db) |
| SSL | Let's Encrypt (certbot, ACME HTTP-01) |
| 既存バナー | `PUBLIC_TEST_MODE=true` で赤色「テスト運用中」バナーあり |

---

## 全7フェーズ・20ステップ

### Phase 1: インフラ準備 (手動作業)

| # | 内容 | リスク |
|---|------|--------|
| 1 | ConoHa VPSインスタンス作成 (Ubuntu 24, 最小プラン) | 低 |
| 2 | SSH設定 (`~/.ssh/config` に `givers-conoha-dev` 追加) | 低 |
| 3 | DEVサーバー初期設定 (Docker, UFW) | 低 |
| 4 | DNS設定 (`dev.givers.work` → DEVサーバーIP の Aレコード) | 低 |

#### Step 1: ConoHa VPSインスタンス作成
- ConoHa コントロールパネルでサーバーを追加
- SSH鍵を設定 (新規鍵 `~/.ssh/givers_conoha_dev_key` を生成するか、既存鍵を流用)
- IPアドレスを控える
- セキュリティグループ: `default`, `IPv4v6-SSH`, `IPv4v6-Web` を割り当て

#### Step 2: SSH設定
`~/.ssh/config` に追加:
```
Host givers-conoha-dev
  HostName <DEV_IP_ADDRESS>
  User root
  IdentityFile ~/.ssh/givers_conoha_dev_key
  IdentitiesOnly yes
```

#### Step 3: DEVサーバー初期設定
```bash
ssh givers-conoha-dev
curl -fsSL https://get.docker.com | sh
systemctl enable docker
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

#### Step 4: DNS設定
ConoHa DNSパネルで Aレコードを追加:

| タイプ | 名称 | TTL  | 値 |
|--------|------|------|----|
| A      | dev  | 3600 | <DEV_IP_ADDRESS> |

---

### Phase 2: 環境変数ファイル作成

バックエンド・フロントエンドそれぞれに `.env.dev` 系ファイルを用意し、本番 (`.env.prod`) とは独立した環境変数を渡す。

| # | ファイル | 説明 |
|---|---------|------|
| 5 | `.env.dev.example` | バックエンド用テンプレート (リポジトリにコミット) |
| 6 | `.env.dev` | バックエンド用実ファイル (**gitignore対象**) |
| 7 | `frontend/.env.dev.example` | フロントエンド用テンプレート (リポジトリにコミット) |
| 8 | `frontend/.env.dev` | フロントエンド用実ファイル (**gitignore対象**) |
| 9 | `.gitignore` 更新 | `.env.dev` を追加 |

#### Step 5: `.env.dev.example` (バックエンド用テンプレート)

```env
# ===========================================
# GIVErS DEV環境変数 (.env.dev)
# コピーして .env.dev を作成し、値を設定する
# ===========================================

# --- PostgreSQL ---
POSTGRES_USER=givers
POSTGRES_PASSWORD=CHANGE_ME_DEV_PASSWORD
POSTGRES_DB=givers
DATABASE_URL=postgres://givers:CHANGE_ME_DEV_PASSWORD@db:5432/givers?sslmode=disable

# --- Backend ---
FRONTEND_URL=https://dev.givers.work
BACKEND_URL=https://dev.givers.work
AUTH_REQUIRED=true
SESSION_SECRET=CHANGE_ME_32BYTE_RANDOM_BASE64
LOG_LEVEL=debug

# --- Stripe (サンドボックス) ---
# Stripe Dashboard → テストモード → APIキー から取得
STRIPE_SECRET_KEY=sk_test_...
# Stripe Dashboard → テストモード → Webhook → DEV用エンドポイントの署名シークレット
STRIPE_WEBHOOK_SECRET=whsec_...

# --- OAuth ---
# Google: 本番と同じクライアント (Cloud ConsoleでDEV用コールバックURLを追加済み)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
# GitHub: DEV専用OAuth App (1 App = 1 callback URL の制約あり)
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
# Discord: 本番と同じクライアント (Developer PortalでDEV用リダイレクトURLを追加済み)
DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=

# --- Admin ---
HOST_EMAILS=
OFFICIAL_DOMAIN=https://givers.work

# --- Frontend (docker-compose経由でバックエンドに渡す分) ---
PUBLIC_API_URL=https://dev.givers.work
PUBLIC_MOCK_MODE=false
PUBLIC_PLATFORM_PROJECT_ID=
PUBLIC_OFFICIAL_URL=https://givers.work

# --- Domain / SSL ---
DOMAIN=dev.givers.work
CERTBOT_EMAIL=your-email@example.com
```

#### Step 6: `.env.dev` (バックエンド実ファイル)
`.env.dev.example` をコピーし、実際の値を設定する。

#### Step 7: `frontend/.env.dev.example` (フロントエンド用テンプレート)

```env
# ===========================================
# GIVErS DEV フロントエンド環境変数
# コピーして frontend/.env.dev を作成
# Astro ビルド時に PUBLIC_* が埋め込まれる
# ===========================================

PUBLIC_MOCK_MODE=false
PUBLIC_API_URL=https://dev.givers.work
PUBLIC_PLATFORM_PROJECT_ID=
PUBLIC_OFFICIAL_URL=https://givers.work
PUBLIC_GITHUB_REPO=https://github.com/SeijiShii/givers
PUBLIC_TEST_MODE=true
PUBLIC_DEV_MODE=true
# DEV環境ではGA無効 (空にする)
PUBLIC_GA_MEASUREMENT_ID=
```

本番との主な差分:

| 変数 | 本番 (.env.prod) | DEV (.env.dev) |
|------|-----------------|----------------|
| `PUBLIC_API_URL` | `https://givers.work` | `https://dev.givers.work` |
| `PUBLIC_DEV_MODE` | (なし) | `true` |
| `PUBLIC_GA_MEASUREMENT_ID` | `G-XXXXXXX` | (空 = 無効) |

#### Step 8: `frontend/.env.dev` (フロントエンド実ファイル)
`frontend/.env.dev.example` をコピーして作成。

#### Step 9: `.gitignore` 更新
`.env.dev` を追加 (ルート直下と frontend/ の両方がマッチ):
```
.env.dev
```

---

### Phase 3: Docker / Nginx 設定

| # | ファイル | 内容 |
|---|---------|------|
| 10 | `nginx/conf.d/default.conf.dev` | `server_name dev.givers.work` + SSL証明書パス変更 |
| 11 | `nginx/conf.d/default.conf.dev.initial` | SSL初回取得用 HTTP-only 設定 |
| 12 | `docker-compose.dev.yml` | DEV用Nginx設定マウント, `Dockerfile.dev-server` 使用, `env_file: .env.dev` |
| 13 | `frontend/Dockerfile.dev-server` | `.env.dev` → `.env` にリネームしてビルド |

#### Step 10: DEV用Nginx設定
`default.conf` をベースに以下を変更:
- `server_name dev.givers.work;` (port 80 / 443 の2箇所)
- `ssl_certificate /etc/letsencrypt/live/dev.givers.work/fullchain.pem;`
- `ssl_certificate_key /etc/letsencrypt/live/dev.givers.work/privkey.pem;`

#### Step 11: DEV用Nginx初期設定
`default.conf.initial` をベースに `server_name dev.givers.work;` に変更。

#### Step 12: `docker-compose.dev.yml`
`docker-compose.prod.yml` をベースに以下を変更:
```yaml
services:
  nginx:
    volumes:
      - ./nginx/conf.d/default.conf.dev:/etc/nginx/conf.d/default.conf:ro
      # ...

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile.dev-server

  backend:
    env_file: .env.dev    # ← 本番は .env.prod

  db:
    env_file: .env.dev    # ← DB接続情報も .env.dev から
```

#### Step 13: `frontend/Dockerfile.dev-server`
`Dockerfile.prod` をベースに `.env.dev` → `.env` にリネームしてビルド:
```dockerfile
RUN mv .env .env.orig 2>/dev/null || true && \
    mv .env.dev .env 2>/dev/null || true
```
Astro は `.env` を読み込むため、ビルド前にリネームが必要。

---

### Phase 4: DEVバナー表示 (フロントエンド)

| # | ファイル | 内容 |
|---|---------|------|
| 14 | `src/i18n/ja.json`, `en.json` | `devBanner` キー追加 (本番リンク付き) |
| 15 | `src/layouts/BaseLayout.astro` | `PUBLIC_DEV_MODE` による青色DEVバナー表示ロジック |
| 16 | `src/styles/global.css` | `.dev-banner` CSS (青色, z-index:10000, リンクは金色) |

#### Step 14: i18nキー追加
- ja.json: `"devBanner": "これは開発環境です。本番サイトは <a href=\"https://givers.work\">givers.work</a> をご覧ください。"`
- en.json: `"devBanner": "This is a development environment. Visit <a href=\"https://givers.work\">givers.work</a> for the production site."`

#### Step 15: BaseLayout.astro にDEVバナーロジック追加
```typescript
const devMode = import.meta.env.PUBLIC_DEV_MODE === 'true';
const officialUrl = import.meta.env.PUBLIC_OFFICIAL_URL || 'https://givers.work';
```

テンプレート部分 (既存testBannerの直後):
```astro
{devMode && (
  <div class="dev-banner">
    <Fragment set:html={t('devBanner')} />
  </div>
)}
```

#### Step 16: DEVバナー用CSS
```css
.dev-banner {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  background: rgba(30, 100, 200, 0.9);
  color: #fff;
  text-align: center;
  padding: 0.5rem 1rem;
  font-size: 0.9rem;
  font-weight: bold;
  z-index: 10000;
}
.dev-banner a {
  color: #ffd700;
  text-decoration: underline;
}
body:has(.dev-banner) {
  padding-top: 2.5rem;
}
body:has(.test-banner):has(.dev-banner) {
  padding-top: 5rem;
}
body:has(.dev-banner) .test-banner {
  top: 2.5rem;
}
```

---

### Phase 5: デプロイスクリプト

| # | ファイル | 内容 |
|---|---------|------|
| 17 | `scripts/deploy-dev.sh` | `deploy.sh` ベース, `REMOTE_HOST=givers-conoha-dev`, compose → `docker-compose.dev.yml` |
| 18 | `scripts/init-ssl-dev.sh` | `dev.givers.work` のSSL証明書取得 |

#### Step 17: `scripts/deploy-dev.sh`
`deploy.sh` をベースに以下を変更:
- `REMOTE_HOST="givers-conoha-dev"`
- `REMOTE_DIR="/opt/givers-dev"`
- `LOCAL_TMP="/tmp/givers-deploy-dev"`
- compose ファイル: `docker-compose.dev.yml`
- 環境変数ファイル: `.env.dev` と `frontend/.env.dev` を転送
- リモート実行コマンド: `docker compose -f docker-compose.dev.yml ...`

#### Step 18: `scripts/init-ssl-dev.sh`
`init-ssl.sh` をベースに以下を変更:
- compose ファイル: `docker-compose.dev.yml`
- certbot `-d dev.givers.work` のみ (`www` 不要)
- nginx設定切り替え: `default.conf.dev.initial` ↔ `default.conf.dev`

---

### Phase 6: DBダンプスクリプト

| # | ファイル | 内容 |
|---|---------|------|
| 19 | `scripts/dump-prod-db.sh` | SSH経由で `pg_dump` → ローカルにストリーム |

```bash
#!/bin/bash
set -e

REMOTE_HOST="givers-conoha-root"
REMOTE_DIR="/opt/givers"
COMPOSE_FILE="docker-compose.prod.yml"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DUMP_FILENAME="givers_prod_${TIMESTAMP}.sql"
LOCAL_OUTPUT_DIR="${1:-.}"
LOCAL_DUMP_PATH="${LOCAL_OUTPUT_DIR}/${DUMP_FILENAME}"

# リモートでダンプ → ローカルにストリーム (リモートにファイルを残さない)
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && \
  docker compose -f $COMPOSE_FILE exec -T db \
  pg_dump -U givers -d givers --no-owner --no-privileges \
  --exclude-table-data=sessions" \
  > "$LOCAL_DUMP_PATH"
```

設計ポイント:
- `--no-owner --no-privileges` でDEV/ローカル復元を容易に
- `--exclude-table-data=sessions` でセッション情報を除外
- タイムスタンプ付きファイル名で世代管理

---

### Phase 7: ドキュメント

| # | ファイル | 内容 |
|---|---------|------|
| 20 | `docs/setup/dev-server-deployment.md` | DEVサーバー構築・運用手順書 |
| 21 | `TODO.md` 更新 | 完了タスクのチェック |

---

## 外部サービスのDEV設定 (詳細手順)

### 仕組み

OAuthコールバックURLは `BACKEND_URL` 環境変数から動的に構築される:
```go
// backend/internal/handler/auth_handler.go
redirectBase := os.Getenv("BACKEND_URL")  // DEV: "https://dev.givers.work"
googleConfig.RedirectURL = redirectBase + "/api/auth/google/callback"
githubConfig.RedirectURL = redirectBase + "/api/auth/github/callback"
discordConfig.RedirectURL = redirectBase + "/api/auth/discord/callback"
```

Stripeのオンボーディング return/refresh URL も `FRONTEND_URL` から構築される:
```go
// backend/internal/service/stripe_service.go
returnURL := s.frontendURL + "/api/stripe/onboarding/return?project_id=" + projectID
```

**→ バックエンド側は `.env.dev` の URL を設定するだけで自動的にDEV用URLになる。**
**→ 問題は外部サービス側にDEV用コールバックURLを登録する手動作業。**

### 方針

| サービス | 方針 | 理由 |
|---------|------|------|
| Google OAuth | 既存クライアントにコールバックURL追加 | 1クライアントに複数URIを登録可能 |
| GitHub OAuth | **DEV専用OAuth Appを新規作成** | 1 App = 1 callback URL の制約あり |
| Discord OAuth | 既存アプリにリダイレクトURL追加 | 1アプリに複数URLを登録可能 |
| Stripe | **サンドボックス環境を使用** | 本番の決済に影響させない |

---

### 1. Google OAuth — 既存クライアントにURL追加

1. [Google Cloud Console](https://console.cloud.google.com/) を開く
2. **APIとサービス** → **認証情報** → 既存の OAuth 2.0 クライアント ID をクリック
3. **承認済みのリダイレクト URI** に追加:
   ```
   https://dev.givers.work/api/auth/google/callback
   ```
4. **保存**

`.env.dev` の設定:
```env
# 本番と同じクライアントID/Secret
GOOGLE_CLIENT_ID=（本番と同じ値）
GOOGLE_CLIENT_SECRET=（本番と同じ値）
```

> **注意**: OAuth 同意画面が「テスト」モードの場合、テストユーザーとして登録されたGoogleアカウントのみログイン可能。必要に応じてテストユーザーを追加する。

---

### 2. GitHub OAuth — DEV専用OAuth Appを新規作成

GitHub OAuth App は **1アプリ = 1コールバックURL** の制約がある。

1. [GitHub Developer Settings](https://github.com/settings/developers) を開く
2. **OAuth Apps** → **New OAuth App**
3. 以下を入力:

   | 項目 | 値 |
   |------|-----|
   | Application name | `GIVErS DEV` |
   | Homepage URL | `https://dev.givers.work` |
   | Authorization callback URL | `https://dev.givers.work/api/auth/github/callback` |

4. **Register application** → **Client ID** と **Client Secret** を控える

`.env.dev` の設定:
```env
# DEV専用の新しいクライアントID/Secret
GITHUB_CLIENT_ID=（DEV用に新規発行した値）
GITHUB_CLIENT_SECRET=（DEV用に新規発行した値）
```

---

### 3. Discord OAuth — 既存アプリにURL追加

1. [Discord Developer Portal](https://discord.com/developers/applications) を開く
2. 既存のアプリケーション → **OAuth2** タブ
3. **Redirects** に追加:
   ```
   https://dev.givers.work/api/auth/discord/callback
   ```
4. **Save Changes**

`.env.dev` の設定:
```env
# 本番と同じクライアントID/Secret
DISCORD_CLIENT_ID=（本番と同じ値）
DISCORD_CLIENT_SECRET=（本番と同じ値）
```

---

### 4. Stripe — サンドボックス環境を使用

DEV環境では Stripe の**サンドボックス (テストモード)** を使用する。
サンドボックスは本番とは完全に分離されており、実際の課金は発生しない。

#### 4a. APIキーの取得

1. [Stripe Dashboard](https://dashboard.stripe.com/) にログイン
2. 右上の **「テストモード」** トグルがオンであることを確認 (オレンジ色の「テストモード」バナーが表示される)
3. **開発者** → **APIキー**
4. **シークレットキー** (`sk_test_...`) を控える

`.env.dev` の設定:
```env
STRIPE_SECRET_KEY=sk_test_...
```

> Stripe は `sk_test_*` / `sk_live_*` でテストと本番が完全に分離されている。
> DEV環境で `sk_test_*` を使う限り、本番の課金・アカウントに一切影響しない。

#### 4b. Webhookエンドポイントの登録 (DEV専用)

Webhook は**エンドポイントURLごとに個別の署名シークレット (`whsec_*`)** が発行される。
DEV用を本番とは別に登録する。

1. Stripe Dashboard → **テストモード**に切り替え
2. **開発者** → **Webhook** → **エンドポイントを追加**
3. 以下を設定:

   | 項目 | 値 |
   |------|-----|
   | エンドポイントURL | `https://dev.givers.work/api/webhooks/stripe` |
   | リッスンするイベント | 下記参照 |

4. リッスンするイベントを選択:
   - `payment_intent.succeeded`
   - `customer.subscription.created`
   - `customer.subscription.deleted`
   - `invoice.payment_succeeded`

5. **エンドポイントを追加** → 表示される **署名シークレット (`whsec_*`)** を控える

`.env.dev` の設定:
```env
STRIPE_WEBHOOK_SECRET=whsec_...  # DEV専用 (本番とは異なる値)
```

#### 4c. Connected Account (Stripe Connect)

Connected Account の作成・オンボーディングはサンドボックス内で動作する。
`FRONTEND_URL` を `https://dev.givers.work` に設定するだけで、
return/refresh URL は自動的にDEV用になる。**Stripe側の追加設定は不要。**

サンドボックスで作成された Connected Account はテスト用であり、
本番の Connected Account とは完全に分離されている。

#### 4d. テスト用カード番号

DEV環境で決済テストする際に使えるカード:

| カード番号 | 結果 |
|-----------|------|
| `4242 4242 4242 4242` | 成功 |
| `4000 0000 0000 0002` | カード拒否 |
| `4000 0025 0000 3155` | 3Dセキュア認証要求 |

有効期限: 未来の任意の日付、CVC: 任意の3桁、郵便番号: 任意

---

### 5. 外部サービス設定チェックリスト

- [ ] Google Cloud Console: リダイレクトURI `https://dev.givers.work/api/auth/google/callback` を追加
- [ ] GitHub: DEV専用 OAuth App `GIVErS DEV` を新規作成し Client ID/Secret を取得
- [ ] Discord Developer Portal: リダイレクトURL `https://dev.givers.work/api/auth/discord/callback` を追加
- [ ] Stripe Dashboard (テストモード): Webhook `https://dev.givers.work/api/webhooks/stripe` を登録し `whsec_*` を取得
- [ ] `.env.dev` に全ての値を設定
- [ ] `frontend/.env.dev` に `PUBLIC_DEV_MODE=true` 等を設定
- [ ] DEVサーバーにデプロイ後、各プロバイダーのログインを動作確認
- [ ] Stripe テストカード `4242...` で決済フローを動作確認

---

## リスクと対策

| リスク | 影響度 | 対策 |
|--------|--------|------|
| 間違って本番にデプロイ | **高** | `REMOTE_HOST` をハードコード、スクリプト冒頭で確認表示 |
| 本番DBダンプに個人情報 | **高** | sessions除外, サニタイズオプション検討 |
| Stripe 本番キーをDEVに設定 | **高** | `.env.dev.example` に `sk_test_...` と明記、`sk_live_*` を使わないこと |
| GitHub OAuth 本番Appのcallback変更 | 中 | DEV専用Appを新規作成 (本番Appは触らない) |
| Let's Encrypt レート制限 | 低 | サブドメインは独立カウント |

---

## 推奨実装順序

```
Phase 2 (環境変数ファイル) → Phase 4 (DEVバナー) → Phase 3 (Docker/Nginx)
    → Phase 5 (デプロイスクリプト) → Phase 6 (DBダンプ)
        → Phase 1 (ConoHaインスタンス作成・手動) → Phase 7 (ドキュメント)
```

コード系を先に全て揃えてから、手動インフラ作業をすると
`deploy-dev.sh --init` → `.env` 設定 → `init-ssl-dev.sh` で一気に完了できる。

外部サービスの設定 (OAuth URL追加, Stripe Webhook登録) は
DEVサーバーのSSL設定が完了してから行う (URLが `https://` で有効でないとコールバックが動作しないため)。

---

## 成功基準

- [ ] `dev.givers.work` に HTTPS でアクセスできる
- [ ] 青色DEVバナー + 本番リンクが表示される
- [ ] 本番 `givers.work` にはDEVバナーが表示されない
- [ ] `deploy-dev.sh` でデプロイできる
- [ ] `dump-prod-db.sh` で本番DBダンプがDLできる
- [ ] Google / GitHub / Discord の各ログインが動作する
- [ ] Stripe サンドボックスでテスト決済が動作する
- [ ] DEV環境構築手順が `docs/setup/dev-server-deployment.md` に文書化されている
