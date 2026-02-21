package model

import "time"

// ContactMessage represents a message submitted via the contact form.
type ContactMessage struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name,omitempty"`
	Message   string    `json:"message"`
	Status    string    `json:"status"` // "unread" | "read"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ContactListOptions carries filter and pagination parameters for listing contact messages.
type ContactListOptions struct {
	// Status filters by message status: "", "all", "unread", "read".
	// Empty string and "all" return all messages.
	Status string
	Limit  int
	Offset int
}
