package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/app/device/command"
	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

func TestUpdateDeviceStatusCommand_Handle(t *testing.T) {
	ctx := context.Background()
	online := &devicedtos.UpdateDeviceStatusReq{DeviceID: "esp-1", Status: models.DeviceStatusOnline}
	offline := &devicedtos.UpdateDeviceStatusReq{DeviceID: "esp-1", Status: models.DeviceStatusOffline}

	t.Run("unknown device going online is inserted with last_seen", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindByID(mock.Anything, "esp-1").Return(nil, database.ErrNotFound).Once()
		repo.EXPECT().
			Upsert(mock.Anything, mock.MatchedBy(func(d *models.Device) bool {
				return d.Status == models.DeviceStatusOnline && !d.StatusChangedAt.IsZero() && d.LastSeen.Equal(d.StatusChangedAt)
			}), []string{"status", "status_changed_at", "last_seen"}).
			Return(nil).Once()

		require.NoError(t, command.NewUpdateDeviceStatusCommand(repo).Handle(ctx, online))
	})

	t.Run("going offline does not move last_seen", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindByID(mock.Anything, "esp-1").
			Return(&models.Device{DeviceID: "esp-1", Status: models.DeviceStatusOnline}, nil).Once()
		repo.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"status", "status_changed_at"}).Return(nil).Once()

		require.NoError(t, command.NewUpdateDeviceStatusCommand(repo).Handle(ctx, offline))
	})

	t.Run("unchanged status is a no-op", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindByID(mock.Anything, "esp-1").
			Return(&models.Device{DeviceID: "esp-1", Status: models.DeviceStatusOnline}, nil).Once()

		require.NoError(t, command.NewUpdateDeviceStatusCommand(repo).Handle(ctx, online))
		repo.AssertNotCalled(t, "Upsert")
	})

	t.Run("find failure maps to internal server", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindByID(mock.Anything, "esp-1").Return(nil, errors.New("db down")).Once()

		assert.ErrorIs(t, command.NewUpdateDeviceStatusCommand(repo).Handle(ctx, online), svcerr.ErrInternalServer)
	})

	t.Run("upsert failure maps to internal server", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindByID(mock.Anything, "esp-1").Return(nil, database.ErrNotFound).Once()
		repo.EXPECT().Upsert(mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("db down")).Once()

		assert.ErrorIs(t, command.NewUpdateDeviceStatusCommand(repo).Handle(ctx, online), svcerr.ErrInternalServer)
	})
}
