package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	GoogleID  string    `json:"-"`
	GitHubID  string    `json:"-"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
