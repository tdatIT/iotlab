package devicedtos

import (
	"encoding/json"
	"time"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
)

// ListDevicesReq is the (empty) request of the list-devices endpoint.
type ListDevicesReq struct{}

// DeviceIDReq carries the {id} path parameter of the per-device read endpoints.
type DeviceIDReq struct {
	DeviceID string `param:"id" validate:"required,max=64"`
}

// TelemetryStatsReq selects the telemetry window of a device.
type TelemetryStatsReq struct {
	DeviceID string `param:"id"    validate:"required,max=64"`
	Range    string `query:"range" validate:"required,oneof=15m 30m 60m"`
}

// SensorStatsReq selects the sensor window of a device.
type SensorStatsReq struct {
	DeviceID string `param:"id"    validate:"required,max=64"`
	Range    string `query:"range" validate:"required,oneof=30m 60m 90m"`
}

// Stat is the min/max/avg of one metric over a window.
type Stat struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// DeviceDTO is the device representation returned to API callers.
type DeviceDTO struct {
	DeviceID        string          `json:"device_id"`
	Name            *string         `json:"name"`
	Location        *string         `json:"location"`
	Status          string          `json:"status"`
	StatusChangedAt time.Time       `json:"status_changed_at"`
	LastSeen        time.Time       `json:"last_seen"`
	IP              *string         `json:"ip"`
	FW              *string         `json:"fw"`
	FirstSeen       time.Time       `json:"first_seen"`
	Metadata        json.RawMessage `json:"metadata"          swaggertype:"object"`
}

// TelemetryDTO is one telemetry report of a device.
type TelemetryDTO struct {
	Time           time.Time `json:"time"`
	FW             *string   `json:"fw"`
	UptimeS        *int64    `json:"uptime_s"`
	ResetReason    *string   `json:"reset_reason"`
	RSSIDbm        *int16    `json:"rssi_dbm"`
	IP             *string   `json:"ip"`
	HeapFree       *int32    `json:"heap_free"`
	HeapMin        *int32    `json:"heap_min"`
	WifiReconnects *int32    `json:"wifi_reconnects"`
	MQTTReconnects *int32    `json:"mqtt_reconnects"`
	SensorErrors   *int32    `json:"sensor_errors"`
}

// TelemetrySummary aggregates the reports of a window. A stat is null when no report in the window carries the metric.
type TelemetrySummary struct {
	RSSIDbm  *Stat `json:"rssi_dbm"`
	HeapFree *Stat `json:"heap_free"`
	HeapMin  *Stat `json:"heap_min"`
	Samples  int64 `json:"samples"`
}

// TelemetryStatsDTO is the telemetry series of a window plus its summary.
type TelemetryStatsDTO struct {
	Range   string           `json:"range"`
	Summary TelemetrySummary `json:"summary"`
	Points  []*TelemetryDTO  `json:"points"`
}

// SensorDTO is one temperature/humidity reading of a device.
type SensorDTO struct {
	Time        time.Time `json:"time"`
	TempC       float64   `json:"temp_c"`
	HumidityPct float64   `json:"humidity_pct"`
}

// SensorSummary aggregates the readings of a window. Stats are null when the window has no reading.
type SensorSummary struct {
	TempC       *Stat `json:"temp_c"`
	HumidityPct *Stat `json:"humidity_pct"`
	Samples     int64 `json:"samples"`
}

// SensorStatsDTO is the raw sensor series of a window plus its summary.
type SensorStatsDTO struct {
	Range   string        `json:"range"`
	Summary SensorSummary `json:"summary"`
	Points  []*SensorDTO  `json:"points"`
}

// SensorBucketDTO is one 30-minute aggregate of a device's readings.
type SensorBucketDTO struct {
	Bucket  time.Time `json:"bucket"`
	AvgC    float64   `json:"avg_c"`
	MinC    float64   `json:"min_c"`
	MaxC    float64   `json:"max_c"`
	AvgH    float64   `json:"avg_h"`
	MinH    float64   `json:"min_h"`
	MaxH    float64   `json:"max_h"`
	Samples int64     `json:"samples"`
}

