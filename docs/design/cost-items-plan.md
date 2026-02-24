# コスト項目カスタマイズ 設計書

## 背景・課題

現在の `project_costs` テーブルは以下の**固定カラム**でコスト内訳を管理している。

| カラム | 意味 |
|--------|------|
| `server_cost_monthly` | サーバー費用（月額） |
| `dev_cost_per_day` | 開発者コスト（1日あたり） |
| `dev_days_per_month` | 月あたり稼働日数 |
| `other_cost_monthly` | その他費用（月額） |

**問題点**: ラベルが英語固定・4項目固定のため、
- 個人開発者が「自分の人件費（月額）」を 1 行で書きたい場合に不自由
- 組織が「家賃」「サーバー（AWS）」「サーバー（Cloudflare）」と細分化したい場合に対応できない
- そもそも項目名が日本語で表示されていない

---

## 設計方針

### 核心アイデア

> **コスト項目を「ラベル付きの行」の配列**として扱う。
> 行の数・名前はオーナーが自由に設定できる。

### レイヤー設計

```
システムデフォルト（初期表示用の雛形）
       ↓  オーナーが「テンプレート」として保存
ユーザーレベルのテンプレート（user_cost_presets）
       ↓  プロジェクト作成時に自動コピー
プロジェクトレベルの実際のコスト（project_cost_items）
```

- **システムデフォルト**: ハードコードされた 4 行（現行の内容をラベル付きで再現）
- **ユーザーテンプレート**: 「毎回同じ構成を使う人」向けに保存できる雛形
- **プロジェクト実値**: 実際の金額を持つ行。テンプレートからコピーして生成、独立して変更可能

### `unit_type`（計算単位）

| 値 | 意味 | 必要フィールド |
|----|------|----------------|
| `monthly` | 月額固定 | `amount_monthly` |
| `daily_x_days` | 人日単価 × 稼働日数 | `rate_per_day`, `days_per_month` |

月額換算は `monthly` → `amount_monthly`、`daily_x_days` → `rate_per_day × days_per_month` で計算する。

---

## データモデル

### `user_cost_presets`（ユーザーレベルのテンプレート）

```sql
CREATE TABLE user_cost_presets (
    id           VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id      VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label        VARCHAR(100) NOT NULL,       -- 例: "サーバー費用"
    unit_type    VARCHAR(20) NOT NULL         -- "monthly" | "daily_x_days"
                 CHECK (unit_type IN ('monthly', 'daily_x_days')),
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_cost_presets_user_id ON user_cost_presets(user_id);
```

> テンプレートは「ラベルと計算単位」だけを保存する。金額はプロジェクトごとに異なるので持たない。

### `project_cost_items`（プロジェクトレベルの実値）

```sql
CREATE TABLE project_cost_items (
    id             VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id     VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    label          VARCHAR(100) NOT NULL,
    unit_type      VARCHAR(20) NOT NULL
                   CHECK (unit_type IN ('monthly', 'daily_x_days')),
    amount_monthly INT NOT NULL DEFAULT 0,   -- unit_type = 'monthly' のとき使用
    rate_per_day   INT NOT NULL DEFAULT 0,   -- unit_type = 'daily_x_days' のとき使用
    days_per_month INT NOT NULL DEFAULT 0,   -- unit_type = 'daily_x_days' のとき使用
    sort_order     INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_cost_items_project_id ON project_cost_items(project_id);
```

### 既存テーブルとの移行戦略

| 対象 | 対応 |
|------|------|
| `project_costs` テーブル | 廃止。既存データを `project_cost_items` へマイグレーション |
| `ProjectCosts` モデル | 廃止。`[]ProjectCostItem` に置き換え |
| `project_handler.go` の `costs` フィールド | 配列形式に変更 |

---

## システムデフォルト項目

オーナーがテンプレートを未設定の場合、プロジェクト作成時に以下の 4 行を雛形として提示する（バックエンドで生成、フロントで表示）。

| sort_order | label | unit_type | 備考 |
|------------|-------|-----------|------|
| 0 | サーバー費用 | monthly | 旧 server_cost_monthly |
| 1 | 開発者費用 | daily_x_days | 旧 dev_cost_per_day × dev_days_per_month |
| 2 | その他費用 | monthly | 旧 other_cost_monthly |

> ユーザーテンプレートが 1 件以上存在する場合はシステムデフォルトを使わず、テンプレートを雛形として使う。

---

## API 仕様

### ユーザーテンプレート API

#### `GET /api/me/cost-presets`

ユーザー自身のコスト項目テンプレート一覧を返す。

**認証**: 必須

**レスポンス (200)**
```json
{
  "presets": [
    {
      "id": "uuid",
      "label": "サーバー費用",
      "unit_type": "monthly",
      "sort_order": 0
    },
    {
      "id": "uuid",
      "label": "開発者費用",
      "unit_type": "daily_x_days",
      "sort_order": 1
    }
  ]
}
```

テンプレートが未設定の場合は `"presets": []` を返す（システムデフォルトはフロントが補完）。

