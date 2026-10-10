-- A continuous aggregate cannot be created inside a transaction block, so it is the only statement of this migration.
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
