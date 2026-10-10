// Package locker provides a distributed lock backed by Redis.
package locker

import (
	"context"
	"time"
)

// Locker is a distributed mutex. Acquire returns the token needed to Release the lock, so a holder can never release someone
// else's lock.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
	Release(ctx context.Context, key string, token string) error
}
