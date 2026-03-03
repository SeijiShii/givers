# 寄付返金機能 仕様書

## 概要

プロジェクトオーナーが自プロジェクトへの寄付を返金できる機能。加えて寄付者自身も返金を要求できる。Stripe Refund API を使い全額返金のみ対応。定期寄付の個別決済も返金可能だが、サブスク解約は寄付者が別途行う。

---

## 要件

| # | 内容 |
|---|------|
| R1 | 全額返金のみ（部分返金は対象外） |
| R2 | 単発寄付・定期寄付の各決済回（`donations` テーブルの各レコード）が返金対象 |
| R3 | プロジェクトオーナー: 寄付一覧から直接返金 |
| R4 | 寄付者: 3点メニュー（`⋯`）から返金を要求 |
| R5 | 定期寄付の決済を返金する場合、「定期寄付の解約は次回決済前に寄付者が行う必要があります」の注意書きを表示 |
| R6 | 返金によりサブスクリプションは自動キャンセルされない |
| R7 | 返金済み寄付は月次集計（チャート・ヘルスシグナル）から除外 |
| R8 | 二重返金を防止（同じ寄付に対する複数回の返金操作を排除） |

---

## API エンドポイント

### POST /api/projects/:id/donations/:donationId/refund

プロジェクトオーナー（またはホスト）が寄付を返金する。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| 認可 | プロジェクトオーナー（`project.owner_id == userID`）またはホスト |
| リクエストボディ | なし |

**レスポンス (200)**
```json
{ "ok": true }
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 401 | `{ "error": "unauthorized" }` | 未認証 |
| 403 | `{ "error": "forbidden" }` | オーナーでもホストでもない |
| 404 | `{ "error": "project_not_found" }` | プロジェクトが存在しない |
| 404 | `{ "error": "not_found" }` | 寄付レコードが存在しない |
| 409 | `{ "error": "already_refunded" }` | 返金済みまたは返金処理中 |
| 500 | `{ "error": "refund_failed" }` | Stripe API エラー / DB エラー |

---

### POST /api/me/donations/:donationId/refund

寄付者が自分の寄付を返金する。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| 認可 | 寄付者本人（`donor_type='user'` かつ `donor_id == userID`） |
| リクエストボディ | なし |

**レスポンス (200)**
```json
{ "ok": true }
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 401 | `{ "error": "unauthorized" }` | 未認証 |
| 403 | `{ "error": "forbidden" }` | 寄付者本人ではない、またはトークン寄付 |
| 404 | `{ "error": "not_found" }` | 寄付レコードが存在しない |
| 409 | `{ "error": "already_refunded" }` | 返金済みまたは返金処理中 |
| 500 | `{ "error": "refund_failed" }` | Stripe API エラー / DB エラー |

**制約**: `donor_type='token'`（匿名寄付）は寄付者側からの返金不可（ログイン必須）。オーナー側からは返金可能。

---

## DB スキーマ変更

### マイグレーション: `029_add_refund_columns`

```sql
-- up
ALTER TABLE donations ADD COLUMN IF NOT EXISTS refund_status VARCHAR(20);
ALTER TABLE donations ADD COLUMN IF NOT EXISTS stripe_refund_id TEXT;
CREATE INDEX IF NOT EXISTS idx_donations_refund_status
    ON donations(refund_status) WHERE refund_status IS NOT NULL;

-- down
DROP INDEX IF EXISTS idx_donations_refund_status;
ALTER TABLE donations DROP COLUMN IF EXISTS stripe_refund_id;
ALTER TABLE donations DROP COLUMN IF EXISTS refund_status;
```

### refund_status カラム

| 値 | 意味 |
|---|------|
| `NULL` | 未返金（デフォルト。通常の寄付状態） |
| `'pending'` | 返金処理中（DB に先行書き込み、Stripe API 呼び出し前） |
| `'completed'` | 返金完了（Stripe Refund 成功後に遷移） |

状態遷移:

```
NULL → pending → completed  （正常フロー）
NULL → pending → NULL       （Stripe API 失敗時のロールバック）
```

### stripe_refund_id カラム

Stripe Refund オブジェクトの ID（`re_...`）。`refund_status='completed'` のときのみ値が設定される。監査・問い合わせ対応用。

