# TODO — GIVErS プラットフォーム

最終更新: 2026-02-23

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

- [ ] **マイグレーション (020)**: `ALTER TABLE projects ADD COLUMN share_message TEXT DEFAULT ''`
- [ ] **Project モデル**: `ShareMessage string` フィールド追加
- [ ] **Create / Update API**: `share_message` を受け取り・保存（オーナーのみ更新可）
- [ ] **Get / List API**: `share_message` を返す
- [ ] **フロント — ProjectDetail**: DB の `share_message` をダイアログ初期値に使用
  - 優先順位: localStorage 保存済みメッセージ > DB share_message > プロジェクト名
- [ ] **フロント — プロジェクト編集フォーム**: シェアメッセージ入力欄を追加
- [ ] **テスト**: Project CRUD テストに share_message の検証を追加

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
