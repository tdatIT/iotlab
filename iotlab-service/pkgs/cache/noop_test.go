package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopCache_storesNothingAndAlwaysMisses(t *testing.T) {
	ctx := context.Background()
	c := NewNoopCache()
	require.NotNil(t, c)

	require.NoError(t, c.Set(ctx, "k", []byte("v")))
	require.NoError(t, c.SetWithTTL(ctx, "k", []byte("v"), time.Minute))

	got, ok := c.Get(ctx, "k")
	assert.False(t, ok, "a noop cache never reports a hit")
	assert.Nil(t, got)

	require.NoError(t, c.Delete(ctx, "k"))
}
