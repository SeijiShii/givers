-- =============================================================================
-- GIVErS 集約マイグレーション (001〜021 の最終スキーマ)
-- 新規環境のセットアップ時に使用。全テーブルを一括作成する。
-- =============================================================================

-- ---------------------------------------------------------------------------
-- users
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email       VARCHAR(255) NOT NULL UNIQUE,
    google_id   VARCHAR(255) UNIQUE,
    github_id   VARCHAR(255) UNIQUE,
    name        VARCHAR(255) NOT NULL DEFAULT '',
    suspended_at TIMESTAMP WITH TIME ZONE,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);

-- ---------------------------------------------------------------------------
-- projects
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS projects (
    id                 VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    owner_id           VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name               VARCHAR(255) NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    overview           TEXT NOT NULL DEFAULT '',
    share_message      TEXT NOT NULL DEFAULT '',
    deadline           DATE,
    status             VARCHAR(50) NOT NULL DEFAULT 'active',
    owner_want_monthly INT,
    monthly_target     INT NOT NULL DEFAULT 0,
    stripe_account_id  VARCHAR(255),
    cost_items         JSONB,
    image_url          TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_projects_status   ON projects(status);

-- ---------------------------------------------------------------------------
-- project_alerts
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS project_alerts (
    id                  VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id          VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    warning_threshold   INT NOT NULL DEFAULT 50,
    critical_threshold  INT NOT NULL DEFAULT 20,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(project_id)
);

CREATE INDEX IF NOT EXISTS idx_project_alerts_project_id ON project_alerts(project_id);

-- ---------------------------------------------------------------------------
-- contact_messages
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS contact_messages (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email      VARCHAR(255) NOT NULL,
    name       VARCHAR(255),
    message    TEXT NOT NULL,
    status     VARCHAR(50) NOT NULL DEFAULT 'unread',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_contact_messages_status     ON contact_messages(status);
CREATE INDEX IF NOT EXISTS idx_contact_messages_created_at ON contact_messages(created_at DESC);

-- ---------------------------------------------------------------------------
-- watches
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS watches (
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX IF NOT EXISTS idx_watches_user_id    ON watches(user_id);
CREATE INDEX IF NOT EXISTS idx_watches_project_id ON watches(project_id);

-- ---------------------------------------------------------------------------
-- project_updates
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS project_updates (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    author_id  VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      VARCHAR(500),
    body       TEXT NOT NULL,
    visible    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_project_updates_project_id ON project_updates(project_id);
CREATE INDEX IF NOT EXISTS idx_project_updates_created_at ON project_updates(created_at DESC);

-- ---------------------------------------------------------------------------
-- platform_health (singleton)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS platform_health (
    id                 INT PRIMARY KEY DEFAULT 1,
    monthly_cost       INTEGER NOT NULL DEFAULT 0,
    current_monthly    INTEGER NOT NULL DEFAULT 0,
    warning_threshold  INTEGER NOT NULL DEFAULT 60,
    critical_threshold INTEGER NOT NULL DEFAULT 30,
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (id = 1)
);

INSERT INTO platform_health (id) VALUES (1) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- donations
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS donations (
    id                      VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id              VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    donor_type              VARCHAR(10) NOT NULL CHECK (donor_type IN ('token', 'user')),
    donor_id                VARCHAR(36) NOT NULL,
    amount                  INTEGER NOT NULL CHECK (amount > 0),
    currency                VARCHAR(3) NOT NULL DEFAULT 'jpy',
    message                 TEXT,
    is_recurring            BOOLEAN NOT NULL DEFAULT false,
    stripe_payment_id       TEXT,
    stripe_subscription_id  TEXT,
    paused                  BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX  IF NOT EXISTS idx_donations_project_id                    ON donations(project_id);
CREATE INDEX  IF NOT EXISTS idx_donations_donor                         ON donations(donor_type, donor_id);
CREATE INDEX  IF NOT EXISTS idx_donations_stripe_subscription_id        ON donations(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_stripe_payment_id_unique       ON donations(stripe_payment_id) WHERE stripe_payment_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_stripe_subscription_id_unique  ON donations(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- user_cost_presets
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_cost_presets (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      VARCHAR(100) NOT NULL,
    unit_type  VARCHAR(20) NOT NULL CHECK (unit_type IN ('monthly', 'daily_x_days')),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_cost_presets_user_id ON user_cost_presets(user_id);

-- ---------------------------------------------------------------------------
-- activities
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS activities (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    type       VARCHAR(20) NOT NULL CHECK (type IN ('donation', 'project_created', 'project_updated', 'milestone')),
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id   VARCHAR(36),
    amount     INTEGER,
    rate       INTEGER,
    message    TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activities_project_id ON activities(project_id);
CREATE INDEX IF NOT EXISTS idx_activities_created_at ON activities(created_at DESC);

-- ---------------------------------------------------------------------------
-- sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sessions (
    token      VARCHAR(64) PRIMARY KEY,
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- ---------------------------------------------------------------------------
-- seed: 開発用ダミーユーザー
-- ---------------------------------------------------------------------------
INSERT INTO users (id, email, name, created_at, updated_at)
VALUES ('dev-user-id', 'dev@givers.local', 'Dev User', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
