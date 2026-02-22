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
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// DonationPatch holds fields that can be updated on a donation.
type DonationPatch struct {
	Amount *int
	Paused *bool
}

// ActivityItem represents a single entry in a project's activity feed.
type ActivityItem struct {
	DonorName string    `json:"donor_name"`
	Amount    int       `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message,omitempty"`
}
