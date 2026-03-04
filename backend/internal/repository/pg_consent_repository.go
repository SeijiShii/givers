package repository

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgConsentRepository は ConsentRepository の PostgreSQL 実装
type PgConsentRepository struct {
	pool *pgxpool.Pool
}

// NewPgConsentRepository は PgConsentRepository を生成する
func NewPgConsentRepository(pool *pgxpool.Pool) *PgConsentRepository {
	return &PgConsentRepository{pool: pool}
}

func (r *PgConsentRepository) Upsert(ctx context.Context, c *model.UserConsent) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO user_consents (user_id, terms_version, agreed_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (user_id) DO UPDATE
		   SET terms_version = EXCLUDED.terms_version,
		       agreed_at = EXCLUDED.agreed_at
		 RETURNING agreed_at`,
		c.UserID, c.TermsVersion,
	).Scan(&c.AgreedAt)
}

func (r *PgConsentRepository) GetByUserID(ctx context.Context, userID string) (*model.UserConsent, error) {
	var c model.UserConsent
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, terms_version, agreed_at
		 FROM user_consents WHERE user_id = $1`, userID,
	).Scan(&c.UserID, &c.TermsVersion, &c.AgreedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}
