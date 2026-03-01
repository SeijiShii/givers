package model

import "time"

// Donation represents a single or recurring donation to a project.
type Donation struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	DonorType            string    `json:"donor_type"` // "token" or "user"
	DonorID              string    `json:"donor_id"`
	Amount               int       `json:"amount"`
	Currency             string    `json:"currency"`
	Message              string    `json:"message,omitempty"`
	IsRecurring          bool      `json:"is_recurring"`
	StripePaymentID      string    `json:"-"`
	StripeSubscriptionID string    `json:"-"`
	Paused               bool      `json:"paused"`
	NextBillingMessage   string    `json:"next_billing_message,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// DonationPatch holds fields that can be updated on a donation.
type DonationPatch struct {
	Amount             *int
	Paused             *bool
	NextBillingMessage *string
}

// MonthlySum represents the total donation amount for a single month.
type MonthlySum struct {
	Month  string `json:"month"` // "2026-01"
	Amount int    `json:"amount"`
}

// ChartDataPoint represents one data point for the project chart.
type ChartDataPoint struct {
	Month        string `json:"month"`
	MinAmount    int    `json:"minAmount"`
	TargetAmount int    `json:"targetAmount"`
	ActualAmount int    `json:"actualAmount"`
}

// DonationMessage represents a donation message for project owner viewing.
type DonationMessage struct {
	DonorName   string    `json:"donor_name"`
	Amount      int       `json:"amount"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
	IsRecurring bool      `json:"is_recurring"`
}

// DonationMessageResult holds a page of donation messages with total count.
type DonationMessageResult struct {
	Messages []*DonationMessage `json:"messages"`
	Total    int                `json:"total"`
}

// ActivityItem represents a single entry in the activity feed.
type ActivityItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "donation", "project_created", "project_updated", "milestone"
	ProjectID   string    `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ActorName   *string   `json:"actor_name"`
	Amount      *int      `json:"amount,omitempty"`
	Rate        *int      `json:"rate,omitempty"`
	Message     string    `json:"message,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
