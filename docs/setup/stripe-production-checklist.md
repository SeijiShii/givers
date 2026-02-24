# Stripe 本番移行セキュリティチェックリスト

最終更新: 2026-02-24

> 参照元:
> - [Go-live checklist | Stripe Documentation](https://docs.stripe.com/get-started/checklist/go-live)
> - [Account checklist | Stripe Documentation](https://docs.stripe.com/get-started/account/checklist)
> - [Integration security guide | Stripe Documentation](https://docs.stripe.com/security/guide)

---

## アプリ層（コード変更が必要）

### 対応済み

| # | 項目 | 実装箇所 | 詳細 |
|---|------|----------|------|
| A1 | Webhook 署名検証 | `pkg/stripe/client.go` | HMAC-SHA256 + タイムスタンプ（5分以内）+ `hmac.Equal` 定数時間比較 |
| A2 | Webhook 冪等性 | `internal/service/stripe_service.go` | 重複寄付は `ErrDuplicate` で無視 |
| A3 | Cookie セキュリティ | `internal/handler/auth_handler.go`, `stripe_handler.go` | HttpOnly, SameSite=Lax, 本番時 Secure=true |
| A4 | OAuth CSRF 対策 | `internal/handler/auth_handler.go` | サーバー側 state 検証 + 10分有効期限 + ワンタイム使用 |
| A5 | セッショントークン生成 | `pkg/auth/session.go` | `crypto/rand` 32バイト + hex エンコード |
| A6 | CORS 制御 | `internal/handler/handler.go` | `FRONTEND_URL` によるオリジン制限（ワイルドカードなし） |
| A7 | Stripe Checkout 利用 | `pkg/stripe/client.go` | カード情報がサーバーに触れない設計（PCI 負担最小化） |

### 未対応

#### A8. セキュリティヘッダーミドルウェア追加

**対象ファイル**: `backend/internal/handler/handler.go`

`handler.go` にミドルウェアを追加し、`main.go` で適用する。

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
        w.Header().Set("Content-Security-Policy",
            "default-src 'self'; "+
                "connect-src 'self' https://checkout.stripe.com; "+
                "frame-src https://checkout.stripe.com; "+
                "script-src 'self' https://checkout.stripe.com; "+
                "img-src 'self' https://*.stripe.com; "+
                "style-src 'self' 'unsafe-inline'")
        next.ServeHTTP(w, r)
    })
}
```

#### A9. レートリミット追加

**対象ファイル**: `backend/internal/handler/handler.go`（新規ミドルウェア）, `backend/cmd/server/main.go`

IP ベースのレートリミッターを導入する。

| エンドポイント | 制限 | 理由 |
|---|---|---|
| `POST /api/donations/checkout` | 10 req/min per IP | Checkout Session 乱造防止 |
| `POST /api/webhooks/stripe` | 100 req/min | Stripe からの正常な通知量に十分 |
| その他 API | 60 req/min per IP | 一般的な API 保護 |

実装候補:
- `golang.org/x/time/rate` による Token Bucket
- `sync.Map` で IP ごとの `rate.Limiter` を管理

#### A10. Statement Descriptor 設定

**対象ファイル**: `backend/pkg/stripe/client.go` (`CreateCheckoutSession`)

Checkout Session 作成時に `payment_intent_data[statement_descriptor_suffix]` を設定する。
顧客の銀行明細にプロジェクト名（最大22文字）が表示される。

```go
// 一回限り寄付の場合
params.Set("payment_intent_data[statement_descriptor_suffix]", truncate(projectName, 22))

// 定期寄付の場合（Stripe Dashboard のサブスクリプション設定で対応）
```

#### A11. ログの機密情報除去

**対象ファイル**: `backend/internal/handler/auth_handler.go`, `backend/pkg/auth/middleware.go`

現在 `userID` やセッショントークンプレフィックスがログに出力されている。

対応:
- `userID` → ハッシュ化 or 末尾4文字のみ出力
- セッショントークンのプレフィックスログ → 削除
- Cookie ヘッダーのログ出力 → 削除

#### A12. HTTP サーバー IdleTimeout 追加

**対象ファイル**: `backend/cmd/server/main.go`

```go
server := &http.Server{
    Addr:         ":8080",
    Handler:      h.CORS(mux),
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout:  120 * time.Second,  // 追加
}
```

#### A13. Stripe エラー型の区別

**対象ファイル**: `backend/pkg/stripe/client.go`

Stripe API レスポンスの `error.type` を解析し、種別ごとに適切なエラーを返す。

| Stripe エラー型 | アプリでの対応 |
|---|---|
| `card_error` | ユーザーにカード問題を通知（再試行を促す） |
| `invalid_request_error` | 内部エラーとしてログ（開発バグ） |
| `rate_limit_error` | リトライ or 503 返却 |
| `authentication_error` | アラート通知（API キー問題） |

---

## インフラ層（サーバー設定・運用で対応）

### 対応済み

| # | 項目 | 実装箇所 | 詳細 |
|---|------|----------|------|
| B1 | HTTP タイムアウト | `cmd/server/main.go` | Read/Write 各10秒（Slowloris 対策） |
| B2 | Graceful Shutdown | `cmd/server/main.go` | SIGINT/SIGTERM で5秒猶予 |
| B3 | Docker マルチステージビルド | `backend/Dockerfile` | Alpine + CA 証明書、最小イメージ |
| B4 | 環境変数による秘密管理 | `.env` | API キー等がコードに含まれない |

### 未対応

#### B5. TLS/HTTPS（SSL 終端）

**対象**: nginx 設定ファイル（新規作成）

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location / {
        root /var/www/frontend/dist;
        try_files $uri /index.html;
    }
}
```

