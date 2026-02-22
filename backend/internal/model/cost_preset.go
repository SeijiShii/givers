package model

import "time"

// CostPreset represents a user-level cost item template.
type CostPreset struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Label     string    `json:"label"`
	UnitType  string    `json:"unit_type"` // "monthly" | "daily_x_days"
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CostPresetPatch holds fields that can be updated on a preset.
type CostPresetPatch struct {
	Label    *string
	UnitType *string
}

// DefaultCostPresets returns the system default cost items to pre-populate
// when a user has no saved presets.
func DefaultCostPresets() []*CostPreset {
	return []*CostPreset{
		{Label: "サーバー費用", UnitType: "monthly", SortOrder: 0},
		{Label: "開発者費用", UnitType: "daily_x_days", SortOrder: 1},
		{Label: "その他費用", UnitType: "monthly", SortOrder: 2},
	}
}
