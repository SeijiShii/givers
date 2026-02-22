package repository

import (
	"context"
	"strings"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgDonationRepository struct {
	pool *pgxpool.Pool
}

// NewPgDonationRepository returns a PostgreSQL-backed DonationRepository.
func NewPgDonationRepository(pool *pgxpool.Pool) DonationRepository {
	return &pgDonationRepository{pool: pool}
}

const donationSelectCols = `id, project_id, donor_type, donor_id, amount, currency,
	COALESCE(message, ''), is_recurring, COALESCE(stripe_payment_id, ''),
	COALESCE(stripe_subscription_id, ''), paused, created_at, updated_at`

func scanDonation(scan func(...any) error) (*model.Donation, error) {
	d := &model.Donation{}
	return d, scan(
		&d.ID, &d.ProjectID, &d.DonorType, &d.DonorID,
		&d.Amount, &d.Currency, &d.Message,
		&d.IsRecurring, &d.StripePaymentID, &d.StripeSubscriptionID,
		&d.Paused, &d.CreatedAt, &d.UpdatedAt,
	)
}

func (r *pgDonationRepository) Create(ctx context.Context, d *model.Donation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO donations
		 (project_id, donor_type, donor_id, amount, currency, message, is_recurring,
		  stripe_payment_id, stripe_subscription_id)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), $7, NULLIF($8,''), NULLIF($9,''))`,
		d.ProjectID, d.DonorType, d.DonorID, d.Amount, d.Currency,
		d.Message, d.IsRecurring, d.StripePaymentID, d.StripeSubscriptionID,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrDuplicate
	}
	return err
}

func (r *pgDonationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+donationSelectCols+`
		 FROM donations
		 WHERE donor_type = 'user' AND donor_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.Donation
	for rows.Next() {
		d, err := scanDonation(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *pgDonationRepository) GetByID(ctx context.Context, id string) (*model.Donation, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+donationSelectCols+` FROM donations WHERE id = $1`, id)
	d, err := scanDonation(row.Scan)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *pgDonationRepository) Patch(ctx context.Context, id string, patch model.DonationPatch) error {
	if patch.Amount == nil && patch.Paused == nil {
		return nil
	}

	if patch.Amount != nil && patch.Paused != nil {
		tag, err := r.pool.Exec(ctx,
			`UPDATE donations SET amount = $1, paused = $2, updated_at = NOW() WHERE id = $3`,
			*patch.Amount, *patch.Paused, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}

	if patch.Amount != nil {
		tag, err := r.pool.Exec(ctx,
			`UPDATE donations SET amount = $1, updated_at = NOW() WHERE id = $2`,
			*patch.Amount, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE donations SET paused = $1, updated_at = NOW() WHERE id = $2`,
		*patch.Paused, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgDonationRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM donations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgDonationRepository) DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM donations WHERE stripe_subscription_id = $1`, subscriptionID)
	return err
}

func (r *pgDonationRepository) MigrateToken(ctx context.Context, token string, userID string) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE donations SET donor_type = 'user', donor_id = $1, updated_at = NOW()
		 WHERE donor_type = 'token' AND donor_id = $2`,
		userID, token)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (r *pgDonationRepository) ListActivityByProject(ctx context.Context, projectID string, limit int) ([]*model.ActivityItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
			CASE WHEN d.donor_type = 'user' THEN COALESCE(u.name, '匿名') ELSE '匿名' END,
			d.amount, d.created_at, COALESCE(d.message, '')
		 FROM donations d
		 LEFT JOIN users u ON d.donor_type = 'user' AND d.donor_id = u.id
		 WHERE d.project_id = $1
		 ORDER BY d.created_at DESC
		 LIMIT $2`,
		projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.ActivityItem
	for rows.Next() {
		a := &model.ActivityItem{}
		if err := rows.Scan(&a.DonorName, &a.Amount, &a.CreatedAt, &a.Message); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
