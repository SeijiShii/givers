package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/givers/backend/internal/model"
)

// ---------------------------------------------------------------------------
// Minimal interfaces (only what DonationAlertService needs)
// ---------------------------------------------------------------------------

// AlertProjectRepo provides project owner lookups.
type AlertProjectRepo interface {
	GetOwnerID(ctx context.Context, projectID string) (string, error)
}

// AlertDetectionRepo provides donation pattern queries and alert persistence.
type AlertDetectionRepo interface {
	Create(ctx context.Context, a *model.DonationAlert) error
	List(ctx context.Context, status string, limit, offset int) ([]*model.DonationAlert, int, error)
	UpdateStatus(ctx context.Context, id, status, resolvedBy string) error
	CountRecentByDonor(ctx context.Context, projectID, donorType, donorID string, windowMinutes int) (int, error)
	DailyTotalByDonor(ctx context.Context, projectID, donorType, donorID string) (int, error)
	RefundCountByDonor(ctx context.Context, projectID, donorType, donorID string, windowDays int) (int, error)
	HasRecentAlert(ctx context.Context, projectID, donorType, donorID, alertType string, windowHours int) (bool, error)
}

// StripeAlertDetector is the interface injected into StripeServiceImpl.
type StripeAlertDetector interface {
	CheckDonation(ctx context.Context, projectID, donorType, donorID string, amount int, donationID string) error
}

// ---------------------------------------------------------------------------
// Thresholds (package-level vars for testability / future configurability)
// ---------------------------------------------------------------------------

var (
	// HighFrequencyWindow is the time window in minutes for frequency checks.
	HighFrequencyWindow = 60
	// HighFrequencyWarn triggers a warning alert.
	HighFrequencyWarn = 3
	// HighFrequencyCrit triggers a critical alert.
	HighFrequencyCrit = 5

	// HighAmountWarn is the single-donation amount (JPY) for a warning.
	HighAmountWarn = 100_000
	// HighAmountCrit is the single-donation amount (JPY) for a critical alert.
	HighAmountCrit = 500_000
	// HighAmountDailyWarn is the daily total (JPY) for a warning.
	HighAmountDailyWarn = 300_000

	// RoundTripWindow is the time window in days for refund pattern checks.
	RoundTripWindow = 30
	// RoundTripWarn triggers a warning alert.
	RoundTripWarn = 3
	// RoundTripCrit triggers a critical alert.
	RoundTripCrit = 5
)

// ---------------------------------------------------------------------------
// DonationAlertService
// ---------------------------------------------------------------------------

// DonationAlertService detects suspicious donation patterns and manages alerts.
type DonationAlertService struct {
	projectRepo AlertProjectRepo
	alertRepo   AlertDetectionRepo
}

// NewDonationAlertService creates a new DonationAlertService.
func NewDonationAlertService(pr AlertProjectRepo, ar AlertDetectionRepo) *DonationAlertService {
	return &DonationAlertService{projectRepo: pr, alertRepo: ar}
}

// CheckDonation runs detection checks after a donation is confirmed.
// Errors are logged but never returned — fire-and-forget to avoid breaking the donation flow.
func (s *DonationAlertService) CheckDonation(ctx context.Context, projectID, donorType, donorID string, amount int, donationID string) error {
	s.checkSelfDealing(ctx, projectID, donorType, donorID, amount, donationID)
	s.checkHighFrequency(ctx, projectID, donorType, donorID, amount, donationID)
	s.checkHighAmount(ctx, projectID, donorType, donorID, amount, donationID)
	s.checkRoundTripping(ctx, projectID, donorType, donorID, amount, donationID)
	return nil
}

// List returns alerts for admin viewing.
func (s *DonationAlertService) List(ctx context.Context, status string, limit, offset int) ([]*model.DonationAlert, int, error) {
	return s.alertRepo.List(ctx, status, limit, offset)
}

// UpdateStatus changes the status of an alert.
func (s *DonationAlertService) UpdateStatus(ctx context.Context, id, status, resolvedBy string) error {
	return s.alertRepo.UpdateStatus(ctx, id, status, resolvedBy)
}

// ---------------------------------------------------------------------------
// Detection checks
// ---------------------------------------------------------------------------

