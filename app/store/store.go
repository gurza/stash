// Package store provides key-value storage implementations.
package store

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a key is not found in the store.
var ErrNotFound = errors.New("key not found")

// KeyInfo holds metadata about a stored key.
type KeyInfo struct {
	Key       string    `db:"key"`
	Size      int       `db:"size"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
