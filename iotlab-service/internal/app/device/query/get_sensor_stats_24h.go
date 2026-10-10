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

// IGetSensorStats24hQuery returns the 30-minute temperature/humidity buckets of the last 24h plus their summary.
type IGetSensorStats24hQuery decorator.QueryHandler[*devicedtos.DeviceIDReq, *devicedtos.SensorStats24hDTO]

type getSensorStats24hQuery struct {
	sensorRepo sensorrepo.Repository
}

// NewGetSensorStats24hQuery builds the 24h-stats query.
func NewGetSensorStats24hQuery(sensorRepo sensorrepo.Repository) IGetSensorStats24hQuery {
	return &getSensorStats24hQuery{sensorRepo: sensorRepo}
}

func (q *getSensorStats24hQuery) Handle(
	ctx context.Context,
	req *devicedtos.DeviceIDReq,
) (*devicedtos.SensorStats24hDTO, error) {
	buckets, err := q.sensorRepo.FindBuckets30m(ctx, req.DeviceID, 24*time.Hour)
	if err != nil {
		slog.Error("get sensor 24h stats failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}

	var temp, humidity accumulator
	for _, b := range buckets {
		temp.addBucket(b.MinC, b.MaxC, b.AvgC, b.Samples)
		humidity.addBucket(b.MinH, b.MaxH, b.AvgH, b.Samples)
	}

	return &devicedtos.SensorStats24hDTO{
		Range: dayWindow,
		Summary: devicedtos.SensorSummary{
			TempC:       temp.stat(),
			HumidityPct: humidity.stat(),
			Samples:     temp.n,
		},
		Points: devicedtos.NewSensorBucketDTOs(buckets),
	}, nil
}
