# お知らせ機能 仕様書

## 概要

ホスト（管理者）がサイト全体のユーザーに向けてお知らせを配信する機能。メンテナンス予告、障害報告、サービスに関する告知などを INFO / WARN / ERROR の重要度アイコン付きで投稿・管理できる。ログインユーザーにはヘッダーのチャイムアイコンで新着・未読を通知する。

---

## 要件

| # | 内容 |
|---|------|
| R1 | ホストのみがお知らせを投稿・編集・非表示にできる |
| R2 | お知らせには重要度（INFO / WARN / ERROR）を設定できる |
| R3 | お知らせは「公開」「非表示」の状態を持つ |
| R4 | アプリヘッダーにチャイムアイコン（🔔）を表示する |
| R5 | ログインユーザーには未読件数をバッジで表示する |
| R6 | ユーザーがお知らせを開くと既読としてマークされる |
| R7 | 非ログインユーザーもお知らせ一覧は閲覧可能（未読管理なし） |
| R8 | お知らせ一覧ページと、ヘッダーからのドロップダウンプレビューの2つの表示を持つ |
| R9 | 日時指定投稿ができる（`published_at` を未来日時に設定すると、その時刻まで公開されない） |

---

## ユーザーストーリー

### ホスト（管理者）

- 管理画面からお知らせを新規投稿できる
- 重要度（INFO / WARN / ERROR）を選択できる
- タイトルと本文を入力できる
- 「日時指定」チェックボックスをオンにすると、公開日時を指定して予約投稿できる
- 投稿済みのお知らせを編集できる
- お知らせを非表示にできる（ソフトデリート）
- 非表示にしたお知らせを再公開できる
- お知らせの一覧を管理画面で確認できる（予約中のお知らせも含む）

### ログインユーザー

- ヘッダーのチャイムアイコンで未読のお知らせがあることに気づく
- チャイムアイコンをクリックすると最新のお知らせがドロップダウンで表示される
- ドロップダウンから「すべて見る」でお知らせ一覧ページに遷移できる
- お知らせをクリック/表示すると既読になる
- 重要度アイコンで緊急度を判断できる

### 非ログインユーザー

- ヘッダーのチャイムアイコンからお知らせを確認できる（バッジなし）
- お知らせ一覧ページを閲覧できる

---

## データモデル

### announcements テーブル

| カラム | 型 | デフォルト | NULL | 説明 |
|--------|------|-----------|------|------|
| id | UUID | gen_random_uuid() | NO | 主キー |
| author_id | UUID | — | NO | 投稿者（ホスト）の user ID |
| title | VARCHAR(200) | — | NO | タイトル |
| body | TEXT | '' | NO | 本文 |
| severity | VARCHAR(10) | 'info' | NO | 重要度: `info`, `warn`, `error` |
| visible | BOOLEAN | true | NO | 公開フラグ（false = 非表示） |
| published_at | TIMESTAMPTZ | NOW() | NO | 公開日時（並び順・予約投稿に使用。未来日時 = 予約中） |
| created_at | TIMESTAMPTZ | NOW() | NO | 作成日時 |
| updated_at | TIMESTAMPTZ | NOW() | NO | 更新日時 |

インデックス:
- `idx_announcements_visible_published` ON announcements(visible, published_at DESC) — 公開お知らせの一覧取得用

### announcement_reads テーブル

| カラム | 型 | デフォルト | NULL | 説明 |
|--------|------|-----------|------|------|
| user_id | UUID | — | NO | ユーザー ID |
| announcement_id | UUID | — | NO | お知らせ ID |
| read_at | TIMESTAMPTZ | NOW() | NO | 既読日時 |

主キー: (user_id, announcement_id)
外部キー: user_id → users(id), announcement_id → announcements(id) ON DELETE CASCADE

---

## DB マイグレーション

### マイグレーション: `030_create_announcements`

```sql
-- up
CREATE TABLE IF NOT EXISTS announcements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    severity VARCHAR(10) NOT NULL DEFAULT 'info'
        CHECK (severity IN ('info', 'warn', 'error')),
    visible BOOLEAN NOT NULL DEFAULT true,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_announcements_visible_published
    ON announcements(visible, published_at DESC);

CREATE TABLE IF NOT EXISTS announcement_reads (
    user_id UUID NOT NULL REFERENCES users(id),
    announcement_id UUID NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, announcement_id)
);

-- down
DROP TABLE IF EXISTS announcement_reads;
DROP TABLE IF EXISTS announcements;
```

