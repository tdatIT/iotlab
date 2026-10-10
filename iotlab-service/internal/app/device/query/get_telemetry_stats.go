package query

import (
	"context"
	"log/slog"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	telemetryrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IGetTelemetryStatsQuery returns the telemetry series of a window plus its RSSI/heap summary.
type IGetTelemetryStatsQuery decorator.QueryHandler[*devicedtos.TelemetryStatsReq, *devicedtos.TelemetryStatsDTO]

type getTelemetryStatsQuery struct {
	telemetryRepo telemetryrepo.Repository
}

// NewGetTelemetryStatsQuery builds the telemetry-stats query.
func NewGetTelemetryStatsQuery(telemetryRepo telemetryrepo.Repository) IGetTelemetryStatsQuery {
	return &getTelemetryStatsQuery{telemetryRepo: telemetryRepo}
}

func (q *getTelemetryStatsQuery) Handle(
	ctx context.Context,
	req *devicedtos.TelemetryStatsReq,
) (*devicedtos.TelemetryStatsDTO, error) {
	window, _ := time.ParseDuration(req.Range) // Range is validated against a fixed set of durations at the handler

	rows, err := q.telemetryRepo.FindAllByDevice(ctx, req.DeviceID, window)
	if err != nil {
		slog.Error("get telemetry stats failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}

	var rssi, heapFree, heapMin accumulator
	for _, r := range rows {
		if r.RSSIDbm != nil {
			rssi.add(float64(*r.RSSIDbm))
		}
		if r.HeapFree != nil {
			heapFree.add(float64(*r.HeapFree))
		}
		if r.HeapMin != nil {
			heapMin.add(float64(*r.HeapMin))
		}
	}

	return &devicedtos.TelemetryStatsDTO{
		Range: req.Range,
		Summary: devicedtos.TelemetrySummary{
			RSSIDbm:  rssi.stat(),
			HeapFree: heapFree.stat(),
			HeapMin:  heapMin.stat(),
			Samples:  int64(len(rows)),
		},
		Points: devicedtos.NewTelemetryDTOs(rows),
	}, nil
}
