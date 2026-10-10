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

// IGetSensorStatsQuery returns the raw temperature/humidity series of a window plus its summary.
type IGetSensorStatsQuery decorator.QueryHandler[*devicedtos.SensorStatsReq, *devicedtos.SensorStatsDTO]

type getSensorStatsQuery struct {
	sensorRepo sensorrepo.Repository
}

// NewGetSensorStatsQuery builds the sensor-stats query.
func NewGetSensorStatsQuery(sensorRepo sensorrepo.Repository) IGetSensorStatsQuery {
	return &getSensorStatsQuery{sensorRepo: sensorRepo}
}

func (q *getSensorStatsQuery) Handle(ctx context.Context, req *devicedtos.SensorStatsReq) (*devicedtos.SensorStatsDTO, error) {
	window, _ := time.ParseDuration(req.Range) // Range is validated against a fixed set of durations at the handler

	rows, err := q.sensorRepo.FindAllByDevice(ctx, req.DeviceID, window)
	if err != nil {
		slog.Error("get sensor stats failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}

	var temp, humidity accumulator
	for _, r := range rows {
		temp.add(r.TempC)
		humidity.add(r.HumidityPct)
	}

	return &devicedtos.SensorStatsDTO{
		Range: req.Range,
		Summary: devicedtos.SensorSummary{
			TempC:       temp.stat(),
			HumidityPct: humidity.stat(),
			Samples:     temp.n,
		},
		Points: devicedtos.NewSensorDTOs(rows),
	}, nil
}
