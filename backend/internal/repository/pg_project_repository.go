package repository

import (
	"context"
	"errors"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgProjectRepository は ProjectRepository の PostgreSQL 実装
type PgProjectRepository struct {
	pool *pgxpool.Pool
}

// NewPgProjectRepository は PgProjectRepository を生成する
func NewPgProjectRepository(pool *pgxpool.Pool) *PgProjectRepository {
	return &PgProjectRepository{pool: pool}
}

// List はプロジェクト一覧を取得する
func (r *PgProjectRepository) List(ctx context.Context, limit, offset int) ([]*model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, owner_id, name, description, deadline, status, owner_want_monthly, created_at, updated_at
		 FROM projects ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

// GetByID は ID でプロジェクトを取得する
func (r *PgProjectRepository) GetByID(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx,
		`SELECT id, owner_id, name, description, deadline, status, owner_want_monthly, created_at, updated_at
		 FROM projects WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Load costs
	var c model.ProjectCosts
	err = r.pool.QueryRow(ctx,
		`SELECT id, project_id, server_cost_monthly, dev_cost_per_day, dev_days_per_month, other_cost_monthly, created_at, updated_at
		 FROM project_costs WHERE project_id = $1`, id,
	).Scan(&c.ID, &c.ProjectID, &c.ServerCostMonthly, &c.DevCostPerDay, &c.DevDaysPerMonth, &c.OtherCostMonthly, &c.CreatedAt, &c.UpdatedAt)
	if err == nil {
		p.Costs = &c
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Load alerts (optional)
	var a model.ProjectAlerts
	err = r.pool.QueryRow(ctx,
		`SELECT id, project_id, warning_threshold, critical_threshold, created_at, updated_at
		 FROM project_alerts WHERE project_id = $1`, id,
	).Scan(&a.ID, &a.ProjectID, &a.WarningThreshold, &a.CriticalThreshold, &a.CreatedAt, &a.UpdatedAt)
	if err == nil {
		p.Alerts = &a
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	return &p, nil
}

// ListByOwnerID はオーナーIDでプロジェクト一覧を取得する
func (r *PgProjectRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, owner_id, name, description, deadline, status, owner_want_monthly, created_at, updated_at
		 FROM projects WHERE owner_id = $1 ORDER BY created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

// Create はプロジェクトを作成する
func (r *PgProjectRepository) Create(ctx context.Context, project *model.Project) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO projects (owner_id, name, description, deadline, status, owner_want_monthly)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at, updated_at`,
		project.OwnerID, project.Name, project.Description, project.Deadline, project.Status, project.OwnerWantMonthly,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return err
	}

	if project.Costs != nil {
		project.Costs.ProjectID = project.ID
		if err := r.upsertCosts(ctx, project.Costs); err != nil {
			return err
		}
	}
	if project.Alerts != nil {
		project.Alerts.ProjectID = project.ID
		if err := r.upsertAlerts(ctx, project.Alerts); err != nil {
			return err
		}
	}
	return nil
}

// Update はプロジェクトを更新する
func (r *PgProjectRepository) Update(ctx context.Context, project *model.Project) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE projects SET name = $1, description = $2, deadline = $3, status = $4, owner_want_monthly = $5, updated_at = NOW()
		 WHERE id = $6`,
		project.Name, project.Description, project.Deadline, project.Status, project.OwnerWantMonthly, project.ID,
	)
	if err != nil {
		return err
	}
	if project.Costs != nil {
		project.Costs.ProjectID = project.ID
		if err := r.upsertCosts(ctx, project.Costs); err != nil {
			return err
		}
	}
	if project.Alerts != nil {
		project.Alerts.ProjectID = project.ID
		if err := r.upsertAlerts(ctx, project.Alerts); err != nil {
			return err
		}
	}
	return nil
}

// Delete はプロジェクトを論理削除する（status を "deleted" に更新）。
// 対象が存在しない場合は ErrNotFound を返す。
func (r *PgProjectRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET status='deleted', updated_at=NOW() WHERE id=$1 AND status != 'deleted'`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PgProjectRepository) upsertCosts(ctx context.Context, c *model.ProjectCosts) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO project_costs (project_id, server_cost_monthly, dev_cost_per_day, dev_days_per_month, other_cost_monthly)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (project_id) DO UPDATE SET
		   server_cost_monthly = EXCLUDED.server_cost_monthly,
		   dev_cost_per_day = EXCLUDED.dev_cost_per_day,
		   dev_days_per_month = EXCLUDED.dev_days_per_month,
		   other_cost_monthly = EXCLUDED.other_cost_monthly,
		   updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		c.ProjectID, c.ServerCostMonthly, c.DevCostPerDay, c.DevDaysPerMonth, c.OtherCostMonthly,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *PgProjectRepository) upsertAlerts(ctx context.Context, a *model.ProjectAlerts) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO project_alerts (project_id, warning_threshold, critical_threshold)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (project_id) DO UPDATE SET
		   warning_threshold = EXCLUDED.warning_threshold,
		   critical_threshold = EXCLUDED.critical_threshold,
		   updated_at = NOW()
		 RETURNING id, created_at, updated_at`,
		a.ProjectID, a.WarningThreshold, a.CriticalThreshold,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}
