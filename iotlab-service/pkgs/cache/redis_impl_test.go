package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) (*miniredis.Miniredis, Cache) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{
		Addr:         mr.Addr(),
		MaxRetries:   -1, // no backoff retries so the connection-error cases fail fast
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
	})
	t.Cleanup(func() { _ = client.Close() })

	return mr, NewRedisCache(client)
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	t.Run("missing key returns found=false", func(t *testing.T) {
		_, c := newTestCache(t)

		val, ok := c.Get(ctx, "absent")
		assert.False(t, ok)
		assert.Nil(t, val)
	})

	t.Run("existing key returns value and found=true", func(t *testing.T) {
		_, c := newTestCache(t)
		require.NoError(t, c.Set(ctx, "k", []byte("hello")))

		val, ok := c.Get(ctx, "k")
		require.True(t, ok)
		assert.Equal(t, []byte("hello"), val)
	})

	t.Run("empty value is present, distinguished from missing", func(t *testing.T) {
		_, c := newTestCache(t)
		require.NoError(t, c.Set(ctx, "k", []byte("")))

		val, ok := c.Get(ctx, "k")
		assert.True(t, ok)
		assert.Empty(t, val)
	})

	t.Run("connection error returns found=false", func(t *testing.T) {
		mr, c := newTestCache(t)
		mr.Close() // any error other than redis.Nil must be swallowed as a miss

		val, ok := c.Get(ctx, "k")
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}

func TestSet(t *testing.T) {
	ctx := context.Background()

	t.Run("stores value with no expiry", func(t *testing.T) {
		mr, c := newTestCache(t)
		require.NoError(t, c.Set(ctx, "k", []byte("v")))

		got, err := mr.Get("k")
		require.NoError(t, err)
		assert.Equal(t, "v", got)
		assert.Equal(t, time.Duration(0), mr.TTL("k")) // ttl=0 => persistent
	})

	t.Run("overwrites an existing value", func(t *testing.T) {
		_, c := newTestCache(t)
		require.NoError(t, c.Set(ctx, "k", []byte("first")))
		require.NoError(t, c.Set(ctx, "k", []byte("second")))

		val, ok := c.Get(ctx, "k")
		require.True(t, ok)
		assert.Equal(t, []byte("second"), val)
	})

	t.Run("connection error returns error", func(t *testing.T) {
		mr, c := newTestCache(t)
		mr.Close()

		assert.Error(t, c.Set(ctx, "k", []byte("v")))
	})
}

func TestSetWithTTL(t *testing.T) {
	ctx := context.Background()

	t.Run("value is present before expiry and gone after", func(t *testing.T) {
		mr, c := newTestCache(t)
		require.NoError(t, c.SetWithTTL(ctx, "k", []byte("v"), time.Minute))

		val, ok := c.Get(ctx, "k")
		require.True(t, ok)
		assert.Equal(t, []byte("v"), val)

		mr.FastForward(2 * time.Minute) // TTL elapses

		_, ok = c.Get(ctx, "k")
		assert.False(t, ok)
	})

	t.Run("connection error returns error", func(t *testing.T) {
		mr, c := newTestCache(t)
		mr.Close()

		assert.Error(t, c.SetWithTTL(ctx, "k", []byte("v"), time.Minute))
	})
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("removes an existing key", func(t *testing.T) {
		_, c := newTestCache(t)
		require.NoError(t, c.Set(ctx, "k", []byte("v")))

		require.NoError(t, c.Delete(ctx, "k"))

		_, ok := c.Get(ctx, "k")
		assert.False(t, ok)
	})

	t.Run("deleting a missing key is a no-op", func(t *testing.T) {
		_, c := newTestCache(t)

		assert.NoError(t, c.Delete(ctx, "absent"))
	})
}
