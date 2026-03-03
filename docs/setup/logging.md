# GIVErS ログ管理ガイド

**最終更新**: 2026-03-03

---

## 1. アーキテクチャ概要

```
サービス                ログ出力先              捕捉方法
──────────────────────────────────────────────────────────────
Backend (Go slog)   → stdout (JSON)         → Docker json-file ドライバ
Frontend (Astro)    → stdout                → Docker json-file ドライバ
PostgreSQL          → stdout                → Docker json-file ドライバ
Nginx access        → /var/log/nginx/ + stdout → ホストマウント + Docker
Nginx error         → /var/log/nginx/ + stderr → ホストマウント + Docker
```

- **Backend/Frontend/DB**: アプリケーションは stdout に出力し、Docker の `json-file` ドライバが捕捉・ローテーション
- **Nginx**: ホストの `./logs/nginx/` にマウントされたファイルに直接書き込み + `docker logs` でも確認可能

---

## 2. ログ保存期間のポリシー

GIVErS は寄付（決済）を扱うプラットフォームのため、一般的な Web アプリケーションよりも長い保存期間を設定している。

### 保存期間の根拠

| 観点 | 要件 | 根拠 |
|------|------|------|
| **Stripe チャージバック** | 120日（約4ヶ月） | 不審請求の申し立て期限。調査にアクセスログ（IP・リクエスト）が必要 |
| **セキュリティ調査** | 90日〜1年 | 不正アクセスの発見が遅れるケースに対応。IP・認証ログが証拠になる |
| **デバッグ** | 30〜90日 | 直近の問題調査には十分 |
| **個人情報保護法** | 必要最小限 | ログに IP・メール等が含まれるため、不要な長期保存は避ける |
| **税務（日本）** | 7年 | 法人税法の帳簿保存義務。ただし **取引記録は DB に永続保存済み** なのでログは補助的 |

### ログと DB の役割分担

- **DB（donations テーブル等）**: 寄付金額・日時・寄付者情報を **永続保存**。税務・法的要件はこちらでカバー
- **Stripe**: 全決済記録を Stripe 側でも保持
- **ログ**: デバッグ・セキュリティ調査・チャージバック対応の **補助情報**

### 採用した保存期間

| ログ種別 | 保存期間 | 理由 |
|---------|---------|------|
| **Nginx access / error** | **90日**（logrotate） | チャージバック（120日以内、大半は60日以内）+ セキュリティ調査をカバー |
| **Backend** | **サイズ制限 250MB** (50MB × 5) | JSON ログは中規模トラフィックで約90日分相当 |
| **Frontend / DB** | **サイズ制限 150MB** (50MB × 3) | 補助的なログ。デバッグ用途 |

### ディスク使用量

ConoHa VPS 100GB SSD での見積もり:
- Nginx 90日分（圧縮後）: 約 500MB〜1GB
- Docker ログ上限: ~800MB
- **合計 ~2GB 以下** — 十分余裕あり

---

## 3. ログの保存場所と設定

### 本番サーバー (`/opt/givers`)

| サービス | 保存先 | フォーマット | ローテーション |
|---------|--------|------------|--------------|
| Backend | Docker 内部 (`/var/lib/docker/containers/...`) | JSON (slog) | 50MB × 5 = 250MB 上限 |
| Frontend | Docker 内部 | テキスト | 50MB × 3 = 150MB 上限 |
| DB | Docker 内部 | PostgreSQL | 50MB × 3 = 150MB 上限 |
| Nginx access | `/opt/givers/logs/nginx/access.log` | combined | logrotate 日次 90日保持 |
| Nginx error | `/opt/givers/logs/nginx/error.log` | nginx error | logrotate 日次 90日保持 |

### Docker ログドライバ設定 (`docker-compose.prod.yml`)

```yaml
logging:
  driver: json-file
  options:
    max-size: "50m"
    max-file: "5"   # backend/nginx
    # max-file: "3"  # frontend/db
```

### Nginx logrotate 設定 (`/etc/logrotate.d/givers-nginx`)

