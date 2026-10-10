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

func TestNew_returnsNonNilCaller(t *testing.T) {
	c := New(Config{BaseURL: "http://localhost"})
	require.NotNil(t, c)
}

func TestMakeRequest_returnsNonNilRequest(t *testing.T) {
	c := New(Config{BaseURL: "http://localhost"})
	require.NotNil(t, c.MakeRequest())
}

func TestNew_timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL: srv.URL,
		Timeout: 50 * time.Millisecond,
	})

	_, err := c.MakeRequest().Get("/")
	require.Error(t, err)
}

func TestNew_retryCondition_retriesUntilSuccess(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL:    srv.URL,
		RetryCount: 3,
		RetryWait:  10 * time.Millisecond,
		RetryCondition: func(r *resty.Response, _ error) bool {
			return r != nil && r.StatusCode() == http.StatusInternalServerError
		},
	})

	resp, err := c.MakeRequest().Get("/")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode())
	require.Equal(t, int32(3), hits.Load()) // 2 failures + 1 success
}

func TestNew_noRetryCondition_doesNotRetryOnHTTPError(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL:    srv.URL,
		RetryCount: 3,
		RetryWait:  10 * time.Millisecond,
	})

	resp, err := c.MakeRequest().Get("/")
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode())
	require.Equal(t, int32(1), hits.Load())
}

func TestNew_debug_doesNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL: srv.URL,
		Debug:   true,
	})

	require.NotPanics(t, func() {
		_, _ = c.MakeRequest().Get("/")
	})
}

func TestGetClient_returnsTheUnderlyingRestyClient(t *testing.T) {
	c := New(Config{BaseURL: "http://example.com", Timeout: time.Second})

	client := c.GetClient()

	require.NotNil(t, client)
	require.Same(t, client, c.GetClient(), "the same client is reused across calls")
}
