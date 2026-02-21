package repository

import (
	"context"
	"strings"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContactRepository defines the persistence interface for contact messages.
// It is defined here (in repository) to avoid an import cycle with service.
type ContactRepository interface {
	Save(ctx context.Context, msg *model.ContactMessage) error
	List(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error)
}

// PgContactRepository is the PostgreSQL implementation of ContactRepository.
type PgContactRepository struct {
	pool *pgxpool.Pool
}

// NewPgContactRepository creates a PgContactRepository backed by the given pool.
func NewPgContactRepository(pool *pgxpool.Pool) *PgContactRepository {
	return &PgContactRepository{pool: pool}
}

// Ensure PgContactRepository implements ContactRepository at compile time.
var _ ContactRepository = (*PgContactRepository)(nil)

// Save inserts a new contact_messages row and populates msg.ID and timestamps
// from the database RETURNING clause.
func (r *PgContactRepository) Save(ctx context.Context, msg *model.ContactMessage) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO contact_messages (email, name, message, status)
		 VALUES ($1, NULLIF($2, ''), $3, $4)
		 RETURNING id, created_at, updated_at`,
		msg.Email, msg.Name, msg.Message, msg.Status,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)
}

// List returns contact messages filtered by status and paginated by limit/offset.
// Status "" or "all" returns all messages.
func (r *PgContactRepository) List(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
	var conditions []string
	var args []any

	status := strings.TrimSpace(opts.Status)
	if status != "" && status != "all" {
		args = append(args, status)
		conditions = append(conditions, "status = $1")
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, opts.Limit, opts.Offset)

	query := `SELECT id, email, COALESCE(name, ''), message, status, created_at, updated_at
	          FROM contact_messages ` + where +
		` ORDER BY created_at DESC
		  LIMIT $` + itoa(limitArg) + ` OFFSET $` + itoa(offsetArg)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.ContactMessage
	for rows.Next() {
		var m model.ContactMessage
		if err := rows.Scan(&m.ID, &m.Email, &m.Name, &m.Message, &m.Status, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	return messages, rows.Err()
}

// itoa converts a small positive integer to its string representation.
// Using strconv.Itoa would require importing strconv; this avoids the dependency
// for single-digit parameter indices.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	// Fallback for two-digit indices — sufficient for our query param counts.
	tens := n / 10
	ones := n % 10
	return string(rune('0'+tens)) + string(rune('0'+ones))
}
