package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// mockProjectUpdateRepo — in-memory ProjectUpdateRepository for unit tests
// ---------------------------------------------------------------------------

type mockProjectUpdateRepo struct {
	updates   []*model.ProjectUpdate
	listErr   error
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func newMockProjectUpdateRepo() *mockProjectUpdateRepo {
	return &mockProjectUpdateRepo{}
}

func (r *mockProjectUpdateRepo) ListByProjectID(ctx context.Context, projectID string, includeHidden bool) ([]*model.ProjectUpdate, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var result []*model.ProjectUpdate
	for _, u := range r.updates {
		if u.ProjectID != projectID {
			continue
		}
		if !includeHidden && !u.Visible {
			continue
		}
		result = append(result, u)
	}
	return result, nil
}

func (r *mockProjectUpdateRepo) GetByID(ctx context.Context, id string) (*model.ProjectUpdate, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	for _, u := range r.updates {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (r *mockProjectUpdateRepo) Create(ctx context.Context, update *model.ProjectUpdate) error {
	if r.createErr != nil {
		return r.createErr
	}
	if update.ID == "" {
		update.ID = "generated-id"
	}
	update.CreatedAt = time.Now()
	update.UpdatedAt = time.Now()
	r.updates = append(r.updates, update)
	return nil
}

func (r *mockProjectUpdateRepo) Update(ctx context.Context, update *model.ProjectUpdate) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	for i, u := range r.updates {
		if u.ID == update.ID {
			r.updates[i] = update
			return nil
		}
	}
	return errors.New("not found")
}

func (r *mockProjectUpdateRepo) Delete(ctx context.Context, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	for _, u := range r.updates {
		if u.ID == id {
			u.Visible = false
			return nil
		}
	}
	return errors.New("not found")
}

// ---------------------------------------------------------------------------
// Tests: ListByProjectID
// ---------------------------------------------------------------------------

func TestProjectUpdateRepo_ListByProjectID_ReturnsVisibleUpdates(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	title := "First update"
	_ = repo.Create(ctx, &model.ProjectUpdate{
		ID:        "u1",
		ProjectID: "project-1",
		AuthorID:  "author-1",
		Title:     &title,
		Body:      "Some body text",
		Visible:   true,
	})
	_ = repo.Create(ctx, &model.ProjectUpdate{
		ID:        "u2",
		ProjectID: "project-1",
		AuthorID:  "author-1",
		Body:      "Hidden update",
		Visible:   false,
	})

	got, err := repo.ListByProjectID(ctx, "project-1", false)
	if err != nil {
		t.Fatalf("ListByProjectID returned unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 visible update, got %d", len(got))
	}
	if got[0].ID != "u1" {
		t.Errorf("expected update u1, got %q", got[0].ID)
	}
}

func TestProjectUpdateRepo_ListByProjectID_IncludeHidden(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "a1", Body: "visible", Visible: true})
	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u2", ProjectID: "project-1", AuthorID: "a1", Body: "hidden", Visible: false})

	got, err := repo.ListByProjectID(ctx, "project-1", true)
	if err != nil {
		t.Fatalf("ListByProjectID: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 updates (includeHidden=true), got %d", len(got))
	}
}

func TestProjectUpdateRepo_ListByProjectID_FiltersOtherProjects(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "project-1", AuthorID: "a1", Body: "body", Visible: true})
	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u2", ProjectID: "project-2", AuthorID: "a1", Body: "body", Visible: true})

	got, err := repo.ListByProjectID(ctx, "project-1", false)
	if err != nil {
		t.Fatalf("ListByProjectID: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 update for project-1, got %d", len(got))
	}
}

func TestProjectUpdateRepo_ListByProjectID_EmptyResult(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	got, err := repo.ListByProjectID(ctx, "no-such-project", false)
	if err != nil {
		t.Fatalf("ListByProjectID: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d", len(got))
	}
}

func TestProjectUpdateRepo_ListByProjectID_ReturnsError(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	repo.listErr = errors.New("db error")
	ctx := context.Background()

	_, err := repo.ListByProjectID(ctx, "project-1", false)
	if err == nil {
		t.Error("expected error from ListByProjectID, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: GetByID
// ---------------------------------------------------------------------------

func TestProjectUpdateRepo_GetByID_Found(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "hello", Visible: true})

	got, err := repo.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != "u1" {
		t.Errorf("expected ID=u1, got %q", got.ID)
	}
	if got.Body != "hello" {
		t.Errorf("expected Body=hello, got %q", got.Body)
	}
}

func TestProjectUpdateRepo_GetByID_NotFound(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for missing ID, got nil")
	}
}

