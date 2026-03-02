package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock DonationRepository
// ---------------------------------------------------------------------------

type mockDonationRepository struct {
	listByUserFunc func(ctx context.Context, userID string, limit, offset int) ([]*model.Donation, error)
	getByIDFunc    func(ctx context.Context, id string) (*model.Donation, error)
	patchFunc      func(ctx context.Context, id string, patch model.DonationPatch) error
	deleteFunc     func(ctx context.Context, id string) error
	migrateFunc    func(ctx context.Context, token string, userID string) (int, error)
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
func (m *mockDonationRepository) DeleteByStripeSubscriptionID(ctx context.Context, subscriptionID string) error {
	return nil
}
func (m *mockDonationRepository) MigrateToken(ctx context.Context, token string, userID string) (int, error) {
	if m.migrateFunc != nil {
		return m.migrateFunc(ctx, token, userID)
	}
	return 0, nil
}
func (m *mockDonationRepository) CurrentMonthSumByProject(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockDonationRepository) MonthlySumByProject(ctx context.Context, projectID string) ([]*model.MonthlySum, error) {
	return nil, nil
}
func (m *mockDonationRepository) ListByProject(ctx context.Context, projectID string, limit, offset int) ([]*model.Donation, error) {
	return nil, nil
}
func (m *mockDonationRepository) GetByStripeSubscriptionID(ctx context.Context, subscriptionID string) (*model.Donation, error) {
	return nil, nil
}
func (m *mockDonationRepository) ListMessagesByProject(ctx context.Context, projectID string, limit, offset int, sort, donor string) (*model.DonationMessageResult, error) {
	return &model.DonationMessageResult{Messages: []*model.DonationMessage{}, Total: 0}, nil
}
func (m *mockDonationRepository) ListByProjectForOwner(ctx context.Context, projectID string, limit, offset int, sort, sourceFilter, donorFilter string) (*model.OwnerDonationResult, error) {
	return nil, nil
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
