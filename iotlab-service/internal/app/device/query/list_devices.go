package query

import (
	"context"
	"log/slog"

	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	devicerepo "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/decorator"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// IListDevicesQuery lists every registered device.
type IListDevicesQuery decorator.QueryHandler[*devicedtos.ListDevicesReq, []*devicedtos.DeviceDTO]

type listDevicesQuery struct {
	deviceRepo devicerepo.Repository
}

// NewListDevicesQuery builds the list-devices query.
func NewListDevicesQuery(deviceRepo devicerepo.Repository) IListDevicesQuery {
	return &listDevicesQuery{deviceRepo: deviceRepo}
}

func (q *listDevicesQuery) Handle(ctx context.Context, _ *devicedtos.ListDevicesReq) ([]*devicedtos.DeviceDTO, error) {
	devices, err := q.deviceRepo.FindAll(ctx)
	if err != nil {
		slog.Error("list devices failed", slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}
	return devicedtos.NewDeviceDTOs(devices), nil
}
