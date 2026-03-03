# ConoHa VPS — GIVErS 本番デプロイ手順

**サーバー**: ConoHa VPS v3 (Ubuntu 24, 1GB/2Core)
**ドメイン**: givers.work
**IP**: 163.44.111.156
**最終更新**: 2026-03-01

---

## 1. サーバー初期設定（済）

### SSH 接続

```bash
ssh givers-conoha-root
```

- SSH鍵認証: `~/.ssh/givers_conoha_root_key` (ed25519)
- パスワード認証: 無効化済み

### ConoHa セキュリティグループ

VPSに以下のセキュリティグループを割り当て:
- `default`
- `IPv4v6-SSH` — ポート22（設定済み）
- `IPv4v6-Web` — ポート80/443（**要追加**）

### UFW（OS ファイアウォール）

```bash
ufw allow 22/tcp    # SSH（設定済み）
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

### DNS

ConoHa DNS で設定済み:

| タイプ | 名称 | TTL  | 値               |
|--------|------|------|------------------|
| A      | @    | 3600 | 163.44.111.156   |
| A      | www  | 3600 | 163.44.111.156   |

---

## 2. サーバーに Docker をインストール

```bash
ssh givers-conoha-root
curl -fsSL https://get.docker.com | sh
systemctl enable docker
```

---

## 3. デプロイ方式

**ローカルビルド + SCP 方式** を採用。サーバーに Git は不要。

```
ローカル PC                          ConoHa VPS
─────────────                       ──────────────
1. git pull (対象ブランチ)
2. /tmp にパッケージ作成
   (node_modules/.git 除外)
3. rsync/scp で転送 ──────────────→ /opt/givers/
                                    4. docker compose build
                                    5. docker compose up -d
```

### デプロイスクリプト

```bash
# デフォルト (main ブランチ)
./scripts/deploy.sh

# ブランチ指定
./scripts/deploy.sh feature/xxx
```

スクリプトの処理:
1. ローカルで指定ブランチを pull
2. `/tmp/givers-deploy` に必要ファイルをコピー（node_modules, .git, テスト等を除外）
3. `rsync` でサーバーの `/opt/givers` に転送（`.env` は除外＝サーバー上の設定を保持）
4. サーバーで `docker compose up -d --build`
5. DBマイグレーション実行

オプション:
- `--init` — ファイル転送のみ（初回セットアップ用、ビルド・起動しない）

---

## 4. 初回セットアップ

### 4.1 初回デプロイ（ファイル転送のみ）

```bash
# ローカルから実行 — --init でファイル転送のみ（ビルド・起動しない）
./scripts/deploy.sh --init
```

### 4.2 環境変数の設定

サーバー上で `.env` を作成:

```bash
ssh givers-conoha-root
cd /opt/givers
cp .env.prod.example .env
nano .env    # 各値を本番用に編集
chmod 600 .env
```

**必須の変更項目**:
- `POSTGRES_PASSWORD` — `openssl rand -base64 24` で生成
- `DATABASE_URL` — 上記パスワードと一致させる
- `SESSION_SECRET` — `openssl rand -base64 32` で生成
- `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` — 本番キー
- `CERTBOT_EMAIL` — SSL 証明書の通知先メール
- OAuth の各 Client ID/Secret

### 4.3 SSL 証明書の初回取得

```bash
ssh givers-conoha-root
cd /opt/givers
./scripts/init-ssl.sh
```

処理内容:
1. HTTP-only の nginx 設定で起動
2. certbot で Let's Encrypt 証明書を取得（ACME HTTP-01 チャレンジ）
3. HTTPS 対応の nginx 設定に切り替え
4. 全サービスを起動

---

## 5. 通常のデプロイ更新

ローカルから実行するだけ:

```bash
./scripts/deploy.sh
```

---

## 6. サーバー上での操作

```bash
ssh givers-conoha-root
cd /opt/givers

# 起動
docker compose -f docker-compose.prod.yml up -d

# 停止
docker compose -f docker-compose.prod.yml down

# ログ確認
docker compose -f docker-compose.prod.yml logs -f

