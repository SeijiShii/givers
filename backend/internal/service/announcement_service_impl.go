package service

import (
	"context"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

type announcementServiceImpl struct {
	repo repository.AnnouncementRepository
}

// NewAnnouncementService は AnnouncementService の実装を生成する
func NewAnnouncementService(repo repository.AnnouncementRepository) AnnouncementService {
	return &announcementServiceImpl{repo: repo}
}

func (s *announcementServiceImpl) Create(ctx context.Context, a *model.Announcement) error {
	a.Visible = true
	if a.Severity == "" {
		a.Severity = "info"
	}
	if a.PublishedAt.IsZero() {
		a.PublishedAt = time.Now()
	}
	return s.repo.Insert(ctx, a)
}

func (s *announcementServiceImpl) GetByID(ctx context.Context, id string) (*model.Announcement, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *announcementServiceImpl) Update(ctx context.Context, a *model.Announcement) error {
	a.UpdatedAt = time.Now()
	return s.repo.Update(ctx, a)
}

func (s *announcementServiceImpl) SetVisibility(ctx context.Context, id string, visible bool) error {
	return s.repo.SetVisibility(ctx, id, visible)
}

func (s *announcementServiceImpl) ListVisible(ctx context.Context, userID string, limit int, cursor string) ([]*model.Announcement, string, error) {
	announcements, nextCursor, err := s.repo.ListVisible(ctx, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	if userID != "" && len(announcements) > 0 {
		_ = s.repo.SetReadFlags(ctx, userID, announcements)
	}
	return announcements, nextCursor, nil
}

func (s *announcementServiceImpl) ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
	return s.repo.ListAll(ctx, limit, cursor)
}

func (s *announcementServiceImpl) MarkRead(ctx context.Context, userID, announcementID string) error {
	return s.repo.MarkRead(ctx, userID, announcementID)
}

func (s *announcementServiceImpl) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}
