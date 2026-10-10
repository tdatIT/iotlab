package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"

	"github.com/tdatIT/iotlab-service/pkgs/tracing"
)

func TestInit_setsTheGlobalProviderAndPropagatorAndReturnsAShutdown(t *testing.T) {
	ctx := context.Background()

	shutdown, err := tracing.Init(ctx, tracing.Config{
		Endpoint:       "127.0.0.1:4317",
		Insecure:       true,
		SampleRatio:    1,
		ServiceName:    "test-service",
		ServiceVersion: "v0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, shutdown)
	// Bound the flush: no collector is listening on the endpoint, so an unbounded shutdown would sit in the exporter's retry loop.
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		_ = shutdown(stopCtx)
	})

	// The global propagator must round-trip a W3C traceparent for the HTTP middleware to pick up incoming trace context.
	carrier := map[string]string{}
	otel.GetTextMapPropagator().Inject(ctx, mapCarrier(carrier))
	assert.Contains(t, otel.GetTextMapPropagator().Fields(), "traceparent")

	_, span := otel.Tracer("test").Start(ctx, "span")
	assert.True(t, span.SpanContext().IsValid())
	span.End()
}

func TestInit_secureEndpoint_isAccepted(t *testing.T) {
	shutdown, err := tracing.Init(context.Background(), tracing.Config{
		Endpoint:    "127.0.0.1:4317",
		SampleRatio: 0,
		ServiceName: "test-service",
	})

	require.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	require.NoError(t, shutdown(stopCtx))
}

// mapCarrier is the minimal TextMapCarrier needed to exercise Inject.
type mapCarrier map[string]string

func (c mapCarrier) Get(key string) string { return c[key] }
func (c mapCarrier) Set(key, val string)   { c[key] = val }
func (c mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
