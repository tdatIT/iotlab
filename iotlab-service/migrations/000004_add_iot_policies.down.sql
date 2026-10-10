SELECT remove_retention_policy('device_status_events', if_exists => true);
SELECT remove_retention_policy('device_health_1h',     if_exists => true);
SELECT remove_retention_policy('device_telemetry',     if_exists => true);
SELECT remove_retention_policy('sensor_30m',           if_exists => true);
SELECT remove_retention_policy('sensor_raw',           if_exists => true);

SELECT remove_continuous_aggregate_policy('device_health_1h', if_exists => true);
SELECT remove_continuous_aggregate_policy('sensor_30m',       if_exists => true);
