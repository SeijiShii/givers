# TODO — GIVErS プラットフォーム

最終更新: 2026-03-01

---

## 実装 TODO

### 9. SNS シェア機能 — フロント残作業

バックエンド（`share_message` カラム・API）は実装済み。フロント側の接続が残っている。

- [ ] **フロント — ProjectDetail**: DB の `share_message` をダイアログ初期値に使用
  - 優先順位: localStorage 保存済みメッセージ > DB share_message > プロジェクト名
- [ ] **フロント — プロジェクト編集フォーム**: シェアメッセージ入力欄を追加

### 11. description → overview マイグレーション残作業

バックエンド API・フロント型定義・新規プロジェクトフォームは対応済み。

- [ ] **マイグレーション**: 既存 `description` データを `overview` に移行（既存データがある場合）

### 13. メール送信

- 問い合わせ通知、マジックリンク認証等
- 初期はメールログインなし（追って決める）
- プロバイダー未選定（SendGrid / SES / Resend 等）

---

## フロント改善

- [ ] 新規作成ページで最低金額、見積金額を選ぶのではなく、すべて入力の一択。どちらかに有効な値が入力されていればOK
- [ ] 詳細ページで寄付セクションへスクロールするボタン
- [ ] 詳細ページで見積金額を変更。最低金額も変更できるようにする。
- [ ] 詳細ページチャートで最低金額に見積金額が表示されている不具合
- [ ] 寄付発生による更新を10秒おきなどに行えるか（それとも更新ボタン？）

---

## Stripe 本番移行セキュリティ対策

> 詳細: [setup/stripe-production-checklist.md](../setup/stripe-production-checklist.md)

### アプリ層（P1: 高優先度）

- [ ] **A11. ログの機密情報除去**: `auth_handler.go` の userID・トークンプレフィックスのログ出力を修正

### アプリ層（P2: 中優先度）

- [ ] **A10. Statement Descriptor**: Checkout Session に `statement_descriptor_suffix` を追加
- [ ] **A13. Stripe エラー型区別**: `card_error` / `invalid_request_error` を区別して適切にハンドリング

### インフラ層（P0: 必須 — 本番稼働の前提条件）

- [ ] **B5. TLS/HTTPS**: nginx + Let's Encrypt で SSL 終端、HTTP→HTTPS リダイレクト
- [ ] **B6. 本番 Webhook 登録**: Stripe Dashboard で本番 URL を登録、`STRIPE_WEBHOOK_SECRET` を更新
- [ ] **B7. API キー切替**: `sk_test_` → `sk_live_` に切替、テスト用キーの残存箇所を確認
- [ ] **B8. 2FA 有効化**: Stripe アカウントの二要素認証を有効化

### インフラ層（P1-P2: 高〜中優先度）

- [ ] **B9. DB 接続 SSL 化**: `DATABASE_URL` を `sslmode=require` + 強固なパスワードに変更
- [ ] **B10. PCI コンプライアンス**: Stripe Dashboard で SAQ-A を提出（年次）
- [ ] **B11. Statement Descriptor (Dashboard)**: ビジネス名・明細表示名を設定
- [ ] **B12. チャージバック対策**: 不正取引レビュー・異議申し立て対応フローを整備

---

## インフラ・運用

- [ ] **本番インフラ**: ConoHa に決定 — ローカル動作確認後に構築
- [ ] **`docker-compose.prod.yml`** 作成（本番デプロイ用）


## 実装 TODO（続き）

### 19. プロジェクト詳細でのサブスクリプション寄付管理

プロジェクト詳細画面で、寄付者が既にサブスク寄付中の場合、新規寄付フォームの代わりにサブスク管理 UI を表示する。

> MePage でのサブスク管理（金額変更・一時停止・再開・削除）は実装済み。
> バックエンド（寄付 CRUD・Stripe pause/resume/cancel・Webhook）も実装済み。

#### バックエンド（追加実装）

- [ ] **Stripe 金額変更連携**: `DonationService.Patch` で amount 変更時に Stripe Subscription の price も更新する
  - `pkg/stripe/client.go` に `UpdateSubscriptionAmount` メソッドを追加
- [ ] **次回決済メッセージ**: `donations` テーブルに `next_billing_message TEXT` カラムを追加
  - `DonationPatch` に `NextBillingMessage` フィールド追加
  - `PATCH /api/me/donations/:id` で `next_billing_message` を受付
  - Webhook: `invoice.payment_succeeded` 時にメッセージをアクティビティに記録→カラムをクリア

#### フロントエンド

- [ ] **ProjectDetail — サブスク検知**: `getMyRecurringDonations()` で現プロジェクトへのアクティブサブスクを検出
- [ ] **ProjectDetail — 管理 UI 切替**: サブスク検出時は `DonateForm` の代わりに `SubscriptionManageForm` を表示
  - 現在の金額・状態の表示
  - 金額変更フォーム
  - 一時停止 / 再開トグル
  - キャンセルボタン（確認ダイアログ付き）
  - 次回決済メッセージ入力テキストエリア
- [ ] **api.ts**: `updateRecurringDonation` に `next_billing_message` フィールドを追加

### 20. 寄付メッセージ閲覧（プロジェクトオーナー向け）

プロジェクトオーナーが寄付者からのメッセージを閲覧する UI を実装する。

> DB の `donations.message` / `activities.message` カラム、寄付作成時のメッセージ入力（DonateForm）、`DonationRepository.ListByProject()` は実装済み。

#### バックエンド

- [ ] **メッセージ一覧 API**: `GET /api/projects/:id/messages` を新設（オーナー認証必須）
  - クエリパラメータ: `limit`（50）, `offset`（0）, `sort`（`asc` / `desc`）, `donor`（寄付者名フィルタ）
  - レスポンス: `{ messages: [{ donor_name, amount, message, created_at, is_recurring }], total }`
  - `DonationRepository` にメッセージ付き寄付を JOIN 取得するメソッドを追加

#### フロントエンド

- [ ] **ProjectDetail — メッセージタブ**: オーナーのみ表示される「メッセージ」タブを追加
  - メッセージ一覧（寄付者名・金額・本文・日時）
  - 日時ソート切替（新しい順 / 古い順）
  - 寄付者名フィルタ入力
- [ ] **api.ts**: `getProjectMessages(projectId, params)` を追加
- [ ] **i18n**: メッセージタブ関連の翻訳キーを追加

### 21. 定期寄付にログイン必須

idea.md「継続支援はアカウント必須（解約・変更を確実に管理できるようにするため）」を実装で強制する。
現状、未ログイン（トークン）のまま `is_recurring=true` で Checkout が通ってしまう。

- [ ] **バックエンド**: `stripe_handler.go` Checkout で `is_recurring=true && donorType=="token"` を 400 で拒否
- [ ] **フロントエンド**: `DonateForm` で未ログイン時に月額ボタンを非活性化 + 「月額寄付にはログインが必要です」メッセージを表示