```
/opt/givers/logs/nginx/access.log
/opt/givers/logs/nginx/error.log {
    daily
    missingok
    rotate 90
    compress
    delaycompress
    notifempty
    sharedscripts
    postrotate
        docker compose -f /opt/givers/docker-compose.prod.yml exec -T nginx nginx -s reopen
    endscript
}
```

---

## 4. `scripts/logs.sh` — ログ閲覧ユーティリティ

ローカルまたはリモートサーバーのログを統一的に閲覧するスクリプト。

### オプション一覧

| オプション | 短縮 | 説明 | デフォルト |
|-----------|------|------|-----------|
| `--remote` | `-r` | リモートサーバー (SSH) のログを表示 | ローカル Docker |
| `--tail <N>` | `-n <N>` | 最新 N 行を表示 | 50 |
| `--no-follow` | | リアルタイム追従なし | follow 有効 |
| `--since <datetime>` | | 指定日時以降のログ | なし |
| `--until <datetime>` | | 指定日時以前のログ | なし |
| `-o <path>` | | ログをファイルに保存（自動 no-follow） | なし |
| `-h` | `--help` | ヘルプを表示 | |

### サービス指定

| 引数 | 対象 | 取得方法 |
|------|------|---------|
| `backend` | Go API サーバー | `docker compose logs` |
| `frontend` | Astro SSR | `docker compose logs` |
| `db` | PostgreSQL | `docker compose logs` |
| `nginx` | Nginx アクセスログ | `tail` (ホストファイル) |
| `nginx error` | Nginx エラーログ | `tail` (ホストファイル) |
| (なし) | 全サービス | `docker compose logs` |

### 日時指定フォーマット

**Docker サービス** (backend / frontend / db):
- RFC3339: `2026-03-03T10:00:00`
- 相対指定: `1h`, `30m`, `24h`

**Nginx**:
- combined タイムスタンプ: `03/Mar/2026:10:00`

### 使用例

```bash
# 基本操作
./scripts/logs.sh                           # 全サービスを follow
./scripts/logs.sh backend                   # backend を follow
./scripts/logs.sh nginx                     # nginx access.log を follow
./scripts/logs.sh nginx error               # nginx error.log を follow
./scripts/logs.sh --tail 100 backend        # 最新100行
./scripts/logs.sh --no-follow backend       # 表示して終了

# リモートサーバー
./scripts/logs.sh --remote backend          # SSH 経由でリモートの backend ログ
./scripts/logs.sh --remote nginx            # SSH 経由でリモートの nginx ログ

# 日時指定
./scripts/logs.sh --remote --since "1h" backend                               # 直近1時間
./scripts/logs.sh --remote --since "2026-03-03T10:00:00" backend              # 指定時刻以降
./scripts/logs.sh --remote --since "2026-03-01" --until "2026-03-02" backend  # 期間指定
./scripts/logs.sh --remote --since "03/Mar/2026:10:00" nginx                  # nginx 日時フィルタ

# ファイル出力
./scripts/logs.sh --remote --since "1h" backend -o ./backend-1h.log
./scripts/logs.sh --remote --since "2026-03-01" nginx -o ./nginx-mar01.log
# => "Saved 1234 lines to ./backend-1h.log"
```

---

## 5. ログの内容 — 各サービスの出力例

### Backend (Go slog JSON)

リクエストログ (`RequestLogger` ミドルウェア):
```json
{"time":"2026-03-03T12:00:00Z","level":"INFO","source":{"file":"request_logger.go","line":25},"msg":"request","method":"POST","path":"/api/donations/checkout","status":200,"duration_ms":45,"remote_addr":"203.0.113.5:12345"}
```

エラーログ（スタックトレース付き）:
```json
{"time":"2026-03-03T12:00:01Z","level":"ERROR","source":{"file":"stripe_handler.go","line":89},"msg":"stripe session create failed","error":"card_declined","project_id":"abc-123","stacktrace":"goroutine 42 [running]:\n..."}
```

認証ログ:
```json
{"time":"2026-03-03T12:00:02Z","level":"INFO","msg":"google oauth login success"}
{"time":"2026-03-03T12:00:03Z","level":"WARN","msg":"google callback: state verification failed","state_prefix":"abc12345"}
```

### Nginx アクセスログ (combined)

