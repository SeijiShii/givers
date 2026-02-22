package model

import "time"

// ProjectCostItem represents one line in a project's cost breakdown.
type ProjectCostItem struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	Label         string    `json:"label"`
	UnitType      string    `json:"unit_type"` // "monthly" | "daily_x_days"
	AmountMonthly int       `json:"amount_monthly"`
	RatePerDay    int       `json:"rate_per_day"`
	DaysPerMonth  int       `json:"days_per_month"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// MonthlyAmount returns the monthly cost in JPY for this line item.
func (c *ProjectCostItem) MonthlyAmount() int {
	switch c.UnitType {
	case "daily_x_days":
		return c.RatePerDay * c.DaysPerMonth
	default: // "monthly"
		return c.AmountMonthly
	}
}

// TotalMonthlyAmount sums the monthly amounts of all items.
func TotalMonthlyAmount(items []*ProjectCostItem) int {
	total := 0
	for _, item := range items {
		total += item.MonthlyAmount()
	}
	return total
}

// ProjectCostItemInput is the request payload for one cost item line.
type ProjectCostItemInput struct {
	Label         string `json:"label"`
	UnitType      string `json:"unit_type"`
	AmountMonthly int    `json:"amount_monthly"`
	RatePerDay    int    `json:"rate_per_day"`
	DaysPerMonth  int    `json:"days_per_month"`
}
