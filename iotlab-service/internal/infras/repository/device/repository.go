// Package device is the persistence layer for the device registry.
package device

import (
	"context"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
)

// Repository is the persistence contract for the devices table. It returns database.ErrNotFound / database.ErrDuplicate
// instead of driver errors; the app layer maps them to svcerr.* before returning to a transport boundary.
type Repository interface {
	// Create inserts device; zero-valued status, timestamps and metadata take the column defaults.
	Create(ctx context.Context, device *models.Device) error
	FindByID(ctx context.Context, deviceID string) (*models.Device, error)
	FindAll(ctx context.Context) ([]*models.Device, error)
	// Update overwrites only the non-zero (non-nil) fields of device.
	Update(ctx context.Context, device *models.Device) error
	Delete(ctx context.Context, deviceID string) error
	// Upsert inserts device, or on a device_id conflict overwrites only the given columns of the existing row.
	Upsert(ctx context.Context, device *models.Device, columns ...string) error
}
