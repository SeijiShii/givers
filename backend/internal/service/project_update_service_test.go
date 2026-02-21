package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// mockProjectUpdateRepository — ProjectUpdateRepository のモック
// ---------------------------------------------------------------------------

type mockProjectUpdateRepository struct {
	listFunc   func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error)
	getFunc    func(ctx context.Context, id string) (*model.ProjectUpdate, error)
	createFunc func(ctx context.Context, update *model.ProjectUpdate) error
	updateFunc func(ctx context.Context, update *model.ProjectUpdate) error
	deleteFunc func(ctx context.Context, id string) error
}

func (m *mockProjectUpdateRepository) ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectID, includeHidden)
	}
	return nil, nil
}

func (m *mockProjectUpdateRepository) GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockProjectUpdateRepository) Create(ctx context.Context, update *model.ProjectUpdate) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, update)
	}
	return nil
}

func (m *mockProjectUpdateRepository) Update(ctx context.Context, update *model.ProjectUpdate) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, update)
	}
	return nil
}

func (m *mockProjectUpdateRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests: ProjectUpdateService.ListByProjectID
// ---------------------------------------------------------------------------

func TestProjectUpdateService_ListByProjectID_CallsRepository(t *testing.T) {
	var capturedProjectID string
	var capturedIncludeHidden bool
	want := []*model.ProjectUpdate{
		{ID: "u1", ProjectID: "project-1", Body: "update 1", Visible: true},
		{ID: "u2", ProjectID: "project-1", Body: "update 2", Visible: true},
	}
	mock := &mockProjectUpdateRepository{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			capturedProjectID = projectID
			capturedIncludeHidden = includeHidden
			return want, nil
		},
	}

	svc := NewProjectUpdateService(mock)
	got, err := svc.ListByProjectID(context.Background(), "project-1", false)
	if err != nil {
		t.Fatalf("ListByProjectID returned unexpected error: %v", err)
	}
	if capturedProjectID != "project-1" {
		t.Errorf("expected projectID=project-1, got %q", capturedProjectID)
	}
	if capturedIncludeHidden != false {
		t.Errorf("expected includeHidden=false, got %v", capturedIncludeHidden)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 updates, got %d", len(got))
	}
}

func TestProjectUpdateService_ListByProjectID_IncludeHiddenPassedThrough(t *testing.T) {
	var capturedIncludeHidden bool
	mock := &mockProjectUpdateRepository{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			capturedIncludeHidden = includeHidden
			return nil, nil
		},
	}

	svc := NewProjectUpdateService(mock)
	_, err := svc.ListByProjectID(context.Background(), "project-1", true)
	if err != nil {
		t.Fatalf("ListByProjectID: %v", err)
	}
	if !capturedIncludeHidden {
		t.Error("expected includeHidden=true to be passed through to repository")
	}
}

