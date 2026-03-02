package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewPgSubscriptionRepository returns a PostgreSQL-backed SubscriptionRepository.
func NewPgSubscriptionRepository(pool *pgxpool.Pool) SubscriptionRepository {
	return &pgSubscriptionRepository{pool: pool}
}

const subscriptionSelectCols = `id, project_id, donor_type, donor_id, amount, currency,
	COALESCE(message, ''), COALESCE(stripe_subscription_id, ''), paused,
	COALESCE(next_billing_message, ''), created_at, updated_at`

func scanSubscription(scan func(...any) error) (*model.Subscription, error) {
	s := &model.Subscription{}
	return s, scan(
		&s.ID, &s.ProjectID, &s.DonorType, &s.DonorID,
		&s.Amount, &s.Currency, &s.Message,
		&s.StripeSubscriptionID, &s.Paused, &s.NextBillingMessage,
		&s.CreatedAt, &s.UpdatedAt,
	)
}

func (r *pgSubscriptionRepository) Create(ctx context.Context, s *model.Subscription) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO subscriptions
		 (project_id, donor_type, donor_id, amount, currency, message, stripe_subscription_id)
		 VALUES ($1, $2, $3, $4, $5, NULLIF($6,''), NULLIF($7,''))
		 RETURNING id`,
		s.ProjectID, s.DonorType, s.DonorID, s.Amount, s.Currency,
		s.Message, s.StripeSubscriptionID,
	).Scan(&s.ID)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return ErrDuplicate
	}
	return err
}

func (r *pgSubscriptionRepository) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+subscriptionSelectCols+` FROM subscriptions WHERE id = $1`, id)
	s, err := scanSubscription(row.Scan)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *pgSubscriptionRepository) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*model.Subscription, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+subscriptionSelectCols+` FROM subscriptions WHERE stripe_subscription_id = $1`, stripeSubID)
	s, err := scanSubscription(row.Scan)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *pgSubscriptionRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+subscriptionSelectCols+`
		 FROM subscriptions
		 WHERE donor_type = 'user' AND donor_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*model.Subscription
	for rows.Next() {
		s, err := scanSubscription(rows.Scan)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *pgSubscriptionRepository) Patch(ctx context.Context, id string, patch model.SubscriptionPatch) error {
	if patch.Amount == nil && patch.Paused == nil && patch.NextBillingMessage == nil {
		return nil
	}

	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if patch.Amount != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", argIdx))
		args = append(args, *patch.Amount)
		argIdx++
	}
	if patch.Paused != nil {
		setClauses = append(setClauses, fmt.Sprintf("paused = $%d", argIdx))
		args = append(args, *patch.Paused)
		argIdx++
	}
	if patch.NextBillingMessage != nil {
		setClauses = append(setClauses, fmt.Sprintf("next_billing_message = NULLIF($%d, '')", argIdx))
		args = append(args, *patch.NextBillingMessage)
		argIdx++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE subscriptions SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgSubscriptionRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgSubscriptionRepository) DeleteByStripeSubscriptionID(ctx context.Context, stripeSubID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM subscriptions WHERE stripe_subscription_id = $1`, stripeSubID)
	return err
}

func (r *pgSubscriptionRepository) MigrateToken(ctx context.Context, token string, userID string) (int, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE subscriptions SET donor_type = 'user', donor_id = $1, updated_at = NOW()
		 WHERE donor_type = 'token' AND donor_id = $2`,
		userID, token)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
