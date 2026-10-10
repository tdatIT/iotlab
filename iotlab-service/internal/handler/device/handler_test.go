package device_test

import (
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

	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicehandler "github.com/tdatIT/iotlab-service/internal/handler/device"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	sensormock "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor/mock"
	telemetrymock "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry/mock"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/validate"
)

type repos struct {
	device    *devicemock.MockRepository
	sensor    *sensormock.MockRepository
	telemetry *telemetrymock.MockRepository
}

// newServer mounts the handler on the same paths as the router, with the production validator and error handler.
func newServer(t *testing.T) (*echo.Echo, repos) {
	t.Helper()

	r := repos{
		device:    devicemock.NewMockRepository(t),
		sensor:    sensormock.NewMockRepository(t),
		telemetry: telemetrymock.NewMockRepository(t),
	}
	h := devicehandler.NewDeviceHandler(deviceapp.NewDeviceApplication(r.device, r.sensor, r.telemetry))

	e := echo.New()
	e.Validator = validate.GetValidator()
	e.HTTPErrorHandler = svcerr.ErrorHandlerEchoFn
	e.GET("/devices", h.ListDevices)
	e.GET("/devices/:id/telemetry/current", h.GetCurrentTelemetry)
	e.GET("/devices/:id/telemetry/stats", h.GetTelemetryStats)
	e.GET("/devices/:id/sensor/current", h.GetCurrentSensor)
	e.GET("/devices/:id/sensor/stats", h.GetSensorStats)
	e.GET("/devices/:id/sensor/average-24h", h.GetSensorAverage24h)
	e.GET("/devices/:id/sensor/stats-24h", h.GetSensorStats24h)
	return e, r
}

func get(e *echo.Echo, path string) (int, map[string]any) {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func TestHandler_ListDevices(t *testing.T) {
	t.Run("success wraps the devices in the envelope", func(t *testing.T) {
		e, r := newServer(t)
		r.device.EXPECT().FindAll(mock.Anything).Return([]*models.Device{{DeviceID: "esp-1", Metadata: "{}"}}, nil).Once()

		code, body := get(e, "/devices")

		require.Equal(t, http.StatusOK, code)
		data := body["data"].([]any)
		require.Len(t, data, 1)
		assert.Equal(t, "esp-1", data[0].(map[string]any)["device_id"])
	})

	t.Run("app error is rendered by the error handler", func(t *testing.T) {
		e, r := newServer(t)
		r.device.EXPECT().FindAll(mock.Anything).Return(nil, errors.New("db down")).Once()

		code, _ := get(e, "/devices")

		assert.Equal(t, http.StatusInternalServerError, code)
	})
}

func TestHandler_GetCurrentTelemetry(t *testing.T) {
	t.Run("success returns the newest report", func(t *testing.T) {
		e, r := newServer(t)
		r.telemetry.EXPECT().FindCurrent(mock.Anything, "esp-1").
			Return(&models.DeviceTelemetry{DeviceID: "esp-1", RSSIDbm: new(int16(-48))}, nil).Once()

		code, body := get(e, "/devices/esp-1/telemetry/current")

		require.Equal(t, http.StatusOK, code)
		assert.InDelta(t, -48.0, body["data"].(map[string]any)["rssi_dbm"], 1e-9)
	})

	t.Run("no report is a 404", func(t *testing.T) {
		e, r := newServer(t)
		r.telemetry.EXPECT().FindCurrent(mock.Anything, "esp-1").Return(nil, database.ErrNotFound).Once()

		code, _ := get(e, "/devices/esp-1/telemetry/current")

		assert.Equal(t, http.StatusNotFound, code)
	})
}

func TestHandler_GetTelemetryStats(t *testing.T) {
	t.Run("allowed range reaches the repository with its duration", func(t *testing.T) {
		e, r := newServer(t)
		r.telemetry.EXPECT().FindAllByDevice(mock.Anything, "esp-1", 30*time.Minute).Return(nil, nil).Once()

		code, body := get(e, "/devices/esp-1/telemetry/stats?range=30m")

		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, "30m", body["data"].(map[string]any)["range"])
	})

	for _, path := range []string{
		"/devices/esp-1/telemetry/stats",
		"/devices/esp-1/telemetry/stats?range=10m",
		"/devices/esp-1/telemetry/stats?range=90m",
	} {
		t.Run("rejects "+path, func(t *testing.T) {
			e, _ := newServer(t)

			code, _ := get(e, path)

			assert.Equal(t, http.StatusBadRequest, code)
		})
	}
}

func TestHandler_GetCurrentSensor(t *testing.T) {
	t.Run("success returns the newest reading", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindCurrent(mock.Anything, "esp-1").
			Return(&models.SensorRaw{DeviceID: "esp-1", TempC: 31.6, HumidityPct: 74.5}, nil).Once()

		code, body := get(e, "/devices/esp-1/sensor/current")

		require.Equal(t, http.StatusOK, code)
		assert.InDelta(t, 31.6, body["data"].(map[string]any)["temp_c"], 1e-9)
	})

	t.Run("no reading is a 404", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindCurrent(mock.Anything, "esp-1").Return(nil, database.ErrNotFound).Once()

		code, _ := get(e, "/devices/esp-1/sensor/current")

		assert.Equal(t, http.StatusNotFound, code)
	})
}

func TestHandler_GetSensorStats(t *testing.T) {
	t.Run("allowed range reaches the repository with its duration", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindAllByDevice(mock.Anything, "esp-1", 90*time.Minute).Return(nil, nil).Once()

		code, _ := get(e, "/devices/esp-1/sensor/stats?range=90m")

		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("range outside 30m/60m/90m is a 400", func(t *testing.T) {
		e, _ := newServer(t)

		code, _ := get(e, "/devices/esp-1/sensor/stats?range=15m")

		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("app error is rendered by the error handler", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindAllByDevice(mock.Anything, "esp-1", 30*time.Minute).Return(nil, errors.New("db down")).Once()

		code, _ := get(e, "/devices/esp-1/sensor/stats?range=30m")

		assert.Equal(t, http.StatusInternalServerError, code)
	})
}

func TestHandler_GetSensorAverage24h(t *testing.T) {
	t.Run("success returns the averages", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).
			Return([]*models.Sensor30m{{AvgC: 30, AvgH: 70, Samples: 10}}, nil).Once()

		code, body := get(e, "/devices/esp-1/sensor/average-24h")

		require.Equal(t, http.StatusOK, code)
		assert.InDelta(t, 30.0, body["data"].(map[string]any)["avg_temp_c"], 1e-9)
	})

	t.Run("app error is rendered by the error handler", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(nil, errors.New("db down")).Once()

		code, _ := get(e, "/devices/esp-1/sensor/average-24h")

		assert.Equal(t, http.StatusInternalServerError, code)
	})
}

func TestHandler_GetSensorStats24h(t *testing.T) {
	t.Run("success returns the buckets", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).
			Return([]*models.Sensor30m{{AvgC: 30, Samples: 10}}, nil).Once()

		code, body := get(e, "/devices/esp-1/sensor/stats-24h")

		require.Equal(t, http.StatusOK, code)
		assert.Len(t, body["data"].(map[string]any)["points"], 1)
	})

	t.Run("app error is rendered by the error handler", func(t *testing.T) {
		e, r := newServer(t)
		r.sensor.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(nil, errors.New("db down")).Once()

		code, _ := get(e, "/devices/esp-1/sensor/stats-24h")

		assert.Equal(t, http.StatusInternalServerError, code)
	})
}
