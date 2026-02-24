# GIVErS バックエンド API 仕様

## 基本情報

| 項目 | 内容 |
|------|------|
| Base URL（開発） | `http://localhost:8080` |
| Content-Type | `application/json` |
| 認証方式 | Cookie ベース（`session_id=<UUID>`、HttpOnly） |
| 文字コード | UTF-8 |

---

## エンドポイント一覧

### ヘルス

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/health` | 不要 | ヘルスチェック |

### 認証

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/auth/providers` | 不要 | 有効な認証プロバイダ一覧（環境変数の有無で動的に返す） |
| GET | `/api/auth/{provider}/login` | 不要 | OAuth 開始（provider = google \| github \| apple）。対応する認可 URL にリダイレクト |
| GET | `/api/auth/{provider}/callback` | 不要 | OAuth コールバック。`code` を受け取り、ユーザー取得・セッション確立 |
| POST | `/api/auth/logout` | 必須 | ログアウト。sessions テーブルから該当行を削除、Cookie クリア |

### プロジェクト

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/projects` | 不要 | プロジェクト一覧（`status=active` のみ。クエリ詳細は下記） |
| GET | `/api/projects/:id` | 不要 | プロジェクト詳細 |
| POST | `/api/projects` | 必須 | プロジェクト作成。一般オーナー: `status: draft` → Stripe Connect 完了後に active。ホスト: `status: active`（Connect 不要） |
| PUT | `/api/projects/:id` | 必須（オーナー） | プロジェクト更新 |
| DELETE | `/api/projects/:id` | 必須（オーナー） | プロジェクト削除（論理削除: status → deleted） |
| PATCH | `/api/projects/:id/status` | 必須（オーナーまたはホスト） | 状態変更（`frozen` ↔ `active`） |
| POST | `/api/projects/:id/watch` | 必須 | ウォッチ登録 |
| DELETE | `/api/projects/:id/watch` | 必須 | ウォッチ解除 |

### プロジェクト アップデート

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/projects/:id/updates` | 不要 | アップデート一覧 |
| POST | `/api/projects/:id/updates` | 必須（オーナー） | アップデート投稿 |
| PUT | `/api/projects/:id/updates/:uid` | 必須（投稿者） | アップデート編集 |
| DELETE | `/api/projects/:id/updates/:uid` | 必須（投稿者またはホスト） | アップデート削除 |

### マイページ

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/me` | 必須 | 現在のユーザー情報 |
| GET | `/api/me/projects` | 必須 | 自分のプロジェクト一覧（draft 含む） |
| GET | `/api/me/donations` | 必須 | 自分の寄付履歴 |
| PATCH | `/api/me/donations/:id` | 必須 | 定期寄付の編集（金額変更・一時停止・再開） |
| DELETE | `/api/me/donations/:id` | 必須 | 定期寄付のキャンセル |
| GET | `/api/me/watches` | 必須 | ウォッチ中のプロジェクト一覧 |
| POST | `/api/me/migrate-from-token` | 必須 | 匿名トークンに紐づく寄付を現在ユーザーに移行（冪等。詳細は下記） |

### 決済

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| POST | `/api/donations/checkout` | 不要（匿名寄付あり） | Stripe Checkout Session 作成 |
| GET | `/api/stripe/connect/callback` | 不要（Stripe からのリダイレクト） | Stripe Connect オンボーディング完了コールバック |
| POST | `/api/webhooks/stripe` | 不要（Stripe 署名検証） | Stripe Webhook |

### プラットフォーム

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/host` | 不要 | プラットフォーム健全性（計算済み rate・signal を含む） |
| POST | `/api/contact` | 不要 | サービスホストへの問い合わせ送信 |
| GET | `/api/legal/:type` | 不要 | 法的文書（Markdown）取得。`type` = `terms` \| `privacy` \| `disclaimer`。ファイル未配置なら 404 |