---

## API エンドポイント

### GET /api/announcements

公開中のお知らせ一覧を取得する。ログインユーザーには既読情報を付与。

**抽出条件**: `visible = true AND published_at <= NOW()`
→ 非表示と予約中（未来日時）の両方を除外。スケジューラー不要。

| 項目 | 内容 |
|------|------|
| 認証 | 任意（ログイン時は既読情報付き） |
| クエリパラメータ | `limit`（デフォルト: 20, 最大: 50）、`cursor`（ページネーション） |

**レスポンス (200)**

```json
{
  "announcements": [
    {
      "id": "uuid",
      "title": "サーバーメンテナンスのお知らせ",
      "body": "2026年3月10日 02:00〜04:00 にサーバーメンテナンスを実施します。",
      "severity": "warn",
      "published_at": "2026-03-03T10:00:00Z",
      "is_read": false
    }
  ],
  "next_cursor": "cursor-string-or-null"
}
```

`is_read` は非ログイン時は常に `null`。

---

### GET /api/announcements/unread-count

ログインユーザーの未読件数を取得する。予約中のお知らせはカウント対象外。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |

**レスポンス (200)**

```json
{
  "count": 3
}
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 401 | `{ "error": "unauthorized" }` | 未認証 |

---

### POST /api/announcements/:id/read

お知らせを既読にマークする。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| リクエストボディ | なし |

**レスポンス (200)**

```json
{ "ok": true }
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 401 | `{ "error": "unauthorized" }` | 未認証 |
| 404 | `{ "error": "not_found" }` | お知らせが存在しない |

---

### POST /api/admin/announcements

お知らせを投稿する。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| 認可 | ホストのみ |

**リクエストボディ**

```json
{
  "title": "サーバーメンテナンスのお知らせ",
  "body": "2026年3月10日 02:00〜04:00 にサーバーメンテナンスを実施します。",
  "severity": "info",
  "published_at": "2026-03-10T00:00:00+09:00"
}
```

| フィールド | 必須 | バリデーション |
|-----------|------|---------------|
| title | YES | 1〜200文字 |
| body | NO | 最大5000文字 |
| severity | NO | `info`（デフォルト）, `warn`, `error` |
| published_at | NO | ISO 8601 日時文字列。省略時は `NOW()`（即時公開）。未来日時を指定すると予約投稿。過去日時も指定可能 |

**レスポンス (201)**

```json
{
  "announcement": {
    "id": "uuid",
    "title": "サーバーメンテナンスのお知らせ",
    "body": "...",
    "severity": "info",
    "visible": true,
    "published_at": "2026-03-03T10:00:00Z",
    "created_at": "2026-03-03T10:00:00Z",
    "updated_at": "2026-03-03T10:00:00Z"
  }
}
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 400 | `{ "errors": [{ "field": "title", "message": "required" }] }` | バリデーションエラー |
| 401 | `{ "error": "unauthorized" }` | 未認証 |
| 403 | `{ "error": "forbidden" }` | ホストではない |

---

### PUT /api/admin/announcements/:id

お知らせを編集する。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| 認可 | ホストのみ |

**リクエストボディ**

```json
{
  "title": "【更新】サーバーメンテナンスのお知らせ",
  "body": "メンテナンス時間が変更になりました。",
  "severity": "warn"
}
```

部分更新対応（送信されたフィールドのみ更新）。

**レスポンス (200)**

```json
{
  "announcement": { ... }
}
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 400 | `{ "errors": [...] }` | バリデーションエラー |
| 401 | `{ "error": "unauthorized" }` | 未認証 |
| 403 | `{ "error": "forbidden" }` | ホストではない |
| 404 | `{ "error": "not_found" }` | お知らせが存在しない |

---

### PATCH /api/admin/announcements/:id/visibility

お知らせの公開/非表示を切り替える。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| 認可 | ホストのみ |

**リクエストボディ**

