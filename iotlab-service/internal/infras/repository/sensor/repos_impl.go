package sensor

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
	"github.com/tdatIT/iotlab-service/pkgs/database"
)

const selectSensor = `SELECT time, device_id, temp_c, humidity_pct FROM sensor_raw`

type sensorReposImpl struct {
	db database.Querier
}

// NewSensorRepository returns the pgx-backed SensorRaw repository.
func NewSensorRepository(db database.Querier) Repository {
	return &sensorReposImpl{db: db}
}

func (r *sensorReposImpl) Create(ctx context.Context, reading *models.SensorRaw) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO sensor_raw (time, device_id, temp_c, humidity_pct) VALUES ($1, $2, $3, $4)`,
		reading.Time, reading.DeviceID, reading.TempC, reading.HumidityPct)
	return database.Translate(err)
}

func (r *sensorReposImpl) FindByKey(ctx context.Context, deviceID string, at time.Time) (*models.SensorRaw, error) {
	return r.one(ctx, selectSensor+` WHERE device_id = $1 AND time = $2`, deviceID, at)
}

// Update overwrites only the non-zero readings; a reading of exactly 0 cannot be written back through it.
func (r *sensorReposImpl) Update(ctx context.Context, reading *models.SensorRaw) error {
	tag, err := r.db.Exec(ctx, `UPDATE sensor_raw SET
    temp_c       = COALESCE(NULLIF($3, 0::double precision), temp_c),
    humidity_pct = COALESCE(NULLIF($4, 0::double precision), humidity_pct)
WHERE device_id = $1 AND time = $2`,
		reading.DeviceID, reading.Time, reading.TempC, reading.HumidityPct)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

func (r *sensorReposImpl) Delete(ctx context.Context, deviceID string, at time.Time) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM sensor_raw WHERE device_id = $1 AND time = $2`, deviceID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

func (r *sensorReposImpl) FindAllByDevice(
	ctx context.Context,
	deviceID string,
	window time.Duration,
) ([]*models.SensorRaw, error) {
	rows, err := r.db.Query(ctx,
		selectSensor+` WHERE device_id = $1 AND time >= now() - $2::interval ORDER BY time`, deviceID, window)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.SensorRaw])
}

func (r *sensorReposImpl) FindLast6h(ctx context.Context, deviceID string) ([]*models.SensorRaw, error) {
	return r.FindAllByDevice(ctx, deviceID, 6*time.Hour)
}

func (r *sensorReposImpl) FindLastDay(ctx context.Context, deviceID string) ([]*models.SensorRaw, error) {
	return r.FindAllByDevice(ctx, deviceID, 24*time.Hour)
}

func (r *sensorReposImpl) FindBuckets30m(
	ctx context.Context,
	deviceID string,
	window time.Duration,
) ([]*models.Sensor30m, error) {
	rows, err := r.db.Query(ctx, `SELECT bucket, device_id, avg_c, min_c, max_c, avg_h, min_h, max_h, samples
FROM sensor_30m
WHERE device_id = $1 AND bucket >= now() - $2::interval
ORDER BY bucket`, deviceID, window)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.Sensor30m])
}

func (r *sensorReposImpl) FindCurrent(ctx context.Context, deviceID string) (*models.SensorRaw, error) {
	return r.one(ctx, selectSensor+` WHERE device_id = $1 ORDER BY time DESC LIMIT 1`, deviceID)
}

func (r *sensorReposImpl) one(ctx context.Context, sql string, params ...any) (*models.SensorRaw, error) {
	rows, err := r.db.Query(ctx, sql, params...)
	if err != nil {
		return nil, err
	}
	s, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[models.SensorRaw])
	if err != nil {
		return nil, database.Translate(err)
	}
	return s, nil
}
