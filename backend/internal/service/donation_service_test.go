package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock DonationRepository
// ---------------------------------------------------------------------------

type mockDonationRepository struct {
	listByUserFunc  func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	getByIDFunc     func(ctx context.Context, id string) (*model.Donation, error)
	patchFunc       func(ctx context.Context, id string, patch model.DonationPatch) error
	deleteFunc      func(ctx context.Context, id string) error
	migrateFunc     func(ctx context.Context, token string, userID string) (int, error)
}

func (m *mockDonationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockDonationRepository) GetByID(ctx context.Context, id string) (*model.Donation, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockDonationRepository) Patch(ctx context.Context, id string, patch model.DonationPatch) error {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, id, patch)
	}
	return nil
}
func (m *mockDonationRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}
func (m *mockDonationRepository) Create(ctx context.Context, d *model.Donation) error {
	return nil
}
func (m *mockDonationRepository) MigrateToken(ctx context.Context, token string, userID string) (int, error) {
	if m.migrateFunc != nil {
		return m.migrateFunc(ctx, token, userID)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// DonationService.ListByUser tests
// ---------------------------------------------------------------------------

func TestDonationService_ListByUser_ReturnsUserDonations(t *testing.T) {
	now := time.Now()
	donations := []*model.Donation{
		{ID: "d1", ProjectID: "p1", DonorType: "user", DonorID: "u1", Amount: 1000, CreatedAt: now},
	}
	mock := &mockDonationRepository{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
			return donations, nil
		},
	}
	svc := NewDonationService(mock)

	got, err := svc.ListByUser(context.Background(), "u1", 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 donation, got %d", len(got))
	}
}

func TestDonationService_ListByUser_PropagatesError(t *testing.T) {
	mock := &mockDonationRepository{
		listByUserFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewDonationService(mock)
	_, err := svc.ListByUser(context.Background(), "u1", 20, 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// DonationService.Patch tests
// ---------------------------------------------------------------------------

func TestDonationService_Patch_Success(t *testing.T) {
	var capturedID string
	var capturedPatch model.DonationPatch
	amount := 2000
	paused := true

	mock := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, DonorType: "user", DonorID: "u1", IsRecurring: true}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			capturedID = id
			capturedPatch = patch
			return nil
		},
	}
	svc := NewDonationService(mock)

	patch := model.DonationPatch{Amount: &amount, Paused: &paused}
	if err := svc.Patch(context.Background(), "d1", "u1", patch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "d1" {
		t.Errorf("expected id=d1, got %q", capturedID)
	}
	if capturedPatch.Amount == nil || *capturedPatch.Amount != 2000 {
		t.Error("expected amount=2000")
	}
}

func TestDonationService_Patch_ForbiddenOtherUser(t *testing.T) {
	mock := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, DonorType: "user", DonorID: "other-user"}, nil
		},
	}
	svc := NewDonationService(mock)

	amount := 2000
	if err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Amount: &amount}); err == nil {
		t.Fatal("expected forbidden error for other user's donation")
	}
}

func TestDonationService_Patch_NotFound(t *testing.T) {
	mock := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return nil, repository.ErrNotFound
		},
	}
	svc := NewDonationService(mock)

	amount := 2000
	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Amount: &amount})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DonationService.Delete tests
// ---------------------------------------------------------------------------

func TestDonationService_Delete_Success(t *testing.T) {
	mock := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, DonorType: "user", DonorID: "u1"}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
	svc := NewDonationService(mock)

	if err := svc.Delete(context.Background(), "d1", "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDonationService_Delete_ForbiddenOtherUser(t *testing.T) {
	mock := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{ID: id, DonorType: "user", DonorID: "other-user"}, nil
		},
	}
	svc := NewDonationService(mock)

	if err := svc.Delete(context.Background(), "d1", "u1"); err == nil {
		t.Fatal("expected forbidden error for other user's donation")
	}
}

func TestDonationService_Delete_NotFound(t *testing.T) {
	mock := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return nil, repository.ErrNotFound
		},
	}
	svc := NewDonationService(mock)

	err := svc.Delete(context.Background(), "d1", "u1")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DonationService.MigrateToken tests
// ---------------------------------------------------------------------------

func TestDonationService_MigrateToken_Success(t *testing.T) {
	mock := &mockDonationRepository{
		migrateFunc: func(ctx context.Context, token string, userID string) (int, error) {
			return 3, nil
		},
	}
	svc := NewDonationService(mock)

	result, err := svc.MigrateToken(context.Background(), "token-abc", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MigratedCount != 3 {
		t.Errorf("expected MigratedCount=3, got %d", result.MigratedCount)
	}
	if result.AlreadyMigrated {
		t.Error("expected AlreadyMigrated=false")
	}
}

func TestDonationService_MigrateToken_AlreadyMigrated(t *testing.T) {
	mock := &mockDonationRepository{
		migrateFunc: func(ctx context.Context, token string, userID string) (int, error) {
			return 0, nil
		},
	}
	svc := NewDonationService(mock)

	result, err := svc.MigrateToken(context.Background(), "token-abc", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MigratedCount != 0 {
		t.Errorf("expected MigratedCount=0, got %d", result.MigratedCount)
	}
	if !result.AlreadyMigrated {
		t.Error("expected AlreadyMigrated=true when count=0")
	}
}

func TestDonationService_MigrateToken_InvalidToken(t *testing.T) {
	svc := NewDonationService(&mockDonationRepository{})

	_, err := svc.MigrateToken(context.Background(), "", "u1")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}
