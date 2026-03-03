# DEV環境サーバー構築 — 実装計画書

## 概要

本番環境 (`givers.work`) と同じ構成で、開発・ステージング用サーバーを `dev.givers.work` に構築する。
DEV環境には「開発用」であることを示すバナーを表示し、本番GIVErSへのリンクを併記する。
本番DBダンプのローカルダウンロードスクリプトも用意する。

## 現状の把握

| 項目 | 現状 |
|------|------|
| 本番サーバー | ConoHa VPS v3 (Ubuntu 24, 1GB/2Core), IP: `163.44.111.156` |
| SSH | `givers-conoha-root` |
| デプロイ先 | `/opt/givers` |
| 構成 | Docker Compose (nginx + certbot + frontend + backend + db) |
| SSL | Let's Encrypt (certbot, ACME HTTP-01) |
| 既存バナー | `PUBLIC_TEST_MODE=true` で赤色「テスト運用中」バナーあり |

---

## 全7フェーズ・20ステップ

### Phase 1: インフラ準備 (手動作業)

| # | 内容 | リスク |
|---|------|--------|
| 1 | ConoHa VPSインスタンス作成 (Ubuntu 24, 最小プラン) | 低 |
| 2 | SSH設定 (`~/.ssh/config` に `givers-conoha-dev` 追加) | 低 |
| 3 | DEVサーバー初期設定 (Docker, UFW) | 低 |
| 4 | DNS設定 (`dev.givers.work` → DEVサーバーIP の Aレコード) | 低 |

#### Step 1: ConoHa VPSインスタンス作成
- ConoHa コントロールパネルでサーバーを追加
- SSH鍵を設定 (新規鍵 `~/.ssh/givers_conoha_dev_key` を生成するか、既存鍵を流用)
- IPアドレスを控える
- セキュリティグループ: `default`, `IPv4v6-SSH`, `IPv4v6-Web` を割り当て

#### Step 2: SSH設定
`~/.ssh/config` に追加:
```
Host givers-conoha-dev
  HostName <DEV_IP_ADDRESS>
  User root
  IdentityFile ~/.ssh/givers_conoha_dev_key
  IdentitiesOnly yes
```

#### Step 3: DEVサーバー初期設定
```bash
ssh givers-conoha-dev
curl -fsSL https://get.docker.com | sh
systemctl enable docker
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable
```

#### Step 4: DNS設定
ConoHa DNSパネルで Aレコードを追加:

| タイプ | 名称 | TTL  | 値 |
|--------|------|------|----|
| A      | dev  | 3600 | <DEV_IP_ADDRESS> |

---

### Phase 2: 環境変数テンプレート作成

| # | ファイル | 主な変更点 |
|---|---------|-----------|
| 5 | `.env.dev.example` | URL → `dev.givers.work`, Stripeテストキー, `LOG_LEVEL=debug` |
| 6 | `frontend/.env.dev.example` | `PUBLIC_DEV_MODE=true` (新規), `PUBLIC_GA_MEASUREMENT_ID=` (空) |
| 7 | `frontend/.env.dev` | 実ファイル作成 (.gitignore対象) |
| 8 | `.gitignore` 更新 | `.env.dev` を追加 |

#### Step 5: `.env.dev.example` 作成
`.env.prod.example` をベースに以下を変更:
- `FRONTEND_URL=https://dev.givers.work`
- `BACKEND_URL=https://dev.givers.work`
- `PUBLIC_API_URL=https://dev.givers.work`
- `PUBLIC_OFFICIAL_URL=https://givers.work`
- `STRIPE_SECRET_KEY=sk_test_...`
- `DOMAIN=dev.givers.work`
- `LOG_LEVEL=debug`

#### Step 6: `frontend/.env.dev.example` 作成
```env
PUBLIC_MOCK_MODE=false
PUBLIC_API_URL=https://dev.givers.work
PUBLIC_PLATFORM_PROJECT_ID=
PUBLIC_OFFICIAL_URL=https://givers.work
PUBLIC_GITHUB_REPO=https://github.com/SeijiShii/givers
PUBLIC_TEST_MODE=true
PUBLIC_DEV_MODE=true
PUBLIC_GA_MEASUREMENT_ID=
```

#### Step 7-8: `.env.dev` 実ファイル作成 + `.gitignore` 更新
- `frontend/.env.dev` を `.env.dev.example` からコピーして実際の値を設定
- `.gitignore` に `.env.dev` を追加

