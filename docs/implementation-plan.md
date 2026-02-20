# GIVErS 実装プラン（Astro + React）

## 第一目標と情報提供の優先度

- **日本で寄付の文化を根付かせることを第一目標**とする（idea.md）。
- そのため**日本での寄付に必要な情報の提供を最優先**する。寄付金控除・領収書、日本語の案内・FAQ、デフォルトロケールを日本とするなど、日本の寄付者が安心して寄付するために必要な情報を優先的に整える。

## 技術スタック

| 層 | 技術 |
|----|------|
| インフラ | Docker Compose |
| バックエンド | Go (API サーバー) |
| フロントエンド | Astro + React (Islands) / TypeScript |
| DB | PostgreSQL |
| 決済 | Stripe Connect |
| 認証 | Google / GitHub OAuth。拡張候補: Email（マジックリンクまたはパスワード）、Apple Sign In |

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

## 開発方針: TDD（テスト駆動開発）

- **この先はテスト先行**で実装する（レッド→グリーン→リファクタリング）
- テストはリファクタリング時のコンテキスト・安全網として活用する
- Phase 3 以前のコードは後追いでテストを追加済み（ProjectService, ProjectHandler, 認証ミドルウェア）

## アーキテクチャ概要

- フロント: Astro が静的 HTML を生成。寄付フォーム・設定画面などは React Islands で hydrate
- バックエンド: JSON API を提供。認証・決済・プロジェクト CRUD を担当
- 通信: フロントは fetch で API を呼び出し

## データモデル（主要）

- **users**: id, email, name, google_id, github_id, apple_id, password_hash（いずれも NULL 可のプロバイダ別 ID / パスワードハッシュ）, created_at, updated_at。同一ユーザーは email でリンク（idea.md・phase2-plan 参照）。
- **projects**: id, owner_id, name, description, deadline, monthly_target, status, stripe_account_id, ...
- **project_costs**: project_id, server_cost, dev_cost, other_cost, ...
- **project_alerts**: project_id, warning_threshold, critical_threshold
- **donations**: id, project_id, **donor_type**（'token' \| 'user'）, **donor_id**（token の UUID または user の UUID）, amount, currency, stripe_payment_id, ...  
  ※ donor_type + donor_id の 2 カラムで識別（検索・インデックス・トークン→ユーザー移行が明確）。アカウントなし・ありを問わず全寄付を記録。idea.md の「全ての寄付者の寄付行動履歴を保存する」方針に基づく。将来的なデータ分析も想定。
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
- **認証**: ミドルウェア注入方式。開発中は `AUTH_REQUIRED=false` で認証なし。詳細は [docs/phase3-plan.md](docs/phase3-plan.md)

### Phase 4: 決済（Stripe Connect）

- Stripe Connect で募集者オンボーディング
- 寄付用 Checkout Session / サブスク作成
- Webhook で決済完了・サブスク状態の同期
- フロント: 寄付フォーム（React）、金額・通貨・単発/定期の選択
- **匿名寄付者トークン**: 単発寄付は決済成功時（Webhook または成功リダイレクト）に donor_token を発行し、donations に記録すると同時に Cookie で返す。Cookie は HttpOnly, Secure（本番）, SameSite=Lax、有効期限は 1 年程度。詳細は `docs/mock-implementation-status.md` 12.2「単発寄付時のトークン発行タイミング」。
- **利用停止時の決済**: ホストが利用停止したユーザーに紐づく Stripe サブスクは解約する。**凍結時**: 新規寄付・新規サブスク受付停止、既存サブスクは継続。**削除時**: 新規停止に加え既存サブスクも解約。→ `docs/mock-implementation-status.md` 12.5・`docs/idea.md` 参照。

### Phase 5: プラットフォーム機能

- サービスホストページ（健全性表示: 青/黄/赤）
- プロジェクト単位の達成率・アラート表示
- トップページ: 新着・HOT プロジェクト表示（新着＝created_at 降順、HOT＝達成率降順。→ `docs/mock-implementation-status.md` 12.5）
- プロジェクト間リンク・発見導線
- **利用停止・凍結時のメッセージ**: ホスト権限で利用停止されたアカウント、またはオーナーが凍結・削除したプロジェクトに対して寄付等のアクションを試みたときに、状況を理解できる親切なメッセージを表示する。→ `docs/idea.md`・`docs/user-management-mock-plan.md` 1.3・`docs/mock-implementation-status.md` 12.5 参照。

### Phase 5.5: チャート表示（推移・進捗）

- **プロジェクトページ**: 折れ線グラフで目標金額・実際の寄付額・費やしたコストの時系列推移
- **マイページ**: 月ごと合計寄付額、プロジェクト別寄付額のチャート表示
- 詳細は [docs/charts-plan.md](docs/charts-plan.md) を参照

### Phase 6: 仕上げ

- 公式/自ホストの明示（About に公式 URL と GitHub リンクを記載。自ホストに寛大で強制しない。悪意あるクローン運営について公式は責任を負わない旨の公言を About 等で検討。→ idea.md・mock-implementation-status 12.6）
- 本番用 Docker 設定、環境変数管理
- 基本的な E2E テスト
- ConoHa での運用設定は [docs/conoha-deployment.md](docs/conoha-deployment.md) を参照

---

## 手動動作確認チェックリスト

