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