func (s *DonationAlertService) checkSelfDealing(ctx context.Context, projectID, donorType, donorID string, amount int, donationID string) {
	if donorType != "user" {
		return
	}
	ownerID, err := s.projectRepo.GetOwnerID(ctx, projectID)
	if err != nil {
		slog.Warn("alert: get owner failed", "project_id", projectID, "error", err)
		return
	}
	if donorID != ownerID {
		return
	}
	s.createAlert(ctx, projectID, donorType, donorID, donationID, "self_dealing", "critical", 24,
		map[string]any{"owner_id": ownerID, "amount": amount})
}

func (s *DonationAlertService) checkHighFrequency(ctx context.Context, projectID, donorType, donorID string, amount int, donationID string) {
	count, err := s.alertRepo.CountRecentByDonor(ctx, projectID, donorType, donorID, HighFrequencyWindow)
	if err != nil {
		slog.Warn("alert: count recent failed", "project_id", projectID, "error", err)
		return
	}

	var severity string
	switch {
	case count >= HighFrequencyCrit:
		severity = "critical"
	case count >= HighFrequencyWarn:
		severity = "warning"
	default:
		return
	}

	s.createAlert(ctx, projectID, donorType, donorID, donationID, "high_frequency", severity, 1,
		map[string]any{"count": count, "window_minutes": HighFrequencyWindow, "threshold": HighFrequencyWarn})
}

func (s *DonationAlertService) checkHighAmount(ctx context.Context, projectID, donorType, donorID string, amount int, donationID string) {
	// Single donation check
	if amount >= HighAmountCrit {
		s.createAlert(ctx, projectID, donorType, donorID, donationID, "high_amount", "critical", 24,
			map[string]any{"amount": amount, "threshold": HighAmountCrit})
	} else if amount >= HighAmountWarn {
		s.createAlert(ctx, projectID, donorType, donorID, donationID, "high_amount", "warning", 24,
			map[string]any{"amount": amount, "threshold": HighAmountWarn})
	}

	// Daily total check
	dailyTotal, err := s.alertRepo.DailyTotalByDonor(ctx, projectID, donorType, donorID)
	if err != nil {
		slog.Warn("alert: daily total failed", "project_id", projectID, "error", err)
		return
	}
	if dailyTotal >= HighAmountDailyWarn {
		s.createAlert(ctx, projectID, donorType, donorID, donationID, "high_amount", "warning", 24,
			map[string]any{"daily_total": dailyTotal, "threshold": HighAmountDailyWarn})
	}
}

func (s *DonationAlertService) checkRoundTripping(ctx context.Context, projectID, donorType, donorID string, amount int, donationID string) {
	refundCount, err := s.alertRepo.RefundCountByDonor(ctx, projectID, donorType, donorID, RoundTripWindow)
	if err != nil {
		slog.Warn("alert: refund count failed", "project_id", projectID, "error", err)
		return
	}

	var severity string
	switch {
	case refundCount >= RoundTripCrit:
		severity = "critical"
	case refundCount >= RoundTripWarn:
		severity = "warning"
	default:
		return
	}

	s.createAlert(ctx, projectID, donorType, donorID, donationID, "round_tripping", severity, 24,
		map[string]any{"refund_count": refundCount, "window_days": RoundTripWindow, "threshold": RoundTripWarn})
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func (s *DonationAlertService) createAlert(ctx context.Context, projectID, donorType, donorID, donationID, alertType, severity string, dedupHours int, detailsMap map[string]any) {
	// Deduplication check
	exists, err := s.alertRepo.HasRecentAlert(ctx, projectID, donorType, donorID, alertType, dedupHours)
	if err != nil {
		slog.Warn("alert: dedup check failed", "project_id", projectID, "alert_type", alertType, "error", err)
		return
	}
	if exists {
		return
	}

	detailsJSON, _ := json.Marshal(detailsMap)

	if err := s.alertRepo.Create(ctx, &model.DonationAlert{
		ProjectID:  projectID,
		DonationID: donationID,
		DonorType:  donorType,
		DonorID:    donorID,
		AlertType:  alertType,
		Severity:   severity,
		Details:    string(detailsJSON),
	}); err != nil {
		slog.Warn("alert: create failed", "project_id", projectID, "alert_type", alertType, "error", err)
	}
}
