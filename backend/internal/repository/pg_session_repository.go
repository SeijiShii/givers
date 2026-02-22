package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPgSessionRepository returns a PostgreSQL-backed SessionRepository.
func NewPgSessionRepository(pool *pgxpool.Pool) SessionRepository {
	return &pgSessionRepository{pool: pool}
}

func (r *pgSessionRepository) Create(ctx context.Context, s *model.Session) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES ($1, $2, $3, $4)`,
		s.Token, s.UserID, s.CreatedAt, s.ExpiresAt)
	return err
}

func (r *pgSessionRepository) FindByToken(ctx context.Context, token string) (*model.Session, error) {
	s := &model.Session{}
	err := r.pool.QueryRow(ctx,
		`SELECT token, user_id, created_at, expires_at FROM sessions WHERE token = $1`,
		token).Scan(&s.Token, &s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *pgSessionRepository) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (r *pgSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}
