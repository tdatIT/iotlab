package query

import (
	"context"
	"errors"
	"log/slog"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	telemetryrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IGetCurrentTelemetryQuery reads the newest telemetry report of a device.
type IGetCurrentTelemetryQuery decorator.QueryHandler[*devicedtos.DeviceIDReq, *devicedtos.TelemetryDTO]

type getCurrentTelemetryQuery struct {
	telemetryRepo telemetryrepo.Repository
}

// NewGetCurrentTelemetryQuery builds the current-telemetry query.
func NewGetCurrentTelemetryQuery(telemetryRepo telemetryrepo.Repository) IGetCurrentTelemetryQuery {
	return &getCurrentTelemetryQuery{telemetryRepo: telemetryRepo}
}

func (q *getCurrentTelemetryQuery) Handle(ctx context.Context, req *devicedtos.DeviceIDReq) (*devicedtos.TelemetryDTO, error) {
	row, err := q.telemetryRepo.FindCurrent(ctx, req.DeviceID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, svcerr.ErrRecordNotFound
		}
		slog.Error("get current telemetry failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}
	return devicedtos.NewTelemetryDTO(row), nil
}