---

## Stripe 連携

### 返金処理フロー（2段階ステータス方式）

```
1. DB: SET refund_status = 'pending'
   WHERE refund_status IS NULL  ← 二重返金防止（行ロック）
   │
   ├─ 失敗（ErrAlreadyRefunded） → 409 を返す
   │
2. PaymentIntent ID を解決:
   ├─ stripe_payment_id があれば使用（単発寄付 or 改善後の renewal）
   └─ なければ GET /v1/invoices/{stripe_invoice_id} → payment_intent を取得
   │
3. Stripe API: POST /v1/refunds
   payment_intent={pi_...}
   Stripe-Account: {acct_...}  ← Connected Account の場合のみ
   │
   ├─ 成功 → DB: SET refund_status = 'completed',
   │              stripe_refund_id = 're_...'
   │
   └─ 失敗 → DB: SET refund_status = NULL  ← ロールバック
              → 500 を返す
```

**ポイント**:
- DB を先に書き込むことで「Stripe は返金したが DB 未反映」のケースを排除
- `WHERE refund_status IS NULL` により並行リクエストの二重返金を排除
- Stripe API 失敗時は `refund_status` を NULL に戻し、再試行可能に
- `stripe_account_id` が空のプロジェクト（ホストのプロジェクト）は `Stripe-Account` ヘッダーを省略

### Stripe Refund API

```
POST https://api.stripe.com/v1/refunds
Content-Type: application/x-www-form-urlencoded
Authorization: Basic {base64(STRIPE_SECRET_KEY:)}
Stripe-Account: acct_...   ← Connected Account の場合のみ

payment_intent=pi_...
```

レスポンス:
```json
{
  "id": "re_...",
  "status": "succeeded",
  "payment_intent": "pi_...",
  "amount": 1000,
  "currency": "jpy"
}
```

### Stripe Invoice PaymentIntent 取得（旧レコード fallback 用）

```
GET https://api.stripe.com/v1/invoices/{invoice_id}
Authorization: Basic {base64(STRIPE_SECRET_KEY:)}
Stripe-Account: acct_...   ← Connected Account の場合のみ
```

レスポンスの `payment_intent` フィールドを使用。

---

## Webhook 改善: invoice の payment_intent 保存

### 背景

現在、`invoice.payment_succeeded` Webhook で作成される renewal 寄付レコードには `stripe_payment_id` が設定されていない。返金時に毎回 Invoice API を呼ぶのは非効率。

### 変更内容

`handleInvoicePaymentSucceeded` で donation 作成時に、Invoice オブジェクトの `payment_intent` フィールドを `stripe_payment_id` として保存する。

**変更前**:
```go
d := &model.Donation{
    // ...
    StripeInvoiceID: obj.ID,
    // StripePaymentID は未設定
}
```

**変更後**:
```go
d := &model.Donation{
    // ...
    StripePaymentID: obj.PaymentIntent, // Invoice の payment_intent を保存
    StripeInvoiceID: obj.ID,
}
```

**WebhookEventObject 構造体**に `PaymentIntent` フィールドを追加:
```go
PaymentIntent string `json:"payment_intent"` // invoice イベントで使用
```

### 既存レコードへの影響

- 既存の renewal 寄付（`stripe_payment_id` が空）は引き続き Invoice API fallback で返金可能
- 改善後の新規 renewal 寄付は `stripe_payment_id` が設定済みとなり、追加 API コール不要

---

## 月次集計への影響

返金済み寄付（`refund_status = 'completed'`）は月次集計から除外する。

### 変更対象クエリ

**CurrentMonthSumByProject**（現在月の合計）:
```sql
SELECT COALESCE(SUM(amount), 0)::int FROM donations
WHERE project_id = $1
  AND created_at >= DATE_TRUNC('month', NOW())
  AND refund_status IS DISTINCT FROM 'completed'
```

**MonthlySumByProject**（12ヶ月推移）:
```sql
SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') AS month,
       SUM(amount)::int AS amount
FROM donations
WHERE project_id = $1
  AND created_at >= DATE_TRUNC('month', NOW()) - INTERVAL '11 months'
  AND refund_status IS DISTINCT FROM 'completed'
GROUP BY DATE_TRUNC('month', created_at)
ORDER BY month
```

