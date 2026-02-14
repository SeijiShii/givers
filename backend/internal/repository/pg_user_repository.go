package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgUserRepository は UserRepository の PostgreSQL 実装
type PgUserRepository struct {
	pool *pgxpool.Pool
}

// NewPgUserRepository は PgUserRepository を生成する
func NewPgUserRepository(pool *pgxpool.Pool) *PgUserRepository {
	return &PgUserRepository{pool: pool}
}

// Ping は DB 接続を確認する（DB インターフェース実装）
func (r *PgUserRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// FindByID は ID でユーザーを取得する
func (r *PgUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	var googleID, githubID *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, google_id, github_id, name, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &googleID, &githubID, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if googleID != nil {
		u.GoogleID = *googleID
	}
	if githubID != nil {
		u.GitHubID = *githubID
	}
	return &u, nil
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByGoogleID は Google ID でユーザーを取得する
func (r *PgUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	var u model.User
	var gid, ghID *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, google_id, github_id, name, created_at, updated_at FROM users WHERE google_id = $1`,
		googleID,
	).Scan(&u.ID, &u.Email, &gid, &ghID, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if gid != nil {
		u.GoogleID = *gid
	}
	if ghID != nil {
		u.GitHubID = *ghID
	}
	return &u, nil
}

// FindByGitHubID は GitHub ID でユーザーを取得する
func (r *PgUserRepository) FindByGitHubID(ctx context.Context, githubID string) (*model.User, error) {
	var u model.User
	var gid, ghID *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, google_id, github_id, name, created_at, updated_at FROM users WHERE github_id = $1`,
		githubID,
	).Scan(&u.ID, &u.Email, &gid, &ghID, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if gid != nil {
		u.GoogleID = *gid
	}
	if ghID != nil {
		u.GitHubID = *ghID
	}
	return &u, nil
}

// Create はユーザーを作成する
func (r *PgUserRepository) Create(ctx context.Context, user *model.User) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, github_id, name) VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), $4)
		 RETURNING id, created_at, updated_at`,
		user.Email, user.GoogleID, user.GitHubID, user.Name,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}
