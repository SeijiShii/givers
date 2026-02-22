package model

import "time"

type Project struct {
	ID               string     `json:"id"`
	OwnerID          string     `json:"owner_id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Deadline         *time.Time `json:"deadline,omitempty"`
	Status           string     `json:"status"`
	OwnerWantMonthly *int       `json:"owner_want_monthly,omitempty"` // オーナーの「〇〇円欲しい」表明（月額）
	MonthlyTarget    int        `json:"monthly_target"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	CostItems []*ProjectCostItem `json:"cost_items,omitempty"`
	Alerts    *ProjectAlerts     `json:"alerts,omitempty"`
}

type ProjectAlerts struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	WarningThreshold  int       `json:"warning_threshold"`
	CriticalThreshold int       `json:"critical_threshold"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
