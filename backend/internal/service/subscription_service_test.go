package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock SubscriptionRepository
// ---------------------------------------------------------------------------

type mockSubscriptionRepo struct {
	getByIDFunc                  func(ctx context.Context, id string) (*model.Subscription, error)
	patchFunc                    func(ctx context.Context, id string, patch model.SubscriptionPatch) error
	deleteFunc                   func(ctx context.Context, id string) error
	listByUserFunc               func(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error)
	createFunc                   func(ctx context.Context, s *model.Subscription) error
	getByStripeSubscriptionIDFunc func(ctx context.Context, stripeSubID string) (*model.Subscription, error)
	deleteByStripeSubscriptionIDFunc func(ctx context.Context, stripeSubID string) error
	migrateTokenFunc             func(ctx context.Context, token string, userID string) (int, error)
}

func (m *mockSubscriptionRepo) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockSubscriptionRepo) Patch(ctx context.Context, id string, patch model.SubscriptionPatch) error {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, id, patch)
	}
	return nil
}
func (m *mockSubscriptionRepo) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}
func (m *mockSubscriptionRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*model.Subscription, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}
func (m *mockSubscriptionRepo) Create(ctx context.Context, s *model.Subscription) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, s)
	}
	return nil
}
func (m *mockSubscriptionRepo) GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*model.Subscription, error) {
	if m.getByStripeSubscriptionIDFunc != nil {
		return m.getByStripeSubscriptionIDFunc(ctx, stripeSubID)
	}
	return nil, nil
}
func (m *mockSubscriptionRepo) DeleteByStripeSubscriptionID(ctx context.Context, stripeSubID string) error {
	if m.deleteByStripeSubscriptionIDFunc != nil {
		return m.deleteByStripeSubscriptionIDFunc(ctx, stripeSubID)
	}
	return nil
}
func (m *mockSubscriptionRepo) MigrateToken(ctx context.Context, token string, userID string) (int, error) {
	if m.migrateTokenFunc != nil {
		return m.migrateTokenFunc(ctx, token, userID)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock SubscriptionManager (for subscription_service tests)
// ---------------------------------------------------------------------------

type mockSubManager struct {
	pauseFunc        func(ctx context.Context, subscriptionID string) error
	resumeFunc       func(ctx context.Context, subscriptionID string) error
	cancelFunc       func(ctx context.Context, subscriptionID string) error
	updateAmountFunc func(ctx context.Context, subscriptionID string, newAmount int) error
}

func (m *mockSubManager) PauseSubscription(ctx context.Context, subscriptionID string) error {
	if m.pauseFunc != nil {
		return m.pauseFunc(ctx, subscriptionID)
	}
	return nil
}
func (m *mockSubManager) ResumeSubscription(ctx context.Context, subscriptionID string) error {
	if m.resumeFunc != nil {
		return m.resumeFunc(ctx, subscriptionID)
	}
	return nil
}
func (m *mockSubManager) CancelSubscription(ctx context.Context, subscriptionID string) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(ctx, subscriptionID)
	}
	return nil
}
func (m *mockSubManager) UpdateSubscriptionAmount(ctx context.Context, subscriptionID string, newAmount int) error {
	if m.updateAmountFunc != nil {
		return m.updateAmountFunc(ctx, subscriptionID, newAmount)
	}
	return nil
}

// ---------------------------------------------------------------------------
// SubscriptionService.Patch tests
// ---------------------------------------------------------------------------

func TestSubscriptionService_Patch_Forbidden(t *testing.T) {
	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{ID: id, DonorType: "user", DonorID: "other-user"}, nil
		},
	}
	svc := NewSubscriptionService(repo, nil)

	amount := 2000
	err := svc.Patch(context.Background(), "s1", "u1", model.SubscriptionPatch{Amount: &amount})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSubscriptionService_Patch_Pause(t *testing.T) {
	var capturedSubID string
	paused := true

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				StripeSubscriptionID: "sub_123",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.SubscriptionPatch) error {
			return nil
		},
	}
	sm := &mockSubManager{
		pauseFunc: func(ctx context.Context, subscriptionID string) error {
			capturedSubID = subscriptionID
			return nil
		},
	}
	svc := NewSubscriptionService(repo, sm)

	err := svc.Patch(context.Background(), "s1", "u1", model.SubscriptionPatch{Paused: &paused})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_123" {
		t.Errorf("expected PauseSubscription called with sub_123, got %q", capturedSubID)
	}
}

func TestSubscriptionService_Patch_Resume(t *testing.T) {
	var capturedSubID string
	paused := false

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				Paused: true, StripeSubscriptionID: "sub_456",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.SubscriptionPatch) error {
			return nil
		},
	}
	sm := &mockSubManager{
		resumeFunc: func(ctx context.Context, subscriptionID string) error {
			capturedSubID = subscriptionID
			return nil
		},
	}
	svc := NewSubscriptionService(repo, sm)

	err := svc.Patch(context.Background(), "s1", "u1", model.SubscriptionPatch{Paused: &paused})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_456" {
		t.Errorf("expected ResumeSubscription called with sub_456, got %q", capturedSubID)
	}
}

