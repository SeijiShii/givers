package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPgUserRepository_CreateAndFindByGoogleID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://givers:givers@localhost:5432/givers?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	repo := NewPgUserRepository(pool)

	unique := fmt.Sprintf("%d", time.Now().UnixNano())
	user := &model.User{
		Email:    fmt.Sprintf("test-%s@example.com", unique),
		GoogleID: fmt.Sprintf("google-%s", unique),
		Name:     "Test User",
	}

	err = repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if user.ID == "" {
		t.Error("expected ID to be set after Create")
	}

	found, err := repo.FindByGoogleID(ctx, user.GoogleID)
	if err != nil {
		t.Fatalf("FindByGoogleID failed: %v", err)
	}
	if found.Email != user.Email {
		t.Errorf("expected email %q, got %q", user.Email, found.Email)
	}
	if found.Name != user.Name {
		t.Errorf("expected name %q, got %q", user.Name, found.Name)
	}
}
