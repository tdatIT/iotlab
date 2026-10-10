// Package sensor is the persistence layer for the sensor_raw hypertable.
package sensor

import (
	"context"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
)

// Repository is the persistence contract for raw sensor readings. A row is keyed by (device_id, time). Range reads return
// rows oldest first so they plot directly as a series. It returns database.ErrNotFound / database.ErrDuplicate instead of
// driver errors; the app layer maps them to svcerr.*.
type Repository interface {
	Create(ctx context.Context, reading *models.SensorRaw) error
	FindByKey(ctx context.Context, deviceID string, at time.Time) (*models.SensorRaw, error)
	Update(ctx context.Context, reading *models.SensorRaw) error
	Delete(ctx context.Context, deviceID string, at time.Time) error
	// FindAllByDevice returns the rows of the last window. The caller enforces the allowed window values.
	FindAllByDevice(ctx context.Context, deviceID string, window time.Duration) ([]*models.SensorRaw, error)
	FindLast6h(ctx context.Context, deviceID string) ([]*models.SensorRaw, error)
	FindLastDay(ctx context.Context, deviceID string) ([]*models.SensorRaw, error)
	// FindCurrent returns the newest row of the device.
	FindCurrent(ctx context.Context, deviceID string) (*models.SensorRaw, error)
	// FindBuckets30m returns the sensor_30m buckets that start inside the last window, oldest first.
	FindBuckets30m(ctx context.Context, deviceID string, window time.Duration) ([]*models.Sensor30m, error)
}
