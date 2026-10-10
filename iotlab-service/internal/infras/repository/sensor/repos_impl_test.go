package sensor_test

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
	sensorrepo "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor"
	"github.com/tdatIT/iotlab-service/pkgs/database"
)

var (
	ctx     = context.Background()
	dbErr   = errors.New("db down")
	at      = time.Date(2026, 10, 10, 8, 0, 0, 0, time.UTC)
	columns = []string{"time", "device_id", "temp_c", "humidity_pct"}
)

// newRepo returns the repository over a pgxmock pool that fails the test on any unmet expectation.
func newRepo(t *testing.T) (sensorrepo.Repository, pgxmock.PgxPoolIface) {
	t.Helper()

	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, pool.ExpectationsWereMet()) })
	return sensorrepo.NewSensorRepository(pool), pool
}

func TestSensorRepository_Create(t *testing.T) {
	t.Run("binds the reading", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`INSERT INTO sensor_raw \(time, device_id, temp_c, humidity_pct\)`).
			WithArgs(at, "esp-1", 31.6, 74.5).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		require.NoError(t, repo.Create(ctx, &models.SensorRaw{Time: at, DeviceID: "esp-1", TempC: 31.6, HumidityPct: 74.5}))
	})

	t.Run("unique violation returns database.ErrDuplicate", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`INSERT INTO sensor_raw`).WithArgs(at, "esp-1", 0.0, 0.0).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		assert.ErrorIs(t, repo.Create(ctx, &models.SensorRaw{Time: at, DeviceID: "esp-1"}), database.ErrDuplicate)
	})
}

func TestSensorRepository_FindByKey(t *testing.T) {
	t.Run("existing row is scanned", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM sensor_raw WHERE device_id = \$1 AND time = \$2`).WithArgs("esp-1", at).
			WillReturnRows(pgxmock.NewRows(columns).AddRow(at, "esp-1", 31.6, 74.5))

		got, err := repo.FindByKey(ctx, "esp-1", at)

		require.NoError(t, err)
		assert.InDelta(t, 31.6, got.TempC, 1e-9)
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM sensor_raw`).WithArgs("esp-1", at).WillReturnRows(pgxmock.NewRows(columns))

		_, err := repo.FindByKey(ctx, "esp-1", at)

		assert.ErrorIs(t, err, database.ErrNotFound)
	})

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM sensor_raw`).WithArgs("esp-1", at).WillReturnError(dbErr)

		_, err := repo.FindByKey(ctx, "esp-1", at)

		assert.ErrorIs(t, err, dbErr)
	})
}

func TestSensorRepository_Update(t *testing.T) {
	t.Run("zero readings keep their stored value", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE sensor_raw SET\s+temp_c\s+= COALESCE\(NULLIF\(\$3, 0::double precision\), temp_c\)`).
			WithArgs("esp-1", at, 25.0, 0.0).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		require.NoError(t, repo.Update(ctx, &models.SensorRaw{Time: at, DeviceID: "esp-1", TempC: 25}))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE sensor_raw`).WithArgs("esp-1", at, 25.0, 0.0).WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		assert.ErrorIs(t, repo.Update(ctx, &models.SensorRaw{Time: at, DeviceID: "esp-1", TempC: 25}), database.ErrNotFound)
	})

	t.Run("exec failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`UPDATE sensor_raw`).WithArgs("esp-1", at, 25.0, 0.0).WillReturnError(dbErr)

		assert.ErrorIs(t, repo.Update(ctx, &models.SensorRaw{Time: at, DeviceID: "esp-1", TempC: 25}), dbErr)
	})
}

func TestSensorRepository_Delete(t *testing.T) {
	t.Run("existing row is removed", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM sensor_raw WHERE device_id = \$1 AND time = \$2`).WithArgs("esp-1", at).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		require.NoError(t, repo.Delete(ctx, "esp-1", at))
	})

	t.Run("no row returns database.ErrNotFound", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM sensor_raw`).WithArgs("esp-1", at).WillReturnResult(pgxmock.NewResult("DELETE", 0))

		assert.ErrorIs(t, repo.Delete(ctx, "esp-1", at), database.ErrNotFound)
	})

	t.Run("exec failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectExec(`DELETE FROM sensor_raw`).WithArgs("esp-1", at).WillReturnError(dbErr)

		assert.ErrorIs(t, repo.Delete(ctx, "esp-1", at), dbErr)
	})
}

func TestSensorRepository_RangeReads(t *testing.T) {
	cases := []struct {
		name   string
		window time.Duration
		find   func(sensorrepo.Repository) ([]*models.SensorRaw, error)
	}{
		{"90m window", 90 * time.Minute, func(r sensorrepo.Repository) ([]*models.SensorRaw, error) {
			return r.FindAllByDevice(ctx, "esp-1", 90*time.Minute)
		}},
		{"last 6h", 6 * time.Hour, func(r sensorrepo.Repository) ([]*models.SensorRaw, error) {
			return r.FindLast6h(ctx, "esp-1")
		}},
		{"last day", 24 * time.Hour, func(r sensorrepo.Repository) ([]*models.SensorRaw, error) {
			return r.FindLastDay(ctx, "esp-1")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" passes the window as an interval, oldest first", func(t *testing.T) {
			repo, pool := newRepo(t)
			pool.ExpectQuery(`WHERE device_id = \$1 AND time >= now\(\) - \$2::interval ORDER BY time$`).
				WithArgs("esp-1", tc.window).
				WillReturnRows(pgxmock.NewRows(columns).AddRow(at, "esp-1", 30.0, 70.0).AddRow(at, "esp-1", 31.0, 71.0))

			got, err := tc.find(repo)

			require.NoError(t, err)
			assert.Len(t, got, 2)
		})
	}

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM sensor_raw`).WithArgs("esp-1", time.Hour).WillReturnError(dbErr)

		got, err := repo.FindAllByDevice(ctx, "esp-1", time.Hour)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
	})
}

func TestSensorRepository_FindCurrent(t *testing.T) {
	t.Run("returns the newest row", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`WHERE device_id = \$1 ORDER BY time DESC LIMIT 1`).WithArgs("esp-1").
			WillReturnRows(pgxmock.NewRows(columns).AddRow(at, "esp-1", 31.6, 74.5))

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

func TestSensorRepository_FindBuckets30m(t *testing.T) {
	bucketColumns := []string{"bucket", "device_id", "avg_c", "min_c", "max_c", "avg_h", "min_h", "max_h", "samples"}

	t.Run("reads sensor_30m inside the window, oldest first", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM sensor_30m\s+WHERE device_id = \$1 AND bucket >= now\(\) - \$2::interval\s+ORDER BY bucket$`).
			WithArgs("esp-1", 24*time.Hour).
			WillReturnRows(pgxmock.NewRows(bucketColumns).AddRow(at, "esp-1", 30.0, 29.0, 31.0, 70.0, 60.0, 75.0, int64(120)))

		got, err := repo.FindBuckets30m(ctx, "esp-1", 24*time.Hour)

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, int64(120), got[0].Samples)
	})

	t.Run("query failure surfaces the driver error", func(t *testing.T) {
		repo, pool := newRepo(t)
		pool.ExpectQuery(`FROM sensor_30m`).WithArgs("esp-1", 24*time.Hour).WillReturnError(dbErr)

		got, err := repo.FindBuckets30m(ctx, "esp-1", 24*time.Hour)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
	})
}
