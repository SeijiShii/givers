# Stripe Connect 設定ガイド（Accounts v2 API）

> **2026-02 更新**: Stripe Connect Standard（OAuth）から **Accounts v2 API** に移行しました。
> `STRIPE_CONNECT_CLIENT_ID`（`ca_...`）は不要です。`STRIPE_SECRET_KEY` のみで動作します。

## v2 API 概要

GIVErS プラットフォームは **Stripe Accounts v2 API** を使用して連結アカウントを管理します。

### 新しいフロー

1. プロジェクト作成時に `POST /v2/core/accounts` で連結アカウントを自動作成
2. `POST /v2/core/account_links` でオンボーディング URL を生成
3. プロジェクトオーナーが Stripe のオンボーディングページで本人確認・銀行口座を設定
4. 完了後に `GET /api/stripe/onboarding/return` にリダイレクト → プロジェクトが `active` に

### 設定

| パラメータ | 値 | 説明 |
|-----------|-----|------|
| `dashboard` | `full` | オーナーが Stripe ダッシュボードにフルアクセス可能 |
| `fees_collector` | `stripe` | Stripe が手数料を直接徴収 |
| `losses_collector` | `stripe` | 損失は Stripe が負担 |
| `capabilities` | `card_payments` | カード決済を有効化 |

### 必要な環境変数

```env
STRIPE_SECRET_KEY=sk_test_...    # API 認証
STRIPE_WEBHOOK_SECRET=whsec_...  # Webhook 署名検証
# STRIPE_CONNECT_CLIENT_ID は廃止
```

### API エンドポイント

| メソッド | パス | 説明 |
|---------|------|------|
| `GET` | `/api/stripe/onboarding/return` | オンボーディング完了後のリターン URL |
| `GET` | `/api/stripe/onboarding/refresh` | オンボーディング再開（リンク期限切れ時） |
| `POST` | `/api/donations/checkout` | Stripe Checkout セッション作成 |
| `POST` | `/api/webhooks/stripe` | Webhook 受信 |

---

> **以下は旧ドキュメント（OAuth ベース）です。参考情報として残しています。**

---

## 目次（旧ドキュメント）

