package device_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/domain/models"
	devicerepo "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	"github.com/tdatIT/iotlab-service/pkgs/database"
)

var (
	ctx     = context.Background()
	dbErr   = errors.New("db down")
	columns = []string{
		"device_id", "name", "location", "status", "status_changed_at", "last_seen", "ip", "fw", "first_seen", "metadata",
	}
	seen = time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
)

// newRepo returns the repository over a pgxmock pool that fails the test on any unmet expectation.
func newRepo(t *testing.T) (devicerepo.Repository, pgxmock.PgxPoolIface) {
	t.Helper()

	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.ExpectationsWereMet()) })
	return devicerepo.NewDeviceRepository(pool), pool
}

func deviceRow(id string) []any {
	return []any{id, new("kitchen"), (*string)(nil), "online", seen, seen, new("192.168.1.156"), new("1.2.0"), seen, `{}`}
}

func TestDeviceRepository_Create(t *testing.T) {
	t.Run("binds the device and lets zero values take the column defaults", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`INSERT INTO devices .* COALESCE\(NULLIF\(\$4, ''\), 'offline'\)`).
			WithArgs("esp-1", (*string)(nil), (*string)(nil), "", (*time.Time)(nil), (*time.Time)(nil),
				(*string)(nil), (*string)(nil), (*time.Time)(nil), "").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		require.NoError(t, repo.Create(ctx, &models.Device{DeviceID: "esp-1"}))
	})

	t.Run("unique violation returns database.ErrDuplicate", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`INSERT INTO devices`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		assert.ErrorIs(t, repo.Create(ctx, &models.Device{DeviceID: "esp-1"}), database.ErrDuplicate)
	})
}

func TestDeviceRepository_FindByID(t *testing.T) {
	t.Run("existing row is scanned", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`SELECT .* host\(ip\) AS ip.* FROM devices WHERE device_id = \$1`).WithArgs("esp-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow(deviceRow("esp-1")...))

		got, err := repo.FindByID(ctx, "esp-1")

		require.NoError(t, err)
		assert.Equal(t, "esp-1", got.DeviceID)
		assert.Equal(t, "kitchen", *got.Name)
		assert.Nil(t, got.Location)
		assert.Equal(t, "192.168.1.156", *got.IP)
		assert.True(t, seen.Equal(got.LastSeen))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM devices WHERE device_id = \$1`).WithArgs("nope").WillReturnRows(pgxmock.NewRows(columns))

		got, err := repo.FindByID(ctx, "nope")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, database.ErrNotFound)
	})

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM devices`).WithArgs("esp-1").WillReturnError(dbErr)

		_, err := repo.FindByID(ctx, "esp-1")

		assert.ErrorIs(t, err, dbErr)
	})
}

func TestDeviceRepository_FindAll(t *testing.T) {
	t.Run("returns every device ordered by id", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM devices ORDER BY device_id`).
			WillReturnRows(pgxmock.NewRows(columns).AddRow(deviceRow("esp-1")...).AddRow(deviceRow("esp-2")...))

		got, err := repo.FindAll(ctx)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "esp-2", got[1].DeviceID)
	})

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM devices`).WillReturnError(dbErr)

		got, err := repo.FindAll(ctx)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
	})
}

func TestDeviceRepository_Update(t *testing.T) {
	t.Run("COALESCE keeps the columns of nil fields", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE devices SET\s+name\s+= COALESCE\(\$2, name\)`).
			WithArgs("esp-1", new("kitchen"), (*string)(nil), "", (*time.Time)(nil), (*time.Time)(nil),
				(*string)(nil), (*string)(nil), (*time.Time)(nil), "").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		require.NoError(t, repo.Update(ctx, &models.Device{DeviceID: "esp-1", Name: new("kitchen")}))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE devices`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		assert.ErrorIs(t, repo.Update(ctx, &models.Device{DeviceID: "nope"}), database.ErrNotFound)
	})

	t.Run("exec failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE devices`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(dbErr)

		assert.ErrorIs(t, repo.Update(ctx, &models.Device{DeviceID: "esp-1"}), dbErr)
	})
}

func TestDeviceRepository_Delete(t *testing.T) {
	t.Run("existing row is removed", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM devices WHERE device_id = \$1`).WithArgs("esp-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		require.NoError(t, repo.Delete(ctx, "esp-1"))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM devices`).WithArgs("nope").WillReturnResult(pgxmock.NewResult("DELETE", 0))

		assert.ErrorIs(t, repo.Delete(ctx, "nope"), database.ErrNotFound)
	})

	t.Run("exec failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM devices`).WithArgs("esp-1").WillReturnError(dbErr)

		assert.ErrorIs(t, repo.Delete(ctx, "esp-1"), dbErr)
	})
}

func TestDeviceRepository_Upsert(t *testing.T) {
	anyArgs := []any{pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()}

	t.Run("conflict overwrites only the quoted listed columns", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`INSERT INTO devices .* ON CONFLICT \(device_id\) DO UPDATE SET "last_seen" = EXCLUDED\."last_seen", ` +
			`"fw" = EXCLUDED\."fw"$`).
			WithArgs(anyArgs...).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		require.NoError(t, repo.Upsert(ctx, &models.Device{DeviceID: "esp-1", LastSeen: seen}, "last_seen", "fw"))
	})

	t.Run("no columns leaves an existing row untouched", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`ON CONFLICT \(device_id\) DO NOTHING$`).WithArgs(anyArgs...).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		require.NoError(t, repo.Upsert(ctx, &models.Device{DeviceID: "esp-1"}))
	})

	t.Run("a hostile column name stays a quoted identifier", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DO UPDATE SET "x"";DROP TABLE devices;--" = EXCLUDED\."x"";DROP TABLE devices;--"$`).
			WithArgs(anyArgs...).
			WillReturnError(&pgconn.PgError{Code: "42703"})

		require.Error(t, repo.Upsert(ctx, &models.Device{DeviceID: "esp-1"}, `x";DROP TABLE devices;--`))
	})
}
