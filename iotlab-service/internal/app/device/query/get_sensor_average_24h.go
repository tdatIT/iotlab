package query

import (
	"context"
	"log/slog"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	sensorrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IGetSensorAverage24hQuery returns the average temperature/humidity of the last 24h.
type IGetSensorAverage24hQuery decorator.QueryHandler[*devicedtos.DeviceIDReq, *devicedtos.SensorAverageDTO]

type getSensorAverage24hQuery struct {
	sensorRepo sensorrepo.Repository
}

// NewGetSensorAverage24hQuery builds the 24h-average query.
func NewGetSensorAverage24hQuery(sensorRepo sensorrepo.Repository) IGetSensorAverage24hQuery {
	return &getSensorAverage24hQuery{sensorRepo: sensorRepo}
}

// Handle averages the sensor_30m buckets weighted by their sample count, which equals the average of the raw readings
// without scanning ~5760 rows.
func (q *getSensorAverage24hQuery) Handle(
	ctx context.Context,
	req *devicedtos.DeviceIDReq,
) (*devicedtos.SensorAverageDTO, error) {
	buckets, err := q.sensorRepo.FindBuckets30m(ctx, req.DeviceID, 24*time.Hour)
	if err != nil {
		slog.Error("get sensor 24h average failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}

	var temp, humidity accumulator
	for _, b := range buckets {
		temp.addBucket(b.MinC, b.MaxC, b.AvgC, b.Samples)
		humidity.addBucket(b.MinH, b.MaxH, b.AvgH, b.Samples)
	}

	return &devicedtos.SensorAverageDTO{
		Range:          dayWindow,
		AvgTempC:       temp.avg(),
		AvgHumidityPct: humidity.avg(),
		Samples:        temp.n,
	}, nil
}
