package query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/app/device/query"
	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicemock "github.com/tdatIT/iotlab-service/internal/infras/repository/device/mock"
	sensormock "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor/mock"
	telemetrymock "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry/mock"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

var (
	ctx   = context.Background()
	byID  = &devicedtos.DeviceIDReq{DeviceID: "esp-1"}
	dbErr = errors.New("db down")
)

func TestListDevicesQuery_Handle(t *testing.T) {
	t.Run("maps every device", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindAll(mock.Anything).
			Return([]*models.Device{{DeviceID: "esp-1", Metadata: `{"room":"lab"}`}}, nil).Once()

		got, err := query.NewListDevicesQuery(repo).Handle(ctx, &devicedtos.ListDevicesReq{})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "esp-1", got[0].DeviceID)
		assert.JSONEq(t, `{"room":"lab"}`, string(got[0].Metadata))
	})

	t.Run("no devices returns an empty, non-nil list", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindAll(mock.Anything).Return(nil, nil).Once()

		got, err := query.NewListDevicesQuery(repo).Handle(ctx, &devicedtos.ListDevicesReq{})

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := devicemock.NewMockRepository(t)
		repo.EXPECT().FindAll(mock.Anything).Return(nil, dbErr).Once()

		_, err := query.NewListDevicesQuery(repo).Handle(ctx, &devicedtos.ListDevicesReq{})

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}

