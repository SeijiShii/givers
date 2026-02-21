package model

import "time"

// PlatformHealth represents the singleton row tracking the platform's financial health.
type PlatformHealth struct {
	MonthlyCost       int       `json:"monthly_cost"`
	CurrentMonthly    int       `json:"current_monthly"`
	WarningThreshold  int       `json:"warning_threshold"`
	CriticalThreshold int       `json:"critical_threshold"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Rate returns the achievement rate as an integer percentage (0-100+).
func (p *PlatformHealth) Rate() int {
	if p.MonthlyCost == 0 {
		return 0
	}
	return p.CurrentMonthly * 100 / p.MonthlyCost
}

// Signal returns "green", "yellow", or "red" based on thresholds.
func (p *PlatformHealth) Signal() string {
	rate := p.Rate()
	if rate >= p.WarningThreshold {
		return "green"
	}
	if rate >= p.CriticalThreshold {
		return "yellow"
	}
	return "red"
}
