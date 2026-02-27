package repository

import (
	"context"
	"fmt"
	"time"

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

func scanUser(scan func(...any) error) (*model.User, error) {
	var u model.User
	var googleID, githubID, discordID *string
	if err := scan(&u.ID, &u.Email, &googleID, &githubID, &discordID, &u.Name, &u.SuspendedAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	if googleID != nil {
		u.GoogleID = *googleID
	}
	if githubID != nil {
		u.GitHubID = *githubID
	}
	if discordID != nil {
		u.DiscordID = *discordID
	}
	return &u, nil
}

const userSelectCols = `id, email, google_id, github_id, discord_id, name, suspended_at, created_at, updated_at`

// FindByID は ID でユーザーを取得する
func (r *PgUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE id = $1`, id)
	return scanUser(row.Scan)
}

// FindByGoogleID は Google ID でユーザーを取得する
func (r *PgUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE google_id = $1`, googleID)
	return scanUser(row.Scan)
}

// FindByGitHubID は GitHub ID でユーザーを取得する
func (r *PgUserRepository) FindByGitHubID(ctx context.Context, githubID string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE github_id = $1`, githubID)
	return scanUser(row.Scan)
}

// FindByDiscordID は Discord ID でユーザーを取得する
func (r *PgUserRepository) FindByDiscordID(ctx context.Context, discordID string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE discord_id = $1`, discordID)
	return scanUser(row.Scan)
}

// FindByEmail はメールアドレスでユーザーを取得する
func (r *PgUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM users WHERE email = $1`, email)
	return scanUser(row.Scan)
}

// Create はユーザーを作成する
func (r *PgUserRepository) Create(ctx context.Context, user *model.User) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO users (email, google_id, github_id, discord_id, name) VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5)
		 RETURNING id, created_at, updated_at`,
		user.Email, user.GoogleID, user.GitHubID, user.DiscordID, user.Name,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// UpdateProviderID は指定ユーザーの OAuth プロバイダー ID を更新する
// column は "google_id", "github_id", "discord_id" のいずれか
func (r *PgUserRepository) UpdateProviderID(ctx context.Context, userID, column, value string) error {
	allowed := map[string]bool{"google_id": true, "github_id": true, "discord_id": true}
	if !allowed[column] {
		return fmt.Errorf("invalid provider column: %s", column)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET `+column+` = $1, updated_at = NOW() WHERE id = $2`,
		value, userID)
	return err
}

// List returns users ordered by created_at desc.
func (r *PgUserRepository) List(ctx context.Context, limit, offset int) ([]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+userSelectCols+` FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUser(rows.Scan)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Suspend sets or clears suspended_at for a user.
// suspend=true sets suspended_at=NOW(), suspend=false clears it.
func (r *PgUserRepository) Suspend(ctx context.Context, id string, suspend bool) error {
	var suspendedAt *time.Time
	if suspend {
		now := time.Now()
		suspendedAt = &now
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE users SET suspended_at = $1, updated_at = NOW() WHERE id = $2`,
		suspendedAt, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
