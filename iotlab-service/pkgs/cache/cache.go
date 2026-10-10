// Package cache provides the key/value cache abstraction and its Redis and no-op implementations.
package cache

import (
	"context"
	"time"
)

// Cache is the key/value contract used by read-through call sites. Get reports a miss with ok=false rather than an error.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte) error
	SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