func TestSubscriptionService_Patch_Amount(t *testing.T) {
	var capturedSubID string
	var capturedAmount int
	newAmount := 3000

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				Amount: 1000, StripeSubscriptionID: "sub_amt",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.SubscriptionPatch) error {
			return nil
		},
	}
	sm := &mockSubManager{
		updateAmountFunc: func(ctx context.Context, subscriptionID string, newAmt int) error {
			capturedSubID = subscriptionID
			capturedAmount = newAmt
			return nil
		},
	}
	svc := NewSubscriptionService(repo, sm)

	err := svc.Patch(context.Background(), "s1", "u1", model.SubscriptionPatch{Amount: &newAmount})
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

func TestSubscriptionService_Patch_SameAmount_SkipsStripe(t *testing.T) {
	sameAmount := 1000
	stripeCalled := false

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				Amount: 1000, StripeSubscriptionID: "sub_same",
			}, nil
		},
		patchFunc: func(ctx context.Context, id string, patch model.SubscriptionPatch) error {
			return nil
		},
	}
	sm := &mockSubManager{
		updateAmountFunc: func(ctx context.Context, subscriptionID string, newAmt int) error {
			stripeCalled = true
			return nil
		},
	}
	svc := NewSubscriptionService(repo, sm)

	err := svc.Patch(context.Background(), "s1", "u1", model.SubscriptionPatch{Amount: &sameAmount})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stripeCalled {
		t.Error("expected UpdateSubscriptionAmount NOT to be called when amount unchanged")
	}
}

func TestSubscriptionService_Patch_StripeError_Propagates(t *testing.T) {
	paused := true

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				StripeSubscriptionID: "sub_err",
			}, nil
		},
	}
	patchCalled := false
	repo.patchFunc = func(ctx context.Context, id string, patch model.SubscriptionPatch) error {
		patchCalled = true
		return nil
	}
	sm := &mockSubManager{
		pauseFunc: func(ctx context.Context, subscriptionID string) error {
			return errors.New("stripe api error")
		},
	}
	svc := NewSubscriptionService(repo, sm)

	err := svc.Patch(context.Background(), "s1", "u1", model.SubscriptionPatch{Paused: &paused})
	if err == nil {
		t.Fatal("expected error when Stripe API fails")
	}
	if patchCalled {
		t.Error("expected repo.Patch NOT to be called when Stripe fails")
	}
}

// ---------------------------------------------------------------------------
// SubscriptionService.Delete tests
// ---------------------------------------------------------------------------

func TestSubscriptionService_Delete_Forbidden(t *testing.T) {
	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{ID: id, DonorType: "user", DonorID: "other-user"}, nil
		},
	}
	svc := NewSubscriptionService(repo, nil)

	err := svc.Delete(context.Background(), "s1", "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestSubscriptionService_Delete_CancelsStripe(t *testing.T) {
	var capturedSubID string
	deleteCalled := false

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				StripeSubscriptionID: "sub_del",
			}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			deleteCalled = true
			return nil
		},
	}
	sm := &mockSubManager{
		cancelFunc: func(ctx context.Context, subscriptionID string) error {
			capturedSubID = subscriptionID
			return nil
		},
	}
	svc := NewSubscriptionService(repo, sm)

	err := svc.Delete(context.Background(), "s1", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSubID != "sub_del" {
		t.Errorf("expected CancelSubscription called with sub_del, got %q", capturedSubID)
	}
	if !deleteCalled {
		t.Error("expected repo.Delete to be called after Stripe cancel")
	}
}

func TestSubscriptionService_Delete_NilManager(t *testing.T) {
	deleteCalled := false

	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{
				ID: id, DonorType: "user", DonorID: "u1",
				StripeSubscriptionID: "sub_nil",
			}, nil
		},
		deleteFunc: func(ctx context.Context, id string) error {
			deleteCalled = true
			return nil
		},
	}
	svc := NewSubscriptionService(repo, nil)

	err := svc.Delete(context.Background(), "s1", "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected repo.Delete to be called even with nil SubscriptionManager")
	}
}

// ---------------------------------------------------------------------------
// SubscriptionService.GetByID tests
// ---------------------------------------------------------------------------

func TestSubscriptionService_GetByID_Forbidden(t *testing.T) {
	repo := &mockSubscriptionRepo{
		getByIDFunc: func(ctx context.Context, id string) (*model.Subscription, error) {
			return &model.Subscription{ID: id, DonorType: "user", DonorID: "other-user"}, nil
		},
	}
	svc := NewSubscriptionService(repo, nil)

	_, err := svc.GetByID(context.Background(), "s1", "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}
