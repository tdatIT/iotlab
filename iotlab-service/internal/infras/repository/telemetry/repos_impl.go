package telemetry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
	"github.com/tdatIT/iotlab-service/pkgs/database"
)

const insertTelemetry = `INSERT INTO device_telemetry (time, device_id, fw, uptime_s, reset_reason, rssi_dbm, ip, heap_free, heap_min,
                              wifi_reconnects, mqtt_reconnects, sensor_errors)
VALUES ($1, $2, $3, $4, $5, $6, $7::inet, $8, $9, $10, $11, $12)`

// selectTelemetry renders ip as text so it scans into a plain string.
const selectTelemetry = `SELECT time, device_id, fw, uptime_s, reset_reason, rssi_dbm, host(ip) AS ip, heap_free, heap_min,
       wifi_reconnects, mqtt_reconnects, sensor_errors
FROM device_telemetry`

const updateTelemetry = `UPDATE device_telemetry SET
    fw              = COALESCE($3, fw),
    uptime_s        = COALESCE($4, uptime_s),
    reset_reason    = COALESCE($5, reset_reason),
    rssi_dbm        = COALESCE($6, rssi_dbm),
    ip              = COALESCE($7::inet, ip),
    heap_free       = COALESCE($8, heap_free),
    heap_min        = COALESCE($9, heap_min),
    wifi_reconnects = COALESCE($10, wifi_reconnects),
    mqtt_reconnects = COALESCE($11, mqtt_reconnects),
    sensor_errors   = COALESCE($12, sensor_errors)
WHERE time = $1 AND device_id = $2`

type telemetryReposImpl struct {
	db database.Querier
}

// NewTelemetryRepository returns the pgx-backed DeviceTelemetry repository.
func NewTelemetryRepository(db database.Querier) Repository {
	return &telemetryReposImpl{db: db}
}

func (r *telemetryReposImpl) Create(ctx context.Context, telemetry *models.DeviceTelemetry) error {
	_, err := r.db.Exec(ctx, insertTelemetry, args(telemetry)...)
	return database.Translate(err)
}

func (r *telemetryReposImpl) FindByKey(ctx context.Context, deviceID string, at time.Time) (*models.DeviceTelemetry, error) {
	return r.one(ctx, selectTelemetry+` WHERE device_id = $1 AND time = $2`, deviceID, at)
}

func (r *telemetryReposImpl) Update(ctx context.Context, telemetry *models.DeviceTelemetry) error {
	tag, err := r.db.Exec(ctx, updateTelemetry, args(telemetry)...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

func (r *telemetryReposImpl) Delete(ctx context.Context, deviceID string, at time.Time) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM device_telemetry WHERE device_id = $1 AND time = $2`, deviceID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

func (r *telemetryReposImpl) FindAllByDevice(
	ctx context.Context,
	deviceID string,
	window time.Duration,
) ([]*models.DeviceTelemetry, error) {
	rows, err := r.db.Query(ctx,
		selectTelemetry+` WHERE device_id = $1 AND time >= now() - $2::interval ORDER BY time`, deviceID, window)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.DeviceTelemetry])
}

func (r *telemetryReposImpl) FindLast6h(ctx context.Context, deviceID string) ([]*models.DeviceTelemetry, error) {
	return r.FindAllByDevice(ctx, deviceID, 6*time.Hour)
}

func (r *telemetryReposImpl) FindLastDay(ctx context.Context, deviceID string) ([]*models.DeviceTelemetry, error) {
	return r.FindAllByDevice(ctx, deviceID, 24*time.Hour)
}

func (r *telemetryReposImpl) FindCurrent(ctx context.Context, deviceID string) (*models.DeviceTelemetry, error) {
	return r.one(ctx, selectTelemetry+` WHERE device_id = $1 ORDER BY time DESC LIMIT 1`, deviceID)
}

func (r *telemetryReposImpl) one(ctx context.Context, sql string, params ...any) (*models.DeviceTelemetry, error) {
	rows, err := r.db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	t, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[models.DeviceTelemetry])
	if err != nil {
		return nil, database.Translate(err)
	}
	return t, nil
}

// args binds telemetry in the $1..$12 order shared by insertTelemetry and updateTelemetry.
func args(t *models.DeviceTelemetry) []any {
	return []any{
		t.Time, t.DeviceID, t.FW, t.UptimeS, t.ResetReason, t.RSSIDbm, t.IP,
		t.HeapFree, t.HeapMin, t.WifiReconnects, t.MQTTReconnects, t.SensorErrors,
	}
}
