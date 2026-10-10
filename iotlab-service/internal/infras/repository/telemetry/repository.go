// Package telemetry is the persistence layer for the device_telemetry hypertable.
package telemetry

import (
	"context"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
)

// Repository is the persistence contract for device telemetry. A row is keyed by (device_id, time). Range reads return rows
// oldest first so they plot directly as a series. It returns database.ErrNotFound / database.ErrDuplicate instead of driver
// errors; the app layer maps them to svcerr.*.
type Repository interface {
	Create(ctx context.Context, telemetry *models.DeviceTelemetry) error
	FindByKey(ctx context.Context, deviceID string, at time.Time) (*models.DeviceTelemetry, error)
	Update(ctx context.Context, telemetry *models.DeviceTelemetry) error
	Delete(ctx context.Context, deviceID string, at time.Time) error
	// FindAllByDevice returns the rows of the last window. The caller enforces the allowed window values.
	FindAllByDevice(ctx context.Context, deviceID string, window time.Duration) ([]*models.DeviceTelemetry, error)
	FindLast6h(ctx context.Context, deviceID string) ([]*models.DeviceTelemetry, error)
	FindLastDay(ctx context.Context, deviceID string) ([]*models.DeviceTelemetry, error)
	// FindCurrent returns the newest row of the device.
	FindCurrent(ctx context.Context, deviceID string) (*models.DeviceTelemetry, error)
}
