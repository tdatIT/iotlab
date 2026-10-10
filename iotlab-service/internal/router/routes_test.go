package router_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/config"
	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicehandler "github.com/tdatIT/iotlab-service/internal/handler/device"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	sensormock "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor/mock"
	telemetrymock "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry/mock"
	"github.com/tdatIT/iotlab-service/internal/router"
	"github.com/tdatIT/iotlab-service/pkgs/probe"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/validate"
)

func newRoutedEcho(t *testing.T, readyProbe *probe.Probe, limit config.RateLimit, deviceRepo *devicemock.MockRepository) *echo.Echo {
	t.Helper()

	e := echo.New()
	e.Validator = validate.GetValidator()
	app := deviceapp.NewDeviceApplication(deviceRepo, sensormock.NewMockRepository(t), telemetrymock.NewMockRepository(t))
	router.RegisterRoutes(e, readyProbe, limit, devicehandler.NewDeviceHandler(app))
	return e
}

func TestRegisterRoutes_platformRoutes(t *testing.T) {
	t.Run("liveness always reports ok", func(t *testing.T) {
		e := newRoutedEcho(t, probe.New(time.Second), config.RateLimit{}, devicemock.NewMockRepository(t))

		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/liveness", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "ok", body["status"])
	})

	t.Run("readiness reflects the registered checkers", func(t *testing.T) {
		p := probe.New(time.Second).
			Register("dep", probe.CheckerFunc(func(context.Context) error { return errors.New("down") }))
		e := newRoutedEcho(t, p, config.RateLimit{}, devicemock.NewMockRepository(t))

		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readiness", nil))

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})

	t.Run("swagger ui is served with the generated spec", func(t *testing.T) {
		e := newRoutedEcho(t, probe.New(time.Second), config.RateLimit{}, devicemock.NewMockRepository(t))

		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Swagger UI")
	})

	t.Run("bare swagger path redirects to the ui", func(t *testing.T) {
		e := newRoutedEcho(t, probe.New(time.Second), config.RateLimit{}, devicemock.NewMockRepository(t))

		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/swagger", nil))

		require.Equal(t, http.StatusMovedPermanently, rec.Code)
		assert.Equal(t, "/swagger/index.html", rec.Header().Get("Location"))
	})

	t.Run("metrics is exposed", func(t *testing.T) {
		e := newRoutedEcho(t, probe.New(time.Second), config.RateLimit{}, devicemock.NewMockRepository(t))

		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestRegisterRoutes_deviceRoutesAreRegistered(t *testing.T) {
	e := newRoutedEcho(t, probe.New(time.Second), config.RateLimit{}, devicemock.NewMockRepository(t))

	registered := make(map[string]struct{})
	for _, r := range e.Router().Routes() {
		registered[r.Method+" "+r.Path] = struct{}{}
	}

	for _, want := range []string{
		"GET /metrics",
		"GET /liveness",
		"GET /readiness",
		"GET /swagger",
		"GET /swagger/*",
		"GET /api/v1/devices",
		"GET /api/v1/devices/:id/telemetry/current",
		"GET /api/v1/devices/:id/telemetry/stats",
		"GET /api/v1/devices/:id/sensor/current",
		"GET /api/v1/devices/:id/sensor/stats",
		"GET /api/v1/devices/:id/sensor/average-24h",
		"GET /api/v1/devices/:id/sensor/stats-24h",
	} {
		assert.Contains(t, registered, want)
	}
}

func TestRegisterRoutes_rateLimit(t *testing.T) {
	limit := config.RateLimit{Enabled: true, Rate: 1, Burst: 1, ExpiresIn: time.Minute}

	t.Run("a route rejects the client once its bucket is empty", func(t *testing.T) {
		deviceRepo := devicemock.NewMockRepository(t)
		deviceRepo.EXPECT().FindAll(mock.Anything).Return([]*models.Device{}, nil).Once()
		e := newRoutedEcho(t, probe.New(time.Second), limit, deviceRepo)

		first := httptest.NewRecorder()
		e.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))
		second := httptest.NewRecorder()
		e.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))

		assert.Equal(t, http.StatusOK, first.Code)
		assert.Equal(t, http.StatusTooManyRequests, second.Code)
	})

	t.Run("each route keeps its own bucket", func(t *testing.T) {
		deviceRepo := devicemock.NewMockRepository(t)
		deviceRepo.EXPECT().FindAll(mock.Anything).Return([]*models.Device{}, nil).Once()
		e := newRoutedEcho(t, probe.New(time.Second), limit, deviceRepo)

		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))
		other := httptest.NewRecorder()
		// An invalid range never reaches the repository; the route only has to get past its own limiter.
		e.ServeHTTP(other, httptest.NewRequest(http.MethodGet, "/api/v1/devices/esp-1/sensor/stats?range=1h", nil))

		assert.NotEqual(t, http.StatusTooManyRequests, other.Code, "another route must not share the exhausted bucket")
	})
}
