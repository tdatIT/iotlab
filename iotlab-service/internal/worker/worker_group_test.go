package worker_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/worker"
)

// fnWorker adapts a function to worker.Worker.
type fnWorker func(ctx context.Context) error

func (f fnWorker) Start(ctx context.Context) error { return f(ctx) }

func TestGroup_StartGroup_noWorkers_returnsImmediately(t *testing.T) {
	require.NoError(t, worker.NewGroup().StartGroup(context.Background()))
}

func TestGroup_StartGroup_everyRegisteredWorkerRuns(t *testing.T) {
	var started atomic.Int32

	g := worker.NewGroup()
	g.Register(
		fnWorker(func(context.Context) error { started.Add(1); return nil }),
		fnWorker(func(context.Context) error { started.Add(1); return nil }),
	)
	g.Register(fnWorker(func(context.Context) error { started.Add(1); return nil }))

	require.NoError(t, g.StartGroup(context.Background()))
	assert.Equal(t, int32(3), started.Load())
}

func TestGroup_StartGroup_workerError_cancelsTheOthersAndIsWrapped(t *testing.T) {
	boom := errors.New("boom")
	sibling := make(chan struct{})

	g := worker.NewGroup()
	g.Register(
		fnWorker(func(context.Context) error { return boom }),
		fnWorker(func(ctx context.Context) error {
			<-ctx.Done() // released only because the failing sibling cancelled the group
			close(sibling)
			return ctx.Err()
		}),
	)

	err := g.StartGroup(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "worker group:")
	select {
	case <-sibling:
	case <-time.After(time.Second):
		t.Fatal("sibling worker was not cancelled")
	}
}

func TestGroup_StartGroup_parentCancel_stopsTheGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	g := worker.NewGroup()
	g.Register(fnWorker(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}))

	cancel()
	err := g.StartGroup(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}
