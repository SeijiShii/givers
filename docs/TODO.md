# TODO — GIVErS プラットフォーム

最終更新: 2026-02-22

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

---

## 実装 TODO（仕様未確定 / 外部依存）

### 8. メール送信
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
