package command

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicerepo "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	sensorrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IRecordSensorCommand stores a sensor reading and marks the device as seen.
type IRecordSensorCommand decorator.CommandHandler[*devicedtos.RecordSensorReq]

type recordSensorCommand struct {
	sensorRepo sensorrepo.Repository
	deviceRepo devicerepo.Repository
}

// NewRecordSensorCommand builds the record-sensor command.
func NewRecordSensorCommand(sensorRepo sensorrepo.Repository, deviceRepo devicerepo.Repository) IRecordSensorCommand {
	return &recordSensorCommand{sensorRepo: sensorRepo, deviceRepo: deviceRepo}
}

func (c *recordSensorCommand) Handle(ctx context.Context, req *devicedtos.RecordSensorReq) error {
	reading := &models.SensorRaw{
		Time:        time.UnixMilli(req.Time).UTC(),
		DeviceID:    req.DeviceID,
		TempC:       req.TempC,
		HumidityPct: req.HumidityPct,
	}

	// A redelivered reading hits the (device_id, time) unique index; it is already stored, so it is not an error.
	err := c.sensorRepo.Create(ctx, reading)
	if err != nil && !errors.Is(err, database.ErrDuplicate) {
		slog.Error("record sensor failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return svcerr.ErrInternalServer
	}

	if err := c.deviceRepo.Upsert(ctx, &models.Device{
		DeviceID: req.DeviceID,
		LastSeen: time.Now().UTC(),
	}, "last_seen"); err != nil {
		slog.Error("record sensor: touch device failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return svcerr.ErrInternalServer
	}

	return nil
}
