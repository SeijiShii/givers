# Phase 3 実装プラン: プロジェクト CRUD

## 認証の方針

- **認証ミドルウェアを注入する方式**で実装
- **開発中は認証なし**で動作（ミドルウェアをバイパス）
- 本番では認証ミドルウェアを有効化

### ミドルウェアの切り替え

```
main.go でのルーティング例:

  // 認証不要（常に）
  mux.Handle("GET /api/projects", handler.ListProjects(...))
  mux.Handle("GET /api/projects/{id}", handler.GetProject(...))

  // 認証必要（ミドルウェアで制御）
  if authEnabled {
    mux.Handle("POST /api/projects", authMiddleware(handler.CreateProject(...)))
    mux.Handle("PUT /api/projects/{id}", authMiddleware(handler.UpdateProject(...)))
    mux.Handle("GET /api/me/projects", authMiddleware(handler.MyProjects(...)))
  } else {
    mux.Handle("POST /api/projects", handler.CreateProject(...))  // 開発用: 認証スキップ
    mux.Handle("PUT /api/projects/{id}", handler.UpdateProject(...))
    mux.Handle("GET /api/me/projects", handler.MyProjects(...))
  }
```

### 環境変数

| 変数 | 値 | 動作 |
|------|-----|------|
| AUTH_REQUIRED | `true` | 認証ミドルウェアを適用（本番） |
| AUTH_REQUIRED | `false` または未設定 | 認証なしで通過（開発） |

### ミドルウェアの責務

- **有効時**: セッション Cookie を検証し、`context` に userID をセット。未認証なら 401 を返す
- **無効時**: 開発用のダミー userID を `context` にセットするか、そのまま通過

---

## 実装範囲

- プロジェクト CRUD API
- 費用設定（project_costs）
- アラート閾値（project_alerts）
- フロント: 一覧・詳細・作成・編集・マイページ

## データモデル

- projects テーブル
- project_costs テーブル
- project_alerts テーブル

詳細は実装時に定義。
