# ConoHa サーバーでの GIVErS 運用設定

ConoHa VPS 上で GIVErS を本番運用するときの設定の考え方と手順。**最終更新**: 2025-02

---

## 1. 前提

- **ConoHa VPS** を 1 台用意（例: 1GB メモリ〜。DB・アプリ・リバースプロキシを同居させる想定）。
- ドメインは ConoHa のドメインサービスまたは他社で取得し、VPS のグローバル IP を A レコードで向ける。
- 本番では **Docker Compose** で backend / db を起動し、フロントは **ビルド済み静的ファイル** を nginx 等で配信する想定。

---

## 2. サーバー初期設定

### 2.1 OS・ユーザー

- **OS**: Ubuntu 22.04 LTS を推奨（ConoHa のテンプレートから選択）。
- **root ログイン無効化** し、sudo 可能な一般ユーザー（例: `deploy`）を作成。
- **SSH 鍵認証** にし、パスワードログインは無効化。

### 2.2 ファイアウォール（ufw）

```bash
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP（リダイレクト用）
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

### 2.3 Docker / Docker Compose

- Docker 公式手順で Docker Engine と Docker Compose（v2）をインストール。
- 運用ユーザーを `docker` グループに追加し、sudo なしで `docker compose` を実行可能にする。

---

## 3. アプリケーションの配置

### 3.1 リポジトリの取得

- 本番用には **git clone** または CI でビルドした成果物を配置。
- 配置例: `/home/deploy/givers`（任意）。

### 3.2 本番用 Docker Compose の考え方

- **開発用** `docker-compose.yml` は frontend が `npm run dev` のため、本番では使わない。
- 本番用は次のように分ける想定:
  - **db**: 現状の PostgreSQL イメージのまま。`POSTGRES_PASSWORD` は強力な値に変更。
  - **backend**: 現状の Dockerfile のまま。環境変数のみ本番用に変更。
  - **frontend**: 本番では **Astro をビルド**（`npm run build`）し、生成された静的ファイルを **nginx で配信**（下記 4 節）。  
    または、本番用に「ビルド済み静的ファイルを nginx イメージで配信する」Docker サービスを 1 つ追加する。

### 3.3 本番用環境変数（backend）

| 変数 | 説明 | 例 |
|------|------|-----|
| `DATABASE_URL` | PostgreSQL 接続文字列 | `postgres://givers:強力なパスワード@db:5432/givers?sslmode=disable` |
| `FRONTEND_URL` | フロントの公開 URL（CORS・OAuth リダイレクト用） | `https://your-domain.example.com` |
| `BACKEND_URL` | バックエンドの公開 URL（OAuth コールバック等） | `https://your-domain.example.com` |
| `SESSION_SECRET` | セッション署名用（32 バイト以上・推測困難な文字列） | 環境変数やシークレット管理で設定 |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth（使用する場合） | ConoHa 上では .env や ConoHa のメモに保存しないこと。サーバー内の .env ファイルに記載し、権限を制限。 |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | GitHub OAuth（使用する場合） | 同上 |
| `AUTH_REQUIRED` | 認証を有効にするか | `true` |

- 秘密情報は **.env ファイル** に書き、`chmod 600` で所有者以外読めないようにする。ConoHa の「メモ」に貼らない。

### 3.4 フロントの本番ビルド

- フロントは **API の URL** を本番向けに合わせる必要がある。
- `frontend/.env.production` またはビルド時に `PUBLIC_API_URL=https://your-domain.example.com` を渡す（Astro の `PUBLIC_*` の扱いに合わせる）。
- サーバー上で `cd frontend && npm ci && npm run build` を実行し、`frontend/dist` を nginx のドキュメントルートに指定する（または nginx 用 Docker イメージにコピーする）。

---

## 4. リバースプロキシと SSL

- インターネットからは **80/443** のみ開放し、80 は 443 にリダイレクト。
- **443** で nginx（または Caddy）を動かし、以下を実施:
  - **SSL 終端**: Let's Encrypt（certbot）で証明書を取得・更新。
  - **/**: フロント（静的ファイル）を配信。
  - **/api/** などバックエンド用パス: `http://localhost:8080` などにプロキシ。

### 4.1 nginx 設定例（要約）

- `server_name` にドメインを指定。
- `root` にフロントの `dist` を指定。`location / { try_files $uri $uri/ /index.html; }` で SPA/SSG ルーティングに対応。
- `location /api/ { proxy_pass http://127.0.0.1:8080; proxy_http_version 1.1; proxy_set_header Host $host; proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for; proxy_set_header X-Forwarded-Proto $scheme; }`
- SSL は `listen 443 ssl; ssl_certificate /etc/letsencrypt/live/...; ssl_certificate_key ...;` で指定。

### 4.2 Let's Encrypt（certbot）

- `certbot --nginx -d your-domain.example.com` で証明書取得と nginx 設定の自動編集。
- 自動更新: `certbot renew` を cron で定期実行（例: 毎日）。

---

## 5. メール送信（マジックリンク用）

- マジックリンク方式では **サービス側からメールを送る必要**がある（`docs/mock-implementation-status.md` 3.7 参照）。
- **ConoHa VPS 単体**にはメール送信機能は含まれないため、次のいずれかを利用する想定:
  - **SendGrid / AWS SES / Mailgun 等**: SMTP または API で送信。環境変数で API キー・SMTP 情報を渡す。
  - **ConoHa のメールサービス**（ConoHa WING 等）を契約している場合: その SMTP を利用可能。設定は各サービスの案内に従う。
- バックエンドで「マジックリンク用トークン生成 → メール本文生成 → 送信」を行うレイヤーを実装し、送信先 SMTP/API は環境変数で切り替え可能にすると運用しやすい。

---

## 6. 起動と自動起動

- アプリ用コンテナは **Docker Compose** で起動: `docker compose -f docker-compose.prod.yml up -d`（本番用ファイル名は任意）。
- サーバー再起動時にコンテナも起動するようにする:
  - **systemd**: `docker compose ... up -d` を実行する unit を書き、`multi-user.target` に依存させる。
  - または Docker の「restart: unless-stopped」と、Docker サービス自体の自動起動に頼る方法もある。

---

## 7. 運用上の注意

- **DB のバックアップ**: `pg_dump` を cron で定期実行し、別ストレージ（ConoHa オブジェクトストレージや外部）に退避。
- **ログ**: コンテナの stdout/stderr は `docker compose logs` で確認。長期保存する場合はファイルに落とすかログ収集サービスを検討。
- **監視**: ConoHa の監視機能や、外部の死活監視（HTTPS の /api/health 等）を利用するとよい。
- **セキュリティ**: OS と Docker イメージの定期的な更新、秘密情報の取り扱い（.env の権限、漏洩防止）を徹底する。

---

## 8. 関連ドキュメント

- `docs/implementation-plan.md` - 環境変数一覧、Phase 6 本番用設定
- `docs/phase2-plan.md` - 認証・セッション（Cookie: Secure 本番）
- `docs/mock-implementation-status.md` - Email マジックリンク方針
