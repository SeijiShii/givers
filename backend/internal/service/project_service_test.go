package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
)

// mockProjectRepository は ProjectRepository のモック
type mockProjectRepository struct {
	listFunc           func(ctx context.Context, limit, offset int) ([]*model.Project, error)
	getByIDFunc        func(ctx context.Context, id string) (*model.Project, error)
	listByOwnerIDFunc  func(ctx context.Context, ownerID string) ([]*model.Project, error)
	createFunc         func(ctx context.Context, project *model.Project) error
	updateFunc         func(ctx context.Context, project *model.Project) error
	deleteFunc         func(ctx context.Context, id string) error
}

func (m *mockProjectRepository) List(ctx context.Context, limit, offset int) ([]*model.Project, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, limit, offset)
	}
	return nil, nil
}

func (m *mockProjectRepository) GetByID(ctx context.Context, id string) (*model.Project, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockProjectRepository) ListByOwnerID(ctx context.Context, ownerID string) ([]*model.Project, error) {
	if m.listByOwnerIDFunc != nil {
		return m.listByOwnerIDFunc(ctx, ownerID)
	}
	return nil, nil
}

func (m *mockProjectRepository) Create(ctx context.Context, project *model.Project) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, project)
	}
	return nil
}

func (m *mockProjectRepository) Update(ctx context.Context, project *model.Project) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, project)
	}
	return nil
}

func (m *mockProjectRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockProjectRepository) UpdateStripeConnect(_ context.Context, _, _ string) error {
	return nil
}

func TestProjectService_List(t *testing.T) {
	ctx := context.Background()
	want := []*model.Project{{ID: "1", Name: "P1"}}

	mock := &mockProjectRepository{
		listFunc: func(ctx context.Context, limit, offset int) ([]*model.Project, error) {
			if limit != 10 || offset != 5 {
				t.Errorf("expected limit=10 offset=5, got limit=%d offset=%d", limit, offset)
			}
			return want, nil
		},
	}

	svc := NewProjectService(mock)
	got, err := svc.List(ctx, 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "1" {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestProjectService_GetByID(t *testing.T) {
	ctx := context.Background()
	want := &model.Project{ID: "p1", Name: "Project 1"}

	mock := &mockProjectRepository{
		getByIDFunc: func(ctx context.Context, id string) (*model.Project, error) {
			if id != "p1" {
				return nil, errors.New("not found")
			}
			return want, nil
		},
	}

	svc := NewProjectService(mock)
	got, err := svc.GetByID(ctx, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestProjectService_Create_SetsDefaultStatus(t *testing.T) {
	ctx := context.Background()
	var created *model.Project

	mock := &mockProjectRepository{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			project.ID = "new-id"
			return nil
		},
	}

	svc := NewProjectService(mock)
	p := &model.Project{OwnerID: "u1", Name: "Test", Description: "Desc"}
	if err := svc.Create(ctx, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != "active" {
		t.Errorf("expected status=active, got %q", created.Status)
	}
}

func TestProjectService_Create_PreservesStatus(t *testing.T) {
	ctx := context.Background()
	var created *model.Project

	mock := &mockProjectRepository{
		createFunc: func(ctx context.Context, project *model.Project) error {
			created = project
			return nil
		},
	}

	svc := NewProjectService(mock)
	p := &model.Project{OwnerID: "u1", Name: "Test", Status: "draft"}
	if err := svc.Create(ctx, p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Status != "draft" {
		t.Errorf("expected status=draft, got %q", created.Status)
	}
}
