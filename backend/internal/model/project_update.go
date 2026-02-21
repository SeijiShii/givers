package model

import "time"

// ProjectUpdate はプロジェクト更新情報を表す
type ProjectUpdate struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	AuthorID   string    `json:"-"` // internal, not exposed
	Title      *string   `json:"title,omitempty"`
	Body       string    `json:"body"`
	AuthorName *string   `json:"author_name,omitempty"`
	Visible    bool      `json:"visible"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
