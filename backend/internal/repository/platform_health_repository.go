package repository

import (
	"context"

	"github.com/givers/backend/internal/model"
)

// PlatformHealthRepository handles persistence for the platform_health singleton.
type PlatformHealthRepository interface {
	Get(ctx context.Context) (*model.PlatformHealth, error)
}