```json
{
  "visible": false
}
```

**レスポンス (200)**

```json
{ "ok": true }
```

**エラーレスポンス**

| ステータス | body | 条件 |
|-----------|------|------|
| 401 | `{ "error": "unauthorized" }` | 未認証 |
| 403 | `{ "error": "forbidden" }` | ホストではない |
| 404 | `{ "error": "not_found" }` | お知らせが存在しない |

---

### GET /api/admin/announcements

管理画面用：全お知らせ一覧（非表示・予約中含む）。

| 項目 | 内容 |
|------|------|
| 認証 | 必須 |
| 認可 | ホストのみ |
| クエリパラメータ | `limit`（デフォルト: 20）、`cursor` |

**レスポンス (200)**

```json
{
  "announcements": [
    {
      "id": "uuid",
      "author_id": "user-uuid",
      "title": "サーバーメンテナンスのお知らせ",
      "body": "...",
      "severity": "warn",
      "visible": true,
      "published_at": "2026-03-03T10:00:00Z",
      "created_at": "2026-03-03T10:00:00Z",
      "updated_at": "2026-03-03T10:00:00Z"
    }
  ],
  "next_cursor": "cursor-string-or-null"
}
```

---

## 日時指定投稿（予約投稿）

### 設計方針

**スケジューラー不要方式**: `published_at` カラムのみで実現する。

- 投稿時に `published_at` を未来日時に設定
- 公開一覧の SQL は常に `WHERE visible = true AND published_at <= NOW()` で抽出
- 時刻が到来すれば自動的にクエリ結果に含まれる
- バックグラウンドジョブやcronは一切不要

```
投稿時 published_at = 2026-03-10 09:00 (未来)
  ↓
NOW() < published_at → クエリに含まれない（ユーザーには見えない）
  ↓
時刻到来: NOW() >= published_at → クエリに含まれる（自動公開）
```

### 管理画面での表示

- 管理者一覧（`GET /api/admin/announcements`）には予約中のお知らせも表示される
- 状態列に「🕐 予約中 YYYY/MM/DD HH:mm」と表示
- 予約中のお知らせも編集・非表示操作が可能

---

## バックエンド実装

### Model

```go
// backend/internal/model/announcement.go

type Announcement struct {
    ID          string    `json:"id"`
    AuthorID    string    `json:"author_id"`
    Title       string    `json:"title"`
    Body        string    `json:"body"`
    Severity    string    `json:"severity"`
    Visible     bool      `json:"visible"`
    PublishedAt time.Time `json:"published_at"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    // ユーザー向けレスポンス用（DB カラムではない）
    IsRead      *bool     `json:"is_read,omitempty"`
}
```

### Repository

```go
// backend/internal/repository/announcement_repository.go

type AnnouncementRepository interface {
    Insert(ctx context.Context, a *model.Announcement) error
    GetByID(ctx context.Context, id string) (*model.Announcement, error)
    Update(ctx context.Context, a *model.Announcement) error
    SetVisibility(ctx context.Context, id string, visible bool) error
    ListVisible(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
    ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
    MarkRead(ctx context.Context, userID, announcementID string) error
    UnreadCount(ctx context.Context, userID string) (int, error)
    SetReadFlags(ctx context.Context, userID string, announcements []*model.Announcement) error
}
```

### Service

```go
// backend/internal/service/announcement_service.go

type AnnouncementService interface {
    Create(ctx context.Context, a *model.Announcement) error
    Update(ctx context.Context, a *model.Announcement) error
    SetVisibility(ctx context.Context, id string, visible bool) error
    ListVisible(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error)
    ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
    MarkRead(ctx context.Context, userID, announcementID string) error
    UnreadCount(ctx context.Context, userID string) (int, error)
}
```

### Handler

```go
// backend/internal/handler/announcement_handler.go

type AnnouncementHandler struct {
    svc service.AnnouncementService
}

func NewAnnouncementHandler(svc service.AnnouncementService) *AnnouncementHandler

// ユーザー向け
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request)        // GET /api/announcements
func (h *AnnouncementHandler) UnreadCount(w http.ResponseWriter, r *http.Request)  // GET /api/announcements/unread-count
func (h *AnnouncementHandler) MarkRead(w http.ResponseWriter, r *http.Request)     // POST /api/announcements/:id/read

