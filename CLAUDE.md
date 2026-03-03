# GIVErS — Zero-Fee Donation Platform

## Overview
クリエイターが無料で作品を提供し、ユーザーが自発的に支援できるゼロ手数料の寄付プラットフォーム。プロジェクトのコストと資金目標を透明に表示する。

## Tech Stack
- **Backend**: Go 1.24 + `net/http` + `jackc/pgx/v5`
- **Frontend**: Astro 5 + React 18 (islands architecture)
- **Database**: PostgreSQL 16 (Docker)
- **Auth**: OAuth2 (Google, GitHub, Discord) + Session-based
- **Payment**: Stripe Connect v2 (recurring/one-time)
- **i18n**: ja (default) / en
- **Deployment**: Docker Compose + Nginx + ConoHa VPS

## Architecture (Backend 3-Layer)
```
Handler (HTTP) → Service (Business Logic) → Repository (DB)
```

### Key Services
- `auth_service` — OAuth & session
- `project_service` — Project CRUD
- `donation_service` — 寄付の追跡・履歴
- `stripe_service` — Stripe Connect連携、Webhook
- `subscription_service` — 定期寄付 (pause/resume/cancel)
- `activity_service` — アクティビティフィード
- `milestone_service` — マイルストーン検出

### Key Models
User, Project, Donation, Subscription, ProjectUpdate, Activity, CostPreset, Watch, Contact

## Directory Structure
```
backend/
  cmd/server/          # API server entry
  cmd/migrate/         # Migration runner
  internal/handler/    # HTTP handlers
  internal/service/    # Business logic
  internal/repository/ # Data access
  internal/model/      # Domain entities
  migrations/          # SQL files (28 total)
frontend/
  src/pages/           # Astro routes (i18n prefixed)
  src/components/      # React & Astro components
  src/lib/             # API client, mock-api
  src/i18n/            # i18n strings
  e2e/                 # Playwright tests
docs/                  # Design, setup, legal docs
nginx/                 # Nginx config
scripts/               # deploy.sh, init-ssl.sh
```

## Development Commands
```bash
# DB起動
docker compose up -d db

# Backend
cd backend && go run ./cmd/migrate && go run ./cmd/server  # port 8080

# Frontend
cd frontend && npm run dev    # port 4321
npm run build                 # Production build
npm run test:e2e              # Playwright E2E tests

# Mock mode (backend不要)
# frontend/.env: PUBLIC_MOCK_MODE=true
```

## Environment Variables (重要なもの)
- `DATABASE_URL` — PostgreSQL接続文字列
- `SESSION_SECRET` — セッション署名用 (32+ bytes)
- `FRONTEND_URL` / `BACKEND_URL` — CORS & OAuth redirect
- `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET`
- `GOOGLE_CLIENT_ID/SECRET`, `GITHUB_CLIENT_ID/SECRET`, `DISCORD_CLIENT_ID/SECRET`
- `HOST_EMAILS` — 管理者メールリスト (カンマ区切り)
- `PUBLIC_MOCK_MODE` — フロントのみでUXテスト
- `PUBLIC_API_URL` — Backend APIエンドポイント

## Key Flows

### Donation Flow
1. Frontend → `POST /api/donations/checkout` (rate limit: 10/min/IP)
2. Backend → Stripe Session作成 → `client_secret` 返却
3. Frontend → Stripe Checkout redirect
4. Stripe → `POST /api/webhooks/stripe` (webhook)
5. Backend → donation記録 + milestone判定 + activity作成

### Auth Flow
OAuth → `/api/auth/{provider}/callback` → session作成 (cookie) → DB `sessions` table

## Common Dev Tasks

### 新しいAPIエンドポイント追加
1. `backend/internal/handler/{name}_handler.go` にhandler作成
2. `backend/cmd/server/main.go` でルート登録
3. 必要ならservice/repository層追加

### 新しいページ追加
1. `frontend/src/pages/` に `.astro` ファイル作成
2. i18n対応: `/en/` フォルダにもミラー作成
3. Reactコンポーネントは `client:load` で island化

### DB Migration追加
1. `backend/migrations/{number}_{name}.up.sql` / `.down.sql` 作成
2. `go run ./cmd/migrate` で適用

## Notes
- DB接続: `postgres://givers:givers@localhost:5432/givers` (dev)
- Admin: `HOST_EMAILS` に含まれるユーザーのみ `/admin/*` アクセス可
- Pagination: cursor-based (`next_cursor`)
- License: Apache-2.0