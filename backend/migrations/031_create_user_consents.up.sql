CREATE TABLE IF NOT EXISTS user_consents (
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    terms_version VARCHAR(20) NOT NULL,
    agreed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id)
);