### 管理（ホスト権限必須）

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/admin/users` | 必須（ホスト） | ユーザー一覧 |
| PATCH | `/api/admin/users/:id/suspend` | 必須（ホスト） | ユーザー利用停止・解除（**自分自身は不可 → 400**） |
| GET | `/api/admin/disclosure-export` | 必須（ホスト） | 開示用データ出力（`?type=user&id=xxx` または `?type=project&id=xxx`） |
| GET | `/api/admin/contacts` | 必須（ホスト） | 問い合わせ一覧 |

### PATCH /api/admin/users/:id/suspend — 追加仕様

- **自分自身の停止を禁止**: 対象ユーザー ID がリクエスト元ユーザー ID と一致する場合は **400 Bad Request** を返す。
  ```json
  { "errors": [{ "message": "cannot suspend yourself" }] }
  ```

---

## エラーレスポンス形式

全エンドポイントで統一した形式を使用する。

```json
{
  "errors": [
    { "field": "amount", "message": "must be greater than 0" },
    { "field": "project_id", "message": "project not found" }
  ]
}
```

- `field` は省略可（汎用エラーは省略または `""`）
- 認証エラー（401）: `{ "errors": [{ "message": "unauthorized" }] }`
- 権限エラー（403）: `{ "errors": [{ "message": "forbidden" }] }`
- 未発見（404）: `{ "errors": [{ "message": "not found" }] }`

---

## 主要エンドポイント スキーマ

### GET /api/projects

**クエリパラメータ**

| パラメータ | 型 | デフォルト | 説明 |
|-----------|-----|-----------|------|
| sort | string | `new` | `new`（created_at 降順）/ `hot`（達成率降順） |
| limit | int | 20 | 最大 100 |
| cursor | string | なし | カーソルベースページネーション（前回レスポンスの `next_cursor`） |

**レスポンス (200)**
```json
{
  "projects": [...],
  "next_cursor": "uuid or null"
}
```

### POST /api/projects

**リクエスト**
```json
{
  "name": "string（必須）",
  "overview": "string（任意、Markdown 対応。旧 description を統合）",
  "deadline": "2026-12-31（任意, ISO 8601 date）",
  "owner_want_monthly": 50000,
  "cost_items": [
    {
      "label": "サーバー費用",
      "unit_type": "monthly",
      "amount_monthly": 10000
    },
    {
      "label": "開発者費用",
      "unit_type": "daily_x_days",
      "rate_per_day": 20000,
      "days_per_month": 5
    },
    {
      "label": "その他費用",
      "unit_type": "monthly",
      "amount_monthly": 5000
    }
  ],
  "alerts": {
    "warning_threshold": 60,
    "critical_threshold": 30
  }
}
```

> **`description` → `overview` 統合**: 旧 `description`（カード用短文）と `overview`（詳細用 Markdown）を `overview` 1 カラムに統合。一覧カードでは先頭 N 文字を Markdown ストリップして表示する。詳細は `cost-items-plan.md` 参照。

> **`costs` → `cost_items` 変更**: 旧固定 3 項目オブジェクトから動的行配列に変更。詳細は `cost-items-plan.md` 参照。

**レスポンス (201)**

一般オーナーの場合:
```json
{
  "id": "uuid",
  "status": "draft",
  "stripe_connect_url": "https://connect.stripe.com/..."
}
```

ホスト（サービス運営者）の場合:
```json
{
  "id": "uuid",
  "status": "active"
}
```
> ホストは `HOST_EMAILS` 環境変数で判定。Stripe Connect OAuth は不要で、プラットフォームの Stripe アカウントで直接決済される。

### PUT /api/projects/:id

**リクエスト**: POST /api/projects と同フィールド（全フィールド任意・部分更新）

**レスポンス (200)**: 更新後のプロジェクトオブジェクト

### PATCH /api/projects/:id/status

**リクエスト**
```json
{ "status": "frozen" }
```

`status` は `"active"` または `"frozen"` のみ受け付ける。`deleted` への変更は `DELETE /api/projects/:id` を使う。

### POST /api/donations/checkout

**リクエスト**
```json
{
  "project_id": "uuid（必須）",
  "amount": 1000,
  "currency": "jpy",
  "is_recurring": false,
  "message": "string（任意）",
  "locale": "ja"
}
```

**レスポンス (200)**
```json
{
  "checkout_url": "https://checkout.stripe.com/..."
}
```

### PATCH /api/me/donations/:id

**リクエスト**（変更したいフィールドのみ）
```json
{
  "amount": 2000,
  "paused": true
}
```

**レスポンス (200)**
```json
{ "id": "...", "amount": 2000, "paused": true, "..." : "..." }
```

### GET /api/host

**レスポンス (200)**
```json
{
  "monthly_cost": 50000,
  "current_monthly": 28000,
  "warning_threshold": 60,
  "critical_threshold": 30,
  "rate": 56,
  "signal": "yellow"
}
```

`signal` は `"green"` / `"yellow"` / `"red"` のいずれか。

### POST /api/contact

サービスホストへの問い合わせを送信する。認証不要（未ログインユーザーも可）。

**リクエスト**
```json
{
  "email": "user@example.com",
  "name": "山田 太郎",
  "message": "問い合わせ内容"
}
```

| フィールド | 必須 | 説明 |
|-----------|------|------|
| `email` | ◎ | 返信先メールアドレス |
| `name` | ✕ | 送信者名（任意） |
| `message` | ◎ | 本文（最大 5000 文字） |

**レスポンス (200)**
```json
{ "ok": true }
```

- メッセージは `contact_messages` テーブルに保存する
- `CONTACT_NOTIFY_EMAIL` が設定されている場合、受信時にそのアドレスへ通知メールを送信する（未実装は保存のみ）

### GET /api/admin/contacts

問い合わせ一覧を返す（ホスト権限必須）。

**クエリパラメータ**

| パラメータ | デフォルト | 説明 |
|-----------|-----------|------|
| `limit` | 50 | 最大 200 |
| `cursor` | なし | カーソルベースページネーション |
| `status` | `all` | `all` / `unread` / `read` |

**レスポンス (200)**
```json
{
  "contacts": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "name": "山田 太郎",
      "message": "問い合わせ内容",
      "status": "unread",
      "created_at": "2026-02-21T00:00:00Z"
    }
  ],
  "next_cursor": "uuid or null"
}
```

### GET /api/legal/:type

サーバー上の所定ディレクトリに配置された Markdown ファイルの内容を返す。

**パスパラメータ**

| `type` | ファイル名 | ページ |
|--------|-----------|--------|
| `terms` | `terms.md` | 利用規約 |
| `privacy` | `privacy.md` | プライバシーポリシー |
| `disclaimer` | `disclaimer.md` | 免責事項 |

- ファイルは `LEGAL_DOCS_DIR` 環境変数で指定したディレクトリに配置する（デフォルト: `./legal/`）
- ファイルが存在しない場合は **404** を返す（フロントは「このページはまだ設定されていません」と表示）
- ファイル名・形式は Markdown（`.md`）固定
- `..` や絶対パスを含むパストラバーサルは拒否（400）

**レスポンス (200)**
```json
{
  "type": "terms",
  "content": "# 利用規約\n\n..."
}
```

**レスポンス (404)**
```json
{ "errors": [{ "message": "not found" }] }
```

### GET /api/auth/providers

有効な認証プロバイダの一覧を返す。フロントエンドはこのレスポンスに基づきログインボタンを動的に表示する。

**ルール**
- `google` は常時含まれる（必須プロバイダ。`GOOGLE_CLIENT_ID` 未設定の場合はサーバー起動時にエラー）
- `github` は `GITHUB_CLIENT_ID` が設定されている場合のみ含まれる
- `apple` は `APPLE_CLIENT_ID` が設定されている場合のみ含まれる（将来実装）
- `email` は `ENABLE_EMAIL_LOGIN=true` が設定されている場合のみ含まれる（将来実装）

**レスポンス (200)**
```json
{ "providers": ["google", "github"] }
```

---

## セッション管理

| 項目 | 内容 |
|------|------|
| **方式** | DB `sessions` テーブル。`id（UUID PK）, user_id（FK → users）, expires_at, created_at` |
| **Cookie** | `session_id=<UUID>`（HttpOnly, Secure（本番）, SameSite=Lax）。有効期限 30 日程度 |
| **ログアウト** | `DELETE FROM sessions WHERE id = ?` で確実に無効化 |
| **有効期限** | `expires_at` を毎リクエストで検証。期限切れは 401。必要に応じてスライド延長可 |
| **クリーンアップ** | 期限切れセッションは定期的に削除（バッチ or リクエスト時に古いセッションを削除） |

---

## Stripe Connect フロー（プロジェクト作成時）

### 一般プロジェクトオーナー

1. `POST /api/projects`（status: draft で保存） → `{ "id": "...", "stripe_connect_url": "https://connect.stripe.com/..." }` を返す
2. フロントは `stripe_connect_url` にリダイレクト
3. オーナーが Stripe Connect を完了 → Stripe が `GET /api/stripe/connect/callback?code=...&state=project_id` にリダイレクト
4. バックエンドが `code` を Stripe に送って `stripe_account_id` を取得 → `projects.stripe_account_id` を保存 → `status: draft → active` に更新
5. フロントにリダイレクト（`/projects/<id>`）

**離脱時の扱い**: draft のままマイページに「未接続のプロジェクト」として表示。「Stripe 接続を続ける」リンクを提供。

### サービスホスト（プラットフォーム運営者）

1. `POST /api/projects`（status: **active** で即時保存） → `{ "id": "...", "status": "active" }` を返す
2. Stripe Connect OAuth は不要。`stripe_connect_url` はレスポンスに含まれない
3. 寄付はプラットフォームの Stripe アカウントで直接決済される（`Stripe-Account` ヘッダー省略 → プラットフォーム口座に入金）

**判定方法**: ログインユーザーのメールアドレスが `HOST_EMAILS` 環境変数に含まれる場合、自動的にホストとして扱われる。

---

## トークン→アカウント移行 API（POST /api/me/migrate-from-token）

| 項目 | 内容 |
|------|------|
| **目的** | 匿名寄付時につけたトークン（Cookie）に紐づく寄付を、ログイン中のユーザーに紐づけ直す。「これまでの寄付をアカウントに引き継ぎますか？」フローに対応 |
| **認証** | 必須。セッションのユーザーに移行する |
| **リクエスト** | トークンは **Cookie** で送る（Body は空で可） |
| **処理** | Cookie の `donor_token` に紐づく donations（`donor_type='token'`）を `donor_type='user'`, `donor_id=現在ユーザーID` に UPDATE |
| **冪等** | 同じトークンで複数回呼んでも 2 回目以降はエラーにせず成功扱い |
| **成功時** | 200 OK。`{ "migrated_count": N, "already_migrated": false }` |
| **移行済み** | 200 OK。`{ "migrated_count": 0, "already_migrated": true }` |
| **トークンなし・無効** | 400 Bad Request |

---

## 環境変数

| 変数 | 用途 |
|------|------|
| `DATABASE_URL` | PostgreSQL 接続文字列 |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth（**必須**。未設定の場合はサーバー起動を拒否） |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth（オプション。未設定なら GitHub ログイン無効） |
| `APPLE_CLIENT_ID` / `APPLE_CLIENT_SECRET` / `APPLE_TEAM_ID` / `APPLE_KEY_ID` | Apple Sign In（オプション。将来実装。未設定なら Apple ログイン無効） |
| `ENABLE_EMAIL_LOGIN` | `true` でメールログイン（マジックリンク）を有効化（オプション。将来実装） |
| `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` | Stripe |
| `STRIPE_CONNECT_CLIENT_ID` | Stripe Connect |
| `FRONTEND_URL` | CORS・リダイレクト用 |
| `OFFICIAL_DOMAIN` | 公式ドメイン（自ホスト判定用） |
| `AUTH_REQUIRED` | `true` で認証ミドルウェアを有効化（本番）。未設定または `false` で開発モード（認証スキップ） |
| `HOST_EMAILS` | ホスト権限を持つメールアドレス（カンマ区切り。admin API のアクセス制御 + プロジェクト作成時の Stripe Connect スキップ判定用） |
| `CONTACT_NOTIFY_EMAIL` | 問い合わせ受信時の通知先メールアドレス（オプション。未設定なら DB 保存のみ） |
| `LEGAL_DOCS_DIR` | 利用規約等の Markdown ファイルを配置するディレクトリ（デフォルト: `./legal/`） |
