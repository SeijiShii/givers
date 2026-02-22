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

const projectSelectCols = `id, owner_id, name, description, deadline, status, owner_want_monthly, monthly_target, COALESCE(stripe_account_id, ''), created_at, updated_at`

// List はプロジェクト一覧を取得する
func (r *PgProjectRepository) List(ctx context.Context, limit, offset int) ([]*model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+projectSelectCols+`
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
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.MonthlyTarget, &p.StripeAccountID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

// GetByID は ID でプロジェクトを取得する（コスト項目・アラートも含む）
func (r *PgProjectRepository) GetByID(ctx context.Context, id string) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx,
		`SELECT `+projectSelectCols+` FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.MonthlyTarget, &p.StripeAccountID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Load cost items
	itemRows, err := r.pool.Query(ctx,
		`SELECT id, project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order, created_at, updated_at
		 FROM project_cost_items WHERE project_id = $1 ORDER BY sort_order, created_at`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()
	for itemRows.Next() {
		var item model.ProjectCostItem
		if err := itemRows.Scan(
			&item.ID, &item.ProjectID, &item.Label, &item.UnitType,
			&item.AmountMonthly, &item.RatePerDay, &item.DaysPerMonth,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.CostItems = append(p.CostItems, &item)
	}
	if err := itemRows.Err(); err != nil {
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
		`SELECT `+projectSelectCols+`
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
		if err := rows.Scan(&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.MonthlyTarget, &p.StripeAccountID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

// Create はプロジェクトを作成する
func (r *PgProjectRepository) Create(ctx context.Context, project *model.Project) error {
	project.MonthlyTarget = model.TotalMonthlyAmount(project.CostItems)

	err := r.pool.QueryRow(ctx,
		`INSERT INTO projects (owner_id, name, description, deadline, status, owner_want_monthly, monthly_target)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		project.OwnerID, project.Name, project.Description, project.Deadline,
		project.Status, project.OwnerWantMonthly, project.MonthlyTarget,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return err
	}

	for i, item := range project.CostItems {
		item.ProjectID = project.ID
		item.SortOrder = i
		if err := r.pool.QueryRow(ctx,
			`INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id, created_at, updated_at`,
			item.ProjectID, item.Label, item.UnitType, item.AmountMonthly, item.RatePerDay, item.DaysPerMonth, i,
		).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
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
	project.MonthlyTarget = model.TotalMonthlyAmount(project.CostItems)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE projects SET name=$1, description=$2, deadline=$3, status=$4, owner_want_monthly=$5, monthly_target=$6, updated_at=NOW()
		 WHERE id=$7`,
		project.Name, project.Description, project.Deadline, project.Status,
		project.OwnerWantMonthly, project.MonthlyTarget, project.ID,
	); err != nil {
		return err
	}

	// Replace cost items
	if _, err := tx.Exec(ctx, `DELETE FROM project_cost_items WHERE project_id=$1`, project.ID); err != nil {
		return err
	}
	for i, item := range project.CostItems {
		item.ProjectID = project.ID
		item.SortOrder = i
		if err := tx.QueryRow(ctx,
			`INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id, created_at, updated_at`,
			item.ProjectID, item.Label, item.UnitType, item.AmountMonthly, item.RatePerDay, item.DaysPerMonth, i,
		).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
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

// GetStripeAccountID はプロジェクトの stripe_account_id を返す（StripeService で使用）
func (r *PgProjectRepository) GetStripeAccountID(ctx context.Context, projectID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(stripe_account_id, '') FROM projects WHERE id=$1`, projectID,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateStripeConnect は Stripe Connect 完了後に stripe_account_id と status='active' を保存する
func (r *PgProjectRepository) UpdateStripeConnect(ctx context.Context, projectID, stripeAccountID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET stripe_account_id=$1, status='active', updated_at=NOW() WHERE id=$2`,
		stripeAccountID, projectID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
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
