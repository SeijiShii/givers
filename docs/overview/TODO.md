# TODO — GIVErS プラットフォーム

最終更新: 2026-02-27

---

## 完了済み

### 1. ActivityFeed バックエンド拡張 + フロント接続
- [x] `activities` テーブルのマイグレーション (017)
- [x] ActivityItem モデル拡張（id/type/project_id/project_name/actor_name/amount/rate/message/created_at）
- [x] ActivityRepository + ActivityService + ActivityHandler 新設
- [x] `GET /api/activity?limit=N`（全体フィード）新設
- [x] `GET /api/projects/{id}/activity`（プロジェクト別フィード）を activities テーブルに移行
- [x] フロント `getActivityFeed()` を実 API に接続
- [x] テスト: ActivityService 5件 + ActivityHandler 7件

### 2. ProjectChart — 月別集計 API
- [x] `GET /api/projects/{id}/chart` ハンドラー + ChartHandler
- [x] `MonthlySumByProject` (DATE_TRUNC 集計 SQL) を DonationRepository に追加
- [x] minAmount = cost_items 合計、targetAmount = monthly_target、actualAmount = 月別寄付合計
- [x] フロント `getProjectChart()` を実 API に接続
- [x] テスト: ChartHandler 4件

### 3. PLATFORM_PROJECT_ID 環境変数化
- [x] `PUBLIC_PLATFORM_PROJECT_ID` 環境変数対応（フォールバック: `"mock-4"`）
- [x] `.env.example` に追記

### 4. Project overview TEXT カラム追加
- [x] マイグレーション (018): `ALTER TABLE projects ADD COLUMN overview TEXT`
- [x] Project モデルに `Overview string` 追加
- [x] Create / Update API で overview を受け取り・保存
- [x] Get / List API で overview を返す

### 5. Activity INSERT フック
- [x] 寄付確定時（Webhook）に `donation` イベントを activities に INSERT
- [x] プロジェクト作成時に `project_created` イベントを INSERT
- [x] プロジェクト更新時に `project_updated` イベントを INSERT
- [x] マイルストーン到達時に `milestone` イベントを INSERT（月間達成率 50% / 100%）

### 6. DB セッション管理
- [x] `sessions` テーブル (migration 019) — crypto/rand 32バイト hex トークン
- [x] SessionRepository (interface + pg 実装) + SessionService (TDD: 6テスト)
- [x] `pkg/auth`: HMAC 署名 cookie → DB-backed セッション検証に切り替え
- [x] `SessionValidator` インターフェース導入、`RequireAuth` を DB 検証に変更
- [x] auth/me/stripe handler を `SessionValidator` / `SessionCreatorDeleter` に移行
- [x] Logout: DB からセッション削除 + cookie クリア
- [x] AdminUserService: suspend 時にセッション全削除（強制ログアウト）
- [x] `main.go`: `sessionSecret` 削除、`sessionSvc` に統一

### 7. 開示用データ出力 API（ドラフト）
- [x] `GET /api/admin/disclosure-export?type=user&id=...` — ユーザー情報
- [x] `GET /api/admin/disclosure-export?type=project&id=...` — プロジェクト情報
- [x] `GET /api/admin/disclosure-export?type=donation&id=...` — プロジェクト別寄付一覧 + 合計
- [x] `DonationRepository.ListByProject` 追加
- [x] テスト: 12件（既存7 + donation 5新規）
- 法的要件の確定に依存（弁護士確認後に項目追加の可能性あり）

### 8. SNS シェア機能 — フロントエンド（OGP + シェアボタン UI）
- [x] `BaseLayout.astro`: OGP メタタグ + Twitter Card メタタグ追加
- [x] `frontend/src/lib/site.ts`: `SITE_URL` 定数
- [x] `ShareButtons.tsx`: X / Facebook / LINE シェアボタン（インライン SVG）
- [x] シェアボタン押下 → メッセージ編集ダイアログ表示
- [x] localStorage によるメッセージ一時保存（フォールバック）
- [x] `projects/[id].astro`: SSR でプロジェクト情報取得 → OGP タグ生成
- [x] `index.astro` / `en/index.astro`: ランディングページにシェアボタン追加
- [x] i18n キー追加（share.*）、CSS スタイル追加