func TestProjectUpdateRepo_GetByID_ReturnsError(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	repo.getErr = errors.New("db error")
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "u1")
	if err == nil {
		t.Error("expected error from GetByID, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Create
// ---------------------------------------------------------------------------

func TestProjectUpdateRepo_Create_SetsTimestamps(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	before := time.Now().Add(-time.Second)
	u := &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Body: "body", Visible: true}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if u.CreatedAt.Before(before) {
		t.Errorf("expected CreatedAt to be set, got %v", u.CreatedAt)
	}
	if u.UpdatedAt.Before(before) {
		t.Errorf("expected UpdatedAt to be set, got %v", u.UpdatedAt)
	}
}

func TestProjectUpdateRepo_Create_GeneratesIDWhenEmpty(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	u := &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Body: "body", Visible: true}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == "" {
		t.Error("expected ID to be generated, got empty string")
	}
}

func TestProjectUpdateRepo_Create_StoresUpdate(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	u := &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "content", Visible: true}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(repo.updates) != 1 {
		t.Errorf("expected 1 stored update, got %d", len(repo.updates))
	}
}

func TestProjectUpdateRepo_Create_ReturnsError(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	repo.createErr = errors.New("db error")
	ctx := context.Background()

	err := repo.Create(ctx, &model.ProjectUpdate{ProjectID: "p1", AuthorID: "a1", Body: "body"})
	if err == nil {
		t.Error("expected error from Create, got nil")
	}
}

func TestProjectUpdateRepo_Create_WithOptionalTitle(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	title := "Release v1.0"
	u := &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Title: &title, Body: "details", Visible: true}
	if err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, _ := repo.GetByID(ctx, "u1")
	if got.Title == nil || *got.Title != "Release v1.0" {
		t.Errorf("expected Title=%q, got %v", "Release v1.0", got.Title)
	}
}

// ---------------------------------------------------------------------------
// Tests: Update
// ---------------------------------------------------------------------------

func TestProjectUpdateRepo_Update_ModifiesFields(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "original", Visible: true})

	updated := &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "updated body", Visible: true}
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, "u1")
	if got.Body != "updated body" {
		t.Errorf("expected Body=updated body, got %q", got.Body)
	}
}

func TestProjectUpdateRepo_Update_CanSetVisibleFalse(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "body", Visible: true})

	updated := &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "body", Visible: false}
	if err := repo.Update(ctx, updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, "u1")
	if got.Visible {
		t.Error("expected Visible=false after update, got true")
	}
}

func TestProjectUpdateRepo_Update_NotFound(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	err := repo.Update(ctx, &model.ProjectUpdate{ID: "nonexistent", Body: "body"})
	if err == nil {
		t.Error("expected error for missing update, got nil")
	}
}

func TestProjectUpdateRepo_Update_ReturnsError(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	repo.updateErr = errors.New("db error")
	ctx := context.Background()

	err := repo.Update(ctx, &model.ProjectUpdate{ID: "u1", Body: "body"})
	if err == nil {
		t.Error("expected error from Update, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Delete (soft delete: sets visible=false)
// ---------------------------------------------------------------------------

func TestProjectUpdateRepo_Delete_SetsVisibleFalse(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "body", Visible: true})
	if err := repo.Delete(ctx, "u1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// The mock sets visible=false in-place on the pointer stored in the slice
	// Verify via ListByProjectID excludes it when includeHidden=false
	got, _ := repo.ListByProjectID(ctx, "p1", false)
	if len(got) != 0 {
		t.Errorf("expected 0 visible updates after Delete, got %d", len(got))
	}
}

func TestProjectUpdateRepo_Delete_NotFound(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for missing update, got nil")
	}
}

func TestProjectUpdateRepo_Delete_ReturnsError(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	repo.deleteErr = errors.New("db error")
	ctx := context.Background()

	err := repo.Delete(ctx, "u1")
	if err == nil {
		t.Error("expected error from Delete, got nil")
	}
}

func TestProjectUpdateRepo_Delete_OnlyTargetedUpdate(t *testing.T) {
	repo := newMockProjectUpdateRepo()
	ctx := context.Background()

	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u1", ProjectID: "p1", AuthorID: "a1", Body: "body1", Visible: true})
	_ = repo.Create(ctx, &model.ProjectUpdate{ID: "u2", ProjectID: "p1", AuthorID: "a1", Body: "body2", Visible: true})

	_ = repo.Delete(ctx, "u1")

	got, _ := repo.ListByProjectID(ctx, "p1", false)
	if len(got) != 1 {
		t.Errorf("expected 1 remaining visible update, got %d", len(got))
	}
	if got[0].ID != "u2" {
		t.Errorf("expected remaining update u2, got %q", got[0].ID)
	}
}