func TestProjectUpdateService_ListByProjectID_ReturnsEmptySlice(t *testing.T) {
	mock := &mockProjectUpdateRepository{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			return []*model.ProjectUpdate{}, nil
		},
	}

	svc := NewProjectUpdateService(mock)
	got, err := svc.ListByProjectID(context.Background(), "project-1", false)
	if err != nil {
		t.Fatalf("ListByProjectID: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}

func TestProjectUpdateService_ListByProjectID_PropagatesError(t *testing.T) {
	mock := &mockProjectUpdateRepository{
		listFunc: func(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewProjectUpdateService(mock)
	_, err := svc.ListByProjectID(context.Background(), "project-1", false)
	if err == nil {
		t.Error("expected error from ListByProjectID, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ProjectUpdateService.GetByID
// ---------------------------------------------------------------------------

func TestProjectUpdateService_GetByID_CallsRepository(t *testing.T) {
	var capturedID string
	want := &model.ProjectUpdate{ID: "u1", Body: "body", Visible: true}
	mock := &mockProjectUpdateRepository{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			capturedID = id
			return want, nil
		},
	}

	svc := NewProjectUpdateService(mock)
	got, err := svc.GetByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if capturedID != "u1" {
		t.Errorf("expected id=u1, got %q", capturedID)
	}
	if got.ID != "u1" {
		t.Errorf("expected ID=u1, got %q", got.ID)
	}
}

func TestProjectUpdateService_GetByID_PropagatesError(t *testing.T) {
	mock := &mockProjectUpdateRepository{
		getFunc: func(ctx context.Context, id string) (*model.ProjectUpdate, error) {
			return nil, errors.New("not found")
		},
	}

	svc := NewProjectUpdateService(mock)
	_, err := svc.GetByID(context.Background(), "u1")
	if err == nil {
		t.Error("expected error from GetByID, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ProjectUpdateService.Create
// ---------------------------------------------------------------------------

func TestProjectUpdateService_Create_SetsVisibleTrue(t *testing.T) {
	var capturedUpdate *model.ProjectUpdate
	mock := &mockProjectUpdateRepository{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			return nil
		},
	}

	svc := NewProjectUpdateService(mock)
	u := &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Body: "body"}
	if err := svc.Create(context.Background(), u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !capturedUpdate.Visible {
		t.Error("expected Visible=true for new updates, got false")
	}
}

func TestProjectUpdateService_Create_CallsRepository(t *testing.T) {
	called := false
	mock := &mockProjectUpdateRepository{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			called = true
			return nil
		},
	}

	svc := NewProjectUpdateService(mock)
	if err := svc.Create(context.Background(), &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Body: "body"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !called {
		t.Error("expected repository Create to be called, but it wasn't")
	}
}

func TestProjectUpdateService_Create_PropagatesError(t *testing.T) {
	mock := &mockProjectUpdateRepository{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			return errors.New("db error")
		},
	}

	svc := NewProjectUpdateService(mock)
	err := svc.Create(context.Background(), &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Body: "body"})
	if err == nil {
		t.Error("expected error from Create, got nil")
	}
}

func TestProjectUpdateService_Create_WithTitle(t *testing.T) {
	var capturedUpdate *model.ProjectUpdate
	mock := &mockProjectUpdateRepository{
		createFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			return nil
		},
	}

	svc := NewProjectUpdateService(mock)
	title := "Release v2"
	u := &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Title: &title, Body: "details"}
	if err := svc.Create(context.Background(), u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if capturedUpdate.Title == nil || *capturedUpdate.Title != "Release v2" {
		t.Errorf("expected Title=Release v2, got %v", capturedUpdate.Title)
	}
}

// ---------------------------------------------------------------------------
// Tests: ProjectUpdateService.Update
// ---------------------------------------------------------------------------

func TestProjectUpdateService_Update_CallsRepository(t *testing.T) {
	var capturedUpdate *model.ProjectUpdate
	mock := &mockProjectUpdateRepository{
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			return nil
		},
	}

	svc := NewProjectUpdateService(mock)
	u := &model.ProjectUpdate{ID: "u1", Body: "updated body", Visible: true}
	if err := svc.Update(context.Background(), u); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if capturedUpdate.ID != "u1" {
		t.Errorf("expected ID=u1, got %q", capturedUpdate.ID)
	}
	if capturedUpdate.Body != "updated body" {
		t.Errorf("expected Body=updated body, got %q", capturedUpdate.Body)
	}
}

func TestProjectUpdateService_Update_PropagatesError(t *testing.T) {
	mock := &mockProjectUpdateRepository{
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			return errors.New("db error")
		},
	}

	svc := NewProjectUpdateService(mock)
	err := svc.Update(context.Background(), &model.ProjectUpdate{ID: "u1", Body: "body"})
	if err == nil {
		t.Error("expected error from Update, got nil")
	}
}

func TestProjectUpdateService_Update_SetsUpdatedAt(t *testing.T) {
	before := time.Now().Add(-time.Second)
	var capturedUpdate *model.ProjectUpdate
	mock := &mockProjectUpdateRepository{
		updateFunc: func(ctx context.Context, update *model.ProjectUpdate) error {
			capturedUpdate = update
			return nil
		},
	}

	svc := NewProjectUpdateService(mock)
	u := &model.ProjectUpdate{ID: "u1", Body: "body", Visible: true}
	if err := svc.Update(context.Background(), u); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if capturedUpdate.UpdatedAt.Before(before) {
		t.Errorf("expected UpdatedAt to be set to now, got %v", capturedUpdate.UpdatedAt)
	}
}

// ---------------------------------------------------------------------------
// Tests: ProjectUpdateService.Delete
// ---------------------------------------------------------------------------

func TestProjectUpdateService_Delete_CallsRepository(t *testing.T) {
	var capturedID string
	mock := &mockProjectUpdateRepository{
		deleteFunc: func(ctx context.Context, id string) error {
			capturedID = id
			return nil
		},
	}

	svc := NewProjectUpdateService(mock)
	if err := svc.Delete(context.Background(), "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if capturedID != "u1" {
		t.Errorf("expected id=u1, got %q", capturedID)
	}
}

func TestProjectUpdateService_Delete_PropagatesError(t *testing.T) {
	mock := &mockProjectUpdateRepository{
		deleteFunc: func(ctx context.Context, id string) error {
			return errors.New("db error")
		},
	}

	svc := NewProjectUpdateService(mock)
	if err := svc.Delete(context.Background(), "u1"); err == nil {
		t.Error("expected error from Delete, got nil")
	}
}
