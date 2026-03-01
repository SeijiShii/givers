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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)
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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(mock, nil)

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
	svc := NewDonationService(&mockDonationRepository{}, nil)

	_, err := svc.MigrateToken(context.Background(), "", "u1")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

// ---------------------------------------------------------------------------
// Mock SubscriptionManager
// ---------------------------------------------------------------------------

type mockSubscriptionManager struct {
	pauseFunc        func(ctx context.Context, subID string) error
	resumeFunc       func(ctx context.Context, subID string) error
	cancelFunc       func(ctx context.Context, subID string) error
	updateAmountFunc func(ctx context.Context, subID string, newAmount int) error
}

func (m *mockSubscriptionManager) PauseSubscription(ctx context.Context, subID string) error {
	if m.pauseFunc != nil {
		return m.pauseFunc(ctx, subID)
	}
	return nil
}
func (m *mockSubscriptionManager) ResumeSubscription(ctx context.Context, subID string) error {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, subID)
	}
	return nil
}
func (m *mockSubscriptionManager) CancelSubscription(ctx context.Context, subID string) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, subID)
	}
	return nil
}
func (m *mockSubscriptionManager) UpdateSubscriptionAmount(ctx context.Context, subID string, newAmount int) error {
	if m.updateAmountFunc != nil {
		return m.updateAmountFunc(ctx, subID, newAmount)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stripe subscription integration tests
// ---------------------------------------------------------------------------

func TestDonationService_Patch_PausesStripeSubscription(t *testing.T) {
	var capturedSubID string
	paused := true

	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: true, StripeSubscriptionID: "sub_123",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		pauseFunc: func(ctx context.Context, subID string) error {
			capturedSubID = subID
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Paused: &paused})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_123" {
		t.Errorf("expected PauseSubscription called with sub_123, got %q", capturedSubID)
	}
}

func TestDonationService_Patch_ResumesStripeSubscription(t *testing.T) {
	var capturedSubID string
	paused := false

	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: true, Paused: true, StripeSubscriptionID: "sub_456",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		resumeFunc: func(ctx context.Context, subID string) error {
			capturedSubID = subID
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Paused: &paused})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_456" {
		t.Errorf("expected ResumeSubscription called with sub_456, got %q", capturedSubID)
	}
}

func TestDonationService_Patch_StripeErrorRollsBack(t *testing.T) {
	paused := true
	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: true, StripeSubscriptionID: "sub_789",
			}, nil
		},
	}
	sm := &mockSubscriptionManager{
		pauseFunc: func(ctx context.Context, subID string) error {
			return errors.New("stripe api error")
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Paused: &paused})
	if err == nil {
		t.Fatal("expected error when Stripe API fails")
	}
}

func TestDonationService_Patch_NoStripeCallForNonRecurring(t *testing.T) {
	paused := true
	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: false,
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		pauseFunc: func(ctx context.Context, subID string) error {
			t.Error("PauseSubscription should not be called for non-recurring donation")
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Paused: &paused})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDonationService_Delete_CancelsStripeSubscription(t *testing.T) {
	var capturedSubID string
	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: true, StripeSubscriptionID: "sub_del",
			}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		cancelFunc: func(ctx context.Context, subID string) error {
			capturedSubID = subID
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Delete(context.Background(), "d1", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_del" {
		t.Errorf("expected CancelSubscription called with sub_del, got %q", capturedSubID)
	}
}

func TestDonationService_Delete_StripeErrorPreventsDelete(t *testing.T) {
	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: true, StripeSubscriptionID: "sub_err",
			}, nil
		},
	}
	sm := &mockSubscriptionManager{
		cancelFunc: func(ctx context.Context, subID string) error {
			return errors.New("stripe cancel failed")
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Delete(context.Background(), "d1", "u1")
	if err == nil {
		t.Fatal("expected error when Stripe cancel fails")
	}
}

func TestDonationService_Patch_NilSubscriptionManager_SkipsStripe(t *testing.T) {
	paused := true
	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				IsRecurring: true, StripeSubscriptionID: "sub_nil",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	svc := NewDonationService(repo, nil)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Paused: &paused})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Stripe subscription amount update tests (#19)
// ---------------------------------------------------------------------------

func TestDonationService_Patch_UpdatesStripeAmount(t *testing.T) {
	var capturedSubID string
	var capturedAmount int
	newAmount := 3000

	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				Amount: 1000, IsRecurring: true, StripeSubscriptionID: "sub_amt",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		updateAmountFunc: func(ctx context.Context, subID string, amount int) error {
			capturedSubID = subID
			capturedAmount = amount
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Amount: &newAmount})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_amt" {
		t.Errorf("expected UpdateSubscriptionAmount called with sub_amt, got %q", capturedSubID)
	}
	if capturedAmount != 3000 {
		t.Errorf("expected amount=3000, got %d", capturedAmount)
	}
}

func TestDonationService_Patch_StripeAmountError_ReturnsError(t *testing.T) {
	newAmount := 5000

	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				Amount: 1000, IsRecurring: true, StripeSubscriptionID: "sub_err",
			}, nil
		},
	}
	sm := &mockSubscriptionManager{
		updateAmountFunc: func(ctx context.Context, subID string, amount int) error {
			return errors.New("stripe api error")
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Amount: &newAmount})
	if err == nil {
		t.Fatal("expected error when Stripe amount update fails")
	}
}

func TestDonationService_Patch_SameAmount_SkipsStripeUpdate(t *testing.T) {
	sameAmount := 1000
	stripeCalled := false

	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				Amount: 1000, IsRecurring: true, StripeSubscriptionID: "sub_same",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		updateAmountFunc: func(ctx context.Context, subID string, amount int) error {
			stripeCalled = true
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Amount: &sameAmount})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripeCalled {
		t.Error("expected UpdateSubscriptionAmount NOT to be called when amount unchanged")
	}
}

func TestDonationService_Patch_NonRecurring_SkipsStripeAmountUpdate(t *testing.T) {
	newAmount := 2000
	stripeCalled := false

	repo := &mockDonationRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Donation, error) {
			return &model.Donation{
				ID: id, DonorType: "user", DonorID: "u1",
				Amount: 1000, IsRecurring: false,
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.DonationPatch) error {
			return nil
		},
	}
	sm := &mockSubscriptionManager{
		updateAmountFunc: func(ctx context.Context, subID string, amount int) error {
			stripeCalled = true
			return nil
		},
	}
	svc := NewDonationService(repo, sm)

	err := svc.Patch(context.Background(), "d1", "u1", model.DonationPatch{Amount: &newAmount})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripeCalled {
		t.Error("expected UpdateSubscriptionAmount NOT to be called for non-recurring donation")
	}
}
