package command

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicerepo "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IUpdateDeviceStatusCommand applies an online/offline transition to the device registry.
type IUpdateDeviceStatusCommand decorator.CommandHandler[*devicedtos.UpdateDeviceStatusReq]

type updateDeviceStatusCommand struct {
	deviceRepo devicerepo.Repository
}

// NewUpdateDeviceStatusCommand builds the update-device-status command.
func NewUpdateDeviceStatusCommand(deviceRepo devicerepo.Repository) IUpdateDeviceStatusCommand {
	return &updateDeviceStatusCommand{deviceRepo: deviceRepo}
}

// Handle is a no-op when the status is unchanged (e.g. the retained message is redelivered on reconnect), so
// status_changed_at keeps the time of the real transition. last_seen only moves on "online": "offline" usually comes from
// the broker's last will, not from the device itself.
func (c *updateDeviceStatusCommand) Handle(ctx context.Context, req *devicedtos.UpdateDeviceStatusReq) error {
	current, err := c.deviceRepo.FindByID(ctx, req.DeviceID)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		slog.Error("update device status: find device failed",
			slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return svcerr.ErrInternalServer
	}
	if current != nil && current.Status == req.Status {
		return nil
	}

	now := time.Now().UTC()
	device := &models.Device{DeviceID: req.DeviceID, Status: req.Status, StatusChangedAt: now}
	columns := []string{"status", "status_changed_at"}
	if req.Status == models.DeviceStatusOnline {
		device.LastSeen = now
		columns = append(columns, "last_seen")
	}

	if err := c.deviceRepo.Upsert(ctx, device, columns...); err != nil {
		slog.Error("update device status failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return svcerr.ErrInternalServer
	}
	return nil
}