**プロジェクト一覧の current_monthly_donations サブクエリ**:
```sql
COALESCE((SELECT SUM(amount) FROM donations
  WHERE project_id = p.id
    AND created_at >= DATE_TRUNC('month', NOW())
    AND refund_status IS DISTINCT FROM 'completed'), 0)::int
```

> `IS DISTINCT FROM 'completed'` は NULL（未返金）と `'pending'`（処理中）の両方を含む。処理中の寄付は集計に含める（Stripe 失敗時にロールバックされるため）。

---

## フロントエンド UI 仕様

### 寄付者向け（MePage — 寄付履歴）

**変更点**:
- 寄付履歴テーブルに全決済記録を表示（現在の `!d.is_recurring` フィルタを削除）
- 各行に3点メニュー（`⋯`）を追加

**行の表示パターン**:

| 状態 | 金額表示 | 操作列 |
|------|---------|--------|
| 未返金 | `¥1,000` | `⋯` → 「返金」メニュー |
| 返金処理中 | `¥1,000` | ローディングインジケータ |
| 返金済み | `~~¥1,000~~`（取消線）+ 「返金済み」バッジ | なし |

**3点メニュー**:
- クリックで小さなドロップダウンを表示
- 「返金」オプションのみ（将来的に他の操作を追加可能）
- メニュー外クリックで閉じる

**確認ダイアログ**:

通常の寄付:
> **返金の確認**
> この寄付を返金しますか？この操作は取り消せません。
> [キャンセル] [返金]

定期寄付（`is_recurring = true`）の場合、追加メッセージ:
> **返金の確認**
> この寄付を返金しますか？この操作は取り消せません。
>
> ⚠ 定期寄付の解約は次回決済前に寄付者が行う必要があります。
> [キャンセル] [返金]

**返金後**: 寄付一覧を再取得して画面を更新。

---

### オーナー向け（OwnerDonationHistory — プロジェクト寄付一覧）

**変更点**:
- テーブルに「操作」列を追加

**行の表示パターン**:

| 状態 | 金額表示 | 操作列 |
|------|---------|--------|
| 未返金 | `¥1,000` | 「返金」ボタン |
| 返金処理中 | `¥1,000` | ローディングインジケータ |
| 返金済み | `~~¥1,000~~`（取消線）+ 「返金済み」バッジ | なし |

**確認ダイアログ**:
> **返金の確認**
> この寄付を返金しますか？寄付者に全額が返金されます。この操作は取り消せません。
> [キャンセル] [返金]

**返金後**: 寄付一覧を再取得して画面を更新。

---

## API レスポンスの変更

### GET /api/projects/:id/donations（オーナー向け）

レスポンスの各寄付オブジェクトに `refund_status` フィールドを追加:

```json
{
  "donations": [
    {
      "id": "uuid",
      "donor_name": "山田太郎",
      "donor_type": "user",
      "amount": 1000,
      "currency": "jpy",
      "message": "応援しています！",
      "source": "checkout",
      "is_recurring": false,
      "refund_status": null,
      "created_at": "2026-02-15T10:00:00Z"
    },
    {
      "id": "uuid",
      "donor_name": "田中花子",
      "donor_type": "user",
      "amount": 500,
      "currency": "jpy",
      "message": null,
      "source": "subscription_renewal",
      "is_recurring": true,
      "refund_status": "completed",
      "created_at": "2026-03-01T00:00:00Z"
    }
  ],
  "total": 42
}
```

### GET /api/me/donations（寄付者向け）

レスポンスの各寄付オブジェクトに `refund_status` フィールドを追加:

```json
{
  "donations": [
    {
      "id": "uuid",
      "project_id": "uuid",
      "donor_type": "user",
      "donor_id": "user-uuid",
      "amount": 1000,
      "currency": "jpy",
      "message": "応援しています！",
      "is_recurring": false,
      "source": "checkout",
      "refund_status": null,
      "created_at": "2026-02-15T10:00:00Z",
      "updated_at": "2026-02-15T10:00:00Z"
    }
  ]
}
```

