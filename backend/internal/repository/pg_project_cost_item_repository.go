package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgProjectCostItemRepository は ProjectCostItemRepository の PostgreSQL 実装
type PgProjectCostItemRepository struct {
	pool *pgxpool.Pool
}

// NewPgProjectCostItemRepository は PgProjectCostItemRepository を生成する
func NewPgProjectCostItemRepository(pool *pgxpool.Pool) *PgProjectCostItemRepository {
	return &PgProjectCostItemRepository{pool: pool}
}

// ListByProjectID はプロジェクトのコスト項目一覧を返す
func (r *PgProjectCostItemRepository) ListByProjectID(ctx context.Context, projectID string) ([]*model.ProjectCostItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order, created_at, updated_at
		 FROM project_cost_items WHERE project_id = $1 ORDER BY sort_order, created_at`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.ProjectCostItem
	for rows.Next() {
		var item model.ProjectCostItem
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.Label, &item.UnitType,
			&item.AmountMonthly, &item.RatePerDay, &item.DaysPerMonth,
			&item.SortOrder, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

// ReplaceAll は既存アイテムを全削除して items を挿入し、projects.monthly_target を更新する
func (r *PgProjectCostItemRepository) ReplaceAll(ctx context.Context, projectID string, items []*model.ProjectCostItem) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM project_cost_items WHERE project_id=$1`, projectID); err != nil {
		return err
	}

	for i, item := range items {
		item.ProjectID = projectID
		item.SortOrder = i
		if err := tx.QueryRow(ctx,
			`INSERT INTO project_cost_items (project_id, label, unit_type, amount_monthly, rate_per_day, days_per_month, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 RETURNING id, created_at, updated_at`,
			projectID, item.Label, item.UnitType, item.AmountMonthly, item.RatePerDay, item.DaysPerMonth, i,
		).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return err
		}
	}

	total := model.TotalMonthlyAmount(items)
	if _, err := tx.Exec(ctx,
		`UPDATE projects SET monthly_target=$1, updated_at=NOW() WHERE id=$2`,
		total, projectID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
