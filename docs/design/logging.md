# Backend ログ仕様

## 概要

バックエンドは Go 標準ライブラリ `log/slog` を使用した構造化ログを出力する。

- **フォーマット**: JSON（1行1レコード）
- **出力先**: stdout
- **ログレベル制御**: 環境変数 `LOG_LEVEL`

## 環境変数

| 変数 | 値 | デフォルト | 説明 |
|------|-----|----------|------|
| `LOG_LEVEL` | `DEBUG`, `INFO`, `WARN`, `ERROR` | `INFO` | 最小出力レベル |

## ログレベル設計

| Level | 用途 | 例 |
|-------|------|-----|
| **ERROR** | 内部障害（DB失敗、セッション作成失敗） | `"project create failed"`, `"session create: db insert failed"` |
| **WARN** | 外部/クライアント起因の失敗 | OAuth state 不正、期限切れセッション、milestone fire-and-forget エラー、Stripe 連携失敗 |
| **INFO** | ビジネスイベント・運用情報 | サーバー起動、HTTPリクエストログ、新規ユーザー作成、ログイン成功、マイグレーション完了 |
| **DEBUG** | リクエスト毎のトレース情報 | CORS、auth middleware、セッション検証、cookie/token 詳細 |

## ログ出力例

### INFO（リクエストログ）

```json
{"time":"2026-02-26T10:00:00.000Z","level":"INFO","msg":"request","method":"GET","path":"/api/health","status":200,"duration_ms":1,"remote_addr":"127.0.0.1:52345"}
```

### INFO（ビジネスイベント）

```json
{"time":"2026-02-26T10:00:00.000Z","level":"INFO","msg":"new user created","user_id":"abc123","provider":"google"}
{"time":"2026-02-26T10:00:00.000Z","level":"INFO","msg":"finalize login success","cookie_name":"givers_session","max_age":2592000}
```

### ERROR

```json
{"time":"2026-02-26T10:00:00.000Z","level":"ERROR","msg":"project create failed","error":"pq: duplicate key","user_id":"abc123"}
```

### WARN

```json
{"time":"2026-02-26T10:00:00.000Z","level":"WARN","msg":"auth rejected: validation failed","error":"invalid_session"}
{"time":"2026-02-26T10:00:00.000Z","level":"WARN","msg":"milestone: insert failed","project_id":"proj-1","threshold":50,"error":"context canceled"}
```

### DEBUG（`LOG_LEVEL=DEBUG` のみ）

```json
{"time":"2026-02-26T10:00:00.000Z","level":"DEBUG","msg":"cors","method":"GET","path":"/api/health","origin":"http://localhost:4321","allow_origin":"http://localhost:4321"}
{"time":"2026-02-26T10:00:00.000Z","level":"DEBUG","msg":"session validate: ok","user_id":"abc123"}
```

## アーキテクチャ

```
internal/logging/logging.go   ← slog 初期化 (Setup, Fatal)
internal/handler/request_logger.go ← HTTPリクエストログミドルウェア
```

### ミドルウェアチェーン

```
RequestLogger → SecurityHeaders → CORS → mux
```

`RequestLogger` が最外層で、全リクエストの method / path / status / duration_ms / remote_addr を記録する。

### 初期化

`cmd/server/main.go` と `cmd/migrate/main.go` の先頭で `logging.Setup()` を呼ぶ。
`slog.SetDefault()` によりグローバルに JSON ハンドラが設定される。

## セキュリティポリシー

- **PII（メールアドレス等）** は `DEBUG` レベルのみに記載可。`INFO` / `WARN` / `ERROR` には含めない。
- **シークレット（トークン、ワンタイムコード等）** はログに記載しない。トークンプレフィックス（先頭16文字）のみ `DEBUG` で許可。
- **Cookie ヘッダ** はログに出力しない。

## 推奨運用設定

| 環境 | LOG_LEVEL | 備考 |
|------|-----------|------|
| 開発 | `DEBUG` | CORS / auth / session のトレース情報を含む |
| ステージング | `INFO` | リクエストログ + ビジネスイベント |
| 本番 | `INFO` | 問題発生時に一時的に `DEBUG` に変更可 |

## ログ収集

stdout に出力された JSON は、デプロイ環境に応じて以下のように収集する：

- **Docker / ECS**: コンテナランタイムが stdout を自動収集 → CloudWatch Logs
- **systemd**: `journalctl -u givers-backend` で参照
- **ローカル開発**: ターミナルに直接出力
