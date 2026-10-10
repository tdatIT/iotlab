package cache

import (
	"context"
	"time"
)

// NewNoopCache returns a Cache that stores nothing and reports every Get as a miss. Wire it in when Redis is unavailable at
// startup so read-through call sites can still function (they just fall through to the source of truth on every read).
func NewNoopCache() Cache {
	return noopCache{}
}

type noopCache struct{}

func (noopCache) Get(_ context.Context, _ string) ([]byte, bool)  { return nil, false }
func (noopCache) Set(_ context.Context, _ string, _ []byte) error { return nil }
func (noopCache) SetWithTTL(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}
func (noopCache) Delete(_ context.Context, _ string) error { return nil }