---

## 実装 TODO

### 9. SNS シェア機能 — バックエンド（シェアメッセージ永続化）

プロジェクトオーナーが設定した「おすすめシェアメッセージ」を DB に保存し、
シェアダイアログの初期値として全ユーザーに表示する。

- [x] **マイグレーション (020)**: `ALTER TABLE projects ADD COLUMN share_message TEXT DEFAULT ''`
- [x] **Project モデル**: `ShareMessage string` フィールド追加
- [x] **Create / Update API**: `share_message` を受け取り・保存（オーナーのみ更新可）
- [x] **Get / List API**: `share_message` を返す
- [ ] **フロント — ProjectDetail**: DB の `share_message` をダイアログ初期値に使用
  - 優先順位: localStorage 保存済みメッセージ > DB share_message > プロジェクト名
- [ ] **フロント — プロジェクト編集フォーム**: シェアメッセージ入力欄を追加
- [x] **テスト**: Project CRUD テストに share_message の検証を追加（Create / Update / 未送信時保持 の3件）

### 10. ホスト自身の利用停止を禁止
- [x] **バックエンド**: `AdminUserHandler.Suspend` で、対象ユーザーが自分自身の場合は 400 エラーを返す（TDD: 2テスト追加）
- [x] **フロント**: ユーザー管理一覧で自分自身の行には「利用停止」ボタンを非表示にする

### 11. description と overview を統合
`overview`（Markdown）を主フィールドとし、`description` は overview から自動生成（先頭200文字プレーンテキスト）。

- [x] **バックエンド API**: Create で `overview` を受け取り、description 未指定時は overview から自動生成（`plainTextFromMarkdown`）
  - TDD: 2テスト追加（auto-fill + explicit description 保持）
- [x] **新規プロジェクトフォーム**: 「説明」→「プロジェクト概要」に変更、Markdown プレースホルダー + ヒント + 「後から編集できます」注記
- [x] **フロント型定義**: `CreateProjectInput` / `UpdateProjectInput` に `overview` 追加
- [ ] **マイグレーション**: 既存 `description` データを `overview` に移行（既存データがある場合）

### 12. コスト内訳 UI を自由入力に変更
API は `cost_items`（ラベル・金額の行リスト）で自由入力対応済みだが、
UI がサーバー費/開発費/その他の固定3項目のまま。API に合わせる。

- [x] **フロント型定義**: `ProjectCostItem` / `ProjectCostItemInput` 型追加、`Project` に `cost_items` 追加
- [x] **i18n**: コスト内訳動的行用のキー追加（costItemLabel / costItemAmount / costItemAddRow）
- [x] **新規/編集プロジェクトフォーム**: 固定3項目 → 動的行追加 UI に変更（ラベル+金額+削除ボタン、＋追加ボタン）
- [x] **monthlyTarget 計算を project.monthly_target に統一**（FeaturedProjects / ProjectList / ProjectDetail / NavFinancialHealthMark 4件）
- [x] **mock-api を cost_items 形式に更新**（mock-projects.ts + mock-api.ts createProject/updateProject）
- [x] **プロジェクト詳細ページ**: `cost_items` を個別表示（旧 `costs` 固定表示を置き換え）

### 13. メール送信
- 問い合わせ通知、マジックリンク認証等
- 初期はメールログインなし（追って決める）
- プロバイダー未選定（SendGrid / SES / Resend 等）

---

## Stripe 本番移行セキュリティ対策

> 詳細: [setup/stripe-production-checklist.md](../setup/stripe-production-checklist.md)

### アプリ層（P1: 高優先度）

- [x] **A8. セキュリティヘッダーミドルウェア**: CSP / X-Frame-Options / X-Content-Type-Options / HSTS / Permissions-Policy を `middleware.go` に追加（TDD: 4テスト）
- [x] **A9. レートリミット**: `/api/donations/checkout` に IP ベース制限（10 req/min）を導入（TDD: 5テスト、X-Forwarded-For 対応）
- [ ] **A11. ログの機密情報除去**: `auth_handler.go` の userID・トークンプレフィックスのログ出力を修正