// 管理者向け
func (h *AnnouncementHandler) AdminList(w http.ResponseWriter, r *http.Request)       // GET /api/admin/announcements
func (h *AnnouncementHandler) AdminCreate(w http.ResponseWriter, r *http.Request)     // POST /api/admin/announcements
func (h *AnnouncementHandler) AdminUpdate(w http.ResponseWriter, r *http.Request)     // PUT /api/admin/announcements/:id
func (h *AnnouncementHandler) AdminSetVisibility(w http.ResponseWriter, r *http.Request) // PATCH /api/admin/announcements/:id/visibility
```

### ルーティング登録（main.go）

```go
// ユーザー向け
mux.HandleFunc("GET /api/announcements", announcementHandler.List)
mux.HandleFunc("GET /api/announcements/unread-count", announcementHandler.UnreadCount)
mux.HandleFunc("POST /api/announcements/{id}/read", announcementHandler.MarkRead)

// 管理者向け
mux.HandleFunc("GET /api/admin/announcements", announcementHandler.AdminList)
mux.HandleFunc("POST /api/admin/announcements", announcementHandler.AdminCreate)
mux.HandleFunc("PUT /api/admin/announcements/{id}", announcementHandler.AdminUpdate)
mux.HandleFunc("PATCH /api/admin/announcements/{id}/visibility", announcementHandler.AdminSetVisibility)
```

---

## フロントエンド UI 仕様

### ヘッダーのチャイムアイコン

- **配置**: ヘッダーナビゲーション内、AuthStatus の左隣
- **コンポーネント**: `NavAnnouncementBell`（React island, `client:load`）
- **表示**:
  - 常時: 🔔 チャイムアイコン（SVG ベルアイコン）
  - ログインユーザーで未読あり: 赤丸バッジに未読件数を表示（9+ は「9+」）
  - ログインユーザーで未読なし: バッジなし
  - 非ログインユーザー: バッジなし

```
[🔔]      ← 未読なし / 非ログイン
[🔔 ³]    ← 未読3件
[🔔 9+]   ← 未読10件以上
```

- **クリック動作**: ドロップダウンパネルを開閉（トグル）
- **ドロップダウン外クリック**: パネルを閉じる

### ドロップダウンパネル

- **表示位置**: チャイムアイコンの直下、右寄せ
- **最大表示件数**: 5件（最新順）
- **各項目の表示**:

```
┌─────────────────────────────────────┐
│ お知らせ                       すべて見る │
├─────────────────────────────────────┤
│ ⓘ サーバーメンテナンスのお知らせ     │
│   3月10日 02:00〜04:00に...   3/3  │
│                                ●未読 │
├─────────────────────────────────────┤
│ ⚠ 決済遅延のお知らせ               │
│   現在Stripe側の遅延により...  3/1  │
├─────────────────────────────────────┤
│ ⓘ 新機能リリースのお知らせ          │
│   返金機能を追加しました。     2/28 │
└─────────────────────────────────────┘
```

- **重要度アイコン**:
  - INFO: `ⓘ` 青色（`#3b82f6`）
  - WARN: `⚠` 黄色（`#f59e0b`）
  - ERROR: `⛔` 赤色（`#ef4444`）
- **未読表示**: 未読アイテムは背景色を薄く（`rgba(59, 130, 246, 0.05)`）+ 右に青丸ドット
- **「すべて見る」リンク**: `/announcements` ページへ遷移
- **空の場合**: 「お知らせはありません」を表示

### ドロップダウン既読処理

- ドロップダウンを開いた時点で、表示されている未読のお知らせすべてを既読にマークする
- 既読 API は各お知らせに対して `POST /api/announcements/:id/read` を呼ぶ
- バッジの件数は API レスポンス後に更新

### お知らせ一覧ページ（/announcements）

- **パス**: `/announcements`（ja）、`/en/announcements`（en）
- **レイアウト**: BaseLayout 使用
- **ページタイトル**: 「お知らせ」
- **コンポーネント**: `AnnouncementList`（React island）

