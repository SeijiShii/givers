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
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	Costs  *ProjectCosts  `json:"costs,omitempty"`
	Alerts *ProjectAlerts `json:"alerts,omitempty"`
}

type ProjectCosts struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"project_id"`
	ServerCostMonthly int      `json:"server_cost_monthly"`
	DevCostPerDay    int      `json:"dev_cost_per_day"`
	DevDaysPerMonth  int      `json:"dev_days_per_month"`
	OtherCostMonthly int      `json:"other_cost_monthly"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (c *ProjectCosts) MonthlyTarget() int {
	return c.ServerCostMonthly + (c.DevCostPerDay * c.DevDaysPerMonth) + c.OtherCostMonthly
}

type ProjectAlerts struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"project_id"`
	WarningThreshold  int       `json:"warning_threshold"`
	CriticalThreshold int       `json:"critical_threshold"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
