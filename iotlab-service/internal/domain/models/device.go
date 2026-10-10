package models

import "time"

// Device statuses allowed by the devices.status and device_status_events.status CHECK constraints.
const (
	DeviceStatusOnline  = "online"
	DeviceStatusOffline = "offline"
)

// Device is the registry row of a device and holds its current state only. The db tags name the columns pgx scans into.
type Device struct {
	DeviceID        string    `json:"device_id"         db:"device_id"`
	Name            *string   `json:"name"              db:"name"`
	Location        *string   `json:"location"          db:"location"`
	Status          string    `json:"status"            db:"status"`
	StatusChangedAt time.Time `json:"status_changed_at" db:"status_changed_at"`
	LastSeen        time.Time `json:"last_seen"         db:"last_seen"`
	IP              *string   `json:"ip"                db:"ip"`
	FW              *string   `json:"fw"                db:"fw"`
	FirstSeen       time.Time `json:"first_seen"        db:"first_seen"`
	Metadata        string    `json:"metadata"          db:"metadata"`
}
