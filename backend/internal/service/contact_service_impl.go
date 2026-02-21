package service

import (
	"context"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// contactServiceImpl is the production implementation of ContactService.
type contactServiceImpl struct {
	repo repository.ContactRepository
}

// NewContactService creates a ContactService backed by the given repository.
func NewContactService(repo repository.ContactRepository) ContactService {
	return &contactServiceImpl{repo: repo}
}

// Submit stores a new contact message. It sets the status to "unread" and
// populates CreatedAt/UpdatedAt timestamps before persisting.
func (s *contactServiceImpl) Submit(ctx context.Context, msg *model.ContactMessage) error {
	now := time.Now().UTC()
	msg.Status = "unread"
	msg.CreatedAt = now
	msg.UpdatedAt = now
	return s.repo.Save(ctx, msg)
}

// List returns contact messages according to the given filter/pagination options.
func (s *contactServiceImpl) List(ctx context.Context, opts model.ContactListOptions) ([]*model.ContactMessage, error) {
	return s.repo.List(ctx, opts)
}

// UpdateStatus changes the status of a contact message.
func (s *contactServiceImpl) UpdateStatus(ctx context.Context, id string, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}
