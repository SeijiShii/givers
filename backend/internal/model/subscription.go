package model

import "time"

// Subscription represents a recurring donation subscription (management record).
type Subscription struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	DonorType            string    `json:"donor_type"` // "token" or "user"
	DonorID              string    `json:"donor_id"`
	Amount               int       `json:"amount"`
	Currency             string    `json:"currency"`
	Message              string    `json:"message,omitempty"`
	StripeSubscriptionID string    `json:"-"`
	Paused               bool      `json:"paused"`
	NextBillingMessage   string    `json:"next_billing_message,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// SubscriptionPatch holds fields that can be updated on a subscription.
type SubscriptionPatch struct {
	Amount             *int
	Paused             *bool
	NextBillingMessage *string
}
