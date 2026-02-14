# GIVErS 実装プラン（Astro + React）

## 技術スタック

| 層 | 技術 |
|----|------|
| インフラ | Docker Compose |
| バックエンド | Go (API サーバー) |
| フロントエンド | Astro + React (Islands) / TypeScript |
| DB | PostgreSQL |
| 決済 | Stripe Connect |
| 認証 | Google OAuth 2.0 |

## プロジェクト構成

```
giving_platform/
├── docker-compose.yml
├── backend/                 # Go API
│   ├── cmd/server/
│   ├── internal/
│   │   ├── handler/
│   │   ├── repository/
│   │   ├── service/
│   │   └── model/
│   ├── go.mod
│   └── Dockerfile
├── frontend/                # Astro + React
│   ├── src/
│   │   ├── pages/
│   │   ├── components/
│   │   │   ├── react/       # client:load が必要なコンポーネント
│   │   │   └── astro/       # 静的コンポーネント
│   │   └── layouts/
│   ├── astro.config.mjs
│   ├── package.json
│   └── Dockerfile
└── docs/
    ├── idea.md
    └── implementation-plan.md
```

## アーキテクチャ概要

- フロント: Astro が静的 HTML を生成。寄付フォーム・設定画面などは React Islands で hydrate
- バックエンド: JSON API を提供。認証・決済・プロジェクト CRUD を担当
- 通信: フロントは fetch で API を呼び出し

## データモデル（主要）

- **users**: id, email, google_id, created_at
- **projects**: id, owner_id, name, description, deadline, monthly_target, status, stripe_account_id, ...
- **project_costs**: project_id, server_cost, dev_cost, other_cost, ...
- **project_alerts**: project_id, warning_threshold, critical_threshold
- **donations**: id, project_id, donor_token_or_user_id, amount, currency, stripe_payment_id, ...
- **platform_health**: プラットフォーム全体の健全性（月額必要額、達成率など）

## 実装フェーズ

### Phase 1: 基盤構築

- Docker Compose で backend / frontend / PostgreSQL を起動
- Go: 最小限の API（health check）、DB 接続
- Astro: プロジェクト初期化、React 統合、レイアウト・ナビゲーション
- 開発用 CORS 設定

### Phase 2: 認証・ユーザー

- Google OAuth 2.0 実装（Go）
- トークン（Cookie）による匿名寄付者トラッキング
- アカウント作成時のトークン→ユーザー移行フロー
- フロント: ログイン/ログアウト UI（React Island）

### Phase 3: プロジェクト CRUD

- プロジェクト作成・編集・一覧・詳細 API
- 費用設定（サーバー、開発者、その他）、期限設定
- アラート閾値（WARNING, CRITICAL）設定
- フロント: プロジェクト一覧・詳細・マイページ（設定フォームは React）

### Phase 4: 決済（Stripe Connect）

- Stripe Connect で募集者オンボーディング
- 寄付用 Checkout Session / サブスク作成
- Webhook で決済完了・サブスク状態の同期
- フロント: 寄付フォーム（React）、金額・通貨・単発/定期の選択

### Phase 5: プラットフォーム機能

- サービスホストページ（健全性表示: 青/黄/赤）
- プロジェクト単位の達成率・アラート表示
- トップページ: 新着・HOT プロジェクト表示
- プロジェクト間リンク・発見導線

### Phase 6: 仕上げ

- 公式/自ホストの明示（About、フッター、環境変数）
- 本番用 Docker 設定、環境変数管理
- 基本的な E2E テスト

## フロントエンド詳細（Astro + React）

- React コンポーネントは **TypeScript**（.tsx）で記述
- `src/lib/api.ts` など共通ロジックも TypeScript
- tsconfig は `astro/tsconfigs/strict` を継承

### ページ構成

| パス | 種別 | React Islands |
|-----|------|---------------|
| / | 静的 | プロジェクトカード（リンクのみで静的可） |
| /projects/[id] | 静的+部分動的 | 寄付フォーム、アラート表示 |
| /projects/new | 動的 | プロジェクト作成フォーム |
| /projects/[id]/edit | 動的 | プロジェクト編集フォーム |
| /me | 動的 | マイページ（プロジェクト一覧、寄付履歴） |
| /host | 静的 | プラットフォーム健全性（数値は API から fetch） |
| /about | 静的 | 公式ドメイン等の説明 |

### React Islands の使い分け

- **client:load**: 寄付フォーム、プロジェクト設定フォーム、ログイン状態に依存する UI
- **client:idle**: プロジェクト一覧のフィルタなど、優先度低いインタラクション
- **client:visible**: 無限スクロールや遅延表示が必要な場合

### API 呼び出し

- `src/lib/api.ts` で fetch ラッパーを用意（baseURL、認証ヘッダー、エラーハンドリング）
- 認証: Cookie ベース（HttpOnly）または Bearer トークン

### 多言語化（i18n）

- **対応言語**: 日本語（デフォルト）、英語
- **URL**: `/` 〜 `/about` が日本語、`/en` 〜 `/en/about` が英語
- **翻訳ファイル**: `src/i18n/ja.json`, `src/i18n/en.json`
- **ヘルパー**: `src/lib/i18n.ts` の `t(locale, key, params?)`
- **ロケール切替**: ナビゲーションバー右端に「English」/「日本語」リンク

## バックエンド詳細（Go）

### API 設計（REST）

- `GET /api/health` - ヘルスチェック
- `GET /api/projects` - プロジェクト一覧（クエリ: sort, limit）
- `GET /api/projects/:id` - プロジェクト詳細
- `POST /api/projects` - プロジェクト作成（認証必須）
- `PUT /api/projects/:id` - プロジェクト更新（認証必須）
- `GET /api/me` - 現在のユーザー情報
- `GET /api/me/projects` - 自分のプロジェクト一覧
- `GET /api/me/donations` - 自分の寄付履歴
- `POST /api/auth/google` - Google OAuth コールバック処理
- `POST /api/donations/checkout` - Stripe Checkout Session 作成
- `POST /api/webhooks/stripe` - Stripe Webhook
- `GET /api/host` - プラットフォーム健全性

### ディレクトリ構成（Go）

- `internal/handler`: HTTP ハンドラ
- `internal/service`: ビジネスロジック
- `internal/repository`: DB アクセス
- `internal/model`: エンティティ定義
- `pkg/auth`: 認証ミドルウェア
- `pkg/stripe`: Stripe 連携

## 環境変数

| 変数 | 用途 |
|------|------|
| DATABASE_URL | PostgreSQL 接続文字列 |
| GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET | OAuth |
| STRIPE_SECRET_KEY / STRIPE_WEBHOOK_SECRET | Stripe |
| STRIPE_CONNECT_CLIENT_ID | Stripe Connect |
| FRONTEND_URL | CORS・リダイレクト用 |
| OFFICIAL_DOMAIN | 公式ドメイン（自ホスト判定用） |
