package repository

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgProjectUpdateRepository は ProjectUpdateRepository の PostgreSQL 実装
type PgProjectUpdateRepository struct {
	pool *pgxpool.Pool
}

// NewPgProjectUpdateRepository は PgProjectUpdateRepository を生成する
func NewPgProjectUpdateRepository(pool *pgxpool.Pool) *PgProjectUpdateRepository {
	return &PgProjectUpdateRepository{pool: pool}
}

// ListByProjectID はプロジェクトに属する更新一覧を返す。
// includeHidden=false の場合、visible=true のものだけ返す。
// users テーブルと JOIN して author_name を取得する。
func (r *PgProjectUpdateRepository) ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
	query := `
		SELECT pu.id, pu.project_id, pu.author_id, pu.title, pu.body, pu.visible,
		       pu.created_at, pu.updated_at, u.name AS author_name
		FROM project_updates pu
		JOIN users u ON u.id = pu.author_id
		WHERE pu.project_id = $1`
	if !includeHidden {
		query += " AND pu.visible = true"
	}
	query += " ORDER BY pu.created_at DESC"

	rows, err := r.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var updates []*model.ProjectUpdate
	for rows.Next() {
		var u model.ProjectUpdate
		if err := rows.Scan(
			&u.ID, &u.ProjectID, &u.AuthorID, &u.Title, &u.Body, &u.Visible,
			&u.CreatedAt, &u.UpdatedAt, &u.AuthorName,
		); err != nil {
			return nil, err
		}
		updates = append(updates, &u)
	}
	return updates, rows.Err()
}

// GetByID は ID で更新を取得する
func (r *PgProjectUpdateRepository) GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error) {
	var u model.ProjectUpdate
	err := r.pool.QueryRow(ctx,
		`SELECT pu.id, pu.project_id, pu.author_id, pu.title, pu.body, pu.visible,
		        pu.created_at, pu.updated_at, u.name AS author_name
		 FROM project_updates pu
		 JOIN users u ON u.id = pu.author_id
		 WHERE pu.id = $1`,
		id,
	).Scan(&u.ID, &u.ProjectID, &u.AuthorID, &u.Title, &u.Body, &u.Visible,
		&u.CreatedAt, &u.UpdatedAt, &u.AuthorName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return &u, nil
}

// Create は新しい更新を作成する
func (r *PgProjectUpdateRepository) Create(ctx context.Context, update *model.ProjectUpdate) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO project_updates (project_id, author_id, title, body, visible)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		update.ProjectID, update.AuthorID, update.Title, update.Body, update.Visible,
	).Scan(&update.ID, &update.CreatedAt, &update.UpdatedAt)
}

// Update は title, body, visible, updated_at を更新する
func (r *PgProjectUpdateRepository) Update(ctx context.Context, update *model.ProjectUpdate) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE project_updates
		 SET title = $1, body = $2, visible = $3, updated_at = NOW()
		 WHERE id = $4`,
		update.Title, update.Body, update.Visible, update.ID,
	)
	return err
}

// Delete は visible=false をセットするソフトデリート
func (r *PgProjectUpdateRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE project_updates SET visible = false, updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}