---

### Phase 3: Docker / Nginx 設定

| # | ファイル | 内容 |
|---|---------|------|
| 9 | `nginx/conf.d/default.conf.dev` | `server_name dev.givers.work` + SSL証明書パス変更 |
| 10 | `nginx/conf.d/default.conf.dev.initial` | SSL初回取得用 HTTP-only 設定 |
| 11 | `docker-compose.dev.yml` | DEV用Nginx設定マウント, `Dockerfile.dev-server` 使用 |
| 12 | `frontend/Dockerfile.dev-server` | `.env.dev` → `.env` にリネームしてビルド |

#### Step 9: DEV用Nginx設定
`default.conf` をベースに以下を変更:
- `server_name dev.givers.work;` (port 80 / 443 の2箇所)
- `ssl_certificate /etc/letsencrypt/live/dev.givers.work/fullchain.pem;`
- `ssl_certificate_key /etc/letsencrypt/live/dev.givers.work/privkey.pem;`

#### Step 10: DEV用Nginx初期設定
`default.conf.initial` をベースに `server_name dev.givers.work;` に変更。

#### Step 11: `docker-compose.dev.yml`
`docker-compose.prod.yml` をベースに以下を変更:
```yaml
nginx:
  volumes:
    - ./nginx/conf.d/default.conf.dev:/etc/nginx/conf.d/default.conf:ro
    - certbot_conf:/etc/letsencrypt:ro
    - certbot_www:/var/www/certbot:ro

frontend:
  build:
    context: ./frontend
    dockerfile: Dockerfile.dev-server
```

#### Step 12: `frontend/Dockerfile.dev-server`
`Dockerfile.prod` をベースに `.env.dev` → `.env` にリネームしてビルド:
```dockerfile
RUN mv .env .env.orig 2>/dev/null || true && \
    mv .env.dev .env 2>/dev/null || true
```

---

### Phase 4: DEVバナー表示 (フロントエンド)

| # | ファイル | 内容 |
|---|---------|------|
| 13 | `src/i18n/ja.json`, `en.json` | `devBanner` キー追加 (本番リンク付き) |
| 14 | `src/layouts/BaseLayout.astro` | `PUBLIC_DEV_MODE` による青色DEVバナー表示ロジック |
| 15 | `src/styles/global.css` | `.dev-banner` CSS (青色, z-index:10000, リンクは金色) |

#### Step 13: i18nキー追加
- ja.json: `"devBanner": "これは開発環境です。本番サイトは <a href=\"https://givers.work\">givers.work</a> をご覧ください。"`
- en.json: `"devBanner": "This is a development environment. Visit <a href=\"https://givers.work\">givers.work</a> for the production site."`

#### Step 14: BaseLayout.astro にDEVバナーロジック追加
```typescript
const devMode = import.meta.env.PUBLIC_DEV_MODE === 'true';
const officialUrl = import.meta.env.PUBLIC_OFFICIAL_URL || 'https://givers.work';
```

テンプレート部分 (既存testBannerの直後):
```astro
{devMode && (
  <div class="dev-banner">
    <Fragment set:html={t('devBanner')} />
  </div>
)}
```

#### Step 15: DEVバナー用CSS
```css
.dev-banner {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  background: rgba(30, 100, 200, 0.9);
  color: #fff;
  text-align: center;
  padding: 0.5rem 1rem;
  font-size: 0.9rem;
  font-weight: bold;
  z-index: 10000;
}
.dev-banner a {
  color: #ffd700;
  text-decoration: underline;
}
body:has(.dev-banner) {
  padding-top: 2.5rem;
}
body:has(.test-banner):has(.dev-banner) {
  padding-top: 5rem;
}
body:has(.dev-banner) .test-banner {
  top: 2.5rem;
}
```

---

### Phase 5: デプロイスクリプト

| # | ファイル | 内容 |
|---|---------|------|
| 16 | `scripts/deploy-dev.sh` | `deploy.sh` ベース, `REMOTE_HOST=givers-conoha-dev`, compose → `docker-compose.dev.yml` |
| 17 | `scripts/init-ssl-dev.sh` | `dev.givers.work` のSSL証明書取得 |

