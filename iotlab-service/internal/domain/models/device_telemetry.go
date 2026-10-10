package models

import "time"

// DeviceTelemetry is one health report stored in the device_telemetry hypertable.
type DeviceTelemetry struct {
	Time           time.Time `json:"time"            db:"time"`
	DeviceID       string    `json:"device_id"       db:"device_id"`
	FW             *string   `json:"fw"              db:"fw"`
	UptimeS        *int64    `json:"uptime_s"        db:"uptime_s"`
	ResetReason    *string   `json:"reset_reason"    db:"reset_reason"`
	RSSIDbm        *int16    `json:"rssi_dbm"        db:"rssi_dbm"`
	IP             *string   `json:"ip"              db:"ip"`
	HeapFree       *int32    `json:"heap_free"       db:"heap_free"`
	HeapMin        *int32    `json:"heap_min"        db:"heap_min"`
	WifiReconnects *int32    `json:"wifi_reconnects" db:"wifi_reconnects"`
	MQTTReconnects *int32    `json:"mqtt_reconnects" db:"mqtt_reconnects"`
	SensorErrors   *int32    `json:"sensor_errors"   db:"sensor_errors"`
}

// DeviceHealth1h is a read-only row of the device_health_1h continuous aggregate.
// Aggregates are nullable because every source column is nullable.
type DeviceHealth1h struct {
	Bucket          time.Time `json:"bucket"            db:"bucket"`
	DeviceID        string    `json:"device_id"         db:"device_id"`
	AvgRSSIDbm      *float64  `json:"avg_rssi_dbm"      db:"avg_rssi_dbm"`
	MinRSSIDbm      *int16    `json:"min_rssi_dbm"      db:"min_rssi_dbm"`
	MinHeapFree     *int32    `json:"min_heap_free"     db:"min_heap_free"`
	MinHeapMin      *int32    `json:"min_heap_min"      db:"min_heap_min"`
	MaxUptimeS      *int64    `json:"max_uptime_s"      db:"max_uptime_s"`
	MaxSensorErrors *int32    `json:"max_sensor_errors" db:"max_sensor_errors"`
	Samples         int64     `json:"samples"           db:"samples"`
}
