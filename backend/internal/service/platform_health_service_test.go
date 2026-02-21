package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mock PlatformHealthRepository
// ---------------------------------------------------------------------------

type mockPlatformHealthRepository struct {
	getFunc func(ctx context.Context) (*model.PlatformHealth, error)
}

func (m *mockPlatformHealthRepository) Get(ctx context.Context) (*model.PlatformHealth, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPlatformHealthService_Get_ReturnsHealth(t *testing.T) {
	now := time.Now()
	expected := &model.PlatformHealth{
		MonthlyCost:       50000,
		CurrentMonthly:    28000,
		WarningThreshold:  60,
		CriticalThreshold: 30,
		UpdatedAt:         now,
	}
	mock := &mockPlatformHealthRepository{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return expected, nil
		},
	}
	svc := NewPlatformHealthService(mock)

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.MonthlyCost != 50000 {
		t.Errorf("expected MonthlyCost=50000, got %d", got.MonthlyCost)
	}
	if got.CurrentMonthly != 28000 {
		t.Errorf("expected CurrentMonthly=28000, got %d", got.CurrentMonthly)
	}
}

func TestPlatformHealthService_Get_PropagatesError(t *testing.T) {
	mock := &mockPlatformHealthRepository{
		getFunc: func(ctx context.Context) (*model.PlatformHealth, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewPlatformHealthService(mock)

	_, err := svc.Get(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PlatformHealth model signal/rate tests
// ---------------------------------------------------------------------------

func TestPlatformHealth_Rate(t *testing.T) {
	cases := []struct {
		monthly    int
		current    int
		wantRate   int
	}{
		{50000, 28000, 56},
		{50000, 0, 0},
		{0, 0, 0},
		{100, 100, 100},
		{100, 150, 150}, // over 100%
	}
	for _, c := range cases {
		h := &model.PlatformHealth{MonthlyCost: c.monthly, CurrentMonthly: c.current}
		if got := h.Rate(); got != c.wantRate {
			t.Errorf("Rate(%d, %d) = %d, want %d", c.monthly, c.current, got, c.wantRate)
		}
	}
}

func TestPlatformHealth_Signal(t *testing.T) {
	cases := []struct {
		rate      int
		warning   int
		critical  int
		wantSignal string
	}{
		{56, 60, 30, "yellow"},
		{61, 60, 30, "green"},
		{60, 60, 30, "green"},
		{29, 60, 30, "red"},
		{30, 60, 30, "yellow"},
		{0, 60, 30, "red"},
	}
	for _, c := range cases {
		monthly := 100
		current := c.rate
		h := &model.PlatformHealth{
			MonthlyCost:       monthly,
			CurrentMonthly:    current,
			WarningThreshold:  c.warning,
			CriticalThreshold: c.critical,
		}
		if got := h.Signal(); got != c.wantSignal {
			t.Errorf("Signal(rate=%d, warn=%d, crit=%d) = %q, want %q",
				c.rate, c.warning, c.critical, got, c.wantSignal)
		}
	}
}
