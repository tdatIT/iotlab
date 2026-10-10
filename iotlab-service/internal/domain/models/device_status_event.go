package models

import "time"

// DeviceStatusEvent is one online/offline transition stored in the device_status_events hypertable.
type DeviceStatusEvent struct {
	Time     time.Time `json:"time"      db:"time"`
	DeviceID string    `json:"device_id" db:"device_id"`
	Status   string    `json:"status"    db:"status"`
}
