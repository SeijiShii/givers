package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgActivityRepository struct {
	pool *pgxpool.Pool
}

// NewPgActivityRepository returns a PostgreSQL-backed ActivityRepository.
func NewPgActivityRepository(pool *pgxpool.Pool) ActivityRepository {
	return &pgActivityRepository{pool: pool}
}

func (r *pgActivityRepository) Insert(ctx context.Context, a *model.ActivityItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO activities (type, project_id, actor_id, amount, rate, message)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))`,
		a.Type, a.ProjectID, a.ActorName, a.Amount, a.Rate, a.Message,
	)
	return err
}

// ExistsMilestoneThisMonth checks if a milestone activity at the given rate
// already exists for the project in the current month.
func (r *pgActivityRepository) ExistsMilestoneThisMonth(ctx context.Context, projectID string, rate int) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM activities
			WHERE type = 'milestone'
			  AND project_id = $1
			  AND rate = $2
			  AND created_at >= DATE_TRUNC('month', NOW())
		)`,
		projectID, rate,
	).Scan(&exists)
	return exists, err
}

const activitySelectQuery = `
	SELECT a.id, a.type, a.project_id, p.name,
	       CASE WHEN a.actor_id IS NOT NULL THEN COALESCE(u.name, '匿名') ELSE NULL END,
	       a.amount, a.rate, COALESCE(a.message, ''), a.created_at
	FROM activities a
	JOIN projects p ON a.project_id = p.id
	LEFT JOIN users u ON a.actor_id = u.id`

func (r *pgActivityRepository) ListGlobal(ctx context.Context, limit int) ([]*model.ActivityItem, error) {
	rows, err := r.pool.Query(ctx,
		activitySelectQuery+` ORDER BY a.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

func (r *pgActivityRepository) ListByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
	rows, err := r.pool.Query(ctx,
		activitySelectQuery+` WHERE a.project_id = $1 ORDER BY a.created_at DESC LIMIT $2`,
		projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivities(rows)
}

type scannable interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanActivities(rows scannable) ([]*model.ActivityItem, error) {
	var items []*model.ActivityItem
	for rows.Next() {
		a := &model.ActivityItem{}
		if err := rows.Scan(
			&a.ID, &a.Type, &a.ProjectID, &a.ProjectName,
			&a.ActorName, &a.Amount, &a.Rate, &a.Message, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
