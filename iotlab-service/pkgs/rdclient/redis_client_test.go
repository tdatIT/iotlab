package rdclient_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/pkgs/rdclient"
)

func newMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return mr
}

func TestNewRedisClient_standalone_connectsAndPings(t *testing.T) {
	mr := newMiniredis(t)
	cfg := rdclient.Config{
		Mode:     "standalone",
		Address:  []string{mr.Addr(), "127.0.0.1:1"}, // only the primary address is used
		PoolSize: 2,
	}

	rc, err := rdclient.NewRedisClient(cfg)

	require.NoError(t, err)
	require.NotNil(t, rc.Client())
	require.NoError(t, rc.Client().Ping(context.Background()).Err())
	assert.NoError(t, rc.Close())
}

func TestNewRedisClient_clusterMode_usesEveryAddress(t *testing.T) {
	mr := newMiniredis(t)
	cfg := rdclient.Config{Mode: "cluster", Address: []string{mr.Addr()}}

	rc, err := rdclient.NewRedisClient(cfg)

	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })
	assert.NotNil(t, rc.Client())
}

func TestNewRedisClient_emptyAddressList_fails(t *testing.T) {
	rc, err := rdclient.NewRedisClient(rdclient.Config{})

	require.Error(t, err)
	assert.Nil(t, rc)
	assert.Contains(t, err.Error(), "redis address list is empty")
}

func TestNewRedisClient_sentinelWithoutMasterName_fails(t *testing.T) {
	mr := newMiniredis(t)
	cfg := rdclient.Config{Mode: "sentinel", Address: []string{mr.Addr()}}

	rc, err := rdclient.NewRedisClient(cfg)

	require.Error(t, err)
	assert.Nil(t, rc)
	assert.Contains(t, err.Error(), "master name")
}

func TestNewRedisClient_unreachableServer_failsOnPing(t *testing.T) {
	cfg := rdclient.Config{Address: []string{"127.0.0.1:1"}}

	rc, err := rdclient.NewRedisClient(cfg)

	require.Error(t, err)
	assert.Nil(t, rc)
	assert.Contains(t, err.Error(), "ping failed")
}
