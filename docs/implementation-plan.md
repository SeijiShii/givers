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
| 認証 | Google OAuth（**必須**）+ 環境変数で選択可能なオプションプロバイダ（GitHub / Apple Sign In / Email マジックリンク）。有効プロバイダは `GET /api/auth/providers` で取得しフロントが動的に表示 |

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
- **sessions**: id（UUID）, user_id（FK → users）, expires_at, created_at。**DB sessions テーブル方式**。Cookie は `session_id=<UUID>`（HttpOnly, Secure, SameSite=Lax）。ログアウト時は行を削除して確実に無効化。
- **projects**: id, owner_id, name, ~~description~~→**overview**（Markdown。旧 description と統合）, share_message（シェアメッセージ）, deadline, monthly_target, **status**（`draft` \| `active` \| `frozen` \| `deleted`）, stripe_account_id, ...
  ※ **`draft` ステータスを追加**。プロジェクト作成フォーム送信時に draft で保存 → Stripe Connect 完了後に active に変更。Stripe 未接続のままプロジェクト公開は不可。
  ※ **`description` → `overview` 統合**: カード用短文と詳細ページ用 Markdown を `overview` 1 カラムに統合。一覧カードでは先頭 N 文字を Markdown ストリップして表示。
- ~~**project_costs**~~→**project_cost_items**: project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order, ...（固定 3 項目から動的行に変更。詳細は `cost-items-plan.md` 参照）
- **project_alerts**: project_id, warning_threshold, critical_threshold
- **donations**: id, project_id, **donor_type**（'token' \| 'user'）, **donor_id**（token の UUID または user の UUID）, amount, currency, stripe_payment_id, **is_recurring**（BOOLEAN）, **stripe_subscription_id**（TEXT NULL、定期寄付のみ非 NULL）, created_at, ...
  ※ 単発・定期を同一テーブルで管理。donor_type + donor_id の 2 カラムで識別（検索・インデックス・トークン→ユーザー移行が明確）。アカウントなし・ありを問わず全寄付を記録。idea.md の「全ての寄付者の寄付行動履歴を保存する」方針に基づく。定期寄付の状態管理は stripe_subscription_id + Stripe Webhook で行う。
- **platform_health**: プラットフォーム全体の健全性（月額必要額、達成率など）
- **project_updates**: プロジェクトのアップデート投稿
- **watches**: ウォッチ（ユーザー×プロジェクト）
- **project_mutes**: ミュート（プロジェクトオーナーが寄付者をミュート、プロジェクト単位）
- **contact_messages**: サービスホストへの問い合わせ。`id（UUID PK）, email（NOT NULL）, name（TEXT NULL）, message（TEXT NOT NULL）, status（'unread' | 'read'）, created_at`

### 未定義テーブルの詳細スキーマ

#### project_updates

| カラム | 型 | 説明 |
|--------|----|------|
| id | UUID PK | |
| project_id | UUID FK → projects | |
| author_id | UUID FK → users | 投稿者（プロジェクトオーナー） |
| title | TEXT NULL | タイトル（任意） |
| body | TEXT NOT NULL | 本文（Markdown） |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

#### watches

| カラム | 型 | 説明 |
|--------|----|------|
| user_id | UUID FK → users | |
| project_id | UUID FK → projects | |
| created_at | TIMESTAMPTZ | |
| PK | (user_id, project_id) | 複合主キー |

#### project_mutes

| カラム | 型 | 説明 |
|--------|----|------|
| project_id | UUID FK → projects | |
| muted_user_id | UUID FK → users | ミュートされたユーザー |
| created_at | TIMESTAMPTZ | |
| PK | (project_id, muted_user_id) | 複合主キー |

#### platform_health

| カラム | 型 | 説明 |
|--------|----|------|
| id | INT PK DEFAULT 1 | 常に1行（シングルトン） |
| monthly_cost | INTEGER | 月額必要額（円） |
| current_monthly | INTEGER | 現在の月額達成額（円） |
| warning_threshold | INTEGER | 注意閾値（達成率%、例: 60） |
| critical_threshold | INTEGER | 危険閾値（達成率%、例: 30） |
| updated_at | TIMESTAMPTZ | |

