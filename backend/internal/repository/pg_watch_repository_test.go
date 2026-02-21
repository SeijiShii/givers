package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// mockWatchRepository — in-memory WatchRepository for unit tests
// ---------------------------------------------------------------------------

// watchEntry は Watch テーブルの1行を表す
type watchEntry struct {
	userID    string
	projectID string
}

// mockWatchRepo は WatchRepository のインメモリ実装（テスト用）
type mockWatchRepo struct {
	watches  []watchEntry
	projects map[string]*model.Project // projectID → Project
	watchErr   error
	unwatchErr error
	listErr    error
}

func newMockWatchRepo() *mockWatchRepo {
	return &mockWatchRepo{
		projects: make(map[string]*model.Project),
	}
}

func (r *mockWatchRepo) Watch(ctx context.Context, userID, projectID string) error {
	if r.watchErr != nil {
		return r.watchErr
	}
	// idempotent: skip if already watching
	for _, e := range r.watches {
		if e.userID == userID && e.projectID == projectID {
			return nil
		}
	}
	r.watches = append(r.watches, watchEntry{userID: userID, projectID: projectID})
	return nil
}

func (r *mockWatchRepo) Unwatch(ctx context.Context, userID, projectID string) error {
	if r.unwatchErr != nil {
		return r.unwatchErr
	}
	newList := r.watches[:0]
	for _, e := range r.watches {
		if !(e.userID == userID && e.projectID == projectID) {
			newList = append(newList, e)
		}
	}
	r.watches = newList
	return nil
}

func (r *mockWatchRepo) ListWatchedProjects(ctx context.Context, userID string) ([]*model.Project, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var result []*model.Project
	for _, e := range r.watches {
		if e.userID == userID {
			if p, ok := r.projects[e.projectID]; ok {
				result = append(result, p)
			}
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Tests: Watch
// ---------------------------------------------------------------------------

func TestWatch_AddsEntry(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	if err := repo.Watch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("Watch returned unexpected error: %v", err)
	}
	if len(repo.watches) != 1 {
		t.Errorf("expected 1 watch entry, got %d", len(repo.watches))
	}
	if repo.watches[0].userID != "user-1" || repo.watches[0].projectID != "project-1" {
		t.Errorf("unexpected watch entry: %+v", repo.watches[0])
	}
}

func TestWatch_Idempotent(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	// Watch twice — should only record once
	if err := repo.Watch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("first Watch: %v", err)
	}
	if err := repo.Watch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("second Watch: %v", err)
	}
	if len(repo.watches) != 1 {
		t.Errorf("expected 1 watch entry after duplicate watch, got %d", len(repo.watches))
	}
}

func TestWatch_DifferentUsers(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	_ = repo.Watch(ctx, "user-1", "project-1")
	_ = repo.Watch(ctx, "user-2", "project-1")

	if len(repo.watches) != 2 {
		t.Errorf("expected 2 watch entries for different users, got %d", len(repo.watches))
	}
}

func TestWatch_DifferentProjects(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	_ = repo.Watch(ctx, "user-1", "project-1")
	_ = repo.Watch(ctx, "user-1", "project-2")

	if len(repo.watches) != 2 {
		t.Errorf("expected 2 watch entries for different projects, got %d", len(repo.watches))
	}
}

func TestWatch_ReturnsError(t *testing.T) {
	repo := newMockWatchRepo()
	repo.watchErr = errors.New("db error")
	ctx := context.Background()

	if err := repo.Watch(ctx, "user-1", "project-1"); err == nil {
		t.Error("expected error from Watch, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Unwatch
// ---------------------------------------------------------------------------

func TestUnwatch_RemovesEntry(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	_ = repo.Watch(ctx, "user-1", "project-1")
	if err := repo.Unwatch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("Unwatch returned unexpected error: %v", err)
	}
	if len(repo.watches) != 0 {
		t.Errorf("expected 0 watch entries after Unwatch, got %d", len(repo.watches))
	}
}

func TestUnwatch_Idempotent(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	// Unwatch on a non-existing entry should not error
	if err := repo.Unwatch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("Unwatch on non-existing entry returned error: %v", err)
	}
	if len(repo.watches) != 0 {
		t.Errorf("expected 0 entries, got %d", len(repo.watches))
	}
}

func TestUnwatch_OnlyRemovesTargetEntry(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	_ = repo.Watch(ctx, "user-1", "project-1")
	_ = repo.Watch(ctx, "user-1", "project-2")
	_ = repo.Watch(ctx, "user-2", "project-1")

	if err := repo.Unwatch(ctx, "user-1", "project-1"); err != nil {
		t.Fatalf("Unwatch: %v", err)
	}
	if len(repo.watches) != 2 {
		t.Errorf("expected 2 remaining watch entries, got %d", len(repo.watches))
	}
	for _, e := range repo.watches {
		if e.userID == "user-1" && e.projectID == "project-1" {
			t.Error("unwatched entry still present")
		}
	}
}

func TestUnwatch_ReturnsError(t *testing.T) {
	repo := newMockWatchRepo()
	repo.unwatchErr = errors.New("db error")
	ctx := context.Background()

	if err := repo.Unwatch(ctx, "user-1", "project-1"); err == nil {
		t.Error("expected error from Unwatch, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: ListWatchedProjects
// ---------------------------------------------------------------------------

func TestListWatchedProjects_ReturnsWatchedProjects(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	p1 := &model.Project{ID: "project-1", Name: "Alpha"}
	p2 := &model.Project{ID: "project-2", Name: "Beta"}
	repo.projects["project-1"] = p1
	repo.projects["project-2"] = p2

	_ = repo.Watch(ctx, "user-1", "project-1")
	_ = repo.Watch(ctx, "user-1", "project-2")

	got, err := repo.ListWatchedProjects(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListWatchedProjects: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 projects, got %d", len(got))
	}
}

func TestListWatchedProjects_EmptyForNewUser(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	got, err := repo.ListWatchedProjects(ctx, "user-unknown")
	if err != nil {
		t.Fatalf("ListWatchedProjects: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list for user with no watches, got %d items", len(got))
	}
}

func TestListWatchedProjects_OnlyReturnsCurrentUserProjects(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	p1 := &model.Project{ID: "project-1", Name: "Alpha"}
	p2 := &model.Project{ID: "project-2", Name: "Beta"}
	repo.projects["project-1"] = p1
	repo.projects["project-2"] = p2

	_ = repo.Watch(ctx, "user-1", "project-1")
	_ = repo.Watch(ctx, "user-2", "project-2")

	got, err := repo.ListWatchedProjects(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListWatchedProjects: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 project for user-1, got %d", len(got))
	}
	if got[0].ID != "project-1" {
		t.Errorf("expected project-1, got %q", got[0].ID)
	}
}

func TestListWatchedProjects_ReturnsError(t *testing.T) {
	repo := newMockWatchRepo()
	repo.listErr = errors.New("db error")
	ctx := context.Background()

	_, err := repo.ListWatchedProjects(ctx, "user-1")
	if err == nil {
		t.Error("expected error from ListWatchedProjects, got nil")
	}
}

func TestListWatchedProjects_AfterUnwatch(t *testing.T) {
	repo := newMockWatchRepo()
	ctx := context.Background()

	p1 := &model.Project{ID: "project-1", Name: "Alpha"}
	repo.projects["project-1"] = p1

	_ = repo.Watch(ctx, "user-1", "project-1")
	_ = repo.Unwatch(ctx, "user-1", "project-1")

	got, err := repo.ListWatchedProjects(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListWatchedProjects: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 projects after Unwatch, got %d", len(got))
	}
}
