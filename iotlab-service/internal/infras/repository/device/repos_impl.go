package device

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
	"github.com/tdatIT/iotlab-service/pkgs/database"
)

// insertDevice falls back to the column defaults for zero values: NULLIF turns an empty string into NULL and COALESCE picks
// the default.
const insertDevice = `INSERT INTO devices (device_id, name, location, status, status_changed_at, last_seen, ip, fw, first_seen, metadata)
VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'offline'), COALESCE($5, now()), COALESCE($6, now()), $7::inet, $8,
        COALESCE($9, now()), COALESCE(NULLIF($10, '')::jsonb, '{}'))`

// selectDevice renders ip and metadata as text so they scan into plain strings.
const selectDevice = `SELECT device_id, name, location, status, status_changed_at, last_seen, host(ip) AS ip, fw, first_seen,
       metadata::text AS metadata
FROM devices`

const updateDevice = `UPDATE devices SET
    name              = COALESCE($2, name),
    location          = COALESCE($3, location),
    status            = COALESCE(NULLIF($4, ''), status),
    status_changed_at = COALESCE($5, status_changed_at),
    last_seen         = COALESCE($6, last_seen),
    ip                = COALESCE($7::inet, ip),
    fw                = COALESCE($8, fw),
    first_seen        = COALESCE($9, first_seen),
    metadata          = COALESCE(NULLIF($10, '')::jsonb, metadata)
WHERE device_id = $1`

type deviceReposImpl struct {
	db database.Querier
}

// NewDeviceRepository returns the pgx-backed Device repository.
func NewDeviceRepository(db database.Querier) Repository {
	return &deviceReposImpl{db: db}
}

func (r *deviceReposImpl) Create(ctx context.Context, device *models.Device) error {
	_, err := r.db.Exec(ctx, insertDevice, args(device)...)
	return database.Translate(err)
}

func (r *deviceReposImpl) FindByID(ctx context.Context, deviceID string) (*models.Device, error) {
	rows, err := r.db.Query(ctx, selectDevice+` WHERE device_id = $1`, deviceID)
	if err != nil {
		return nil, err
	}
	d, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[models.Device])
	if err != nil {
		return nil, database.Translate(err)
	}
	return d, nil
}

func (r *deviceReposImpl) FindAll(ctx context.Context) ([]*models.Device, error) {
	rows, err := r.db.Query(ctx, selectDevice+` ORDER BY device_id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[models.Device])
}

func (r *deviceReposImpl) Update(ctx context.Context, device *models.Device) error {
	tag, err := r.db.Exec(ctx, updateDevice, args(device)...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

func (r *deviceReposImpl) Delete(ctx context.Context, deviceID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM devices WHERE device_id = $1`, deviceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// Upsert quotes every column name, so a caller can never inject SQL through columns.
func (r *deviceReposImpl) Upsert(ctx context.Context, device *models.Device, columns ...string) error {
	conflict := ` ON CONFLICT (device_id) DO NOTHING`
	if len(columns) > 0 {
		set := make([]string, 0, len(columns))
		for _, c := range columns {
			col := pgx.Identifier{c}.Sanitize()
			set = append(set, col+" = EXCLUDED."+col)
		}
		conflict = ` ON CONFLICT (device_id) DO UPDATE SET ` + strings.Join(set, ", ")
	}

	_, err := r.db.Exec(ctx, insertDevice+conflict, args(device)...)
	return database.Translate(err)
}

// args binds device in the $1..$10 order shared by insertDevice and updateDevice. Zero times become NULL.
func args(d *models.Device) []any {
	return []any{
		d.DeviceID, d.Name, d.Location, d.Status, nullTime(d.StatusChangedAt), nullTime(d.LastSeen),
		d.IP, d.FW, nullTime(d.FirstSeen), d.Metadata,
	}
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
