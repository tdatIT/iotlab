CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 1) Device registry: current state only
CREATE TABLE devices
(
    device_id         TEXT PRIMARY KEY,
    name              TEXT,
    location          TEXT,
    status            TEXT        NOT NULL DEFAULT 'offline'
        CHECK (status IN ('online', 'offline')),
    status_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen         TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip                INET,
    fw                TEXT,
    first_seen        TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata          JSONB       NOT NULL DEFAULT '{}'
);

-- 2) Sensor readings (DHT22), every 15 s
CREATE TABLE sensor_raw
(
    time         TIMESTAMPTZ      NOT NULL,
    device_id    TEXT             NOT NULL,
    temp_c       DOUBLE PRECISION NOT NULL CHECK (temp_c BETWEEN -40 AND 80),
    humidity_pct DOUBLE PRECISION NOT NULL CHECK (humidity_pct BETWEEN 0 AND 100)
) WITH (
      tsdb.hypertable,
      tsdb.partition_column = 'time',
      tsdb.chunk_interval   = '1 day',
      tsdb.segmentby        = 'device_id',
      tsdb.orderby          = 'time DESC'
      );

CREATE UNIQUE INDEX sensor_raw_device_time_uq ON sensor_raw (device_id, time);

CREATE MATERIALIZED VIEW sensor_30m
    WITH (timescaledb.continuous, timescaledb.materialized_only = false) AS
SELECT time_bucket(INTERVAL '30 minutes', time) AS bucket,
       device_id,
       avg(temp_c)       AS avg_c,
       min(temp_c)       AS min_c,
       max(temp_c)       AS max_c,
       avg(humidity_pct) AS avg_h,
       min(humidity_pct) AS min_h,
       max(humidity_pct) AS max_h,
       count(*)          AS samples
FROM sensor_raw
GROUP BY bucket, device_id
    WITH NO DATA;

-- 3) Device telemetry, every 60 s
CREATE TABLE device_telemetry
(
    time            TIMESTAMPTZ NOT NULL,
    device_id       TEXT        NOT NULL,
    fw              TEXT,
    uptime_s        BIGINT,
    reset_reason    TEXT,
    rssi_dbm        SMALLINT,
    ip              INET,
    heap_free       INTEGER,
    heap_min        INTEGER,
    wifi_reconnects INTEGER,
    mqtt_reconnects INTEGER,
    sensor_errors   INTEGER,
    PRIMARY KEY (device_id, time)
) WITH (
      tsdb.hypertable,
      tsdb.partition_column = 'time',
      tsdb.chunk_interval   = '7 days',
      tsdb.segmentby        = 'device_id',
      tsdb.orderby          = 'time DESC'
      );

CREATE MATERIALIZED VIEW device_health_1h
    WITH (timescaledb.continuous, timescaledb.materialized_only = false) AS
SELECT time_bucket(INTERVAL '1 hour', time) AS bucket,
       device_id,
       avg(rssi_dbm)      AS avg_rssi_dbm,
       min(rssi_dbm)      AS min_rssi_dbm,
       min(heap_free)     AS min_heap_free,
       min(heap_min)      AS min_heap_min,
       max(uptime_s)      AS max_uptime_s,
       max(sensor_errors) AS max_sensor_errors,
       count(*)           AS samples
FROM device_telemetry
GROUP BY bucket, device_id
    WITH NO DATA;

-- 4) Online / offline history
CREATE TABLE device_status_events
(
    time      TIMESTAMPTZ NOT NULL,
    device_id TEXT        NOT NULL,
    status    TEXT        NOT NULL CHECK (status IN ('online', 'offline'))
) WITH (
      tsdb.hypertable,
      tsdb.partition_column = 'time',
      tsdb.chunk_interval   = '30 days',
      tsdb.segmentby        = 'device_id',
      tsdb.orderby          = 'time DESC'
      );

CREATE INDEX device_status_events_device_time_idx ON device_status_events (device_id, time DESC);

-- 5) Policies
SELECT add_continuous_aggregate_policy('sensor_30m',
                                       start_offset      => INTERVAL '3 hours',
                                       end_offset        => INTERVAL '30 minutes',
                                       schedule_interval => INTERVAL '15 minutes');

SELECT add_continuous_aggregate_policy('device_health_1h',
                                       start_offset      => INTERVAL '6 hours',
                                       end_offset        => INTERVAL '1 hour',
                                       schedule_interval => INTERVAL '30 minutes');

SELECT add_retention_policy('sensor_raw',           drop_after => INTERVAL '90 days');
SELECT add_retention_policy('sensor_30m',           drop_after => INTERVAL '2 years');
SELECT add_retention_policy('device_telemetry',     drop_after => INTERVAL '30 days');
SELECT add_retention_policy('device_health_1h',     drop_after => INTERVAL '1 year');
SELECT add_retention_policy('device_status_events', drop_after => INTERVAL '1 year');