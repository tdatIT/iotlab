package command

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicerepo "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	telemetryrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IRecordTelemetryCommand stores a telemetry report and refreshes the device's last_seen, fw and ip.
type IRecordTelemetryCommand decorator.CommandHandler[*devicedtos.RecordTelemetryReq]

type recordTelemetryCommand struct {
	telemetryRepo telemetryrepo.Repository
	deviceRepo    devicerepo.Repository
}

// NewRecordTelemetryCommand builds the record-telemetry command.
func NewRecordTelemetryCommand(
	telemetryRepo telemetryrepo.Repository,
	deviceRepo devicerepo.Repository,
) IRecordTelemetryCommand {
	return &recordTelemetryCommand{telemetryRepo: telemetryRepo, deviceRepo: deviceRepo}
}

func (c *recordTelemetryCommand) Handle(ctx context.Context, req *devicedtos.RecordTelemetryReq) error {
	report := &models.DeviceTelemetry{
		Time:           time.UnixMilli(req.Time).UTC(),
		DeviceID:       req.DeviceID,
		FW:             req.FW,
		UptimeS:        req.UptimeS,
		ResetReason:    req.ResetReason,
		RSSIDbm:        req.RSSIDbm,
		IP:             req.IP,
		HeapFree:       req.HeapFree,
		HeapMin:        req.HeapMin,
		WifiReconnects: req.WifiReconnects,
		MQTTReconnects: req.MQTTReconnects,
		SensorErrors:   req.SensorErrors,
	}

	// A redelivered report hits the (device_id, time) primary key; it is already stored, so it is not an error.
	err := c.telemetryRepo.Create(ctx, report)
	if err != nil && !errors.Is(err, database.ErrDuplicate) {
		slog.Error("record telemetry failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return svcerr.ErrInternalServer
	}

	device := &models.Device{DeviceID: req.DeviceID, LastSeen: time.Now().UTC(), FW: req.FW, IP: req.IP}
	columns := []string{"last_seen"}
	if req.FW != nil {
		columns = append(columns, "fw")
	}
	if req.IP != nil {
		columns = append(columns, "ip")
	}
	if err := c.deviceRepo.Upsert(ctx, device, columns...); err != nil {
		slog.Error("record telemetry: touch device failed",
			slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return svcerr.ErrInternalServer
	}

	return nil
}
