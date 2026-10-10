package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/config"
)

// withFreshPrometheusRegistry swaps in an empty default registerer for the test: newHTTPServer registers collectors there and
// panics on a duplicate.
func withFreshPrometheusRegistry(t *testing.T) {
	t.Helper()
	prev := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = prev })
}

func baseTestConfig() *config.AppConfig {
	return &config.AppConfig{
		Server: config.Server{
			Name:              "test-service",
			CtxDefaultTimeout: time.Second,
		},
		Logger: config.Logger{Level: "error", Format: "json"},
	}
}

func TestNewHTTPServer_defaults_serveRequestsWithARequestID(t *testing.T) {
	withFreshPrometheusRegistry(t)

	e := newHTTPServer(baseTestConfig())
	require.NotNil(t, e)
	assert.NotNil(t, e.Validator)
	assert.NotNil(t, e.HTTPErrorHandler)

	e.GET("/", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get(echo.HeaderXRequestID))
}

func TestNewHTTPServer_panickingHandler_isRecoveredAsA500(t *testing.T) {
	withFreshPrometheusRegistry(t)

	e := newHTTPServer(baseTestConfig())
	e.GET("/boom", func(*echo.Context) error { panic("boom") })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestNewHTTPServer_rateLimitEnabled_rejectsTheBurstOverflow(t *testing.T) {
	withFreshPrometheusRegistry(t)

	cfg := baseTestConfig()
	cfg.RateLimit = config.RateLimit{Enabled: true, Rate: 1, Burst: 1, ExpiresIn: time.Minute}

	e := newHTTPServer(cfg)
	e.GET("/", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	send := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, send())
	assert.Equal(t, http.StatusTooManyRequests, send())
}

func TestNewHTTPServer_consoleLogFormat_isAccepted(t *testing.T) {
	withFreshPrometheusRegistry(t)

	cfg := baseTestConfig()
	cfg.Logger.Format = "console"

	assert.NotNil(t, newHTTPServer(cfg))
}