#### Step 16: `scripts/deploy-dev.sh`
`deploy.sh` をベースに以下を変更:
- `REMOTE_HOST="givers-conoha-dev"`
- `REMOTE_DIR="/opt/givers-dev"`
- `LOCAL_TMP="/tmp/givers-deploy-dev"`
- compose ファイル: `docker-compose.dev.yml`
- DEV用フロントエンド環境変数もコピー: `cp frontend/.env.dev "$LOCAL_TMP/frontend/"`
- リモート実行コマンド: `docker compose -f docker-compose.dev.yml ...`

#### Step 17: `scripts/init-ssl-dev.sh`
`init-ssl.sh` をベースに以下を変更:
- compose ファイル: `docker-compose.dev.yml`
- certbot `-d dev.givers.work` のみ (`www` 不要)
- nginx設定切り替え: `default.conf.dev.initial` ↔ `default.conf.dev`

---

### Phase 6: DBダンプスクリプト

| # | ファイル | 内容 |
|---|---------|------|
| 18 | `scripts/dump-prod-db.sh` | SSH経由で `pg_dump` → ローカルにストリーム |

```bash
#!/bin/bash
set -e

REMOTE_HOST="givers-conoha-root"
REMOTE_DIR="/opt/givers"
COMPOSE_FILE="docker-compose.prod.yml"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DUMP_FILENAME="givers_prod_${TIMESTAMP}.sql"
LOCAL_OUTPUT_DIR="${1:-.}"
LOCAL_DUMP_PATH="${LOCAL_OUTPUT_DIR}/${DUMP_FILENAME}"

# リモートでダンプ → ローカルにストリーム (リモートにファイルを残さない)
ssh "$REMOTE_HOST" "cd $REMOTE_DIR && \
  docker compose -f $COMPOSE_FILE exec -T db \
  pg_dump -U givers -d givers --no-owner --no-privileges \
  --exclude-table-data=sessions" \
  > "$LOCAL_DUMP_PATH"
```

設計ポイント:
- `--no-owner --no-privileges` でDEV/ローカル復元を容易に
- `--exclude-table-data=sessions` でセッション情報を除外
- タイムスタンプ付きファイル名で世代管理

---

### Phase 7: ドキュメント

| # | ファイル | 内容 |
|---|---------|------|
| 19 | `docs/setup/dev-server-deployment.md` | DEVサーバー構築・運用手順書 |
| 20 | `TODO.md` 更新 | 完了タスクのチェック |

---

## 外部サービスのDEV設定 (要手動)

| サービス | 必要な作業 |
|---------|-----------|
| Google OAuth | Cloud Console でコールバックURL `https://dev.givers.work/api/auth/google/callback` 追加 |
| GitHub OAuth | **新規 OAuth App 作成** (1 App = 1 callback URL の制約あり) |
| Discord OAuth | Redirects に `https://dev.givers.work/api/auth/discord/callback` 追加 |
| Stripe Webhook | DEV用エンドポイント `https://dev.givers.work/api/webhooks/stripe` を別途登録 |

---

## リスクと対策

| リスク | 影響度 | 対策 |
|--------|--------|------|
| 間違って本番にデプロイ | **高** | `REMOTE_HOST` をハードコード、スクリプト冒頭で確認表示 |
| 本番DBダンプに個人情報 | **高** | sessions除外, サニタイズオプション検討 |
| Stripe Webhook競合 | 中 | DEV専用 `whsec_*` を別途取得 |
| Let's Encrypt レート制限 | 低 | サブドメインは独立カウント |

---

## 推奨実装順序

```
Phase 2 (環境変数) → Phase 4 (DEVバナー) → Phase 3 (Docker/Nginx)
    → Phase 5 (デプロイスクリプト) → Phase 6 (DBダンプ)
        → Phase 1 (ConoHaインスタンス作成・手動) → Phase 7 (ドキュメント)
```

コード系を先に全て揃えてから、手動インフラ作業をすると
`deploy-dev.sh --init` → `.env` 設定 → `init-ssl-dev.sh` で一気に完了できる。

---

## 成功基準

- [ ] `dev.givers.work` に HTTPS でアクセスできる
- [ ] 青色DEVバナー + 本番リンクが表示される
- [ ] 本番 `givers.work` にはDEVバナーが表示されない
- [ ] `deploy-dev.sh` でデプロイできる
- [ ] `dump-prod-db.sh` で本番DBダンプがDLできる
- [ ] OAuth / Stripe テスト決済が動作する
- [ ] DEV環境構築手順が `docs/setup/dev-server-deployment.md` に文書化されている
