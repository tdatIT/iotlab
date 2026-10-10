package locker

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newTestLocker(t *testing.T) (*miniredis.Miniredis, Locker) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return mr, NewRedisLocker(client)
}

func TestAcquire_freeKey_succeeds(t *testing.T) {
	_, l := newTestLocker(t)

	token, ok, err := l.Acquire(context.Background(), "k", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEmpty(t, token)
}

func TestAcquire_heldKey_fails(t *testing.T) {
	_, l := newTestLocker(t)
	ctx := context.Background()

	_, ok, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	// second caller cannot acquire the same key
	token2, ok2, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.False(t, ok2)
	require.Empty(t, token2)
}

func TestRelease_correctToken_freesLock(t *testing.T) {
	_, l := newTestLocker(t)
	ctx := context.Background()

	token, _, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)

	require.NoError(t, l.Release(ctx, "k", token))

	// lock is free again
	_, ok, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestRelease_wrongToken_keepsLock(t *testing.T) {
	_, l := newTestLocker(t)
	ctx := context.Background()

	_, _, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)

	// releasing with a token we don't own is a no-op (no error)
	require.NoError(t, l.Release(ctx, "k", "not-my-token"))

	// lock is still held
	_, ok, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestAcquire_ttlExpiry_allowsReacquire(t *testing.T) {
	mr, l := newTestLocker(t)
	ctx := context.Background()

	_, ok, err := l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	mr.FastForward(2 * time.Minute) // lock TTL elapses

	_, ok, err = l.Acquire(ctx, "k", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestAcquire_redisUnavailable_returnsTheError(t *testing.T) {
	mr, l := newTestLocker(t)
	mr.Close()

	token, ok, err := l.Acquire(context.Background(), "k", time.Minute)

	require.Error(t, err)
	require.False(t, ok)
	require.Empty(t, token)
}