```
┌─────────────────────────────────────────────────┐
│ お知らせ                                         │
├─────────────────────────────────────────────────┤
│ ⚠ 決済遅延のお知らせ              2026年3月1日   │
│                                                   │
│ 現在Stripe側の遅延により、一部の決済処理に       │
│ 時間がかかっています。復旧次第お知らせします。     │
├─────────────────────────────────────────────────┤
│ ⓘ 新機能リリースのお知らせ       2026年2月28日   │
│                                                   │
│ 返金機能を追加しました。寄付者・プロジェクト       │
│ オーナーともに寄付の返金が可能になりました。       │
├─────────────────────────────────────────────────┤
│              [もっと読み込む]                      │
└─────────────────────────────────────────────────┘
```

- 各アイテム: 重要度アイコン + タイトル + 日付 + 本文
- 未読はタイトル太字 + 薄い背景色
- ページネーション: 「もっと読み込む」ボタン（cursor-based）
- 一覧を開いた時点で表示されているお知らせを既読にマークする

### 管理画面（/admin/announcements）

- **パス**: `/admin/announcements`（ja）、`/en/admin/announcements`（en）
- **認可**: ホストのみアクセス可能
- **コンポーネント**: `AdminAnnouncementList`（React island）

**一覧表示**:

| 重要度 | タイトル | 状態 | 公開日 | 操作 |
|-------|---------|------|-------|------|
| ⓘ | サーバーメンテナンスのお知らせ | 公開中 | 2026/3/3 | [編集] [非表示] |
| ⚠ | 定期メンテナンスのお知らせ | 🕐 予約中 3/10 09:00 | — | [編集] [非表示] |
| ⚠ | 旧お知らせ | 非表示 | 2026/2/1 | [編集] [公開] |

**状態の判定ロジック**:

| 条件 | 表示 |
|------|------|
| `visible = false` | 非表示 |
| `visible = true` かつ `published_at > NOW()` | 🕐 予約中 {日時} |
| `visible = true` かつ `published_at <= NOW()` | 公開中 |

**新規投稿フォーム**:

- タイトル: テキスト入力（必須、200文字以内）
- 本文: textarea（4行）
- 重要度: セレクトボックス（INFO / WARN / ERROR）
- 日時指定: チェックボックス「公開日時を指定する」
  - オフ（デフォルト）: 即時公開
  - オン: `datetime-local` 入力フィールドが表示される（デフォルト値: 現在日時の1時間後、切り上げ）
- 送信ボタン: 日時指定オフ → 「投稿」 / 日時指定オン → 「予約投稿」

```
┌────────────────────────────────────────┐
│ 新しいお知らせを投稿                      │
├────────────────────────────────────────┤
│ タイトル: [________________________]    │
│ 本文:                                    │
│ [                                    ]  │
│ [                                    ]  │
│ 重要度:   [INFO ▾]                       │
│                                          │
│ ☑ 公開日時を指定する                     │
│   公開日時: [2026-03-10T09:00]          │
│                                          │
│              [予約投稿]                   │
└────────────────────────────────────────┘
```

**編集**:

- 投稿フォームと同じフィールドを編集モードで表示
- 予約中のお知らせは日時指定チェックボックスがオンの状態で表示
- 公開済みのお知らせでも `published_at` を未来に変更すると再び予約状態になる
- 「更新」ボタンで保存

---

## i18n 文字列

### 日本語（ja.json）

| キー | 値 |
|------|-----|
| `nav.announcements` | `お知らせ` |
| `announcements.title` | `お知らせ` |
| `announcements.empty` | `お知らせはありません` |
| `announcements.viewAll` | `すべて見る` |
| `announcements.loadMore` | `もっと読み込む` |
| `announcements.severityInfo` | `情報` |
| `announcements.severityWarn` | `注意` |
| `announcements.severityError` | `障害` |
| `announcements.unread` | `未読` |
| `admin.announcements` | `お知らせ管理` |
| `admin.announcementCreate` | `新しいお知らせを投稿` |
| `admin.announcementEdit` | `お知らせを編集` |
| `admin.announcementTitle` | `タイトル` |
| `admin.announcementBody` | `本文` |
| `admin.announcementSeverity` | `重要度` |
| `admin.announcementVisible` | `公開中` |
| `admin.announcementHidden` | `非表示` |
| `admin.announcementPublish` | `公開` |
| `admin.announcementHide` | `非表示にする` |
| `admin.announcementSubmit` | `投稿` |
| `admin.announcementSubmitScheduled` | `予約投稿` |
| `admin.announcementUpdate` | `更新` |
| `admin.announcementSchedule` | `公開日時を指定する` |
| `admin.announcementScheduledAt` | `公開日時` |
| `admin.announcementStatusPublished` | `公開中` |
| `admin.announcementStatusScheduled` | `予約中` |
| `admin.announcementStatusHidden` | `非表示` |

