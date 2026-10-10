package query

import (
	"context"
	"errors"
	"log/slog"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	sensorrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IGetCurrentSensorQuery reads the newest temperature/humidity reading of a device.
type IGetCurrentSensorQuery decorator.QueryHandler[*devicedtos.DeviceIDReq, *devicedtos.SensorDTO]

type getCurrentSensorQuery struct {
	sensorRepo sensorrepo.Repository
}

// NewGetCurrentSensorQuery builds the current-sensor query.
func NewGetCurrentSensorQuery(sensorRepo sensorrepo.Repository) IGetCurrentSensorQuery {
	return &getCurrentSensorQuery{sensorRepo: sensorRepo}
}

func (q *getCurrentSensorQuery) Handle(ctx context.Context, req *devicedtos.DeviceIDReq) (*devicedtos.SensorDTO, error) {
	row, err := q.sensorRepo.FindCurrent(ctx, req.DeviceID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, svcerr.ErrRecordNotFound
		}
		slog.Error("get current sensor failed", slog.String("device_id", req.DeviceID), slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}
	return devicedtos.NewSensorDTO(row), nil
}