# 特定サービスのログ
docker compose -f docker-compose.prod.yml logs -f backend

# コンテナ状態
docker compose -f docker-compose.prod.yml ps
```

---

## 7. SSL 証明書の自動更新

```bash
ssh givers-conoha-root
crontab -e
```

```
0 3 * * * cd /opt/givers && docker compose -f docker-compose.prod.yml run --rm certbot renew && docker compose -f docker-compose.prod.yml exec nginx nginx -s reload
```

---

## 8. 構成図

```
Internet
  │
  ├─ :80  ─→ Nginx ──→ 301 redirect to HTTPS
  └─ :443 ─→ Nginx (SSL)
               ├─ /api/*     → backend:8080  (Go)
               ├─ /uploads/* → backend:8080  (Go)
               └─ /*         → frontend:4321 (Astro SSR)
                                    │
                                    └─ db:5432 (PostgreSQL 16)
```

---

## 9. ファイル構成

```
docker-compose.prod.yml    # 本番用 Compose
.env.prod.example          # 環境変数テンプレート
.env.prod                  # 環境変数（git管理外、サーバー上のみ）
frontend/Dockerfile.prod   # フロント本番ビルド
nginx/conf.d/
  default.conf             # HTTPS nginx 設定
  default.conf.initial     # 初回 SSL 取得用 HTTP 設定
scripts/
  deploy.sh                # ローカル→サーバー デプロイスクリプト
  init-ssl.sh              # SSL 初期設定スクリプト
  logs.sh                  # ログ閲覧ユーティリティ
logrotate/
  nginx                    # nginx 用 logrotate 設定
```

---

## 10. 運用チェックリスト

### 初回セットアップ
- [ ] ConoHa セキュリティグループ `IPv4v6-Web` を追加
- [ ] UFW で 80/443 を許可
- [ ] Docker インストール
- [ ] `./scripts/deploy.sh` で初回デプロイ
- [ ] サーバーで `.env` を作成・設定
- [ ] `./scripts/init-ssl.sh` で SSL 証明書取得
- [ ] `https://givers.work` でアクセス確認
- [ ] `https://givers.work/api/health` で API 確認
- [ ] certbot 自動更新 cron 設定
- [ ] Stripe Webhook URL を `https://givers.work/api/stripe/webhook` に更新
- [ ] OAuth リダイレクト URL を更新

### 通常デプロイ
- [ ] `./scripts/deploy.sh` を実行
- [ ] 動作確認

---

## 12. ログ管理

> 詳細: [logging.md](logging.md)

### ログの場所

| サービス | 保存先 | ローテーション |
|---------|--------|--------------|
| Backend / Frontend / DB | Docker json-file ドライバ | 50MB × 3〜5 ファイル |
| Nginx access/error | `/opt/givers/logs/nginx/` | logrotate 日次 90日保持 |

### ログの確認

```bash
# ローカルからリモートサーバーのログを確認
./scripts/logs.sh --remote backend          # backend を follow
./scripts/logs.sh --remote nginx            # nginx アクセスログ
./scripts/logs.sh --remote --since "1h" backend -o ./out.log  # ファイル出力

# サーバー上で直接確認
cd /opt/givers
docker compose -f docker-compose.prod.yml logs -f backend
tail -f logs/nginx/access.log
```

---

## 13. トラブルシューティング

### SSH 接続できない
- ConoHa セキュリティグループに `IPv4v6-SSH` が割り当てられているか確認
- VPS コンソール（ブラウザ）からログインして `ufw status` を確認

### HTTPS がつながらない
- ConoHa セキュリティグループに `IPv4v6-Web` が必要
- `ufw allow 80/tcp && ufw allow 443/tcp`
- `docker compose logs nginx` でエラー確認

### DNS が引けない
- ConoHa DNS で A レコードが設定されているか確認
- `dig givers.work A @a.conoha-dns.com` で直接確認

### デプロイが遅い
- 初回は Docker イメージのビルドに時間がかかる（Go コンパイル + npm ci）
- 2回目以降は Docker のビルドキャッシュが効く