各フェーズ完了時に、以下を手動で確認する。

### Phase 1: 基盤構築

- [ ] `docker compose up` で db / backend / frontend が起動する
- [ ] フロント http://localhost:4321 にアクセスするとトップページが表示される
- [ ] トップページの「API ステータス」が `ok - GIVErS API` と表示される
- [ ] http://localhost:8080/api/health に GET でアクセスすると `{"status":"ok",...}` が返る
- [ ] DB 停止時、/api/health が unhealthy を返す
- [ ] ナビゲーション（プロジェクト一覧、ホスト、About）で各ページに遷移できる
- [ ] ロケール切替（日本語/English）で言語が切り替わる
- [ ] ソース変更時にホットリロードが動作する

### Phase 2: 認証・ユーザー

- [ ] ナビに「Google」「GitHub」ログインボタンが表示される（拡張で Apple・Email も追加可能。`docs/mock-implementation-status.md` 3.7 参照）
- [ ] 「Google」クリックで Google 認証画面にリダイレクトされる（GOOGLE_CLIENT_ID 設定時）
- [ ] 「GitHub」クリックで GitHub 認証画面にリダイレクトされる（GITHUB_CLIENT_ID 設定時）
- [ ] 認証完了後、フロントにリダイレクトされ、ユーザー名と「ログアウト」が表示される
- [ ] ログアウト後、「Google でログイン」が再表示される
- [ ] 未ログインで GET /api/me が 401 を返す
- [ ] ログイン中に GET /api/me がユーザー情報を返す
- [ ] /me ページにアクセスできる
- [ ] 英語ページ（/en）でもログイン/ログアウトが動作する

### Phase 3: プロジェクト CRUD

- [ ] プロジェクト一覧に登録済みプロジェクトが表示される
- [ ] プロジェクト詳細ページで内容・達成率が表示される
- [ ] ログイン後にプロジェクト作成フォームから新規作成できる
- [ ] 自分のプロジェクトを編集できる
- [ ] 他者のプロジェクトは編集できない
- [ ] マイページに自分のプロジェクト一覧が表示される
- [ ] 費用設定・アラート閾値の保存・表示が正しい

### Phase 4: 決済（Stripe Connect）

- [ ] 募集者が Stripe Connect にオンボーディングできる
- [ ] プロジェクト詳細の「寄付する」で Stripe Checkout に遷移する
- [ ] 単発寄付が完了し、プロジェクトに反映される
- [ ] サブスク（月額等）の申し込みができる
- [ ] Webhook で決済完了が DB に記録される
- [ ] 寄付履歴がマイページに表示される

### Phase 5: プラットフォーム機能

- [ ] /host でプラットフォーム健全性（青/黄/赤）が表示される
- [ ] プロジェクトページに達成率・アラート状態が表示される
- [ ] トップページに新着・HOT プロジェクトが表示される
- [ ] プロジェクト間のリンクで他プロジェクトを発見できる
- [ ] 利用停止アカウントで寄付等を試みたときに、状況が分かる親切なメッセージが表示される
- [ ] 凍結・削除されたプロジェクトに寄付等を試みたときに、状況が分かる親切なメッセージが表示される

### Phase 6: 仕上げ

- [ ] About / フッターに公式ドメインが明記されている
- [ ] 自ホスト版で「自ホスト版です」の表示ができる
- [ ] 本番用環境変数で動作する
- [ ] E2E テストが通る

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
- `POST /api/me/migrate-from-token` - トークンに紐づく寄付を現在ユーザーに移行（冪等。詳細は下記「トークン→アカウント移行 API」）
- `POST /api/auth/google` - Google OAuth コールバック処理
- `POST /api/donations/checkout` - Stripe Checkout Session 作成
- `POST /api/webhooks/stripe` - Stripe Webhook
- `GET /api/host` - プラットフォーム健全性

### トークン→アカウント移行 API（POST /api/me/migrate-from-token）

| 項目 | 内容 |
|------|------|
| **目的** | 匿名寄付時につけたトークン（Cookie）に紐づく寄付を、ログイン中のユーザーに紐づけ直す。idea.md の「これまでの寄付をアカウントに引き継ぎますか？」に対応。 |
| **認証** | 必須。セッションのユーザーに移行する。 |
| **リクエスト** | トークンは **Cookie** で送る（Body は空で可。または `{ "token": "..." }` でオプション送信も可）。 |
| **処理** | Cookie の donor_token に紐づく donations（donor_type='token', donor_id=token）を、donor_type='user', donor_id=現在ユーザーID に UPDATE。該当が 0 件の場合は何もしない。 |
| **冪等** | **冪等とする**。同じトークンで複数回呼んでも、2 回目以降は「すでに移行済み」としてエラーにせず成功扱い。 |
| **すでに移行済み** | 移行済みのトークンで再呼び出し時は **200 OK** を返す。body で `{ "migrated_count": 0, "already_migrated": true }` のようにし、フロントはエラー表示せず「引き継ぎ済みです」等の表示に利用する。 |
| **成功時** | 200 OK。`{ "migrated_count": N, "already_migrated": false }`（N は移行した寄付件数）。 |
| **トークンなし・無効** | Cookie に有効なトークンがない場合は 400 または 200（migrated_count: 0）のいずれかで統一（実装時にどちらにするか決定）。 |

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
