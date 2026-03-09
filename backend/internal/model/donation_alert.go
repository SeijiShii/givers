package model

import "time"

// DonationAlert represents a flagged suspicious donation pattern.
type DonationAlert struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	DonationID string     `json:"donation_id,omitempty"`
	DonorType  string     `json:"donor_type"`
	DonorID    string     `json:"donor_id"`
	AlertType  string     `json:"alert_type"` // "self_dealing", "high_frequency", "high_amount", "round_tripping"
	Severity   string     `json:"severity"`   // "warning", "critical"
	Status     string     `json:"status"`     // "new", "acknowledged", "resolved"
	Details    string     `json:"details"`    // raw JSON
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy string     `json:"resolved_by,omitempty"`
}

// DonationAlertListResult holds a page of alerts with total count.
type DonationAlertListResult struct {
	Alerts []*DonationAlert `json:"alerts"`
	Total  int              `json:"total"`
}