---

## i18n 文字列

### 日本語（ja.json）

**me セクション**:
| キー | 値 |
|------|-----|
| `me.refund` | `返金` |
| `me.refundConfirmTitle` | `返金の確認` |
| `me.refundConfirmMessage` | `この寄付を返金しますか？この操作は取り消せません。` |
| `me.refundSubscriptionWarning` | `定期寄付の解約は次回決済前に寄付者が行う必要があります。` |
| `me.refunded` | `返金済み` |
| `me.refundFailed` | `返金に失敗しました。時間をおいて再度お試しください。` |
| `me.alreadyRefunded` | `すでに返金済みです` |

**projects セクション**:
| キー | 値 |
|------|-----|
| `projects.donationHistoryRefund` | `返金` |
| `projects.donationHistoryRefunded` | `返金済み` |
| `projects.donationHistoryRefundConfirmTitle` | `返金の確認` |
| `projects.donationHistoryRefundConfirmMessage` | `この寄付を返金しますか？寄付者に全額が返金されます。この操作は取り消せません。` |

### 英語（en.json）

**me セクション**:
| キー | 値 |
|------|-----|
| `me.refund` | `Refund` |
| `me.refundConfirmTitle` | `Confirm Refund` |
| `me.refundConfirmMessage` | `Refund this donation? This action cannot be undone.` |
| `me.refundSubscriptionWarning` | `The donor must cancel their recurring donation before the next billing date.` |
| `me.refunded` | `Refunded` |
| `me.refundFailed` | `Refund failed. Please try again later.` |
| `me.alreadyRefunded` | `Already refunded` |

**projects セクション**:
| キー | 値 |
|------|-----|
| `projects.donationHistoryRefund` | `Refund` |
| `projects.donationHistoryRefunded` | `Refunded` |
| `projects.donationHistoryRefundConfirmTitle` | `Confirm Refund` |
| `projects.donationHistoryRefundConfirmMessage` | `Refund this donation? The full amount will be returned to the donor. This action cannot be undone.` |

---

## 変更ファイル一覧

### 新規ファイル

| ファイル | 内容 |
|---------|------|
| `backend/migrations/029_add_refund_columns.up.sql` | DB マイグレーション（refund_status, stripe_refund_id） |
| `backend/migrations/029_add_refund_columns.down.sql` | ロールバック |
| `backend/internal/service/refund_service.go` | 返金サービス（2段階ステータス方式） |
| `backend/internal/handler/refund_handler.go` | 返金ハンドラー（オーナー/寄付者） |

### 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `backend/pkg/stripe/client.go` | `CreateRefund`, `GetInvoicePaymentIntent` メソッド追加。`WebhookEventObject` に `PaymentIntent` フィールド追加 |
| `backend/internal/model/donation.go` | `RefundStatus`, `StripeRefundID` フィールド追加 |
| `backend/internal/repository/errors.go` | `ErrAlreadyRefunded` 追加 |
| `backend/internal/repository/donation_repository.go` | `SetRefundPending`, `CompleteRefund`, `ClearRefundPending` メソッド追加 |
| `backend/internal/repository/pg_donation_repository.go` | 上記メソッド実装、scan/select 更新、集計クエリ更新 |
| `backend/internal/service/stripe_service.go` | `handleInvoicePaymentSucceeded` で `StripePaymentID` 保存 |
| `backend/cmd/server/main.go` | 返金エンドポイント2つのルーティング登録 |
| `frontend/src/lib/api.ts` | 型更新、`refundDonation`/`ownerRefundDonation` 関数追加 |
| `frontend/src/lib/mock-api.ts` | モック実装追加 |
| `frontend/src/i18n/ja.json` | 返金関連文字列追加 |
| `frontend/src/i18n/en.json` | 返金関連文字列追加 |
| `frontend/src/components/react/MePage.tsx` | 3点メニュー、返金 UI、確認ダイアログ |
| `frontend/src/components/react/OwnerDonationHistory.tsx` | 返金ボタン、確認ダイアログ |
| `frontend/src/pages/me.astro` | MePage への Props 追加 |
| `frontend/src/pages/en/me.astro` | MePage への Props 追加（英語版） |
