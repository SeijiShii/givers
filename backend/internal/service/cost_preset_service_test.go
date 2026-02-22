package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// ---------------------------------------------------------------------------
// Mock CostPresetRepository
// ---------------------------------------------------------------------------

type mockCostPresetRepository struct {
	listByUserIDFunc func(ctx context.Context, userID string) ([]*model.CostPreset, error)
	getByIDFunc      func(ctx context.Context, id string) (*model.CostPreset, error)
	createFunc       func(ctx context.Context, preset *model.CostPreset) error
	updateFunc       func(ctx context.Context, preset *model.CostPreset) error
	deleteFunc       func(ctx context.Context, id string) error
	reorderFunc      func(ctx context.Context, userID string, ids []string) error
}

func (m *mockCostPresetRepository) ListByUserID(ctx context.Context, userID string) ([]*model.CostPreset, error) {
	if m.listByUserIDFunc != nil {
		return m.listByUserIDFunc(ctx, userID)
	}
	return nil, nil
}
func (m *mockCostPresetRepository) GetByID(ctx context.Context, id string) (*model.CostPreset, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, repository.ErrNotFound
}
func (m *mockCostPresetRepository) Create(ctx context.Context, preset *model.CostPreset) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, preset)
	}
	return nil
}
func (m *mockCostPresetRepository) Update(ctx context.Context, preset *model.CostPreset) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, preset)
	}
	return nil
}
func (m *mockCostPresetRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}
func (m *mockCostPresetRepository) Reorder(ctx context.Context, userID string, ids []string) error {
	if m.reorderFunc != nil {
		return m.reorderFunc(ctx, userID, ids)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCostPresetService_List_ReturnsUserPresets(t *testing.T) {
	ctx := context.Background()
	presets := []*model.CostPreset{
		{ID: "p1", UserID: "u1", Label: "サーバー費用", UnitType: "monthly"},
	}
	mock := &mockCostPresetRepository{
		listByUserIDFunc: func(_ context.Context, userID string) ([]*model.CostPreset, error) {
			if userID != "u1" {
				t.Errorf("expected userID=u1, got %q", userID)
			}
			return presets, nil
		},
	}
	svc := NewCostPresetService(mock)

	got, err := svc.List(ctx, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "p1" {
		t.Errorf("expected 1 preset with id=p1, got %v", got)
	}
}

func TestCostPresetService_List_ReturnsDefaultsWhenEmpty(t *testing.T) {
	ctx := context.Background()
	mock := &mockCostPresetRepository{
		listByUserIDFunc: func(_ context.Context, _ string) ([]*model.CostPreset, error) {
			return nil, nil // no presets saved
		},
	}
	svc := NewCostPresetService(mock)

	got, err := svc.List(ctx, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defaults := model.DefaultCostPresets()
	if len(got) != len(defaults) {
		t.Errorf("expected %d defaults, got %d", len(defaults), len(got))
	}
}

func TestCostPresetService_Create_Success(t *testing.T) {
	ctx := context.Background()
	var created *model.CostPreset
	mock := &mockCostPresetRepository{
		createFunc: func(_ context.Context, preset *model.CostPreset) error {
			preset.ID = "new-id"
			created = preset
			return nil
		},
	}
	svc := NewCostPresetService(mock)

	p, err := svc.Create(ctx, "u1", "カスタム費用", "monthly")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "new-id" {
		t.Errorf("expected id=new-id, got %q", p.ID)
	}
	if created.UserID != "u1" || created.Label != "カスタム費用" || created.UnitType != "monthly" {
		t.Errorf("unexpected created preset: %+v", created)
	}
}

func TestCostPresetService_Create_InvalidUnitType(t *testing.T) {
	ctx := context.Background()
	svc := NewCostPresetService(&mockCostPresetRepository{})

	_, err := svc.Create(ctx, "u1", "Label", "invalid_type")
	if err == nil {
		t.Error("expected error for invalid unit_type")
	}
}

func TestCostPresetService_Update_Success(t *testing.T) {
	ctx := context.Background()
	existing := &model.CostPreset{ID: "p1", UserID: "u1", Label: "Old", UnitType: "monthly"}
	mock := &mockCostPresetRepository{
		getByIDFunc: func(_ context.Context, id string) (*model.CostPreset, error) {
			if id == "p1" {
				return existing, nil
			}
			return nil, repository.ErrNotFound
		},
		updateFunc: func(_ context.Context, preset *model.CostPreset) error {
			return nil
		},
	}
	svc := NewCostPresetService(mock)

	newLabel := "New Label"
	err := svc.Update(ctx, "p1", "u1", model.CostPresetPatch{Label: &newLabel})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if existing.Label != "New Label" {
		t.Errorf("expected label=New Label, got %q", existing.Label)
	}
}

func TestCostPresetService_Update_Forbidden(t *testing.T) {
	ctx := context.Background()
	mock := &mockCostPresetRepository{
		getByIDFunc: func(_ context.Context, id string) (*model.CostPreset, error) {
			return &model.CostPreset{ID: id, UserID: "other-user"}, nil
		},
	}
	svc := NewCostPresetService(mock)

	label := "X"
	err := svc.Update(ctx, "p1", "u1", model.CostPresetPatch{Label: &label})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCostPresetService_Update_NotFound(t *testing.T) {
	ctx := context.Background()
	svc := NewCostPresetService(&mockCostPresetRepository{})

	label := "X"
	err := svc.Update(ctx, "no-such", "u1", model.CostPresetPatch{Label: &label})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCostPresetService_Delete_Success(t *testing.T) {
	ctx := context.Background()
	var deletedID string
	mock := &mockCostPresetRepository{
		getByIDFunc: func(_ context.Context, id string) (*model.CostPreset, error) {
			return &model.CostPreset{ID: id, UserID: "u1"}, nil
		},
		deleteFunc: func(_ context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	svc := NewCostPresetService(mock)

	if err := svc.Delete(ctx, "p1", "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "p1" {
		t.Errorf("expected Delete called with p1, got %q", deletedID)
	}
}

func TestCostPresetService_Delete_Forbidden(t *testing.T) {
	ctx := context.Background()
	mock := &mockCostPresetRepository{
		getByIDFunc: func(_ context.Context, id string) (*model.CostPreset, error) {
			return &model.CostPreset{ID: id, UserID: "other-user"}, nil
		},
	}
	svc := NewCostPresetService(mock)

	err := svc.Delete(ctx, "p1", "u1")
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestCostPresetService_Reorder_Success(t *testing.T) {
	ctx := context.Background()
	var capturedIDs []string
	mock := &mockCostPresetRepository{
		reorderFunc: func(_ context.Context, userID string, ids []string) error {
			capturedIDs = ids
			return nil
		},
	}
	svc := NewCostPresetService(mock)

	ids := []string{"p3", "p1", "p2"}
	if err := svc.Reorder(ctx, "u1", ids); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedIDs) != 3 || capturedIDs[0] != "p3" {
		t.Errorf("unexpected reorder ids: %v", capturedIDs)
	}
}
