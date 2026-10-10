package httpclient

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func TestCaller_circuitBreaker_opensAfterFailures(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL: srv.URL,
		CircuitBreaker: &CBConfig{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			ResetTimeout:     time.Hour,
			// default policy: trip on 5xx
		},
	})

	// two 5xx responses trip the circuit
	_, _ = c.MakeRequest().Get("/")
	_, _ = c.MakeRequest().Get("/")
	require.Equal(t, int32(2), hits.Load())

	// circuit now open — request is rejected without reaching the server
	_, err := c.MakeRequest().Get("/")
	require.Error(t, err)
	require.Equal(t, int32(2), hits.Load())
}

func TestCaller_circuitBreaker_recoversThroughHalfOpen(t *testing.T) {
	var healthy atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if healthy.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL: srv.URL,
		CircuitBreaker: &CBConfig{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			ResetTimeout:     30 * time.Millisecond,
		},
	})

	// one failure opens the circuit
	_, _ = c.MakeRequest().Get("/")
	_, err := c.MakeRequest().Get("/")
	require.Error(t, err) // blocked while open

	// after the reset timeout the circuit goes half-open; server is healthy now
	time.Sleep(45 * time.Millisecond)
	healthy.Store(true)

	resp, err := c.MakeRequest().Get("/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
}

func TestCaller_circuitBreaker_customPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests) // 429 — not caught by the default 5xx policy
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL: srv.URL,
		CircuitBreaker: &CBConfig{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			ResetTimeout:     time.Hour,
			Policies: []resty.CircuitBreakerPolicy{
				func(resp *resty.Response) bool { return resp.StatusCode() == http.StatusTooManyRequests },
			},
		},
	})

	_, _ = c.MakeRequest().Get("/") // 429 trips the custom policy
	_, err := c.MakeRequest().Get("/")
	require.Error(t, err) // circuit open
}

func TestCaller_noCircuitBreaker_behavesNormally(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}) // no CircuitBreaker

	for range 3 {
		resp, err := c.MakeRequest().Get("/")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
	}
	require.Equal(t, int32(3), hits.Load())
}
