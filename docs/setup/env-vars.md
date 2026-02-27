# 環境変数リファレンス

GIVErS プラットフォームで使用する全環境変数の一覧、意味、入手方法をまとめたドキュメントです。

---

## 目次

- [バックエンド環境変数](#バックエンド環境変数)
  - [DATABASE_URL](#database_url)
  - [FRONTEND_URL](#frontend_url)
  - [BACKEND_URL](#backend_url)
  - [SESSION_SECRET](#session_secret)
  - [AUTH_REQUIRED](#auth_required)
  - [GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET](#google_client_id--google_client_secret)
  - [GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET](#github_client_id--github_client_secret)
  - [DISCORD_CLIENT_ID / DISCORD_CLIENT_SECRET](#discord_client_id--discord_client_secret)
  - [STRIPE_SECRET_KEY](#stripe_secret_key)
  - [STRIPE_WEBHOOK_SECRET](#stripe_webhook_secret)
  - ~~STRIPE_CONNECT_CLIENT_ID~~ (v2 API 移行により廃止)
  - [HOST_EMAILS](#host_emails)
  - [OFFICIAL_DOMAIN](#official_domain)
- [フロントエンド環境変数](#フロントエンド環境変数)
  - [PUBLIC_MOCK_MODE](#public_mock_mode)
  - [PUBLIC_API_URL](#public_api_url)
  - [PUBLIC_PLATFORM_PROJECT_ID](#public_platform_project_id)
  - [PUBLIC_OFFICIAL_URL](#public_official_url)
  - [PUBLIC_GITHUB_REPO](#public_github_repo)
- [環境別の設定例](#環境別の設定例)

---

## バックエンド環境変数

設定ファイル: `backend/.env`（`.gitignore` 済み）
テンプレート: `.env.example`

### DATABASE_URL

| 項目 | 内容 |
|------|------|
| **必須** | はい |
| **形式** | `postgres://ユーザー:パスワード@ホスト:ポート/DB名?sslmode=disable` |
| **用途** | PostgreSQL データベースへの接続文字列 |

**入手方法:**

- **ローカル開発（Docker Compose）**: `docker-compose.yml` で定義された DB サービスの接続情報をそのまま使用。デフォルト: `postgres://givers:givers@localhost:5432/givers?sslmode=disable`
- **本番**: ConoHa VPS 上の PostgreSQL または Docker 内 DB の接続情報を指定。`sslmode=require` を推奨。

---

### FRONTEND_URL

| 項目 | 内容 |
|------|------|
| **必須** | はい |
| **形式** | `https://example.com`（末尾スラッシュなし） |
| **用途** | CORS 許可オリジン、OAuth 認証後のリダイレクト先、Stripe Checkout の成功/キャンセル URL |

**入手方法:**

- **ローカル開発**: `http://localhost:4321`（Astro のデフォルトポート）
- **本番**: 取得したドメインの URL（例: `https://givers.work`）

---

### BACKEND_URL

| 項目 | 内容 |
|------|------|
| **必須** | はい（OAuth 使用時） |
| **形式** | `https://api.example.com`（末尾スラッシュなし） |
| **用途** | OAuth プロバイダーへのコールバック URL 生成に使用 |

**入手方法:**

- **ローカル開発**: `http://localhost:8080`
- **本番**: バックエンド API のドメイン URL。フロントと同一ドメインの場合はそのドメイン、サブドメインの場合は `https://api.example.com` 等。

---

### SESSION_SECRET

| 項目 | 内容 |
|------|------|
| **必須** | はい |
| **形式** | 32文字以上のランダム文字列 |
| **用途** | セッション cookie の暗号化/署名に使用 |

**入手方法:**

以下のコマンドで生成:

```bash
openssl rand -base64 32
```

生成された文字列をそのまま `.env` に設定する。**起動のたびに変えるとセッションが無効になるため、固定値を使うこと。**

- **ローカル開発**: `dev-secret-change-in-production-32bytes`（開発用の固定値で OK）
- **本番**: 上記コマンドで生成した値を使用

---

### AUTH_REQUIRED

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（デフォルト: `false`） |
| **形式** | `true` または `false` |
| **用途** | `true`: 認証ミドルウェアが有効（本番用）。`false`: 認証をスキップし、開発用ダミーユーザーで動作 |

**入手方法:**

手動で設定。

- **ローカル開発**: `false`（OAuth 設定なしでも全機能を試せる）
- **本番**: `true`

---

### GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET

| 項目 | 内容 |
|------|------|
| **必須** | はい（本番）。開発時は `AUTH_REQUIRED=false` なら不要 |
| **形式** | `GOOGLE_CLIENT_ID`: `xxxxxxxxxxxx.apps.googleusercontent.com` / `GOOGLE_CLIENT_SECRET`: `GOCSPX-...` |
| **用途** | Google OAuth2 によるユーザーログイン |

**入手方法:**

1. [Google Cloud Console](https://console.cloud.google.com/) を開く
2. プロジェクトを作成（または既存を選択）
3. **「APIとサービス」→「OAuth 同意画面」** を設定（アプリ名・スコープ `email`, `profile`, `openid`）
4. **「APIとサービス」→「認証情報」→「+ 認証情報を作成」→「OAuth クライアント ID」**
5. アプリケーションの種類: **ウェブ アプリケーション**
6. **承認済みのリダイレクト URI** に追加:
   - ローカル: `http://localhost:8080/api/auth/google/callback`
   - 本番: `https://your-domain/api/auth/google/callback`
7. 表示される **クライアント ID** と **クライアントシークレット** をコピー

> 詳細手順: [oauth2-setup.md](oauth2-setup.md) の「1. Google OAuth2」セクション

---

### GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（オプション。未設定時は GitHub ログインボタンが非表示） |
| **形式** | `GITHUB_CLIENT_ID`: `Iv1.xxxxxxxxxxxxxxxx` / `GITHUB_CLIENT_SECRET`: 40文字の英数字 |
| **用途** | GitHub OAuth2 によるユーザーログイン |

**入手方法:**

1. GitHub → **Settings** → **Developer settings** → **OAuth Apps** → **New OAuth App**
2. 以下を入力:
   - Application name: `GIVErS`
   - Homepage URL: サービスの URL
   - Authorization callback URL: `http://localhost:8080/api/auth/github/callback`（ローカル）
3. **Register application** をクリック
4. **Generate a new client secret** をクリック（**表示は一度きり**）
5. Client ID と Client secret をコピー

> **注意**: GitHub OAuth App は 1つの App に 1つのコールバック URL しか設定できない。ローカル用と本番用で別の App を作成する。

> 詳細手順: [oauth2-setup.md](oauth2-setup.md) の「2. GitHub OAuth2」セクション

---

### DISCORD_CLIENT_ID / DISCORD_CLIENT_SECRET

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（オプション。未設定時は Discord ログインボタンが非表示） |
| **形式** | `DISCORD_CLIENT_ID`: 数字列（18〜20桁） / `DISCORD_CLIENT_SECRET`: 32文字の英数字 |
| **用途** | Discord OAuth2 によるユーザーログイン |

**入手方法:**

1. [Discord Developer Portal](https://discord.com/developers/applications) → **New Application**
2. アプリ名（例: `GIVErS`）を入力 → **Create**
3. 左メニュー **「OAuth2」→「General」**
4. **Client ID** をコピー
5. **「Reset Secret」** → 表示される **Client Secret** をコピー（**表示は一度きり**）
6. **「Redirects」** に追加:
   - ローカル: `http://localhost:8080/api/auth/discord/callback`
   - 本番: `https://your-domain/api/auth/discord/callback`

> **注意**: Discord はメールアドレスを非公開にできます。取得できない場合は `username@discord.invalid` をフォールバックとして使用します。

> 詳細手順: [oauth2-setup.md](oauth2-setup.md) の「3. Discord OAuth2」セクション

---

### STRIPE_SECRET_KEY

| 項目 | 内容 |
|------|------|
| **必須** | はい（決済機能を使う場合） |
| **形式** | テスト: `sk_test_...` / 本番: `sk_live_...` |
| **用途** | Stripe API の認証。Checkout Session 作成、Webhook 処理、Connect OAuth コード交換に使用 |

**入手方法:**

1. [Stripe ダッシュボード](https://dashboard.stripe.com/) にログイン
2. **開発者** → **API キー** を開く
3. **シークレットキー** をコピー

> **重要**: テストキー（`sk_test_`）と本番キー（`sk_live_`）を混在させない。本番キーを使うにはアカウントの本人確認が必要。

> 詳細手順: [stripe-connect-setup.md](stripe-connect-setup.md) の「2. ホスト側の Stripe アカウント作成」セクション

---

### STRIPE_WEBHOOK_SECRET

| 項目 | 内容 |
|------|------|
| **必須** | はい（Webhook 受信時） |
| **形式** | `whsec_...` |
| **用途** | Stripe Webhook のシグネチャ検証。改ざんされたリクエストの拒否に使用 |

**入手方法:**

**本番:**
1. Stripe ダッシュボード → **開発者** → **Webhook** → **エンドポイントを追加**
2. URL: `https://your-domain/api/webhooks/stripe`
3. 受信するイベント: `payment_intent.succeeded`, `customer.subscription.created`, `customer.subscription.deleted`
4. 作成後に表示される **署名シークレット**（`whsec_...`）をコピー

**ローカル開発（Stripe CLI）:**
```bash
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```
出力される `whsec_...` を `.env` に設定。

> 詳細手順: [stripe-connect-setup.md](stripe-connect-setup.md) の「4. Webhook エンドポイントの登録」セクション

---

---

> **注意**: `STRIPE_CONNECT_CLIENT_ID`（`ca_...`）は不要になりました。Accounts v2 API では `STRIPE_SECRET_KEY` のみで連結アカウントの作成・オンボーディングが可能です。

---

### HOST_EMAILS

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（未設定時はホスト判定なし） |
| **形式** | カンマ区切りのメールアドレス |
| **用途** | サービスホスト（プラットフォーム運営者）の判定。一致するメールでログインしたユーザーは `role: "host"` となり、管理機能やホスト専用プロジェクト作成フローが有効になる |

**入手方法:**

手動で設定。プラットフォーム運営者のメールアドレスを指定する。

```env
HOST_EMAILS=admin@example.com,host@givers.co.jp
```

> ホストのプロジェクトは Stripe Connect OAuth をスキップし、プラットフォームの Stripe アカウントで直接決済される。

---

### OFFICIAL_DOMAIN

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（将来の自ホスト判定用） |
| **形式** | `https://givers.example.com` |
| **用途** | 自ホスト判定用（将来実装予定） |

**入手方法:**

手動で設定。サービスの公式ドメインを指定する。

---

## フロントエンド環境変数

設定ファイル: `frontend/.env`
テンプレート: `frontend/.env.example`

Astro のルールにより、フロントエンドでアクセスする環境変数はすべて `PUBLIC_` プレフィックスが必要。

### PUBLIC_MOCK_MODE

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（デフォルト: `false`） |
| **形式** | `true` または `false` |
| **用途** | `true`: API サーバーなしでモックデータでフロントエンドが動作（UX 検証・デザイン検討用） |

**入手方法:** 手動で設定。

---

### PUBLIC_API_URL

| 項目 | 内容 |
|------|------|
| **必須** | はい（`PUBLIC_MOCK_MODE=false` の場合） |
| **形式** | `http://localhost:8080` |
| **用途** | バックエンド API のベース URL |

**入手方法:**

- **ローカル開発**: `http://localhost:8080`
- **本番**: バックエンドの URL（例: `https://givers.work` または `https://api.givers.work`）

---

### PUBLIC_PLATFORM_PROJECT_ID

| 項目 | 内容 |
|------|------|
| **必須** | いいえ（デフォルト: `mock-4`） |
| **形式** | プロジェクト ID（UUID） |
| **用途** | GIVErS プラットフォーム自体への寄付用プロジェクトの ID |

**入手方法:**

プラットフォームプロジェクトを作成した後、その ID を設定する。未設定時は `mock-4` がフォールバック値として使われる。

---

### PUBLIC_OFFICIAL_URL

| 項目 | 内容 |
|------|------|
| **必須** | いいえ |
| **形式** | `https://givers.example.com` |
| **用途** | About ページで表示する公式サイト URL |

**入手方法:** 手動で設定。

---

### PUBLIC_GITHUB_REPO

| 項目 | 内容 |
|------|------|
| **必須** | いいえ |
| **形式** | `https://github.com/example/givers` |
| **用途** | About ページで表示する GitHub リポジトリ URL |

**入手方法:** 手動で設定。

---

## 環境別の設定例

### ローカル開発（最小構成）

```env
# backend/.env
DATABASE_URL=postgres://givers:givers@localhost:5432/givers?sslmode=disable
FRONTEND_URL=http://localhost:4321
AUTH_REQUIRED=false
SESSION_SECRET=dev-secret-change-in-production-32bytes
```

```env
# frontend/.env
PUBLIC_MOCK_MODE=true
```

この構成では OAuth や Stripe の設定なしで全画面を確認できる。

### ローカル開発（API + 認証あり）

```env
# backend/.env
DATABASE_URL=postgres://givers:givers@localhost:5432/givers?sslmode=disable
FRONTEND_URL=http://localhost:4321
BACKEND_URL=http://localhost:8080
AUTH_REQUIRED=true
SESSION_SECRET=dev-secret-change-in-production-32bytes

GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxx

STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
# STRIPE_CONNECT_CLIENT_ID は廃止（v2 API 移行済み）

HOST_EMAILS=admin@example.com
```

```env
# frontend/.env
PUBLIC_MOCK_MODE=false
PUBLIC_API_URL=http://localhost:8080
```

### 本番

```env
# backend/.env
DATABASE_URL=postgres://givers:password@db:5432/givers?sslmode=require
FRONTEND_URL=https://givers.work
BACKEND_URL=https://givers.work
AUTH_REQUIRED=true
SESSION_SECRET=<openssl rand -base64 32 で生成>

GOOGLE_CLIENT_ID=xxxxxxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxx
GITHUB_CLIENT_ID=Iv1.xxxxxxxxxxxxxxxx
GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
DISCORD_CLIENT_ID=xxxxxxxxxxxxxxxxxxxx
DISCORD_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
# STRIPE_CONNECT_CLIENT_ID は廃止（v2 API 移行済み）

HOST_EMAILS=admin@givers.work
OFFICIAL_DOMAIN=https://givers.work
```

```env
# frontend/.env
PUBLIC_MOCK_MODE=false
PUBLIC_API_URL=https://givers.work
PUBLIC_PLATFORM_PROJECT_ID=<プラットフォームプロジェクトの UUID>
PUBLIC_OFFICIAL_URL=https://givers.work
PUBLIC_GITHUB_REPO=https://github.com/example/givers
```

---

## 関連ドキュメント

- [oauth2-setup.md](oauth2-setup.md) — Google / GitHub / Discord OAuth の詳細設定手順
- [stripe-connect-setup.md](stripe-connect-setup.md) — Stripe Connect の詳細設定手順
- [launch-setup-order.md](launch-setup-order.md) — 本番リリース前の外部サービス設定順序