---

#### `POST /api/me/cost-presets`

テンプレート項目を追加する。

**認証**: 必須

**リクエスト**
```json
{
  "label": "AWS費用",
  "unit_type": "monthly"
}
```

| フィールド | 必須 | 制約 |
|-----------|------|------|
| `label` | ◎ | 1〜100文字 |
| `unit_type` | ◎ | `"monthly"` または `"daily_x_days"` |

**レスポンス (201)**
```json
{
  "id": "uuid",
  "label": "AWS費用",
  "unit_type": "monthly",
  "sort_order": 3
}
```

> `sort_order` は既存の最大値 + 1 で自動採番。

---

#### `PUT /api/me/cost-presets/:id`

テンプレート項目のラベルまたは unit_type を更新する。

**認証**: 必須（本人のみ）

**リクエスト**（変更したいフィールドのみ）
```json
{
  "label": "サーバー費用（AWS）",
  "unit_type": "monthly"
}
```

**レスポンス (200)**
```json
{
  "id": "uuid",
  "label": "サーバー費用（AWS）",
  "unit_type": "monthly",
  "sort_order": 0
}
```

---

#### `DELETE /api/me/cost-presets/:id`

テンプレート項目を削除する。

**認証**: 必須（本人のみ）

**レスポンス (200)**
```json
{ "ok": true }
```

---

#### `PUT /api/me/cost-presets/reorder`

テンプレートの並び順を一括変更する。

**認証**: 必須（本人のみ）

**リクエスト**
```json
{
  "ids": ["uuid-c", "uuid-a", "uuid-b"]
}
```

- `ids` に含まれる全 ID が本人のものであることをバックエンドで検証
- 先頭から 0, 1, 2… と `sort_order` を更新する

**レスポンス (200)**
```json
{ "ok": true }
```

---

### プロジェクト API 変更点

プロジェクト作成・更新時の `costs` フィールドを**固定オブジェクトから配列**に変更する。

#### `POST /api/projects`（変更後）

```json
{
  "name": "オープンソースの軽量エディタ",
  "description": "...",
  "deadline": "2027-03-31",
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

| フィールド | 必須 | 制約 |
|-----------|------|------|
| `cost_items` | ✕ | 省略時は空配列（目標金額 0 円） |
| `cost_items[].label` | ◎ | 1〜100文字 |
| `cost_items[].unit_type` | ◎ | `"monthly"` \| `"daily_x_days"` |
| `cost_items[].amount_monthly` | `monthly` 時のみ | 0 以上の整数（円） |
| `cost_items[].rate_per_day` | `daily_x_days` 時のみ | 0 以上の整数（円） |
| `cost_items[].days_per_month` | `daily_x_days` 時のみ | 0〜31 |

**月額換算ロジック（サーバーサイド）**

```
monthly_total = Σ cost_items[i].monthly_amount()

monthly_amount() =
  unit_type == "monthly"      → amount_monthly
  unit_type == "daily_x_days" → rate_per_day × days_per_month
```

`monthly_total` がプロジェクトの月額目標として `projects.monthly_target` に保存される（非正規化キャッシュ。コスト変更時に更新）。

#### `GET /api/projects/:id`（レスポンスの変更）

`costs` オブジェクトを廃止し、`cost_items` 配列を返す。

```json
{
  "id": "uuid",
  "name": "...",
  "cost_items": [
    {
      "id": "uuid",
      "label": "サーバー費用",
      "unit_type": "monthly",
      "amount_monthly": 10000,
      "rate_per_day": 0,
      "days_per_month": 0,
      "monthly_amount": 10000,
      "sort_order": 0
    },
    {
      "id": "uuid",
      "label": "開発者費用",
      "unit_type": "daily_x_days",
      "amount_monthly": 0,
      "rate_per_day": 20000,
      "days_per_month": 5,
      "monthly_amount": 100000,
      "sort_order": 1
    }
  ],
  "monthly_target": 115000,
  "..."
}
```

> `monthly_amount` は計算済みの値（表示用）。フロント側で計算させない。

---

## UX フロー

### プロジェクト作成フォーム（コスト内訳セクション）

```
┌─────────────────────────────────────────┐
│ コスト内訳                               │
│                                          │
│ ┌──────────────┬───────────┬──────────┐  │
│ │ 項目名        │ 計算方法  │  月額    │  │
│ ├──────────────┼───────────┼──────────┤  │
│ │ サーバー費用  │ 月額固定  │ 10,000円 │🗑│
│ ├──────────────┼───────────┼──────────┤  │
│ │ 開発者費用    │ 人日×日数 │──────── │🗑│
│ │              │ 単価: 20,000円/日     │  │
│ │              │ 日数: 5日/月          │  │
│ │              │ → 100,000円/月        │  │
│ ├──────────────┼───────────┼──────────┤  │
│ │ その他費用    │ 月額固定  │  5,000円 │🗑│
│ └──────────────┴───────────┴──────────┘  │
│                                          │
│ [+ 項目を追加]  [テンプレートから読込]   │
│                                          │
│ 月額合計: 115,000円                      │
└─────────────────────────────────────────┘
```

- 行の追加・削除・並び替え（drag or ↑↓ボタン）が可能
- 「テンプレートから読込」→ユーザーが保存したテンプレートを上書き適用
- 項目名は自由入力（例: 「Cloudflare R2 費用」「デザイン費」「自分の生活費」）

### ユーザー設定 → テンプレート管理

```
ユーザー設定 > コスト項目テンプレート

