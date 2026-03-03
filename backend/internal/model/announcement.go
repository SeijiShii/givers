package model

import "time"

// Announcement はホストからユーザーへのお知らせを表す
type Announcement struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Severity    string    `json:"severity"`    // "info", "warn", "error"
	Visible     bool      `json:"visible"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// ユーザー向けレスポンス用（DB カラムではない）
	IsRead *bool `json:"is_read,omitempty"`
}
