// Package devicedtos holds the device requests and responses exchanged with the app layer. Workers and HTTP handlers validate
// the tags at their boundary; the app layer trusts these constraints and does not re-validate them.
package devicedtos

// RecordSensorReq is one DHT22 reading of a device. Time is Unix epoch milliseconds.
type RecordSensorReq struct {
	DeviceID    string  `validate:"required"`
	TempC       float64 `validate:"gte=-40,lte=80"`
	HumidityPct float64 `validate:"gte=0,lte=100"`
	Time        int64   `validate:"gt=0"`
}

// RecordTelemetryReq is one health report of a device. Time is Unix epoch milliseconds; nil fields are stored as NULL.
type RecordTelemetryReq struct {
	DeviceID       string `validate:"required"`
	Time           int64  `validate:"gt=0"`
	FW             *string
	UptimeS        *int64
	ResetReason    *string
	RSSIDbm        *int16
	IP             *string `validate:"omitempty,ip"`
	HeapFree       *int32
	HeapMin        *int32
	WifiReconnects *int32
	MQTTReconnects *int32
	SensorErrors   *int32
}

// UpdateDeviceStatusReq is an online/offline transition reported for a device.
type UpdateDeviceStatusReq struct {
	DeviceID string `validate:"required"`
	Status   string `validate:"oneof=online offline"`
}
