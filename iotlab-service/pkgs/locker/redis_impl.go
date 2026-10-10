package locker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// releaseScript deletes the key only if it still holds our token. This makes release atomic and prevents a client from deleting a
// lock that another client acquired after this one's TTL expired.
var releaseScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
end
return 0
`)

type redisLocker struct {
	client redis.UniversalClient
}

// NewRedisLocker returns a Locker backed by Redis SET-NX with a Lua release.
func NewRedisLocker(client redis.UniversalClient) Locker {
	return &redisLocker{client: client}
}

// Acquire tries to grab the lock with `SET key <token> NX PX ttl`. It returns ok=false (without error) when the key is already
// held by someone else.
func (l *redisLocker) Acquire(ctx context.Context, key string, ttl time.Duration) (string, bool, error) {
	token := uuid.NewString()

	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	return token, true, nil
}

// Release removes the lock only when `token` matches the current holder. A mismatch (the lock expired or is held by another
// client) is not an error.
func (l *redisLocker) Release(ctx context.Context, key, token string) error {
	return releaseScript.Run(ctx, l.client, []string{key}, token).Err()
}