1. [アーキテクチャ概要](#1-アーキテクチャ概要)
2. [ホスト側の Stripe アカウント作成](#2-ホスト側の-stripe-アカウント作成)
3. [Connect Standard の有効化](#3-connect-standard-の有効化)
4. [Webhook エンドポイントの登録](#4-webhook-エンドポイントの登録)
5. [環境変数の設定](#5-環境変数の設定)
6. [プロジェクトオーナーの連携フロー](#6-プロジェクトオーナーの連携フロー)
7. [寄付の決済フロー](#7-寄付の決済フロー)
8. [テスト環境での動作確認](#8-テスト環境での動作確認)
9. [本番切り替え](#9-本番切り替え)
10. [トラブルシューティング](#10-トラブルシューティング)

---

## 1. アーキテクチャ概要

### 登場人物

| 役割 | 説明 | Stripe 上の位置 |
|------|------|-----------------|
| **ホスト（プラットフォーム運営者）** | GIVErS を運営する | プラットフォームアカウント |
| **プロジェクトオーナー** | 寄付を受け取るプロジェクトの作成者 | Connected Account（`acct_...`） |
| **寄付者** | 寄付をする人 | Customer（匿名 or ログインユーザー） |

### 資金の流れ

```
寄付者
  │
  │  Stripe Checkout で決済
  ▼
┌────────────────────────────┐
│  プロジェクトオーナーの     │   ← 寄付金は直接オーナーに入る
│  Stripe アカウント (acct_) │
│  （Connected Account）     │
└────────────────────────────┘
        │
        │ Stripe 決済手数料 3.6% のみ差し引き
        ▼
  オーナーの銀行口座に自動入金
```

**ポイント**: GIVErS は手数料ゼロ。Stripe の決済手数料（日本: 3.6%）のみが発生し、
プラットフォームは中間マージンを取りません。

### Connect Standard を選んだ理由

| | Standard | Express |
|---|---|---|
| オーナーの操作 | 既存の Stripe アカウントで OAuth 連携 | Stripe が新規アカウントを作成 |
| KYC（本人確認） | オーナーが自分の Stripe で完了済み | Stripe のオンボーディングで実施 |
| ダッシュボード | オーナーが自分の Stripe ダッシュボードで管理 | 制限付きダッシュボード |
| 手数料設定 | プラットフォームは手数料を取れない（GIVE の理念と合致） | アプリケーション手数料を設定可能 |
| 実装の複雑さ | OAuth フローのみ | アカウント作成 + オンボーディング API |

Standard は「手数料ゼロ」「オーナーが自分の Stripe を使う」という GIVErS の方針に最適。

---

## 2. ホスト側の Stripe アカウント作成

### 2-1. アカウント登録

1. [stripe.com/jp](https://stripe.com/jp) にアクセス
2. メールアドレスでアカウントを作成
3. **ビジネスの種類** を選択:
   - 個人運営の場合: 「個人事業主」
   - 法人の場合: 「株式会社」等
4. ビジネス情報を入力:
   - ビジネス名: `GIVErS`（または正式名称）
   - 業種: `ソフトウェア` または `非営利`
   - ウェブサイト: `https://your-domain.example.com`
5. 本人確認書類をアップロード（運転免許証 / マイナンバーカード等）

> **注意**: 本人確認が完了するまで本番 API キーは使用できません。
> テストモード（`sk_test_...`）は即座に利用可能です。

### 2-2. API キーの取得

1. Stripe ダッシュボード → **開発者** → **API キー** を開く
2. 以下のキーをメモ:

| キー | 形式 | 用途 |
|------|------|------|
| シークレットキー（テスト） | `sk_test_...` | 開発環境用。サーバー側で使用 |
| シークレットキー（本番） | `sk_live_...` | 本番環境用。サーバー側で使用 |
| 公開可能キー（テスト） | `pk_test_...` | 現在は未使用（将来フロントで使う可能性あり） |

> **重要**: シークレットキーは絶対に公開しない。Git にコミットしない。
> `.env` に記載し、`.gitignore` に含める。

---

## 3. Connect Standard の有効化

### 3-1. Connect 設定画面を開く

1. Stripe ダッシュボード → **Connect** → **設定** を開く
   - 初回は「Connect を始める」のようなウィザードが表示される場合がある
2. **プラットフォームプロフィール** を設定:
   - プラットフォーム名: `GIVErS`
   - アイコン: プラットフォームのロゴをアップロード
   - これはプロジェクトオーナーが OAuth 認可する際に表示される

### 3-2. OAuth 設定

1. **Connect** → **設定** → **OAuth** セクションを開く
2. **リダイレクト URI** を追加:

```
https://your-domain.example.com/api/stripe/connect/callback
```

開発環境の場合:
```
http://localhost:8080/api/stripe/connect/callback
```

> 複数のリダイレクト URI を登録可能。開発用と本番用を両方追加しておくと便利。

3. **`client_id`（`ca_...`）** をメモ:
   - この値はテスト/本番で共通（環境は API キーで決まる）
   - 環境変数 `STRIPE_CONNECT_CLIENT_ID` として使用

### 3-3. Connect アカウントタイプの確認

Stripe ダッシュボードの Connect 設定で:

- **アカウントタイプ**: 「Standard」が選択されていることを確認
- **ブランディング**: プラットフォーム名とアイコンが正しいことを確認

### OAuth 認可画面のイメージ

プロジェクトオーナーが Connect を承認する際に表示される画面:

```
┌─────────────────────────────────────┐
│                                     │
│  [GIVErS ロゴ]                      │
│                                     │
│  GIVErS が以下のアクセスを          │
│  リクエストしています:              │
│                                     │
│  ✓ お支払いの受け取り               │
│  ✓ アカウント情報の読み取り         │
│                                     │
│  [許可する]  [拒否する]             │
│                                     │
└─────────────────────────────────────┘
```

---

## 4. Webhook エンドポイントの登録

Webhook は Stripe から GIVErS サーバーへのイベント通知です。
寄付の確定、サブスクリプションの作成/解約などをリアルタイムで受信します。

### 4-1. エンドポイントを追加

1. Stripe ダッシュボード → **開発者** → **Webhook** → **エンドポイントを追加**
2. URL を入力:

```
https://your-domain.example.com/api/webhooks/stripe
```

3. **受信するイベント** を選択:

| イベント | 用途 |
|----------|------|
| `payment_intent.succeeded` | 一回寄付の決済完了 → donations テーブルに記録 |
| `payment_intent.payment_failed` | 決済失敗 → エラーログ記録 |
| `customer.subscription.created` | 定期寄付の開始 → recurring_donations テーブルに記録 |
| `customer.subscription.deleted` | 定期寄付の解約 → ステータス更新 |
| `account.updated` | Connected Account の状態変更（KYC 完了等） |

4. **署名シークレット**（`whsec_...`）をメモ → `STRIPE_WEBHOOK_SECRET`

### 4-2. Connect Webhook について

Connect Standard では、Connected Account のイベントを受信するために:

- Webhook の「Connect イベントを受信する」オプションを有効にする
- または、Connected Account ごとに Webhook を登録する（推奨しない）

> **注意**: Stripe ダッシュボードで Webhook 登録時に
> 「Connect アプリケーションからのイベント」のチェックボックスがある場合は有効にする。

---

## 5. 環境変数の設定

### 必要な環境変数

```env
# Stripe API キー
STRIPE_SECRET_KEY=sk_test_...          # テスト環境用
# STRIPE_SECRET_KEY=sk_live_...        # 本番環境用

# Stripe Connect
STRIPE_CONNECT_CLIENT_ID=ca_...        # Connect OAuth の client_id

# Stripe Webhook
STRIPE_WEBHOOK_SECRET=whsec_...        # Webhook 署名検証用シークレット
```

### コード側の読み込み

`backend/cmd/server/main.go`:
```go
stripeClient := pkgstripe.NewClient(
    os.Getenv("STRIPE_SECRET_KEY"),
    os.Getenv("STRIPE_CONNECT_CLIENT_ID"),
    os.Getenv("STRIPE_WEBHOOK_SECRET"),
)
```

### Stripe 未設定時の動作

環境変数が空の場合:
- `GenerateConnectURL()` → 空文字を返す（Connect ボタンが非表示）
- `CreateCheckoutSession()` → `stripe: not configured` エラー
- `VerifyWebhookSignature()` → `stripe: not configured` エラー

つまり、Stripe を設定しなくてもアプリは起動し、寄付以外の機能は使える。

---

## 5.5. サービスホストのプロジェクト（Connect 不要）

サービスホスト（`HOST_EMAILS` 環境変数に含まれるユーザー）がプロジェクトを作成する場合、
Stripe Connect OAuth は不要です。

### ホストプロジェクトの動作

| 項目 | 一般オーナー | サービスホスト |
|------|-------------|---------------|
| ログイン方法 | Google/GitHub OAuth2 | Google/GitHub OAuth2（同一） |
| プロジェクト作成時の status | `draft` | `active`（即時公開） |
| Stripe Connect OAuth | 必須 | 不要（スキップ） |
| 使用する Stripe アカウント | オーナー自身の Connected Account (`acct_...`) | プラットフォームの Stripe アカウント |
| `stripe_account_id` | `acct_...`（Connected Account） | 空（NULL） |
| 決済時の `Stripe-Account` ヘッダー | 設定する | 設定しない（プラットフォーム直接入金） |
| 入金先 | オーナーの銀行口座 | プラットフォーム運営者の銀行口座 |

### 判定の仕組み

```
ユーザーがログイン
  │
  ▼
HostMiddleware: メールアドレスが HOST_EMAILS に含まれるか？
  │
  ├── YES → コンテキストに is_host=true を設定
  │         → プロジェクト作成時: status=active, Connect URL なし
  │         → 決済時: Stripe-Account ヘッダー省略（プラットフォーム口座へ直接入金）
  │
  └── NO  → 通常のプロジェクトオーナーフロー
            → プロジェクト作成時: status=draft, Connect URL を返す
            → 決済時: Stripe-Account ヘッダーに acct_... を設定
```

### 資金の流れ（ホストプロジェクト）

```
寄付者
  │
  │  Stripe Checkout で決済（Stripe-Account ヘッダーなし）
  ▼
┌────────────────────────────────┐
│  プラットフォームの Stripe      │   ← 寄付金はプラットフォームに入る
│  アカウント                     │
│ （STRIPE_SECRET_KEY の持ち主）  │
└────────────────────────────────┘
        │
        │ Stripe 決済手数料 3.6% のみ差し引き
        ▼
  プラットフォーム運営者の銀行口座に自動入金
```

---

## 6. プロジェクトオーナーの連携フロー

### フロー全体図

```
1. オーナーがプロジェクトを新規作成
   │
   ▼
2. サーバーが Connect OAuth URL を生成
   GET https://connect.stripe.com/oauth/authorize?
     response_type=code&
     client_id=ca_...&
     scope=read_write&
     state={project_id}
   │
   ▼
3. オーナーが Stripe の OAuth 画面で「許可する」をクリック
   │
   ▼
4. Stripe がコールバック URL にリダイレクト
   GET /api/stripe/connect/callback?code=ac_...&state={project_id}
   │
   ▼
5. サーバーが code を stripe_account_id (acct_...) に交換
   POST https://connect.stripe.com/oauth/token
   │
   ▼
6. stripe_account_id を projects テーブルに保存
   UPDATE projects SET stripe_account_id = 'acct_...' WHERE id = ?
   │
   ▼
7. オーナーをプロジェクトページにリダイレクト
   → /projects/{id}?stripe_connected=1
```

### バックエンドの実装

**Connect URL 生成** (`backend/pkg/stripe/client.go`):
```go
func (c *RealClient) GenerateConnectURL(projectID string) string {
    if c.ConnectClientID == "" {
        return ""
    }
    v := url.Values{}
    v.Set("response_type", "code")
    v.Set("client_id", c.ConnectClientID)
    v.Set("scope", "read_write")
    v.Set("state", projectID)        // CSRF 対策 + プロジェクト紐付け
    return "https://connect.stripe.com/oauth/authorize?" + v.Encode()
}
```

**コールバック処理** (`backend/internal/handler/stripe_handler.go`):
```go
func (h *StripeHandler) ConnectCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    projectID := r.URL.Query().Get("state")

    // code → acct_... に交換し、projects テーブルに保存
    if err := h.svc.CompleteConnect(r.Context(), code, projectID); err != nil {
        http.Redirect(w, r, frontendURL+"/projects/"+projectID+"?stripe_error=1", 302)
        return
    }
    http.Redirect(w, r, frontendURL+"/projects/"+projectID+"?stripe_connected=1", 302)
}
```

**コード交換** (`backend/pkg/stripe/client.go`):
```go
func (c *RealClient) ExchangeConnectCode(ctx context.Context, code string) (string, error) {
    // POST https://connect.stripe.com/oauth/token
    // Basic Auth: sk_test_... (or sk_live_...)
    // Body: code=ac_...&grant_type=authorization_code
    // Response: { "stripe_user_id": "acct_..." }
    return result.StripeUserID, nil
}
```

### データベース

```sql
-- migration 015
ALTER TABLE projects ADD COLUMN IF NOT EXISTS stripe_account_id VARCHAR(255);
```

`stripe_account_id` が NULL のプロジェクトは寄付を受け付けられない。

---

## 7. 寄付の決済フロー

### 一回寄付

```
寄付者がフォームで金額を入力
  │
  ▼
POST /api/donations/checkout
  { project_id, amount, currency: "jpy" }
  │
  ▼
サーバー: Stripe Checkout Session を作成
  mode: "payment"
  Stripe-Account: {project の acct_...}  ← Connected Account に直接入金
  metadata: { project_id, donor_type, donor_id, message }
  │
  ▼
寄付者を Stripe Checkout ページにリダイレクト
  https://checkout.stripe.com/c/pay/...
  │
  ▼
寄付者がカード情報を入力して決済
  │
  ▼
Stripe → Webhook: payment_intent.succeeded
  │
  ▼
サーバー: donations テーブルに記録 + アクティビティ追加
```

### 定期寄付（月額サポート）

```
寄付者がフォームで金額を入力（月額）
  │
  ▼
POST /api/donations/checkout
  { project_id, amount, currency: "jpy", is_recurring: true }
  │
  ▼
サーバー: Stripe Checkout Session を作成
  mode: "subscription"
  line_items[0][price_data][recurring][interval]: "month"
  Stripe-Account: {project の acct_...}
  │
  ▼
（以降は一回寄付と同様だが、毎月自動課金される）
```

### 重要: Product/Price の動的生成

Stripe の Subscription には Product と Price が必要ですが、
**Stripe Checkout の `price_data` パラメータ** を使うことで、
事前にダッシュボードで商品を作る必要はありません。

```go
// CreateCheckoutSession 内
data.Set("line_items[0][price_data][product_data][name]", "月次サポート")
data.Set("line_items[0][price_data][currency]", "jpy")
data.Set("line_items[0][price_data][unit_amount]", "1000")  // ¥1,000
data.Set("line_items[0][price_data][recurring][interval]", "month")
```

Stripe が自動的に Product と Price を Connected Account 上に作成します。

---

## 8. テスト環境での動作確認

### 8-1. Stripe CLI で Webhook をテスト

ローカル開発ではサーバーが公開されていないため、Stripe CLI でトンネルを張ります。

```bash
# Stripe CLI のインストール
# macOS
brew install stripe/stripe-cli/stripe

# Linux
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | \
  gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | \
  sudo tee /etc/apt/sources.list.d/stripe.list
sudo apt update && sudo apt install stripe

# ログイン
stripe login

# Webhook をローカルサーバーに転送
stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

出力される `whsec_...` を `.env` の `STRIPE_WEBHOOK_SECRET` に設定。

### 8-2. テスト用 Connect フロー

1. `.env` に テスト用キーを設定:
   ```env
   STRIPE_SECRET_KEY=sk_test_...
   STRIPE_CONNECT_CLIENT_ID=ca_...
   STRIPE_WEBHOOK_SECRET=whsec_...   # stripe listen の出力値
   ```

2. サーバーを起動:
   ```bash
   cd backend && go run ./cmd/server
   ```

3. プロジェクトを作成 → Connect URL にリダイレクトされる

4. テストモードでは **Stripe のテスト Connect アカウント** で OAuth を完了できる:
   - 「Skip this account form」（テスト用の簡略化フォーム）が表示される
   - 完了すると `acct_...` がプロジェクトに紐付く

### 8-3. テスト用カード番号

| カード番号 | 結果 |
|-----------|------|
| `4242 4242 4242 4242` | 決済成功 |
| `4000 0000 0000 0002` | 決済失敗（カード拒否） |
| `4000 0025 0000 3155` | 3D セキュア認証を要求 |

有効期限: 未来の任意の日付、CVC: 任意の3桁

### 8-4. Webhook イベントの手動トリガー

```bash
# 決済成功イベントをトリガー
stripe trigger payment_intent.succeeded

# サブスクリプション作成イベント
stripe trigger customer.subscription.created
```

---

## 9. 本番切り替え

### チェックリスト

- [ ] Stripe アカウントの本人確認が完了している
- [ ] `.env` のキーをテスト用（`sk_test_`）→ 本番用（`sk_live_`）に切り替え
- [ ] Stripe ダッシュボードで **本番モード** の Webhook エンドポイントを登録
  - テスト用とは別に、本番用 `whsec_...` が発行される
- [ ] Connect のリダイレクト URI に本番ドメインが登録されている
- [ ] `STRIPE_WEBHOOK_SECRET` を本番用に更新

### 環境変数（本番）

```env
STRIPE_SECRET_KEY=sk_live_...
STRIPE_CONNECT_CLIENT_ID=ca_...          # テスト/本番共通
STRIPE_WEBHOOK_SECRET=whsec_...          # 本番 Webhook 用
```

### 注意事項

- **テストキーと本番キーを混在させない**。テストで作られたデータは本番に存在しない。
- Connect Standard では、オーナーのアカウントもテスト/本番が分かれる。
  テストで Connect したオーナーは、本番では再度 Connect が必要。
- Webhook エンドポイントはテスト用と本番用で別々に管理すること。

---

## 10. トラブルシューティング

### Connect OAuth が失敗する

| 症状 | 原因 | 対処 |
|------|------|------|
| `invalid_redirect_uri` | リダイレクト URI が未登録 | Stripe ダッシュボード → Connect → 設定 → OAuth で URI を追加 |
| `invalid_client` | `STRIPE_CONNECT_CLIENT_ID` が間違っている | `ca_...` の値を確認 |
| Connect ページが表示されない | `STRIPE_CONNECT_CLIENT_ID` が空 | `.env` に設定して再起動 |
| `stripe_error=1` でリダイレクト | code 交換に失敗 | サーバーログを確認。`STRIPE_SECRET_KEY` が正しいか確認 |

### Webhook が受信できない

| 症状 | 原因 | 対処 |
|------|------|------|
| Webhook ログに `HTTP 400` | 署名検証失敗 | `STRIPE_WEBHOOK_SECRET` がエンドポイントの `whsec_...` と一致しているか確認 |
| Webhook ログに `HTTP 404` | URL が間違っている | `/api/webhooks/stripe` にパスが正しいか確認 |
| ローカルで Webhook が来ない | トンネルなし | `stripe listen --forward-to localhost:8080/api/webhooks/stripe` を実行 |
| イベントが処理されない | 未対応のイベント | `stripe_service.go` の `ProcessWebhook` で対応イベントを確認 |

### 寄付が記録されない

1. `STRIPE_WEBHOOK_SECRET` が `.env` に設定されているか確認（コメントアウトされていないか）
2. Stripe ダッシュボード → **イベントとログ** で Webhook の送信状態を確認
3. サーバーログで `payment_intent.succeeded` の処理結果を確認
4. metadata に `project_id` が含まれているか確認
5. `donations` テーブルを直接確認:
   ```sql
   SELECT * FROM donations ORDER BY created_at DESC LIMIT 5;
   ```

> **注意: Checkout Session の metadata は PaymentIntent / Subscription にコピーされない**
>
> Stripe Checkout Session 作成時に `metadata[key]` で設定した値は Checkout Session オブジェクトにのみ保存される。
> Webhook で受信する `payment_intent.succeeded` の PaymentIntent や `customer.subscription.created` の Subscription には **自動的にコピーされない**。
>
> 正しい設定方法:
> - 一回寄付: `payment_intent_data[metadata][key]` を使用
> - 定期寄付: `subscription_data[metadata][key]` を使用
>
> ```go
> // NG: Checkout Session にのみ保存される
> data.Set("metadata[project_id]", projectID)
>
> // OK: PaymentIntent に伝播される
> data.Set("payment_intent_data[metadata][project_id]", projectID)
>
> // OK: Subscription に伝播される
> data.Set("subscription_data[metadata][project_id]", projectID)
> ```

### Connected Account に入金されない

- **Stripe Connect Standard では入金はオーナーの Stripe アカウント設定に依存**
- オーナーが Stripe ダッシュボードで銀行口座を設定しているか確認
- Stripe の自動入金スケジュール（通常 2〜7 営業日）を確認

---

## 関連ファイル

| ファイル | 内容 |
|----------|------|
| `backend/pkg/stripe/client.go` | Stripe API クライアント（raw HTTP、SDK 不使用） |
| `backend/internal/service/stripe_service.go` | ビジネスロジック（Connect, Checkout, Webhook） |
| `backend/internal/handler/stripe_handler.go` | HTTP ハンドラー（3エンドポイント） |
| `backend/migrations/015_add_stripe_to_projects.up.sql` | `stripe_account_id` カラム追加 |
| `.env.example` | 環境変数テンプレート |
| `docs/setup/launch-setup-order.md` | 本番リリース前の設定順序 |

---

## API エンドポイント一覧

| メソッド | パス | 用途 |
|----------|------|------|
| `GET` | `/api/stripe/connect/callback` | OAuth コールバック（Stripe → GIVErS） |
| `POST` | `/api/donations/checkout` | Checkout Session 作成 → URL を返す |
| `POST` | `/api/webhooks/stripe` | Webhook 受信（署名検証 + イベント処理） |
