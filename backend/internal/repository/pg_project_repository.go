package repository

import (
	"context"
	"encoding/json"
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

const projectSelectCols = `p.id, p.owner_id, p.name, p.description, p.overview, p.share_message, p.deadline, p.status, p.owner_want_monthly, p.monthly_target, COALESCE(p.stripe_account_id, ''), p.cost_items, p.image_url, p.created_at, p.updated_at, COALESCE((SELECT SUM(amount) FROM donations WHERE project_id = p.id AND created_at >= DATE_TRUNC('month', NOW())), 0)::int`

func scanProject(row pgx.Row) (*model.Project, error) {
	var p model.Project
	var costItemsJSON []byte
	if err := row.Scan(
		&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Overview, &p.ShareMessage,
		&p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.MonthlyTarget,
		&p.StripeAccountID, &costItemsJSON, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
		&p.CurrentMonthlyDonations,
	); err != nil {
		return nil, err
	}
	if len(costItemsJSON) > 0 {
		_ = json.Unmarshal(costItemsJSON, &p.CostItems)
	}
	return &p, nil
}

func scanProjects(rows pgx.Rows) ([]*model.Project, error) {
	var projects []*model.Project
	for rows.Next() {
		var p model.Project
		var costItemsJSON []byte
		if err := rows.Scan(
			&p.ID, &p.OwnerID, &p.Name, &p.Description, &p.Overview, &p.ShareMessage,
			&p.Deadline, &p.Status, &p.OwnerWantMonthly, &p.MonthlyTarget,
			&p.StripeAccountID, &costItemsJSON, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
			&p.CurrentMonthlyDonations,
		); err != nil {
			return nil, err
		}
		if len(costItemsJSON) > 0 {
			_ = json.Unmarshal(costItemsJSON, &p.CostItems)
		}
		projects = append(projects, &p)
	}
	return projects, rows.Err()
}

// List はプロジェクト一覧を取得する。sort は "new"（デフォルト）または "hot"（達成率降順）。
// cursor はカーソルベースページネーション用（前回最後のプロジェクト ID）。
func (r *PgProjectRepository) List(ctx context.Context, sort string, limit int, cursor string) (*model.ProjectListResult, error) {
	// limit+1 をフェッチして next_cursor の有無を判定
	fetchLimit := limit + 1
	var rows pgx.Rows
	var err error

	switch sort {
	case "hot":
		if cursor == "" {
			rows, err = r.pool.Query(ctx,
				`SELECT `+projectSelectCols+`
				 FROM projects p
				 LEFT JOIN LATERAL (
				   SELECT COALESCE(SUM(amount), 0) AS total
				   FROM donations
				   WHERE project_id = p.id
				     AND created_at >= date_trunc('month', NOW())
				 ) d ON true
				 WHERE p.status = 'active'
				 ORDER BY CASE WHEN p.monthly_target > 0 THEN d.total::float / p.monthly_target ELSE 0 END DESC,
				          p.created_at DESC
				 LIMIT $1`, fetchLimit)
		} else {
			// hot ソートでのカーソル: cursor ID より後のものを取得
			// 達成率は変動するため、cursor の位置を created_at で近似する
			rows, err = r.pool.Query(ctx,
				`SELECT `+projectSelectCols+`
				 FROM projects p
				 LEFT JOIN LATERAL (
				   SELECT COALESCE(SUM(amount), 0) AS total
				   FROM donations
				   WHERE project_id = p.id
				     AND created_at >= date_trunc('month', NOW())
				 ) d ON true
				 WHERE p.status = 'active'
				   AND (p.created_at, p.id) < ((SELECT created_at FROM projects WHERE id = $2), $2)
				 ORDER BY CASE WHEN p.monthly_target > 0 THEN d.total::float / p.monthly_target ELSE 0 END DESC,
				          p.created_at DESC
				 LIMIT $1`, fetchLimit, cursor)
		}
	default:
		if cursor == "" {
			rows, err = r.pool.Query(ctx,
				`SELECT `+projectSelectCols+`
				 FROM projects p WHERE p.status = 'active' ORDER BY p.created_at DESC, p.id DESC LIMIT $1`, fetchLimit)
		} else {
			rows, err = r.pool.Query(ctx,
				`SELECT `+projectSelectCols+`
				 FROM projects p
				 WHERE p.status = 'active'
				   AND (p.created_at, p.id) < ((SELECT created_at FROM projects WHERE id = $2), $2)
				 ORDER BY p.created_at DESC, p.id DESC
				 LIMIT $1`, fetchLimit, cursor)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects, err := scanProjects(rows)
	if err != nil {
		return nil, err
	}

	result := &model.ProjectListResult{}
	if len(projects) > limit {
		result.NextCursor = projects[limit-1].ID
		result.Projects = projects[:limit]
	} else {
		result.Projects = projects
	}
	return result, nil
}

// GetByID は ID でプロジェクトを取得する（コスト項目・アラートも含む）
func (r *PgProjectRepository) GetByID(ctx context.Context, id string) (*model.Project, error) {
	p, err := scanProject(r.pool.QueryRow(ctx,
		`SELECT `+projectSelectCols+` FROM projects p WHERE p.id = $1`, id,
	))
	if err != nil {
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

	return p, nil
}

// ListByOwnerID はオーナーIDでプロジェクト一覧を取得する
func (r *PgProjectRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+projectSelectCols+`
		 FROM projects p WHERE p.owner_id = $1 ORDER BY p.created_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProjects(rows)
}

func marshalCostItems(items []model.CostItem) []byte {
	if len(items) == 0 {
		return nil
	}
	b, _ := json.Marshal(items)
	return b
}

// Create はプロジェクトを作成する
func (r *PgProjectRepository) Create(ctx context.Context, project *model.Project) error {
	project.MonthlyTarget = model.TotalMonthly(project.CostItems)

	err := r.pool.QueryRow(ctx,
		`INSERT INTO projects (owner_id, name, description, overview, share_message, deadline, status, owner_want_monthly, monthly_target, cost_items, image_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, created_at, updated_at`,
		project.OwnerID, project.Name, project.Description, project.Overview, project.ShareMessage, project.Deadline,
		project.Status, project.OwnerWantMonthly, project.MonthlyTarget, marshalCostItems(project.CostItems), project.ImageURL,
	).Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
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

// Update はプロジェクトを更新する
func (r *PgProjectRepository) Update(ctx context.Context, project *model.Project) error {
	project.MonthlyTarget = model.TotalMonthly(project.CostItems)

	if _, err := r.pool.Exec(ctx,
		`UPDATE projects SET name=$1, description=$2, overview=$3, share_message=$4, deadline=$5, status=$6, owner_want_monthly=$7, monthly_target=$8, cost_items=$9, image_url=$10, updated_at=NOW()
		 WHERE id=$11`,
		project.Name, project.Description, project.Overview, project.ShareMessage, project.Deadline, project.Status,
		project.OwnerWantMonthly, project.MonthlyTarget, marshalCostItems(project.CostItems), project.ImageURL, project.ID,
	); err != nil {
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

// SaveStripeAccountID は stripe_account_id のみを保存する（status は変更しない）
func (r *PgProjectRepository) SaveStripeAccountID(ctx context.Context, projectID, stripeAccountID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET stripe_account_id=$1, updated_at=NOW() WHERE id=$2`,
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

// ActivateProject はプロジェクトの status を 'active' に更新する
func (r *PgProjectRepository) ActivateProject(ctx context.Context, projectID string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET status='active', updated_at=NOW() WHERE id=$1`,
		projectID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetMonthlyTarget returns the monthly_target for a project (used by MilestoneService).
func (r *PgProjectRepository) GetMonthlyTarget(ctx context.Context, projectID string) (int, error) {
	var target int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(monthly_target, 0) FROM projects WHERE id = $1`, projectID,
	).Scan(&target)
	return target, err
}

// UpdateImageURL はプロジェクトの image_url のみを更新する
func (r *PgProjectRepository) UpdateImageURL(ctx context.Context, projectID, imageURL string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE projects SET image_url=$1, updated_at=NOW() WHERE id=$2`,
		imageURL, projectID,
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
