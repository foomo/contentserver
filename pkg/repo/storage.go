package repo

import (
	"context"
)

// Storage defines the contract for snapshot persistence backends.
// Implementations must be safe for concurrent use.
type Storage interface {
	// Write stores data with the given key.
	Write(ctx context.Context, key string, data []byte) error

	// Read retrieves data for the given key.
	// Returns os.ErrNotExist if the key does not exist.
	Read(ctx context.Context, key string) ([]byte, error)

	// List returns keys matching the given prefix, sorted alphabetically descending (newest first).
	List(ctx context.Context, prefix string) ([]string, error)

	// Delete removes the data for the given key.
	// Returns nil if the key does not exist (idempotent).
	Delete(ctx context.Context, key string) error

	// Close releases any resources held by the storage backend.
	Close() error
}
