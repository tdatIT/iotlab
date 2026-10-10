package models

import "time"

// SensorRaw is one DHT22 reading stored in the sensor_raw hypertable.
type SensorRaw struct {
	Time        time.Time `json:"time"         db:"time"`
	DeviceID    string    `json:"device_id"    db:"device_id"`
	TempC       float64   `json:"temp_c"       db:"temp_c"`
	HumidityPct float64   `json:"humidity_pct" db:"humidity_pct"`
}

// Sensor30m is a read-only row of the sensor_30m continuous aggregate.
type Sensor30m struct {
	Bucket   time.Time `json:"bucket"    db:"bucket"`
	DeviceID string    `json:"device_id" db:"device_id"`
	AvgC     float64   `json:"avg_c"     db:"avg_c"`
	MinC     float64   `json:"min_c"     db:"min_c"`
	MaxC     float64   `json:"max_c"     db:"max_c"`
	AvgH     float64   `json:"avg_h"     db:"avg_h"`
	MinH     float64   `json:"min_h"     db:"min_h"`
	MaxH     float64   `json:"max_h"     db:"max_h"`
	Samples  int64     `json:"samples"   db:"samples"`
}
