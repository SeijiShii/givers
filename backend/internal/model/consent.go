package model

import "time"

// UserConsent はユーザーの利用規約同意記録を表す
type UserConsent struct {
	UserID       string    `json:"user_id"`
	TermsVersion string    `json:"terms_version"`
	AgreedAt     time.Time `json:"agreed_at"`
}