[現在のテンプレート]
  ☰  サーバー費用     月額固定     [編集] [削除]
  ☰  開発者費用       人日×日数    [編集] [削除]
  ☰  その他費用       月額固定     [編集] [削除]

[+ 項目を追加]

※ テンプレートは新規プロジェクト作成時の初期値として使われます。
```

- テンプレート未設定時はシステムデフォルト（上記の3行）が新規作成時の初期値
- テンプレートを保存しておくと「毎回同じ構成」のオーナーが楽になる

---

## 移行・後方互換

### DB マイグレーション

```
migration 012: user_cost_presets テーブルを作成
migration 013: project_cost_items テーブルを作成
migration 014: project_costs → project_cost_items へデータ移行 + project_costs を削除
```

移行 SQL（014）のイメージ:

```sql
INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
SELECT
  project_id,
  'サーバー費用',
  'monthly',
  server_cost_monthly,
  0, 0, 0
FROM project_costs
WHERE server_cost_monthly > 0;

INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
SELECT
  project_id,
  '開発者費用',
  'daily_x_days',
  0,
  dev_cost_per_day,
  dev_days_per_month,
  1
FROM project_costs
WHERE dev_cost_per_day > 0 OR dev_days_per_month > 0;

INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
SELECT
  project_id,
  'その他費用',
  'monthly',
  other_cost_monthly,
  0, 0, 2
FROM project_costs
WHERE other_cost_monthly > 0;

DROP TABLE project_costs;
```

### `projects` テーブルへの `monthly_target` カラム追加

```sql
ALTER TABLE projects ADD COLUMN monthly_target INT NOT NULL DEFAULT 0;

-- 既存プロジェクトの monthly_target を project_cost_items から計算して更新
UPDATE projects p SET monthly_target = (
  SELECT COALESCE(SUM(
    CASE unit_type
      WHEN 'monthly'      THEN amount_monthly
      WHEN 'daily_x_days' THEN rate_per_day * days_per_month
    END
  ), 0)
  FROM project_cost_items
  WHERE project_id = p.id
);
```

---

## 新規エンドポイント一覧

| Method | Path | 認証 | 説明 |
|--------|------|------|------|
| GET | `/api/me/cost-presets` | 必須 | テンプレート一覧 |
| POST | `/api/me/cost-presets` | 必須 | テンプレート追加 |
| PUT | `/api/me/cost-presets/:id` | 必須（本人） | テンプレート更新 |
| DELETE | `/api/me/cost-presets/:id` | 必須（本人） | テンプレート削除 |
| PUT | `/api/me/cost-presets/reorder` | 必須（本人） | 並び順変更 |

既存エンドポイントの変更:

| Method | Path | 変更内容 |
|--------|------|----------|
| POST | `/api/projects` | `costs` オブジェクト → `cost_items` 配列 |
| PUT | `/api/projects/:id` | 同上 |
| GET | `/api/projects/:id` | `costs` → `cost_items` 配列 + `monthly_target` |
| GET | `/api/projects` | 一覧に `monthly_target` を追加（`cost_items` は詳細のみ） |

---

## 実装優先度

| 優先度 | 対象 | 理由 |
|--------|------|------|
| P0 | DB マイグレーション (012-014) | 他の実装のベース |
| P0 | `POST/PUT /api/projects` の `cost_items` 対応 | プロジェクト作成に必要 |
| P0 | `GET /api/projects/:id` の `cost_items` 返却 | フロント表示に必要 |
| P1 | `GET/POST/PUT/DELETE /api/me/cost-presets` | テンプレート管理 |
| P1 | `PUT /api/me/cost-presets/reorder` | 並び替え |
| P2 | フロント: テンプレート管理ページ | 設定画面 |

---

## 未決事項

| 論点 | 選択肢 | 現時点の仮決め |
|------|--------|----------------|
| コスト項目の上限数 | 無制限 / 10 項目まで / 20 項目まで | **10 項目まで**（バリデーション） |
| テンプレートの上限数 | 同上 | **10 項目まで** |
| ラベルの最大文字数 | 50 / 100 文字 | **100 文字** |
| `projects` テーブルへの `monthly_target` キャッシュ | 正規化なし（毎回集計）/ 非正規化（キャッシュ） | **キャッシュ**（一覧表示で毎回 JOIN させない） |
| 既存 `project_costs` の後方互換 | API レスポンスに両方を返す移行期間 / 即廃止 | **即廃止**（フロントはまだ本番接続していないため） |
