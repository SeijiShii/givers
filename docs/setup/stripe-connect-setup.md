# Stripe Connect 設定ガイド（Accounts v2 API）

> **2026-02 更新**: Stripe Connect Standard（OAuth）から **Accounts v2 API** に移行しました。
> `STRIPE_CONNECT_CLIENT_ID`（`ca_...`）は不要です。`STRIPE_SECRET_KEY` のみで動作します。

## 目次

1. [アーキテクチャ概要](#1-アーキテクチャ概要)
2. [ホスト側の Stripe アカウント作成](#2-ホスト側の-stripe-アカウント作成)
3. [Connect（Accounts v2 API）の設定](#3-connectaccounts-v2-apiの設定)
4. [Webhook エンドポイントの登録](#4-webhook-エンドポイントの登録)
5. [環境変数の設定](#5-環境変数の設定)
6. [プロジェクトオーナーのオンボーディングフロー](#6-プロジェクトオーナーのオンボーディングフロー)
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

## 3. Connect（Accounts v2 API）の設定

GIVErS は **Stripe Accounts v2 API** を使用して連結アカウントを管理します。
旧 Connect Standard（OAuth / `ca_...`）は使用しません。

### v2 API のフロー

1. プロジェクト作成時に `POST /v2/core/accounts` で連結アカウントを自動作成
2. `POST /v2/core/account_links` でオンボーディング URL を生成
3. プロジェクトオーナーが Stripe のオンボーディングページで本人確認・銀行口座を設定
4. 完了後に `GET /api/stripe/onboarding/return` にリダイレクト → プロジェクトが `active` に

### v2 アカウント作成パラメータ

| パラメータ | 値 | 説明 |
|-----------|-----|------|
| `dashboard` | `full` | オーナーが Stripe ダッシュボードにフルアクセス可能 |
| `identity.country` | `jp` | アカウントの国 |
| `identity.entity_type` | `individual` | 個人事業主 |
| `defaults.responsibilities.fees_collector` | `stripe` | Stripe が手数料を直接徴収 |
| `defaults.responsibilities.losses_collector` | `stripe` | 損失は Stripe が負担 |
| `configuration.merchant.capabilities.card_payments.requested` | `true` | カード決済を有効化 |

### Stripe ダッシュボードでの Connect 設定

1. Stripe ダッシュボード → **Connect** → **設定** を開く
2. **プラットフォームプロフィール** を設定:
   - プラットフォーム名: `GIVErS`
   - アイコン: プラットフォームのロゴをアップロード

> **注意**: v2 API では OAuth 設定（`client_id` / リダイレクト URI）は不要です。
> アカウント作成・オンボーディングはすべて API 経由で行います。

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

| イベント | 用途 | 実装状況 |
|----------|------|----------|
| `payment_intent.succeeded` | 一回寄付の決済完了 → donations テーブルに記録 | 実装済み |
| `customer.subscription.created` | 定期寄付の開始 → recurring_donations テーブルに記録 | 実装済み |
| `customer.subscription.deleted` | 定期寄付の解約 → レコード削除 | 実装済み |
| `invoice.payment_succeeded` | 定期寄付の継続課金成功 → 次回メッセージの記録・クリア | 実装済み |
| `checkout.session.completed` | Checkout Session 完了通知（Stripe 推奨） | 未実装（検討中） |
| `checkout.session.expired` | Checkout Session 期限切れ | 未実装（検討中） |

> **注意**: 上記以外のイベント（`payment_intent.payment_failed`, `account.updated` 等）は
> 現在のコードでは処理していません。受信しても無視されます（エラーにはなりません）。
> 必要に応じて `stripe_service.go` の `ProcessWebhook` に追加してください。

4. **署名シークレット**（`whsec_...`）をメモ → `STRIPE_WEBHOOK_SECRET`

### 4-2. Connect Webhook について

Connected Account のイベントを受信するために:

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

# Stripe Webhook
STRIPE_WEBHOOK_SECRET=whsec_...        # Webhook 署名検証用シークレット

# ※ STRIPE_CONNECT_CLIENT_ID は v2 API 移行により廃止。不要です。
```

### コード側の読み込み

`backend/cmd/server/main.go`:
```go
stripeClient := pkgstripe.NewClient(
    os.Getenv("STRIPE_SECRET_KEY"),
    os.Getenv("STRIPE_WEBHOOK_SECRET"),
)
```

### Stripe 未設定時の動作

環境変数が空の場合:
- `CreateConnectedAccount()` → `stripe: not configured` エラー
- `CreateCheckoutSession()` → `stripe: not configured` エラー
- `VerifyWebhookSignature()` → `stripe: not configured` エラー

つまり、Stripe を設定しなくてもアプリは起動し、寄付以外の機能は使える。

---

## 5.5. サービスホストのプロジェクト（Connect 不要）

サービスホスト（`HOST_EMAILS` 環境変数に含まれるユーザー）がプロジェクトを作成する場合、
Stripe Connect オンボーディングは不要です。

### ホストプロジェクトの動作

| 項目 | 一般オーナー | サービスホスト |
|------|-------------|---------------|
| ログイン方法 | Google/GitHub OAuth2 | Google/GitHub OAuth2（同一） |
| プロジェクト作成時の status | `draft` | `active`（即時公開） |
| Stripe Connect | v2 API でアカウント作成 + オンボーディング | 不要（スキップ） |
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
            → プロジェクト作成時: status=draft, オンボーディング URL を返す
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

## 6. プロジェクトオーナーのオンボーディングフロー

### ユーザー向けフロー案内（プロジェクト作成画面・FAQ に表示）

プロジェクト作成フォームの下部に、一般オーナー（ホスト以外）にのみ以下の案内が表示される。
ホスト（`HOST_EMAILS` に含まれるユーザー）にはこのセクション全体が非表示。
同じ内容を FAQ ページ（Q9）にも掲載。

> **Stripe アカウントの接続について**
>
> 寄付金を受け取るには、Stripe（決済サービス）のアカウント設定が必要です。
> GIVErS が手数料を取ることはありません。
>
> **プロジェクト作成後の流れ**
> 1. このフォームを送信すると、Stripe の設定ページに移動します
> 2. Stripe のページで本人確認と銀行口座の登録を行います
> 3. 設定が完了すると自動的に GIVErS に戻り、プロジェクトが公開されます
> 4. 寄付金は Stripe 手数料（3.6%）を差し引いた額があなたの銀行口座に直接入金されます。GIVErS は手数料を一切取りません

#### 表示条件

| ユーザー種別 | 表示 | 理由 |
|------------|------|------|
| ホスト | 非表示 | Connect 不要（プラットフォーム口座に直接入金） |
| 一般オーナー | 表示 | Stripe オンボーディングが必要 |
| 既存プロジェクト編集時 | 非表示 | オンボーディングは初回のみ |

実装: `ProjectForm.tsx` で `getMe()` を呼び `role === "host"` を判定。
i18n キー: `projects.stripeConnectTitle`, `stripeConnectDesc`, `stripeConnectFlowTitle`, `stripeConnectStep1`〜`Step4`

### フロー全体図（v2 API）

```
1. オーナーがプロジェクトを新規作成
   │
   ▼
2. サーバーが v2 API で Connected Account を作成
   POST https://api.stripe.com/v2/core/accounts
   Body: { contact_email, display_name, dashboard: "full", ... }
   Response: { "id": "acct_..." }
   │
   ▼
3. stripe_account_id を projects テーブルに保存
   UPDATE projects SET stripe_account_id = 'acct_...' WHERE id = ?
   │
   ▼
4. サーバーが Account Link（オンボーディング URL）を生成
   POST https://api.stripe.com/v2/core/account_links
   Body: { account: "acct_...", use_case: { type: "account_onboarding", ... } }
   Response: { "url": "https://connect.stripe.com/setup/..." }
   │
   ▼
5. オーナーが Stripe のオンボーディングページで本人確認・銀行口座を設定
   │
   ▼
6. 完了後に return URL にリダイレクト
   GET /api/stripe/onboarding/return?project_id={id}
   │
   ▼
7. サーバーが v2 API でオンボーディング完了を確認
   GET https://api.stripe.com/v2/core/accounts/{acct_id}?include=requirements
   requirements.currently_due が空 → プロジェクトを active に
   │
   ▼
8. フロントエンドのプロジェクトページにリダイレクト
   → /projects/{id}?stripe_connected=1
```

### バックエンドの実装

**アカウント作成** (`backend/pkg/stripe/client.go`):
```go
func (c *RealClient) CreateConnectedAccount(ctx context.Context, params CreateAccountParams) (string, error) {
    body := map[string]any{
        "contact_email": params.Email,
        "display_name":  params.DisplayName,
        "dashboard":     "full",
        "identity": map[string]any{
            "country":     "jp",
            "entity_type": "individual",
        },
        // ... capabilities, defaults
    }
    // POST https://api.stripe.com/v2/core/accounts (JSON body)
    // Stripe-Version: 2025-04-30.basil
    return result.ID, nil  // "acct_..."
}
```

**Account Link 生成** (`backend/pkg/stripe/client.go`):
```go
func (c *RealClient) CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error) {
    body := map[string]any{
        "account": accountID,
        "use_case": map[string]any{
            "type": "account_onboarding",
            "account_onboarding": map[string]any{
                "configurations": []string{"merchant"},
                "return_url":     returnURL,
                "refresh_url":    refreshURL,
            },
        },
    }
    // POST https://api.stripe.com/v2/core/account_links (JSON body)
    return result.URL, nil
}
```

**オンボーディング完了確認** (`backend/pkg/stripe/client.go`):
```go
func (c *RealClient) GetAccountOnboarded(ctx context.Context, accountID string) (bool, error) {
    // GET https://api.stripe.com/v2/core/accounts/{id}?include=requirements
    // requirements.currently_due が空なら完了
    return len(result.Requirements.CurrentlyDue) == 0, nil
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
  payment_intent_data[metadata]: { project_id, donor_type, donor_id, message }
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
  subscription_data[metadata]: { project_id, donor_type, donor_id, message }
  │
  ▼
（以降は一回寄付と同様だが、毎月自動課金される）
  │
  ▼
毎月の課金成功時:
  Stripe → Webhook: invoice.payment_succeeded
  → 次回メッセージの記録・クリア処理
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

### 8-2. テスト用 Connect フロー（v2 API）

1. `.env` に テスト用キーを設定:
   ```env
   STRIPE_SECRET_KEY=sk_test_...
   STRIPE_WEBHOOK_SECRET=whsec_...   # stripe listen の出力値
   ```

2. サーバーを起動:
   ```bash
   cd backend && go run ./cmd/server
   ```

3. プロジェクトを作成 → v2 API でアカウントが自動作成され、オンボーディング URL にリダイレクトされる

4. テストモードでは Stripe のオンボーディングフォームが簡略化される:
   - テスト用の情報が自動入力される
   - 完了すると `acct_...` のオンボーディングが完了し、プロジェクトが `active` に

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
- [ ] `STRIPE_WEBHOOK_SECRET` を本番用に更新

### 環境変数（本番）

```env
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...          # 本番 Webhook 用
```

### 注意事項

- **テストキーと本番キーを混在させない**。テストで作られたデータは本番に存在しない。
- v2 API で作成した Connected Account もテスト/本番で分かれる。
  テストで作成したアカウントは本番では再度作成・オンボーディングが必要。
- Webhook エンドポイントはテスト用と本番用で別々に管理すること。

---

## 10. トラブルシューティング

### オンボーディングが完了しない

| 症状 | 原因 | 対処 |
|------|------|------|
| オンボーディング URL にリダイレクトされない | アカウント作成に失敗 | サーバーログで `stripe create account` エラーを確認。`STRIPE_SECRET_KEY` が正しいか確認 |
| オンボーディング後も `draft` のまま | `requirements.currently_due` が残っている | オーナーに追加情報の入力を依頼。Stripe ダッシュボードで Connected Account の状態を確認 |
| `stripe_error=1` でリダイレクト | Account Link 作成 or オンボーディング確認に失敗 | サーバーログを確認。`STRIPE_SECRET_KEY` が正しいか確認 |
| リンク期限切れ | Account Link は一定時間で無効になる | `/api/stripe/onboarding/refresh` にアクセスして新しいリンクを生成 |

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

- v2 API で作成した Connected Account でも入金はオーナーの Stripe アカウント設定に依存
- オーナーが Stripe ダッシュボードで銀行口座を設定しているか確認
- Stripe の自動入金スケジュール（通常 2〜7 営業日）を確認

---

## 関連ファイル

| ファイル | 内容 |
|----------|------|
| `backend/pkg/stripe/client.go` | Stripe API クライアント（raw HTTP、SDK 不使用） |
| `backend/internal/service/stripe_service.go` | ビジネスロジック（Connect, Checkout, Webhook） |
| `backend/internal/handler/stripe_handler.go` | HTTP ハンドラー（オンボーディング + Webhook） |
| `backend/migrations/015_add_stripe_to_projects.up.sql` | `stripe_account_id` カラム追加 |
| `.env.example` | 環境変数テンプレート |
| `docs/setup/launch-setup-order.md` | 本番リリース前の設定順序 |

---

## API エンドポイント一覧

| メソッド | パス | 用途 |
|----------|------|------|
| `GET` | `/api/stripe/onboarding/return` | オンボーディング完了後のリターン URL |
| `GET` | `/api/stripe/onboarding/refresh` | オンボーディング再開（リンク期限切れ時） |
| `POST` | `/api/donations/checkout` | Checkout Session 作成 → URL を返す |
| `POST` | `/api/webhooks/stripe` | Webhook 受信（署名検証 + イベント処理） |
