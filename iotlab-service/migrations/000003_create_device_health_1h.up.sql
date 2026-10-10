-- A continuous aggregate cannot be created inside a transaction block, so it is the only statement of this migration.
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