```
203.0.113.5 - - [03/Mar/2026:12:00:00 +0000] "GET /api/projects HTTP/1.1" 200 4523 "https://givers.work/" "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
203.0.113.5 - - [03/Mar/2026:12:00:01 +0000] "POST /api/donations/checkout HTTP/1.1" 200 128 "-" "Mozilla/5.0..."
```

### Nginx エラーログ

```
2026/03/03 12:00:00 [error] 29#29: *1234 upstream timed out (110: Connection timed out), client: 203.0.113.5, server: givers.work, request: "GET /api/projects HTTP/1.1", upstream: "http://backend:8080/api/projects"
```

### Frontend (Astro SSR)

```
Listening on http://0.0.0.0:4321
[SSR] GET /projects/abc-123 200 32ms
```

### DB (PostgreSQL)

```
2026-03-03 12:00:00.000 UTC [1] LOG:  database system is ready to accept connections
2026-03-03 12:00:05.000 UTC [29] ERROR:  duplicate key value violates unique constraint "donations_pkey"
2026-03-03 12:00:10.000 UTC [29] STATEMENT:  INSERT INTO donations (id, ...) VALUES ($1, ...)
```

---

## 6. ローテーション設定の詳細

### Docker json-file ドライバ

- `max-size: "50m"` — 1ファイルの最大サイズ
- `max-file: "5"` or `"3"` — 保持するファイル数
- ローテーション: Docker が自動で実施。最大サイズ到達時に新ファイルを作成し、古いファイルを削除
- `docker compose logs` は全ファイルから統合して表示

### Nginx logrotate

- **頻度**: 日次 (`daily`)
- **保持**: 90世代 (`rotate 90`)
- **圧縮**: gzip (`compress`)、直近1世代は非圧縮 (`delaycompress`)
- **通知**: ローテーション後に `nginx -s reopen` でログファイル再オープン
- **実行**: OS の cron が `/etc/logrotate.d/` を日次実行

### ディスク使用量の確認

```bash
# nginx ログ
ssh givers-conoha-root "du -sh /opt/givers/logs/"

# Docker ログ全体
ssh givers-conoha-root "docker system df -v"

# 特定コンテナのログサイズ
ssh givers-conoha-root "docker inspect --format='{{.LogPath}}' \$(docker compose -f /opt/givers/docker-compose.prod.yml ps -q backend) | xargs ls -lh"
```

---

## 7. トラブルシューティング

### ログが出力されない

```bash
# コンテナが起動しているか確認
ssh givers-conoha-root "cd /opt/givers && docker compose -f docker-compose.prod.yml ps"

# Docker ログドライバ設定を確認
ssh givers-conoha-root "docker inspect --format='{{.HostConfig.LogConfig}}' \$(docker compose -f /opt/givers/docker-compose.prod.yml ps -q backend)"

# nginx ログディレクトリの権限を確認
ssh givers-conoha-root "ls -la /opt/givers/logs/nginx/"
```

### ディスク容量不足

```bash
# ディスク使用状況
ssh givers-conoha-root "df -h"

# Docker の不要なイメージ・キャッシュを削除
ssh givers-conoha-root "docker system prune -f"

# 古い Docker ログを手動で切り詰め（緊急時のみ）
ssh givers-conoha-root "truncate -s 0 \$(docker inspect --format='{{.LogPath}}' \$(docker compose -f /opt/givers/docker-compose.prod.yml ps -q backend))"
```

### logrotate が動作しない

```bash
# logrotate 設定の構文チェック
ssh givers-conoha-root "logrotate -d /etc/logrotate.d/givers-nginx"

# 手動実行
ssh givers-conoha-root "logrotate -f /etc/logrotate.d/givers-nginx"

# logrotate のステータス確認
ssh givers-conoha-root "cat /var/lib/logrotate/status | grep givers"
```

### `logs.sh` が動作しない

```bash
# 実行権限を確認
chmod +x scripts/logs.sh

# SSH 接続を確認（--remote 使用時）
ssh givers-conoha-root "echo ok"

# Docker Compose が動作するか確認
ssh givers-conoha-root "cd /opt/givers && docker compose -f docker-compose.prod.yml ps"
```
