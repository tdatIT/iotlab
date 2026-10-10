// Package device is the application layer for the IoT devices: use-case orchestration only, no transport or persistence
// details.
package device

import (
	"github.com/tdatIT/iotlab-service/internal/app/device/command"
	"github.com/tdatIT/iotlab-service/internal/app/device/query"
	devicerepo "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	sensorrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor"
	telemetryrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry"
)

// Application bundles every device command and query. The repositories are injected once here and reused by every handler.
type Application struct {
	Command commands
	Query   queries
}

type commands struct {
	RecordSensor       command.IRecordSensorCommand
	RecordTelemetry    command.IRecordTelemetryCommand
	UpdateDeviceStatus command.IUpdateDeviceStatusCommand
}

type queries struct {
	ListDevices         query.IListDevicesQuery
	GetCurrentTelemetry query.IGetCurrentTelemetryQuery
	GetTelemetryStats   query.IGetTelemetryStatsQuery
	GetCurrentSensor    query.IGetCurrentSensorQuery
	GetSensorStats      query.IGetSensorStatsQuery
	GetSensorAverage24h query.IGetSensorAverage24hQuery
	GetSensorStats24h   query.IGetSensorStats24hQuery
}

// NewDeviceApplication builds every device command and query, injecting the repositories once.
func NewDeviceApplication(
	deviceRepo devicerepo.Repository,
	sensorRepo sensorrepo.Repository,
	telemetryRepo telemetryrepo.Repository,
) *Application {
	return &Application{
		Command: commands{
			RecordSensor:       command.NewRecordSensorCommand(sensorRepo, deviceRepo),
			RecordTelemetry:    command.NewRecordTelemetryCommand(telemetryRepo, deviceRepo),
			UpdateDeviceStatus: command.NewUpdateDeviceStatusCommand(deviceRepo),
		},
		Query: queries{
			ListDevices:         query.NewListDevicesQuery(deviceRepo),
			GetCurrentTelemetry: query.NewGetCurrentTelemetryQuery(telemetryRepo),
			GetTelemetryStats:   query.NewGetTelemetryStatsQuery(telemetryRepo),
			GetCurrentSensor:    query.NewGetCurrentSensorQuery(sensorRepo),
			GetSensorStats:      query.NewGetSensorStatsQuery(sensorRepo),
			GetSensorAverage24h: query.NewGetSensorAverage24hQuery(sensorRepo),
			GetSensorStats24h:   query.NewGetSensorStats24hQuery(sensorRepo),
		},
	}
}
