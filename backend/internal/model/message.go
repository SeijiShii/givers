package model

import "time"

// MessageThread はホストとユーザー間のメッセージスレッドを表す
type MessageThread struct {
	ID                string    `json:"id"`
	HostUserID        string    `json:"host_user_id"`
	ParticipantUserID string    `json:"participant_user_id"`
	Subject           string    `json:"subject"`
	Active            bool      `json:"active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	// レスポンス用（DB カラムではない）
	UnreadCount     int    `json:"unread_count,omitempty"`
	LastMessageBody string `json:"last_message_body,omitempty"`
	LastMessageAt   string `json:"last_message_at,omitempty"`
	ParticipantName string `json:"participant_name,omitempty"`
	HostName        string `json:"host_name,omitempty"`
}

// Message はスレッド内の個別メッセージを表す
type Message struct {
	ID         string    `json:"id"`
	ThreadID   string    `json:"thread_id"`
	SenderID   string    `json:"sender_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	SenderName string    `json:"sender_name,omitempty"`
}
