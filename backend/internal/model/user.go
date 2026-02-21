package model

import "time"

type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	GoogleID    string     `json:"-"`
	GitHubID    string     `json:"-"`
	Name        string     `json:"name"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// IsSuspended returns true if the user account is currently suspended.
func (u *User) IsSuspended() bool {
	return u.SuspendedAt != nil
}
