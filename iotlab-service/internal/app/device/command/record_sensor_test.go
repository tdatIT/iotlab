package command_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/app/device/command"
	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	sensormock "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor/mock"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

func TestRecordSensorCommand_Handle(t *testing.T) {
	ctx := context.Background()
	req := &devicedtos.RecordSensorReq{DeviceID: "esp-1", TempC: 31.6, HumidityPct: 74.5, Time: 1791620022386}

	t.Run("stores the reading in UTC and touches the device last_seen", func(t *testing.T) {
		sensorRepo := sensormock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		sensorRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(s *models.SensorRaw) bool {
				return s.DeviceID == "esp-1" && s.TempC == 31.6 && s.HumidityPct == 74.5 &&
					s.Time.Equal(time.UnixMilli(1791620022386)) && s.Time.Location() == time.UTC
			})).
			Return(nil).Once()
		deviceRepo.EXPECT().
			Upsert(mock.Anything, mock.MatchedBy(func(d *models.Device) bool {
				return d.DeviceID == "esp-1" && !d.LastSeen.IsZero()
			}), []string{"last_seen"}).
			Return(nil).Once()

		require.NoError(t, command.NewRecordSensorCommand(sensorRepo, deviceRepo).Handle(ctx, req))
	})

	t.Run("duplicate reading is not an error", func(t *testing.T) {
		sensorRepo := sensormock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		sensorRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(database.ErrDuplicate).Once()
		deviceRepo.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen"}).Return(nil).Once()

		assert.NoError(t, command.NewRecordSensorCommand(sensorRepo, deviceRepo).Handle(ctx, req))
	})

	t.Run("store failure maps to internal server and skips the device", func(t *testing.T) {
		sensorRepo := sensormock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		sensorRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db down")).Once()

		err := command.NewRecordSensorCommand(sensorRepo, deviceRepo).Handle(ctx, req)

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
		deviceRepo.AssertNotCalled(t, "Upsert")
	})

	t.Run("device touch failure maps to internal server", func(t *testing.T) {
		sensorRepo := sensormock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		sensorRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
		deviceRepo.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen"}).Return(errors.New("db down")).Once()

		err := command.NewRecordSensorCommand(sensorRepo, deviceRepo).Handle(ctx, req)

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}