// SensorStats24hDTO is the 30-minute bucket series of the last 24h plus its summary.
type SensorStats24hDTO struct {
	Range   string             `json:"range"`
	Summary SensorSummary      `json:"summary"`
	Points  []*SensorBucketDTO `json:"points"`
}

// SensorAverageDTO is the average temperature/humidity of the last 24h. Averages are null when the window has no reading.
type SensorAverageDTO struct {
	Range          string   `json:"range"`
	AvgTempC       *float64 `json:"avg_temp_c"`
	AvgHumidityPct *float64 `json:"avg_humidity_pct"`
	Samples        int64    `json:"samples"`
}

// NewDeviceDTO maps a model to its DTO, returning nil for a nil model.
func NewDeviceDTO(d *models.Device) *DeviceDTO {
	if d == nil {
		return nil
	}
	return &DeviceDTO{
		DeviceID:        d.DeviceID,
		Name:            d.Name,
		Location:        d.Location,
		Status:          d.Status,
		StatusChangedAt: d.StatusChangedAt,
		LastSeen:        d.LastSeen,
		IP:              d.IP,
		FW:              d.FW,
		FirstSeen:       d.FirstSeen,
		Metadata:        json.RawMessage(d.Metadata),
	}
}

// NewDeviceDTOs maps a slice of models, returning an empty (never nil) slice so the JSON payload is always an array.
func NewDeviceDTOs(devices []*models.Device) []*DeviceDTO {
	result := make([]*DeviceDTO, 0, len(devices))
	for _, d := range devices {
		result = append(result, NewDeviceDTO(d))
	}
	return result
}

// NewTelemetryDTO maps a model to its DTO, returning nil for a nil model.
func NewTelemetryDTO(t *models.DeviceTelemetry) *TelemetryDTO {
	if t == nil {
		return nil
	}
	return &TelemetryDTO{
		Time:           t.Time,
		FW:             t.FW,
		UptimeS:        t.UptimeS,
		ResetReason:    t.ResetReason,
		RSSIDbm:        t.RSSIDbm,
		IP:             t.IP,
		HeapFree:       t.HeapFree,
		HeapMin:        t.HeapMin,
		WifiReconnects: t.WifiReconnects,
		MQTTReconnects: t.MQTTReconnects,
		SensorErrors:   t.SensorErrors,
	}
}

// NewTelemetryDTOs maps a slice of models, returning an empty (never nil) slice.
func NewTelemetryDTOs(rows []*models.DeviceTelemetry) []*TelemetryDTO {
	result := make([]*TelemetryDTO, 0, len(rows))
	for _, r := range rows {
		result = append(result, NewTelemetryDTO(r))
	}
	return result
}

// NewSensorDTO maps a model to its DTO, returning nil for a nil model.
func NewSensorDTO(s *models.SensorRaw) *SensorDTO {
	if s == nil {
		return nil
	}
	return &SensorDTO{Time: s.Time, TempC: s.TempC, HumidityPct: s.HumidityPct}
}

// NewSensorDTOs maps a slice of models, returning an empty (never nil) slice.
func NewSensorDTOs(rows []*models.SensorRaw) []*SensorDTO {
	result := make([]*SensorDTO, 0, len(rows))
	for _, r := range rows {
		result = append(result, NewSensorDTO(r))
	}
	return result
}

// NewSensorBucketDTOs maps a slice of bucket models, returning an empty (never nil) slice.
func NewSensorBucketDTOs(buckets []*models.Sensor30m) []*SensorBucketDTO {
	result := make([]*SensorBucketDTO, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, &SensorBucketDTO{
			Bucket:  b.Bucket,
			AvgC:    b.AvgC,
			MinC:    b.MinC,
			MaxC:    b.MaxC,
			AvgH:    b.AvgH,
			MinH:    b.MinH,
			MaxH:    b.MaxH,
			Samples: b.Samples,
		})
	}
	return result
}
