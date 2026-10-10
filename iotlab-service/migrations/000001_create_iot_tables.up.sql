CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Device registry: current state only
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

-- Sensor readings (DHT22), every 15 s
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

-- Device telemetry, every 60 s
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

-- Online / offline history
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
