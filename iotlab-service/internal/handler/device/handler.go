// Package device is the HTTP transport for the IoT devices. Its handlers bind, validate, delegate to the application layer,
// and let the global error handler render any svcerr.
package device

import (
	"log/slog"

	"github.com/labstack/echo/v5"

	"github.com/tdatIT/iotlab-service/internal/app/device"
	"github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/response"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// Handler serves the device HTTP endpoints.
type Handler struct {
	app *device.Application
}

// NewDeviceHandler builds the device HTTP handler.
func NewDeviceHandler(app *device.Application) *Handler {
	return &Handler{app: app}
}

// ListDevices handles GET /api/v1/devices.
//
//	@Summary		List devices
//	@Description	Returns every registered device ordered by device id.
//	@Tags			devices
//	@Produce		json
//	@Success		200	{object}	response.BaseRes{data=[]devicedtos.DeviceDTO}	"Devices"
//	@Failure		429	{object}	response.BaseRes								"Rate limit exceeded"
//	@Failure		500	{object}	response.BaseRes								"Internal error"
//	@Router			/devices [get]
func (h *Handler) ListDevices(c *echo.Context) error {
	data, err := h.app.Query.ListDevices.Handle(c.Request().Context(), &devicedtos.ListDevicesReq{})
	if err != nil {
		return err
	}
	return success(c, data)
}

// GetCurrentTelemetry handles GET /api/v1/devices/:id/telemetry/current.
//
//	@Summary		Get current telemetry
//	@Description	Returns the newest telemetry report of the device.
//	@Tags			devices
//	@Produce		json
//	@Param			id	path		string											true	"Device ID"
//	@Success		200	{object}	response.BaseRes{data=devicedtos.TelemetryDTO}	"Newest report"
//	@Failure		400	{object}	response.BaseRes								"Invalid device id"
//	@Failure		404	{object}	response.BaseRes								"Device has no telemetry"
//	@Failure		429	{object}	response.BaseRes								"Rate limit exceeded"
//	@Failure		500	{object}	response.BaseRes								"Internal error"
//	@Router			/devices/{id}/telemetry/current [get]
func (h *Handler) GetCurrentTelemetry(c *echo.Context) error {
	var req devicedtos.DeviceIDReq
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	data, err := h.app.Query.GetCurrentTelemetry.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return success(c, data)
}

// GetTelemetryStats handles GET /api/v1/devices/:id/telemetry/stats.
//
//	@Summary		Get telemetry stats
//	@Description	Returns the telemetry series of the window plus an RSSI/heap min/max/avg summary.
//	@Tags			devices
//	@Produce		json
//	@Param			id		path		string												true	"Device ID"
//	@Param			range	query		string												true	"Window"	Enums(15m, 30m, 60m)
//	@Success		200		{object}	response.BaseRes{data=devicedtos.TelemetryStatsDTO}	"Series and summary"
//	@Failure		400		{object}	response.BaseRes									"Invalid device id or range"
//	@Failure		429		{object}	response.BaseRes									"Rate limit exceeded"
//	@Failure		500		{object}	response.BaseRes									"Internal error"
//	@Router			/devices/{id}/telemetry/stats [get]
func (h *Handler) GetTelemetryStats(c *echo.Context) error {
	var req devicedtos.TelemetryStatsReq
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	data, err := h.app.Query.GetTelemetryStats.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return success(c, data)
}

// GetCurrentSensor handles GET /api/v1/devices/:id/sensor/current.
//
//	@Summary		Get current temperature/humidity
//	@Description	Returns the newest sensor reading of the device.
//	@Tags			devices
//	@Produce		json
//	@Param			id	path		string										true	"Device ID"
//	@Success		200	{object}	response.BaseRes{data=devicedtos.SensorDTO}	"Newest reading"
//	@Failure		400	{object}	response.BaseRes							"Invalid device id"
//	@Failure		404	{object}	response.BaseRes							"Device has no reading"
//	@Failure		429	{object}	response.BaseRes							"Rate limit exceeded"
//	@Failure		500	{object}	response.BaseRes							"Internal error"
//	@Router			/devices/{id}/sensor/current [get]
func (h *Handler) GetCurrentSensor(c *echo.Context) error {
	var req devicedtos.DeviceIDReq
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	data, err := h.app.Query.GetCurrentSensor.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return success(c, data)
}

// GetSensorStats handles GET /api/v1/devices/:id/sensor/stats.
//
//	@Summary		Get temperature/humidity stats
//	@Description	Returns the raw sensor series of the window plus a min/max/avg summary.
//	@Tags			devices
//	@Produce		json
//	@Param			id		path		string											true	"Device ID"
//	@Param			range	query		string											true	"Window"	Enums(30m, 60m, 90m)
//	@Success		200		{object}	response.BaseRes{data=devicedtos.SensorStatsDTO}	"Series and summary"
//	@Failure		400		{object}	response.BaseRes								"Invalid device id or range"
//	@Failure		429		{object}	response.BaseRes								"Rate limit exceeded"
//	@Failure		500		{object}	response.BaseRes								"Internal error"
//	@Router			/devices/{id}/sensor/stats [get]
func (h *Handler) GetSensorStats(c *echo.Context) error {
	var req devicedtos.SensorStatsReq
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	data, err := h.app.Query.GetSensorStats.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return success(c, data)
}

// GetSensorAverage24h handles GET /api/v1/devices/:id/sensor/average-24h.
//
//	@Summary		Get 24h average temperature/humidity
//	@Description	Returns the average temperature and humidity of the last 24h, computed from the 30-minute buckets.
//	@Tags			devices
//	@Produce		json
//	@Param			id	path		string												true	"Device ID"
//	@Success		200	{object}	response.BaseRes{data=devicedtos.SensorAverageDTO}	"Averages"
//	@Failure		400	{object}	response.BaseRes									"Invalid device id"
//	@Failure		429	{object}	response.BaseRes									"Rate limit exceeded"
//	@Failure		500	{object}	response.BaseRes									"Internal error"
//	@Router			/devices/{id}/sensor/average-24h [get]
func (h *Handler) GetSensorAverage24h(c *echo.Context) error {
	var req devicedtos.DeviceIDReq
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	data, err := h.app.Query.GetSensorAverage24h.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return success(c, data)
}

// GetSensorStats24h handles GET /api/v1/devices/:id/sensor/stats-24h.
//
//	@Summary		Get 24h temperature/humidity stats
//	@Description	Returns the 30-minute buckets of the last 24h plus a min/max/avg summary.
//	@Tags			devices
//	@Produce		json
//	@Param			id	path		string												true	"Device ID"
//	@Success		200	{object}	response.BaseRes{data=devicedtos.SensorStats24hDTO}	"Buckets and summary"
//	@Failure		400	{object}	response.BaseRes									"Invalid device id"
//	@Failure		429	{object}	response.BaseRes									"Rate limit exceeded"
//	@Failure		500	{object}	response.BaseRes									"Internal error"
//	@Router			/devices/{id}/sensor/stats-24h [get]
func (h *Handler) GetSensorStats24h(c *echo.Context) error {
	var req devicedtos.DeviceIDReq
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	data, err := h.app.Query.GetSensorStats24h.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	return success(c, data)
}

func bindAndValidate(c *echo.Context, req any) error {
	if err := c.Bind(req); err != nil {
		slog.Error("bind request failed", slog.String("error", err.Error()))
		return svcerr.ErrBadRequest
	}
	if err := c.Validate(req); err != nil {
		slog.Error("validate request failed", slog.String("error", err.Error()))
		return err
	}
	return nil
}

func success(c *echo.Context, data any) error {
	res := response.SuccessRes
	res.Data = data
	return res.JSON(c)
}