### 英語（en.json）

| キー | 値 |
|------|-----|
| `nav.announcements` | `Announcements` |
| `announcements.title` | `Announcements` |
| `announcements.empty` | `No announcements` |
| `announcements.viewAll` | `View all` |
| `announcements.loadMore` | `Load more` |
| `announcements.severityInfo` | `Info` |
| `announcements.severityWarn` | `Warning` |
| `announcements.severityError` | `Incident` |
| `announcements.unread` | `Unread` |
| `admin.announcements` | `Announcement Management` |
| `admin.announcementCreate` | `Post new announcement` |
| `admin.announcementEdit` | `Edit announcement` |
| `admin.announcementTitle` | `Title` |
| `admin.announcementBody` | `Body` |
| `admin.announcementSeverity` | `Severity` |
| `admin.announcementVisible` | `Published` |
| `admin.announcementHidden` | `Hidden` |
| `admin.announcementPublish` | `Publish` |
| `admin.announcementHide` | `Hide` |
| `admin.announcementSubmit` | `Post` |
| `admin.announcementSubmitScheduled` | `Schedule` |
| `admin.announcementUpdate` | `Update` |
| `admin.announcementSchedule` | `Schedule publish date` |
| `admin.announcementScheduledAt` | `Publish date` |
| `admin.announcementStatusPublished` | `Published` |
| `admin.announcementStatusScheduled` | `Scheduled` |
| `admin.announcementStatusHidden` | `Hidden` |

---

## 変更ファイル一覧

### 新規ファイル

| ファイル | 内容 |
|---------|------|
| `backend/migrations/030_create_announcements.up.sql` | announcements, announcement_reads テーブル作成 |
| `backend/migrations/030_create_announcements.down.sql` | ロールバック |
| `backend/internal/model/announcement.go` | Announcement 構造体 |
| `backend/internal/repository/announcement_repository.go` | AnnouncementRepository インターフェース |
| `backend/internal/repository/pg_announcement_repository.go` | PostgreSQL 実装 |
| `backend/internal/service/announcement_service.go` | AnnouncementService |
| `backend/internal/handler/announcement_handler.go` | HTTP ハンドラー |
| `frontend/src/components/react/NavAnnouncementBell.tsx` | ヘッダーのチャイムアイコン + ドロップダウン |
| `frontend/src/components/react/AnnouncementList.tsx` | お知らせ一覧ページコンポーネント |
| `frontend/src/components/react/AdminAnnouncementList.tsx` | 管理画面お知らせコンポーネント |
| `frontend/src/pages/announcements.astro` | お知らせ一覧ページ（ja） |
| `frontend/src/pages/en/announcements.astro` | お知らせ一覧ページ（en） |
| `frontend/src/pages/admin/announcements.astro` | 管理画面お知らせページ（ja） |
| `frontend/src/pages/en/admin/announcements.astro` | 管理画面お知らせページ（en） |

### 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `backend/cmd/server/main.go` | お知らせ関連エンドポイント7つのルーティング登録 |
| `frontend/src/layouts/BaseLayout.astro` | ヘッダーに `NavAnnouncementBell` コンポーネント追加 |
| `frontend/src/lib/api.ts` | お知らせ関連の型定義・API 関数追加 |
| `frontend/src/lib/mock-api.ts` | お知らせモック実装追加 |
| `frontend/src/i18n/ja.json` | お知らせ関連文字列追加 |
| `frontend/src/i18n/en.json` | お知らせ関連文字列追加 |
| `frontend/src/styles/global.css` | チャイムアイコン・バッジ・ドロップダウンのスタイル追加 |