## 実装フェーズ

### Phase 1: 基盤構築

- Docker Compose で backend / frontend / PostgreSQL を起動
- Go: 最小限の API（health check）、DB 接続
- Astro: プロジェクト初期化、React 統合、レイアウト・ナビゲーション
- 開発用 CORS 設定

### Phase 2: 認証・ユーザー

- Google OAuth 2.0 実装（Go）。`GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` は**必須**（未設定ならサーバー起動拒否）
- GitHub OAuth など追加プロバイダは環境変数設定のみで有効化（`GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` など）
- `GET /api/auth/providers` でフロントエンドに有効プロバイダ一覧を返す（フロントはこれを見てボタンを動的表示）
- トークン（Cookie）による匿名寄付者トラッキング
- アカウント作成時のトークン→ユーザー移行フロー
- フロント: ログイン/ログアウト UI（React Island）

### Phase 3: プロジェクト CRUD

- プロジェクト作成・編集・一覧・詳細 API
- **説明フィールド**: `overview` 1 カラム（Markdown）で統合。新規作成フォームでも Markdown 入力可。「後から編集できます」注記を表示
- **費用設定**: `cost_items` 配列（ラベル・金額の自由入力行）。UI も動的行追加に対応（`cost-items-plan.md` 参照）
- アラート閾値（WARNING, CRITICAL）設定
- フロント: プロジェクト一覧・詳細・マイページ（設定フォームは React）
- **認証**: ミドルウェア注入方式。開発中は `AUTH_REQUIRED=false` で認証なし。詳細は [docs/phase3-plan.md](docs/phase3-plan.md)

### Phase 4: 決済（Stripe Connect）

- **Stripe Connect オンボーディングは一般プロジェクトオーナーのプロジェクト作成時に必須**。作成フォームの最終ステップで Stripe Connect にリダイレクト。オンボーディング完了後にプロジェクトが公開される。Stripe 未接続のままプロジェクト公開は不可。
- **サービスホスト（`HOST_EMAILS` に含まれるユーザー）は Connect 不要**。プロジェクトは `status: active` で即時公開され、プラットフォームの Stripe アカウントで直接決済される。
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
- **開示用データの出力**: 第三者情報開示請求等に備え、管理画面から**ユーザーID または プロジェクトID 指定で開示用データ（JSON）を出力**できるようにする。ホストのみ実行可能。→ `docs/legal-risk-considerations.md` 4・`docs/user-management-mock-plan.md` 1.4。
- **ホスト自身の利用停止を禁止**: `PATCH /api/admin/users/:id/suspend` で対象が自分自身の場合は 400 エラー。フロントでも自分の行の「利用停止」ボタンを非表示 or disabled にする。
- **問い合わせフォーム** (`/contact`): メールアドレス必須でホストにメッセージを送信。認証不要。メッセージは DB 保存し、`CONTACT_NOTIFY_EMAIL` 設定時は受信通知メールを送信。
- **問い合わせ閲覧** (`/host/contacts`): ホスト権限で問い合わせ一覧を閲覧・既読管理できる管理ページ。

### Phase 5.5: チャート表示（推移・進捗）

- **プロジェクトページ**: 折れ線グラフで目標金額・実際の寄付額・費やしたコストの時系列推移
- **マイページ**: 月ごと合計寄付額、プロジェクト別寄付額のチャート表示
- 詳細は [docs/charts-plan.md](docs/charts-plan.md) を参照

### Phase 6: 仕上げ

