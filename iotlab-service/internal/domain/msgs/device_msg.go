package msgs

// Device payloads carry no device_id: it is the {device_id} segment of the topic.
// Time fields are Unix epoch milliseconds set by the device.
// The status topic `homelab/iot/devices/{device_id}/status` (retained) carries a plain-text body, "online" or
// "offline", so it has no struct here — compare it against models.DeviceStatusOnline / models.DeviceStatusOffline.

// SensorPayload is the wire format consumed on `homelab/iot/env/{device_id}/sensor`.
type SensorPayload struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
	Time int64   `json:"time"`
}

// TelemetryPayload is the wire format consumed on `homelab/iot/devices/{device_id}/telemetry`.
// Optional fields are pointers so a missing key maps to NULL in device_telemetry instead of zero.
type TelemetryPayload struct {
	Time           int64   `json:"time"`
	FW             *string `json:"fw"`
	UptimeS        *int64  `json:"uptime_s"`
	ResetReason    *string `json:"reset_reason"`
	RSSIDbm        *int16  `json:"rssi_dbm"`
	IP             *string `json:"ip"`
	HeapFree       *int32  `json:"heap_free"`
	HeapMin        *int32  `json:"heap_min"`
	WifiReconnects *int32  `json:"wifi_reconnects"`
	MQTTReconnects *int32  `json:"mqtt_reconnects"`
	SensorErrors   *int32  `json:"sensor_errors"`
}
