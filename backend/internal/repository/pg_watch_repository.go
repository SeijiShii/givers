package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgWatchRepository は WatchRepository の PostgreSQL 実装
type PgWatchRepository struct {
	pool *pgxpool.Pool
}

// NewPgWatchRepository は PgWatchRepository を生成する
func NewPgWatchRepository(pool *pgxpool.Pool) *PgWatchRepository {
	return &PgWatchRepository{pool: pool}
}

// Watch はウォッチを登録する（冪等: 既に存在する場合は無視）
func (r *PgWatchRepository) Watch(ctx context.Context, userID, projectID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO watches (user_id, project_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, project_id) DO NOTHING`,
		userID, projectID,
	)
	return err
}

// Unwatch はウォッチを解除する（冪等: 存在しない場合は無視）
func (r *PgWatchRepository) Unwatch(ctx context.Context, userID, projectID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM watches WHERE user_id = $1 AND project_id = $2`,
		userID, projectID,
	)
	return err
}

// ListWatchedProjects はユーザーがウォッチしているプロジェクト一覧を返す
func (r *PgWatchRepository) ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.owner_id, p.name, p.description, p.deadline, p.status, p.owner_want_monthly, p.created_at, p.updated_at
		 FROM projects p
		 INNER JOIN watches w ON w.project_id = p.id
		 WHERE w.user_id = $1
		 ORDER BY w.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(
			&p.ID, &p.OwnerID, &p.Name, &p.Description,
			&p.Deadline, &p.Status, &p.OwnerWantMonthly,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}