- 公式/自ホストの明示（About に公式 URL と GitHub リンクを記載。自ホストに寛大で強制しない。悪意あるクローン運営について公式は責任を負わない旨の公言を About 等で検討。→ idea.md・mock-implementation-status 12.6）
- **法的文書ページ** (`/terms`, `/privacy`, `/disclaimer`): `LEGAL_DOCS_DIR` に Markdown ファイルを配置すると表示される。`GET /api/legal/:type` で内容取得 → フロントでレンダリング。ファイルが存在しないページはリンクを非表示。自ホスト向けにサンプル Markdown テンプレートを `legal/` ディレクトリに同梱する。
- 本番用 Docker 設定、環境変数管理
- 基本的な E2E テスト
- ConoHa での運用設定は [docs/conoha-deployment.md](docs/conoha-deployment.md) を参照

---

## 手動動作確認チェックリスト

各フェーズ完了時に、以下を手動で確認する。

### Phase 1: 基盤構築

- [ ] `docker compose up` で db / backend / frontend が起動する
- [ ] トップページの「API ステータス」が `ok - GIVErS API` と表示される
- [ ] http://localhost:8080/api/health に GET でアクセスすると `{"status":"ok",...}` が返る
- [ ] DB 停止時、/api/health が unhealthy を返す
- [ ] ナビゲーション（プロジェクト一覧、ホスト、About）で各ページに遷移できる
- [ ] ロケール切替（日本語/English）で言語が切り替わる
- [ ] ソース変更時にホットリロードが動作する

### Phase 2: 認証・ユーザー

- [ ] ナビに Google ログインボタンが常時表示される（Google は必須）
- [ ] GitHub・Apple・Email は `GET /api/auth/providers` で有効と返された場合のみボタンが表示される
- [ ] 「Google」クリックで Google 認証画面にリダイレクトされる
- [ ] 「GitHub」クリックで GitHub 認証画面にリダイレクトされる（`GITHUB_CLIENT_ID` 設定時のみ表示）
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
- [ ] 費用設定（動的行追加 UI）・アラート閾値の保存・表示が正しい
- [ ] 新規作成フォームの説明欄が Markdown 入力可で「後から編集できます」注記がある
- [ ] 一覧カードに overview の先頭テキスト（Markdown ストリップ済み）が表示される

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
- [ ] ホストが管理画面からユーザーIDまたはプロジェクトIDを指定して開示用データ（JSON）を出力できる
- [ ] ユーザー管理一覧で自分自身の行に「利用停止」ボタンが非表示 or disabled
- [ ] 自分自身を停止しようとすると 400 エラーが返る
- [ ] `/contact` にアクセスするとメールアドレス入力必須の問い合わせフォームが表示される
- [ ] フォーム送信後に「送信しました」メッセージが表示される
- [ ] `/host/contacts` でホストが問い合わせ一覧を閲覧できる（未読/既読管理）

### Phase 6: 仕上げ

- [ ] About / フッターに公式ドメインが明記されている
- [ ] 自ホスト版で「自ホスト版です」の表示ができる
- [ ] `LEGAL_DOCS_DIR` に `terms.md` を配置すると `/terms` に内容が表示される
- [ ] ファイル未配置の法的文書ページはフッターリンクが非表示になる
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
| /contact | 静的+動的 | ホストへの問い合わせフォーム（送信フォームは React Island） |
| /host/contacts | 動的 | 問い合わせ一覧（ホスト権限必須） |
| /terms | 静的 | 利用規約（Markdown レンダリング。ファイル未配置なら「設定されていません」表示） |
| /privacy | 静的 | プライバシーポリシー（同上） |
| /disclaimer | 静的 | 免責事項（同上） |

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

**API 仕様（エンドポイント一覧・スキーマ・エラー形式・セッション管理・Stripe Connect フロー・環境変数）は [docs/api-specs.md](api-specs.md) を参照。**

### ディレクトリ構成

- `internal/handler`: HTTP ハンドラ
- `internal/service`: ビジネスロジック
- `internal/repository`: DB アクセス
- `internal/model`: エンティティ定義
- `pkg/auth`: 認証ミドルウェア
- `pkg/stripe`: Stripe 連携
