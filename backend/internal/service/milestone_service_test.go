package service

import (
	"context"
	"errors"
	"testing"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockMilestoneProjectRepo struct {
	getMonthlyTargetFunc func(ctx context.Context, projectID string) (int, error)
}

func (m *mockMilestoneProjectRepo) GetMonthlyTarget(ctx context.Context, projectID string) (int, error) {
	if m.getMonthlyTargetFunc != nil {
		return m.getMonthlyTargetFunc(ctx, projectID)
	}
	return 0, nil
}

type mockMilestoneDonationRepo struct {
	currentMonthSumFunc func(ctx context.Context, projectID string) (int, error)
}

func (m *mockMilestoneDonationRepo) CurrentMonthSumByProject(ctx context.Context, projectID string) (int, error) {
	if m.currentMonthSumFunc != nil {
		return m.currentMonthSumFunc(ctx, projectID)
	}
	return 0, nil
}

type mockMilestoneActivityRepo struct {
	existsFunc func(ctx context.Context, projectID string, rate int) (bool, error)
	insertFunc func(ctx context.Context, a *model.ActivityItem) error
}

func (m *mockMilestoneActivityRepo) ExistsMilestoneThisMonth(ctx context.Context, projectID string, rate int) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, projectID, rate)
	}
	return false, nil
}

func (m *mockMilestoneActivityRepo) Insert(ctx context.Context, a *model.ActivityItem) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, a)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMilestoneService_NotifyDonation_50Percent(t *testing.T) {
	var recorded []*model.ActivityItem
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneDonationRepo{
			currentMonthSumFunc: func(_ context.Context, _ string) (int, error) { return 5000, nil },
		},
		&mockMilestoneActivityRepo{
			existsFunc: func(_ context.Context, _ string, _ int) (bool, error) { return false, nil },
			insertFunc: func(_ context.Context, a *model.ActivityItem) error {
				recorded = append(recorded, a)
				return nil
			},
		},
	)

	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recorded) != 1 {
		t.Fatalf("expected 1 milestone, got %d", len(recorded))
	}
	if recorded[0].Type != "milestone" {
		t.Errorf("expected type=milestone, got %q", recorded[0].Type)
	}
	if recorded[0].Rate == nil || *recorded[0].Rate != 50 {
		t.Errorf("expected rate=50, got %v", recorded[0].Rate)
	}
	if recorded[0].ProjectID != "proj-1" {
		t.Errorf("expected ProjectID=proj-1, got %q", recorded[0].ProjectID)
	}
}

func TestMilestoneService_NotifyDonation_100Percent(t *testing.T) {
	var recorded []*model.ActivityItem
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneDonationRepo{
			currentMonthSumFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneActivityRepo{
			existsFunc: func(_ context.Context, _ string, _ int) (bool, error) { return false, nil },
			insertFunc: func(_ context.Context, a *model.ActivityItem) error {
				recorded = append(recorded, a)
				return nil
			},
		},
	)

	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Both 50% and 100% should fire
	if len(recorded) != 2 {
		t.Fatalf("expected 2 milestones (50%% and 100%%), got %d", len(recorded))
	}
	// 100% first (checked first as higher threshold), then 50%
	rates := make([]int, len(recorded))
	for i, r := range recorded {
		if r.Rate != nil {
			rates[i] = *r.Rate
		}
	}
	if rates[0] != 100 || rates[1] != 50 {
		t.Errorf("expected rates [100, 50], got %v", rates)
	}
}

func TestMilestoneService_NotifyDonation_AlreadyRecorded_NoDuplicate(t *testing.T) {
	var recorded []*model.ActivityItem
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneDonationRepo{
			currentMonthSumFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneActivityRepo{
			existsFunc: func(_ context.Context, _ string, rate int) (bool, error) {
				// Both 50% and 100% already recorded
				return true, nil
			},
			insertFunc: func(_ context.Context, a *model.ActivityItem) error {
				recorded = append(recorded, a)
				return nil
			},
		},
	)

	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recorded) != 0 {
		t.Errorf("expected no milestones (already recorded), got %d", len(recorded))
	}
}

func TestMilestoneService_NotifyDonation_ZeroTarget_Skip(t *testing.T) {
	var recorded []*model.ActivityItem
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) { return 0, nil },
		},
		&mockMilestoneDonationRepo{},
		&mockMilestoneActivityRepo{
			insertFunc: func(_ context.Context, a *model.ActivityItem) error {
				recorded = append(recorded, a)
				return nil
			},
		},
	)

	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recorded) != 0 {
		t.Errorf("expected no milestones for zero target, got %d", len(recorded))
	}
}

func TestMilestoneService_NotifyDonation_Below50Percent_NoMilestone(t *testing.T) {
	var recorded []*model.ActivityItem
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneDonationRepo{
			currentMonthSumFunc: func(_ context.Context, _ string) (int, error) { return 4999, nil },
		},
		&mockMilestoneActivityRepo{
			insertFunc: func(_ context.Context, a *model.ActivityItem) error {
				recorded = append(recorded, a)
				return nil
			},
		},
	)

	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recorded) != 0 {
		t.Errorf("expected no milestones below 50%%, got %d", len(recorded))
	}
}

func TestMilestoneService_NotifyDonation_ProjectRepoError_NoError(t *testing.T) {
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) {
				return 0, errors.New("db error")
			},
		},
		&mockMilestoneDonationRepo{},
		&mockMilestoneActivityRepo{},
	)

	// Should not propagate error
	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestMilestoneService_NotifyDonation_100Already50Not_OnlyInserts50(t *testing.T) {
	var recorded []*model.ActivityItem
	svc := NewMilestoneService(
		&mockMilestoneProjectRepo{
			getMonthlyTargetFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneDonationRepo{
			currentMonthSumFunc: func(_ context.Context, _ string) (int, error) { return 10000, nil },
		},
		&mockMilestoneActivityRepo{
			existsFunc: func(_ context.Context, _ string, rate int) (bool, error) {
				// 100% already recorded, 50% not
				return rate == 100, nil
			},
			insertFunc: func(_ context.Context, a *model.ActivityItem) error {
				recorded = append(recorded, a)
				return nil
			},
		},
	)

	if err := svc.NotifyDonation(context.Background(), "proj-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recorded) != 1 {
		t.Fatalf("expected 1 milestone (50%% only), got %d", len(recorded))
	}
	if recorded[0].Rate == nil || *recorded[0].Rate != 50 {
		t.Errorf("expected rate=50, got %v", recorded[0].Rate)
	}
}
