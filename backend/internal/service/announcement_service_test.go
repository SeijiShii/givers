package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// mockAnnouncementRepository — AnnouncementRepository のモック
// ---------------------------------------------------------------------------

type mockAnnouncementRepository struct {
	insertFunc        func(ctx context.Context, a *model.Announcement) error
	getByIDFunc       func(ctx context.Context, id string) (*model.Announcement, error)
	updateFunc        func(ctx context.Context, a *model.Announcement) error
	setVisibilityFunc func(ctx context.Context, id string, visible bool) error
	listVisibleFunc   func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
	listAllFunc       func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error)
	markReadFunc      func(ctx context.Context, userID, announcementID string) error
	unreadCountFunc   func(ctx context.Context, userID string) (int, error)
	setReadFlagsFunc  func(ctx context.Context, userID string, announcements []*model.Announcement) error
}

func (m *mockAnnouncementRepository) Insert(ctx context.Context, a *model.Announcement) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, a)
	}
	return nil
}

func (m *mockAnnouncementRepository) GetByID(ctx context.Context, id string) (*model.Announcement, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockAnnouncementRepository) Update(ctx context.Context, a *model.Announcement) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, a)
	}
	return nil
}

func (m *mockAnnouncementRepository) SetVisibility(ctx context.Context, id string, visible bool) error {
	if m.setVisibilityFunc != nil {
		return m.setVisibilityFunc(ctx, id, visible)
	}
	return nil
}

func (m *mockAnnouncementRepository) ListVisible(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
	if m.listVisibleFunc != nil {
		return m.listVisibleFunc(ctx, limit, cursor)
	}
	return nil, "", nil
}

func (m *mockAnnouncementRepository) ListAll(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
	if m.listAllFunc != nil {
		return m.listAllFunc(ctx, limit, cursor)
	}
	return nil, "", nil
}

func (m *mockAnnouncementRepository) MarkRead(ctx context.Context, userID, announcementID string) error {
	if m.markReadFunc != nil {
		return m.markReadFunc(ctx, userID, announcementID)
	}
	return nil
}

func (m *mockAnnouncementRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	if m.unreadCountFunc != nil {
		return m.unreadCountFunc(ctx, userID)
	}
	return 0, nil
}

