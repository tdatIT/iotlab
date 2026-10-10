package device_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	sensormock "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor/mock"
	telemetrymock "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry/mock"
)

func TestNewDeviceApplication_wiresEveryHandler(t *testing.T) {
	app := deviceapp.NewDeviceApplication(
		devicemock.NewMockRepository(t),
		sensormock.NewMockRepository(t),
		telemetrymock.NewMockRepository(t),
	)

	assert.NotNil(t, app.Command.RecordSensor)
	assert.NotNil(t, app.Command.RecordTelemetry)
	assert.NotNil(t, app.Command.UpdateDeviceStatus)
	assert.NotNil(t, app.Query.ListDevices)
	assert.NotNil(t, app.Query.GetCurrentTelemetry)
	assert.NotNil(t, app.Query.GetTelemetryStats)
	assert.NotNil(t, app.Query.GetCurrentSensor)
	assert.NotNil(t, app.Query.GetSensorStats)
	assert.NotNil(t, app.Query.GetSensorAverage24h)
	assert.NotNil(t, app.Query.GetSensorStats24h)
}
