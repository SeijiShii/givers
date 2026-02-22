package repository

import "errors"

// ErrNotFound is returned when a requested record does not exist in the database.
var ErrNotFound = errors.New("not found")

// ErrDuplicate is returned when a unique constraint violation occurs (e.g. duplicate stripe_payment_id).
var ErrDuplicate = errors.New("duplicate")
