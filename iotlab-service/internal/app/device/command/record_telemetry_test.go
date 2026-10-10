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
	telemetrymock "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry/mock"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

func TestRecordTelemetryCommand_Handle(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the report and refreshes last_seen, fw and ip", func(t *testing.T) {
		telemetryRepo := telemetrymock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		req := &devicedtos.RecordTelemetryReq{
			DeviceID: "esp-1", Time: 1791620046520, FW: new("1.2.0-dht22"), IP: new("192.168.1.156"), RSSIDbm: new(int16(-48)),
		}
		telemetryRepo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(m *models.DeviceTelemetry) bool {
				return m.DeviceID == "esp-1" && *m.RSSIDbm == -48 && m.Time.UnixMilli() == 1791620046520
			})).
			Return(nil).Once()
		deviceRepo.EXPECT().
			Upsert(mock.Anything, mock.MatchedBy(func(d *models.Device) bool {
				return d.DeviceID == "esp-1" && *d.FW == "1.2.0-dht22" && *d.IP == "192.168.1.156"
			}), []string{"last_seen", "fw", "ip"}).
			Return(nil).Once()

		require.NoError(t, command.NewRecordTelemetryCommand(telemetryRepo, deviceRepo).Handle(ctx, req))
	})

	t.Run("missing fw and ip leave those device columns untouched", func(t *testing.T) {
		telemetryRepo := telemetrymock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		telemetryRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(database.ErrDuplicate).Once()
		deviceRepo.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen"}).Return(nil).Once()

		err := command.NewRecordTelemetryCommand(telemetryRepo, deviceRepo).
			Handle(ctx, &devicedtos.RecordTelemetryReq{DeviceID: "esp-1", Time: 1})

		assert.NoError(t, err)
	})

	t.Run("store failure maps to internal server and skips the device", func(t *testing.T) {
		telemetryRepo := telemetrymock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		telemetryRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db down")).Once()

		err := command.NewRecordTelemetryCommand(telemetryRepo, deviceRepo).
			Handle(ctx, &devicedtos.RecordTelemetryReq{DeviceID: "esp-1", Time: 1})

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
		deviceRepo.AssertNotCalled(t, "Upsert")
	})

	t.Run("device touch failure maps to internal server", func(t *testing.T) {
		telemetryRepo := telemetrymock.NewMockRepository(t)
		deviceRepo := devicemock.NewMockRepository(t)
		telemetryRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
		deviceRepo.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen"}).Return(errors.New("db down")).Once()

		err := command.NewRecordTelemetryCommand(telemetryRepo, deviceRepo).
			Handle(ctx, &devicedtos.RecordTelemetryReq{DeviceID: "esp-1", Time: 1})

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}