### アプリ層（P2: 中優先度）

- [ ] **A10. Statement Descriptor**: Checkout Session に `statement_descriptor_suffix` を追加
- [x] **A12. IdleTimeout 追加**: `main.go` の `http.Server` に `IdleTimeout: 120s` を設定
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
- [x] **法的文書**ドラフト作成済み（`backend/legal/` に terms.md / privacy.md / disclaimer.md）
  - `legal_handler.go` でファイル配信する仕組みも実装済み — 弁護士レビュー後に内容修正

---

## 設計判断（確定済み）

- [x] **Stripe Connect 接続タイプ** — Standard（無料プラン）に決定
- [x] **セッション管理方式** — DB session テーブル管理に決定（強制ログアウト対応）
- [x] **シェアメッセージ保存** — projects テーブルに `share_message` カラム追加（オーナーが設定、全ユーザーに初期値表示）


### 14. PJ詳細画面から画像アップロード
- [x] **フロント — ProjectDetail**: オーナー専用の画像アップロード/削除 UI を追加
  - ファイル選択 + D&D ゾーン、プレビュー、アップロード/削除/キャンセルボタン
  - 型・サイズバリデーション（JPEG/PNG/WebP、2MB以下）
  - `uploadProjectImage()` / `deleteProjectImage()` API 呼び出し
  - バックエンド API (`POST/DELETE /api/projects/:id/image`) は実装済み

### 15. PJ詳細画面で見積金額を更新 + アップデート自動投稿
- [x] **フロント — ProjectDetail**: オーナー専用のコスト内訳インライン編集 UI を追加
  - 動的行エディタ（label / unit_price / quantity + 追加/削除 + 合計表示）
  - 保存時に `updateProject(id, { cost_items })` で更新
  - **保存成功後に `createProjectUpdate()` でアップデートを自動投稿**（旧→新の月額見積差分を本文に記載）
  - i18n キー追加: `projects.editCostItems`, `projects.costUpdateTitle`, `projects.costUpdateBody`
  - バックエンド（`PUT /api/projects/:id` で `cost_items` 更新 + `monthly_target` 自動再計算）は実装済み

### 16. Discord OAuth ログイン
- [x] **DB マイグレーション (023)**: `ALTER TABLE users ADD COLUMN discord_id VARCHAR(255) UNIQUE` + index
- [x] **バックエンド**: model / repository / service / handler / providers / routes 全て実装
  - Google/GitHub パターン準拠: FindByDiscordID → FindByEmail → Create
  - メール非公開時は `username@discord.invalid` をフォールバック
  - 環境変数: `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET`
  - エンドポイント: `GET /api/auth/discord/login`, `GET /api/auth/discord/callback`
  - `GET /api/auth/providers` に `discord` を条件付き追加
- [x] **フロント**: `getDiscordLoginUrl()` + AuthStatus に Discord ボタン追加 + モック対応

### 17. アプリバー健全性表示 — 3色（緑/黄/赤）
- [x] **フロント — NavFinancialHealthMark**: `getHostHealth()` API で 3状態判定に変更
  - 旧: `getProject(PLATFORM_PROJECT_ID)` → 2状態（reached / not-reached）
  - 新: `GET /api/host` の `signal` フィールドで 3状態（green / yellow / red）
  - 色: 緑 `#2ecc71`（healthy）、黄 `#f1c40f`（warning）、赤 `#e74c3c`（critical）
  - CSS クラスを `--reached/--not-reached` → `--green/--yellow/--red` に変更
  - バックエンド (`GET /api/host` が `signal` を返却) は実装済み

### 18. シェア機能 — クリップボードコピーボタン
- [x] **フロント — ShareButtons**: X / Facebook / LINE ボタンの隣にクリップボードコピーボタンを追加
  - メッセージ編集ダイアログ → 「コピー」ボタンで `navigator.clipboard.writeText()` 実行
  - コピー後に「コピーしました」フィードバック表示（1.5秒）
  - i18n キー追加: `share.copy`, `share.copied`