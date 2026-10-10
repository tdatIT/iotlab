package telemetry_test

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
	telemetryrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry"
	"github.com/tdatIT/iotlab-service/pkgs/database"
)

var (
	ctx     = context.Background()
	dbErr   = errors.New("db down")
	at      = time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	columns = []string{
		"time", "device_id", "fw", "uptime_s", "reset_reason", "rssi_dbm", "ip", "heap_free", "heap_min",
		"wifi_reconnects", "mqtt_reconnects", "sensor_errors",
	}
	anyArgs = []any{pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		pgxmock.AnyArg()}
)

// newRepo returns the repository over a pgxmock pool that fails the test on any unmet expectation.
func newRepo(t *testing.T) (telemetryrepo.Repository, pgxmock.PgxPoolIface) {
	t.Helper()

	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.ExpectationsWereMet()) })
	return telemetryrepo.NewTelemetryRepository(pool), pool
}

func row(ts time.Time) []any {
	return []any{ts, "esp-1", new("1.2.0"), new(int64(10443)), new("poweron"), new(int16(-48)), new("192.168.1.156"),
		new(int32(226368)), new(int32(143620)), new(int32(0)), new(int32(0)), (*int32)(nil)}
}

func TestTelemetryRepository_Create(t *testing.T) {
	t.Run("binds every column in order", func(t *testing.T) {
		repo, pool := newRepo(t)
		report := &models.DeviceTelemetry{Time: at, DeviceID: "esp-1", RSSIDbm: new(int16(-48)), IP: new("192.168.1.156")}
		pool.ExpectExec(`INSERT INTO device_telemetry .* \$7::inet`).
			WithArgs(at, "esp-1", (*string)(nil), (*int64)(nil), (*string)(nil), report.RSSIDbm, report.IP,
				(*int32)(nil), (*int32)(nil), (*int32)(nil), (*int32)(nil), (*int32)(nil)).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		require.NoError(t, repo.Create(ctx, report))
	})

	t.Run("primary key violation returns database.ErrDuplicate", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`INSERT INTO device_telemetry`).WithArgs(anyArgs...).WillReturnError(&pgconn.PgError{Code: "23505"})

		assert.ErrorIs(t, repo.Create(ctx, &models.DeviceTelemetry{Time: at, DeviceID: "esp-1"}), database.ErrDuplicate)
	})
}

func TestTelemetryRepository_FindByKey(t *testing.T) {
	t.Run("existing row is scanned", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`host\(ip\) AS ip.* WHERE device_id = \$1 AND time = \$2`).WithArgs("esp-1", at).
			WillReturnRows(pgxmock.NewRows(columns).AddRow(row(at)...))

		got, err := repo.FindByKey(ctx, "esp-1", at)

		require.NoError(t, err)
		assert.Equal(t, int16(-48), *got.RSSIDbm)
		assert.Nil(t, got.SensorErrors)
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`WHERE device_id = \$1 AND time = \$2`).WithArgs("esp-1", at).WillReturnRows(pgxmock.NewRows(columns))

		_, err := repo.FindByKey(ctx, "esp-1", at)

		assert.ErrorIs(t, err, database.ErrNotFound)
	})

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM device_telemetry`).WithArgs("esp-1", at).WillReturnError(dbErr)

		_, err := repo.FindByKey(ctx, "esp-1", at)

		assert.ErrorIs(t, err, dbErr)
	})
}

func TestTelemetryRepository_Update(t *testing.T) {
	t.Run("COALESCE keeps the columns of nil fields", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE device_telemetry SET .* WHERE time = \$1 AND device_id = \$2`).WithArgs(anyArgs...).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		require.NoError(t, repo.Update(ctx, &models.DeviceTelemetry{Time: at, DeviceID: "esp-1", SensorErrors: new(int32(5))}))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE device_telemetry`).WithArgs(anyArgs...).WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		assert.ErrorIs(t, repo.Update(ctx, &models.DeviceTelemetry{Time: at, DeviceID: "esp-1"}), database.ErrNotFound)
	})

	t.Run("exec failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE device_telemetry`).WithArgs(anyArgs...).WillReturnError(dbErr)

		assert.ErrorIs(t, repo.Update(ctx, &models.DeviceTelemetry{Time: at, DeviceID: "esp-1"}), dbErr)
	})
}

func TestTelemetryRepository_Delete(t *testing.T) {
	t.Run("existing row is removed", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM device_telemetry WHERE device_id = \$1 AND time = \$2`).WithArgs("esp-1", at).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		require.NoError(t, repo.Delete(ctx, "esp-1", at))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM device_telemetry`).WithArgs("esp-1", at).WillReturnResult(pgxmock.NewResult("DELETE", 0))

		assert.ErrorIs(t, repo.Delete(ctx, "esp-1", at), database.ErrNotFound)
	})

	t.Run("exec failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM device_telemetry`).WithArgs("esp-1", at).WillReturnError(dbErr)

		assert.ErrorIs(t, repo.Delete(ctx, "esp-1", at), dbErr)
	})
}

func TestTelemetryRepository_RangeReads(t *testing.T) {
	cases := []struct {
		name   string
		window time.Duration
		find   func(telemetryrepo.Repository) ([]*models.DeviceTelemetry, error)
	}{
		{"15m window", 15 * time.Minute, func(r telemetryrepo.Repository) ([]*models.DeviceTelemetry, error) {
			return r.FindAllByDevice(ctx, "esp-1", 15*time.Minute)
		}},
		{"last 6h", 6 * time.Hour, func(r telemetryrepo.Repository) ([]*models.DeviceTelemetry, error) {
			return r.FindLast6h(ctx, "esp-1")
		}},
		{"last day", 24 * time.Hour, func(r telemetryrepo.Repository) ([]*models.DeviceTelemetry, error) {
			return r.FindLastDay(ctx, "esp-1")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" passes the window as an interval, oldest first", func(t *testing.T) {
			repo, pool := newRepo(t)
			pool.ExpectQuery(`WHERE device_id = \$1 AND time >= now\(\) - \$2::interval ORDER BY time$`).
				WithArgs("esp-1", tc.window).
				WillReturnRows(pgxmock.NewRows(columns).AddRow(row(at)...).AddRow(row(at.Add(time.Minute))...))

			got, err := tc.find(repo)

			require.NoError(t, err)
			assert.Len(t, got, 2)
		})
	}

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM device_telemetry`).WithArgs("esp-1", time.Hour).WillReturnError(dbErr)

		got, err := repo.FindAllByDevice(ctx, "esp-1", time.Hour)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
	})
}

func TestTelemetryRepository_FindCurrent(t *testing.T) {
	t.Run("returns the newest row", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`WHERE device_id = \$1 ORDER BY time DESC LIMIT 1`).WithArgs("esp-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow(row(at)...))

		got, err := repo.FindCurrent(ctx, "esp-1")

		require.NoError(t, err)
		assert.True(t, at.Equal(got.Time))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`ORDER BY time DESC LIMIT 1`).WithArgs("esp-1").WillReturnRows(pgxmock.NewRows(columns))

		_, err := repo.FindCurrent(ctx, "esp-1")

		assert.ErrorIs(t, err, database.ErrNotFound)
	})
}
