package service

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// ContactService defines the business logic for contact form submissions.
type ContactService interface {
	// Submit stores a new contact message. The msg.ID and timestamps will be
	// populated by the implementation.
	Submit(ctx context.Context, msg *model.ContactMessage) error

	// List returns contact messages according to the given options.
	List(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error)
}