func (m *mockAnnouncementRepository) SetReadFlags(ctx context.Context, userID string, announcements []*model.Announcement) error {
	if m.setReadFlagsFunc != nil {
		return m.setReadFlagsFunc(ctx, userID, announcements)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.Create
// ---------------------------------------------------------------------------

func TestAnnouncementService_Create_SetsDefaults(t *testing.T) {
	var captured *model.Announcement
	mock := &mockAnnouncementRepository{
		insertFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	a := &model.Announcement{
		AuthorID: "host-1",
		Title:    "Test",
		Body:     "Body",
	}
	if err := svc.Create(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !captured.Visible {
		t.Error("expected Visible=true for new announcement")
	}
	if captured.Severity != "info" {
		t.Errorf("expected default Severity=info, got %q", captured.Severity)
	}
	if captured.PublishedAt.IsZero() {
		t.Error("expected PublishedAt to be set")
	}
}

func TestAnnouncementService_Create_PreservesSeverity(t *testing.T) {
	var captured *model.Announcement
	mock := &mockAnnouncementRepository{
		insertFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	a := &model.Announcement{
		AuthorID: "host-1",
		Title:    "Incident",
		Severity: "error",
	}
	if err := svc.Create(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if captured.Severity != "error" {
		t.Errorf("expected Severity=error, got %q", captured.Severity)
	}
}

func TestAnnouncementService_Create_PreservesScheduledPublishedAt(t *testing.T) {
	var captured *model.Announcement
	mock := &mockAnnouncementRepository{
		insertFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	futureTime := time.Now().Add(24 * time.Hour)
	a := &model.Announcement{
		AuthorID:    "host-1",
		Title:       "Scheduled",
		PublishedAt: futureTime,
	}
	if err := svc.Create(context.Background(), a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !captured.PublishedAt.Equal(futureTime) {
		t.Errorf("expected PublishedAt to be preserved for scheduled posting, got %v", captured.PublishedAt)
	}
}

func TestAnnouncementService_Create_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		insertFunc: func(ctx context.Context, a *model.Announcement) error {
			return errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	err := svc.Create(context.Background(), &model.Announcement{AuthorID: "h1", Title: "T"})
	if err == nil {
		t.Error("expected error from Create, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.GetByID
// ---------------------------------------------------------------------------

func TestAnnouncementService_GetByID_CallsRepository(t *testing.T) {
	var capturedID string
	want := &model.Announcement{ID: "a1", Title: "Test"}
	mock := &mockAnnouncementRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Announcement, error) {
			capturedID = id
			return want, nil
		},
	}

	svc := NewAnnouncementService(mock)
	got, err := svc.GetByID(context.Background(), "a1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if capturedID != "a1" {
		t.Errorf("expected id=a1, got %q", capturedID)
	}
	if got.ID != "a1" {
		t.Errorf("expected ID=a1, got %q", got.ID)
	}
}

func TestAnnouncementService_GetByID_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Announcement, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewAnnouncementService(mock)
	_, err := svc.GetByID(context.Background(), "a1")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.Update
// ---------------------------------------------------------------------------

func TestAnnouncementService_Update_SetsUpdatedAt(t *testing.T) {
	before := time.Now().Add(-time.Second)
	var captured *model.Announcement
	mock := &mockAnnouncementRepository{
		updateFunc: func(ctx context.Context, a *model.Announcement) error {
			captured = a
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	a := &model.Announcement{ID: "a1", Title: "Updated"}
	if err := svc.Update(context.Background(), a); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if captured.UpdatedAt.Before(before) {
		t.Errorf("expected UpdatedAt to be set to now, got %v", captured.UpdatedAt)
	}
}

func TestAnnouncementService_Update_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		updateFunc: func(ctx context.Context, a *model.Announcement) error {
			return errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	err := svc.Update(context.Background(), &model.Announcement{ID: "a1"})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.SetVisibility
// ---------------------------------------------------------------------------

func TestAnnouncementService_SetVisibility_CallsRepository(t *testing.T) {
	var capturedID string
	var capturedVisible bool
	mock := &mockAnnouncementRepository{
		setVisibilityFunc: func(ctx context.Context, id string, visible bool) error {
			capturedID = id
			capturedVisible = visible
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	if err := svc.SetVisibility(context.Background(), "a1", false); err != nil {
		t.Fatalf("SetVisibility: %v", err)
	}
	if capturedID != "a1" {
		t.Errorf("expected id=a1, got %q", capturedID)
	}
	if capturedVisible {
		t.Error("expected visible=false, got true")
	}
}

func TestAnnouncementService_SetVisibility_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		setVisibilityFunc: func(ctx context.Context, id string, visible bool) error {
			return errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	err := svc.SetVisibility(context.Background(), "a1", true)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.ListVisible
// ---------------------------------------------------------------------------

func TestAnnouncementService_ListVisible_WithUserID_SetsReadFlags(t *testing.T) {
	announcements := []*model.Announcement{
		{ID: "a1", Title: "Test"},
		{ID: "a2", Title: "Test2"},
	}
	var readFlagsCalled bool
	mock := &mockAnnouncementRepository{
		listVisibleFunc: func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
			return announcements, "next", nil
		},
		setReadFlagsFunc: func(ctx context.Context, userID string, as []*model.Announcement) error {
			readFlagsCalled = true
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	got, nextCursor, err := svc.ListVisible(context.Background(), "user-1", 20, "")
	if err != nil {
		t.Fatalf("ListVisible: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 announcements, got %d", len(got))
	}
	if nextCursor != "next" {
		t.Errorf("expected cursor=next, got %q", nextCursor)
	}
	if !readFlagsCalled {
		t.Error("expected SetReadFlags to be called for authenticated user")
	}
}

func TestAnnouncementService_ListVisible_WithoutUserID_SkipsReadFlags(t *testing.T) {
	var readFlagsCalled bool
	mock := &mockAnnouncementRepository{
		listVisibleFunc: func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
			return []*model.Announcement{}, "", nil
		},
		setReadFlagsFunc: func(ctx context.Context, userID string, as []*model.Announcement) error {
			readFlagsCalled = true
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	_, _, err := svc.ListVisible(context.Background(), "", 20, "")
	if err != nil {
		t.Fatalf("ListVisible: %v", err)
	}
	if readFlagsCalled {
		t.Error("expected SetReadFlags NOT to be called for unauthenticated user")
	}
}

func TestAnnouncementService_ListVisible_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		listVisibleFunc: func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
			return nil, "", errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	_, _, err := svc.ListVisible(context.Background(), "", 20, "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.ListAll
// ---------------------------------------------------------------------------

func TestAnnouncementService_ListAll_CallsRepository(t *testing.T) {
	want := []*model.Announcement{{ID: "a1"}}
	mock := &mockAnnouncementRepository{
		listAllFunc: func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
			return want, "next", nil
		},
	}

	svc := NewAnnouncementService(mock)
	got, cursor, err := svc.ListAll(context.Background(), 20, "")
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1, got %d", len(got))
	}
	if cursor != "next" {
		t.Errorf("expected cursor=next, got %q", cursor)
	}
}

func TestAnnouncementService_ListAll_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		listAllFunc: func(ctx context.Context, limit int, cursor string) ([]*model.Announcement, string, error) {
			return nil, "", errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	_, _, err := svc.ListAll(context.Background(), 20, "")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.MarkRead
// ---------------------------------------------------------------------------

func TestAnnouncementService_MarkRead_CallsRepository(t *testing.T) {
	var capturedUserID, capturedAnnouncementID string
	mock := &mockAnnouncementRepository{
		markReadFunc: func(ctx context.Context, userID, announcementID string) error {
			capturedUserID = userID
			capturedAnnouncementID = announcementID
			return nil
		},
	}

	svc := NewAnnouncementService(mock)
	if err := svc.MarkRead(context.Background(), "user-1", "a1"); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if capturedUserID != "user-1" {
		t.Errorf("expected userID=user-1, got %q", capturedUserID)
	}
	if capturedAnnouncementID != "a1" {
		t.Errorf("expected announcementID=a1, got %q", capturedAnnouncementID)
	}
}

func TestAnnouncementService_MarkRead_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		markReadFunc: func(ctx context.Context, userID, announcementID string) error {
			return errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	err := svc.MarkRead(context.Background(), "user-1", "a1")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: AnnouncementService.UnreadCount
// ---------------------------------------------------------------------------

func TestAnnouncementService_UnreadCount_CallsRepository(t *testing.T) {
	mock := &mockAnnouncementRepository{
		unreadCountFunc: func(ctx context.Context, userID string) (int, error) {
			return 5, nil
		},
	}

	svc := NewAnnouncementService(mock)
	count, err := svc.UnreadCount(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if count != 5 {
		t.Errorf("expected count=5, got %d", count)
	}
}

func TestAnnouncementService_UnreadCount_PropagatesError(t *testing.T) {
	mock := &mockAnnouncementRepository{
		unreadCountFunc: func(ctx context.Context, userID string) (int, error) {
			return 0, errors.New("db error")
		},
	}

	svc := NewAnnouncementService(mock)
	_, err := svc.UnreadCount(context.Background(), "user-1")
	if err == nil {
		t.Error("expected error, got nil")
	}
}