Let's Encrypt + certbot で証明書を取得・自動更新。
[Qualys SSL Labs](https://www.ssllabs.com/ssltest/) で TLS 設定を検証。

#### B6. 本番用 Webhook エンドポイント登録

**対象**: Stripe Dashboard → Developers → Webhooks

1. 本番の Webhook URL を登録: `https://yourdomain.com/api/webhooks/stripe`
2. 監視するイベントを選択:
   - `payment_intent.succeeded`
   - `customer.subscription.created`
   - `customer.subscription.deleted`
3. 生成された `whsec_...` を `.env` の `STRIPE_WEBHOOK_SECRET` に設定
4. テスト環境の Webhook とは別に管理

#### B7. API キー切替・ローテーション

**対象**: Stripe Dashboard → Developers → API Keys

1. **本番キー取得**: `sk_live_...` / `pk_live_...` を取得
2. `.env` の `STRIPE_SECRET_KEY` を `sk_live_...` に変更
3. テスト環境で使っていたキーが他の場所に保存されていないか確認
4. Stripe の API 更新メーリングリストに登録

#### B8. Stripe アカウント 2FA 有効化

**対象**: Stripe Dashboard → Settings → Security

- アカウントオーナーの 2FA を有効化
- チームメンバーにも 2FA を要求
- ログイン情報は共有せず、適切なロールを割り当て

#### B9. DB 接続 SSL 化

**対象**: `.env` の `DATABASE_URL`

```
# 開発（現在）
DATABASE_URL=postgres://givers:givers@localhost:5432/givers?sslmode=disable

# 本番（変更後）
DATABASE_URL=postgres://givers:<strong_password>@db:5432/givers?sslmode=require
```

- パスワードを `openssl rand -base64 32` 等で強固なものに変更
- `sslmode=require` に変更

#### B10. PCI コンプライアンス証明

**対象**: Stripe Dashboard → Settings → Compliance

- Stripe Checkout 利用のため SAQ-A（最も簡易な自己評価）が該当
- 年次で Stripe ダッシュボードから確認・提出

#### B11. Statement Descriptor 設定（ダッシュボード側）

**対象**: Stripe Dashboard → Settings → Public details

- ビジネス名・明細表示名を設定（例: `GIVERS`）
- 5-22文字、特殊文字 `< > \ ' "` 不可
- 顧客の銀行明細に表示される名称

#### B12. 不正利用・チャージバック対策体制

**対象**: 運用プロセス

- Stripe Dashboard で支払いを定期的にレビュー
- 不正取引の報告フローを整備
- チャージバック（異議申し立て）への証拠提出体制を構築
- Stripe Radar の設定確認

---

## 優先度マトリクス

| 優先度 | 項目 | カテゴリ | 理由 |
|--------|------|----------|------|
| **P0 — 必須** | B5. TLS/HTTPS | インフラ | Stripe 必須要件 |
| **P0 — 必須** | B7. API キー切替 | インフラ | 本番稼働の前提条件 |
| **P0 — 必須** | B6. 本番 Webhook 登録 | インフラ | 決済処理に必須 |
| **P0 — 必須** | B8. 2FA 有効化 | インフラ | Stripe 推奨（アカウント保護） |
| **P1 — 高** | A9. レートリミット | アプリ | 不正利用・API 課金リスク |
| **P1 — 高** | A8. セキュリティヘッダー | アプリ | クリックジャッキング・XSS 防止 |
| **P1 — 高** | A11. ログ機密情報除去 | アプリ | PII 漏洩リスク |
| **P1 — 高** | B9. DB 接続 SSL | インフラ | 通信経路の保護 |
| **P2 — 中** | A10. Statement Descriptor | アプリ | 顧客 UX（明細の可読性） |
| **P2 — 中** | A12. IdleTimeout | アプリ | サーバーリソース保護 |
| **P2 — 中** | A13. Stripe エラー型区別 | アプリ | エラー体験の改善 |
| **P2 — 中** | B10. PCI 証明 | インフラ | 年次コンプライアンス |
| **P2 — 中** | B11. Statement Descriptor (Dashboard) | インフラ | 顧客 UX |
| **P3 — 低** | B12. チャージバック対策 | インフラ | 運用開始後に整備 |
