package service

import (
	"context"

	"github.com/givers/backend/internal/model"
	"github.com/givers/backend/internal/repository"
)

// CurrentTermsVersion は現在の利用規約バージョン。
// 利用規約を更新した場合はこの値を変更する。次回のプロジェクト作成時に再同意が求められる。
const CurrentTermsVersion = "2026-03-04"

// ConsentService はユーザーの利用規約同意に関するビジネスロジックのインターフェース
type ConsentService interface {
	// RecordConsent は現在の利用規約バージョンへの同意を記録する
	RecordConsent(ctx context.Context, userID string) error
	// HasCurrentConsent はユーザーが現在の利用規約バージョンに同意しているかを返す
	HasCurrentConsent(ctx context.Context, userID string) (bool, error)
}

type consentService struct {
	repo repository.ConsentRepository
}

// NewConsentService は ConsentService を生成する
func NewConsentService(repo repository.ConsentRepository) ConsentService {
	return &consentService{repo: repo}
}

func (s *consentService) RecordConsent(ctx context.Context, userID string) error {
	return s.repo.Upsert(ctx, &model.UserConsent{
		UserID:       userID,
		TermsVersion: CurrentTermsVersion,
	})
}

func (s *consentService) HasCurrentConsent(ctx context.Context, userID string) (bool, error) {
	c, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if c == nil {
		return false, nil
	}
	return c.TermsVersion == CurrentTermsVersion, nil
}
