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
- **残り**: 寄付確定時・プロジェクト作成/更新時に activities テーブルへの INSERT を各ハンドラーに組み込む

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

---

## 実装 TODO（仕様未確定 / 外部依存）

### 5. Activity INSERT フック
- [x] 寄付確定時（Webhook）に `donation` イベントを activities に INSERT
- [x] プロジェクト作成時に `project_created` イベントを INSERT
- [x] プロジェクト更新時に `project_updated` イベントを INSERT
- [ ] マイルストーン到達時に `milestone` イベントを INSERT（仕様未定）

### 6. 開示用データ出力 API
- `GET /api/admin/disclosure-export` — ルート登録済み
- 法的要件の確定に依存（弁護士確認後）

### 7. メール送信
- 問い合わせ通知、マジックリンク認証等
- プロバイダー未選定（SendGrid / SES / Resend 等）

---

## インフラ・運用

- [ ] **`docker-compose.prod.yml`** 作成（本番デプロイ用）
- [ ] **法的文書**のコンテンツ作成（利用規約・プライバシーポリシー）
  - `legal_handler.go` でファイル配信する仕組みは実装済み

---

## 設計判断（未確定）

- [ ] **Stripe Connect 接続タイプ** — Standard vs Express の最終決定
- [ ] **セッション管理方式** — signed cookie のみ vs DB session テーブル
  - DB テーブルなら強制ログアウト（アカウント停止時等）が可能
