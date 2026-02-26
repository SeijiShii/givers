package service

import (
	"context"
	"log/slog"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Minimal interfaces (only what MilestoneService needs)
// ---------------------------------------------------------------------------

type MilestoneProjectRepo interface {
	GetMonthlyTarget(ctx context.Context, projectID string) (int, error)
}

type MilestoneDonationRepo interface {
	CurrentMonthSumByProject(ctx context.Context, projectID string) (int, error)
}

type MilestoneActivityRepo interface {
	ExistsMilestoneThisMonth(ctx context.Context, projectID string, rate int) (bool, error)
	Insert(ctx context.Context, a *model.ActivityItem) error
}

// ---------------------------------------------------------------------------
// MilestoneService
// ---------------------------------------------------------------------------

var milestoneThresholds = []int{100, 50} // checked high→low

type MilestoneService struct {
	projectRepo  MilestoneProjectRepo
	donationRepo MilestoneDonationRepo
	activityRepo MilestoneActivityRepo
}

func NewMilestoneService(
	pr MilestoneProjectRepo,
	dr MilestoneDonationRepo,
	ar MilestoneActivityRepo,
) *MilestoneService {
	return &MilestoneService{projectRepo: pr, donationRepo: dr, activityRepo: ar}
}

// NotifyDonation checks milestone thresholds and inserts activity records.
// Errors are swallowed (fire-and-forget) so they never break the donation flow.
func (s *MilestoneService) NotifyDonation(ctx context.Context, projectID string) error {
	target, err := s.projectRepo.GetMonthlyTarget(ctx, projectID)
	if err != nil {
		slog.Warn("milestone: get monthly target failed", "project_id", projectID, "error", err)
		return nil
	}
	if target <= 0 {
		return nil
	}

	sum, err := s.donationRepo.CurrentMonthSumByProject(ctx, projectID)
	if err != nil {
		slog.Warn("milestone: get month sum failed", "project_id", projectID, "error", err)
		return nil
	}

	rate := sum * 100 / target

	for _, threshold := range milestoneThresholds {
		if rate < threshold {
			continue
		}
		exists, err := s.activityRepo.ExistsMilestoneThisMonth(ctx, projectID, threshold)
		if err != nil {
			slog.Warn("milestone: exists check failed", "project_id", projectID, "threshold", threshold, "error", err)
			continue
		}
		if exists {
			continue
		}
		t := threshold
		if err := s.activityRepo.Insert(ctx, &model.ActivityItem{
			Type:      "milestone",
			ProjectID: projectID,
			Rate:      &t,
		}); err != nil {
			slog.Warn("milestone: insert failed", "project_id", projectID, "threshold", threshold, "error", err)
		}
	}
	return nil
}