func TestGetCurrentTelemetryQuery_Handle(t *testing.T) {
	t.Run("returns the newest report", func(t *testing.T) {
		repo := telemetrymock.NewMockRepository(t)
		repo.EXPECT().FindCurrent(mock.Anything, "esp-1").
			Return(&models.DeviceTelemetry{DeviceID: "esp-1", RSSIDbm: new(int16(-48))}, nil).Once()

		got, err := query.NewGetCurrentTelemetryQuery(repo).Handle(ctx, byID)

		require.NoError(t, err)
		assert.Equal(t, int16(-48), *got.RSSIDbm)
	})

	t.Run("no report maps to record not found", func(t *testing.T) {
		repo := telemetrymock.NewMockRepository(t)
		repo.EXPECT().FindCurrent(mock.Anything, "esp-1").Return(nil, database.ErrNotFound).Once()

		_, err := query.NewGetCurrentTelemetryQuery(repo).Handle(ctx, byID)

		assert.ErrorIs(t, err, svcerr.ErrRecordNotFound)
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := telemetrymock.NewMockRepository(t)
		repo.EXPECT().FindCurrent(mock.Anything, "esp-1").Return(nil, dbErr).Once()

		_, err := query.NewGetCurrentTelemetryQuery(repo).Handle(ctx, byID)

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}

func TestGetTelemetryStatsQuery_Handle(t *testing.T) {
	t.Run("summarises the window and skips null metrics", func(t *testing.T) {
		repo := telemetrymock.NewMockRepository(t)
		repo.EXPECT().FindAllByDevice(mock.Anything, "esp-1", 15*time.Minute).Return([]*models.DeviceTelemetry{
			{RSSIDbm: new(int16(-40)), HeapFree: new(int32(1000))},
			{RSSIDbm: new(int16(-60)), HeapFree: new(int32(3000))},
			{},
		}, nil).Once()

		got, err := query.NewGetTelemetryStatsQuery(repo).
			Handle(ctx, &devicedtos.TelemetryStatsReq{DeviceID: "esp-1", Range: "15m"})

		require.NoError(t, err)
		assert.Equal(t, "15m", got.Range)
		assert.Equal(t, int64(3), got.Summary.Samples)
		assert.Len(t, got.Points, 3)
		assert.Equal(t, &devicedtos.Stat{Min: -60, Max: -40, Avg: -50}, got.Summary.RSSIDbm)
		assert.Equal(t, &devicedtos.Stat{Min: 1000, Max: 3000, Avg: 2000}, got.Summary.HeapFree)
		assert.Nil(t, got.Summary.HeapMin, "no report carries heap_min")
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := telemetrymock.NewMockRepository(t)
		repo.EXPECT().FindAllByDevice(mock.Anything, "esp-1", time.Hour).Return(nil, dbErr).Once()

		_, err := query.NewGetTelemetryStatsQuery(repo).
			Handle(ctx, &devicedtos.TelemetryStatsReq{DeviceID: "esp-1", Range: "60m"})

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}

func TestGetCurrentSensorQuery_Handle(t *testing.T) {
	t.Run("returns the newest reading", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindCurrent(mock.Anything, "esp-1").
			Return(&models.SensorRaw{DeviceID: "esp-1", TempC: 31.6, HumidityPct: 74.5}, nil).Once()

		got, err := query.NewGetCurrentSensorQuery(repo).Handle(ctx, byID)

		require.NoError(t, err)
		assert.InDelta(t, 31.6, got.TempC, 1e-9)
	})

	t.Run("no reading maps to record not found", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindCurrent(mock.Anything, "esp-1").Return(nil, database.ErrNotFound).Once()

		_, err := query.NewGetCurrentSensorQuery(repo).Handle(ctx, byID)

		assert.ErrorIs(t, err, svcerr.ErrRecordNotFound)
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindCurrent(mock.Anything, "esp-1").Return(nil, dbErr).Once()

		_, err := query.NewGetCurrentSensorQuery(repo).Handle(ctx, byID)

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}

func TestGetSensorStatsQuery_Handle(t *testing.T) {
	t.Run("summarises the window", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindAllByDevice(mock.Anything, "esp-1", 90*time.Minute).Return([]*models.SensorRaw{
			{TempC: 30, HumidityPct: 70},
			{TempC: 32, HumidityPct: 80},
		}, nil).Once()

		got, err := query.NewGetSensorStatsQuery(repo).Handle(ctx, &devicedtos.SensorStatsReq{DeviceID: "esp-1", Range: "90m"})

		require.NoError(t, err)
		assert.Equal(t, int64(2), got.Summary.Samples)
		assert.Equal(t, &devicedtos.Stat{Min: 30, Max: 32, Avg: 31}, got.Summary.TempC)
		assert.Equal(t, &devicedtos.Stat{Min: 70, Max: 80, Avg: 75}, got.Summary.HumidityPct)
		assert.Len(t, got.Points, 2)
	})

	t.Run("empty window has null stats and an empty series", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindAllByDevice(mock.Anything, "esp-1", 30*time.Minute).Return(nil, nil).Once()

		got, err := query.NewGetSensorStatsQuery(repo).Handle(ctx, &devicedtos.SensorStatsReq{DeviceID: "esp-1", Range: "30m"})

		require.NoError(t, err)
		assert.Nil(t, got.Summary.TempC)
		assert.NotNil(t, got.Points)
		assert.Empty(t, got.Points)
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindAllByDevice(mock.Anything, "esp-1", time.Hour).Return(nil, dbErr).Once()

		_, err := query.NewGetSensorStatsQuery(repo).Handle(ctx, &devicedtos.SensorStatsReq{DeviceID: "esp-1", Range: "60m"})

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}

// buckets is two 30-minute aggregates with different sample counts, so a weighted average differs from a plain one.
var buckets = []*models.Sensor30m{
	{AvgC: 30, MinC: 29, MaxC: 31, AvgH: 70, MinH: 60, MaxH: 75, Samples: 100},
	{AvgC: 33, MinC: 32, MaxC: 35, AvgH: 80, MinH: 78, MaxH: 90, Samples: 50},
}

func TestGetSensorAverage24hQuery_Handle(t *testing.T) {
	t.Run("weights each bucket average by its sample count", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(buckets, nil).Once()

		got, err := query.NewGetSensorAverage24hQuery(repo).Handle(ctx, byID)

		require.NoError(t, err)
		assert.Equal(t, "24h", got.Range)
		assert.Equal(t, int64(150), got.Samples)
		assert.InDelta(t, 31.0, *got.AvgTempC, 1e-9)
		assert.InDelta(t, 220.0/3, *got.AvgHumidityPct, 1e-9)
	})

	t.Run("no bucket gives null averages", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(nil, nil).Once()

		got, err := query.NewGetSensorAverage24hQuery(repo).Handle(ctx, byID)

		require.NoError(t, err)
		assert.Nil(t, got.AvgTempC)
		assert.Zero(t, got.Samples)
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(nil, dbErr).Once()

		_, err := query.NewGetSensorAverage24hQuery(repo).Handle(ctx, byID)

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}

func TestGetSensorStats24hQuery_Handle(t *testing.T) {
	t.Run("summarises the buckets and returns them as the series", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(buckets, nil).Once()

		got, err := query.NewGetSensorStats24hQuery(repo).Handle(ctx, byID)

		require.NoError(t, err)
		assert.Len(t, got.Points, 2)
		assert.Equal(t, int64(150), got.Summary.Samples)
		assert.InDelta(t, 29.0, got.Summary.TempC.Min, 1e-9)
		assert.InDelta(t, 35.0, got.Summary.TempC.Max, 1e-9)
		assert.InDelta(t, 31.0, got.Summary.TempC.Avg, 1e-9)
		assert.InDelta(t, 60.0, got.Summary.HumidityPct.Min, 1e-9)
		assert.InDelta(t, 90.0, got.Summary.HumidityPct.Max, 1e-9)
	})

	t.Run("repo failure maps to internal server", func(t *testing.T) {
		repo := sensormock.NewMockRepository(t)
		repo.EXPECT().FindBuckets30m(mock.Anything, "esp-1", 24*time.Hour).Return(nil, dbErr).Once()

		_, err := query.NewGetSensorStats24hQuery(repo).Handle(ctx, byID)

		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}
