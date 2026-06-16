CREATE DATABASE IF NOT EXISTS ballistics;

USE ballistics;

CREATE TABLE IF NOT EXISTS sensor_data (
    id UUID DEFAULT generateUUIDv4(),
    device_id String,
    timestamp DateTime64(9) DEFAULT now64(),
    bowstring_tension Float64,
    arm_deformation Float64,
    arrow_initial_velocity Float64,
    penetration_depth Float64,
    temperature Float64 DEFAULT 0,
    humidity Float64 DEFAULT 0
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (device_id, timestamp)
TTL timestamp + INTERVAL 1 YEAR;

CREATE TABLE IF NOT EXISTS simulation_results (
    id UUID DEFAULT generateUUIDv4(),
    timestamp DateTime64(9) DEFAULT now64(),
    device_id String,
    initial_velocity Float64,
    launch_angle Float64,
    flight_time Float64,
    max_height Float64,
    range Float64,
    impact_velocity Float64,
    kinetic_energy Float64,
    trajectory String,
    armor_type String,
    penetration_depth Float64,
    penetration_success Bool
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (device_id, timestamp)
TTL timestamp + INTERVAL 1 YEAR;

CREATE TABLE IF NOT EXISTS alerts (
    id UUID DEFAULT generateUUIDv4(),
    timestamp DateTime64(9) DEFAULT now64(),
    device_id String,
    alert_type String,
    alert_level String,
    message String,
    sensor_value Float64,
    threshold Float64,
    acknowledged Bool DEFAULT false
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (device_id, timestamp)
TTL timestamp + INTERVAL 1 YEAR;

CREATE TABLE IF NOT EXISTS armor_performance (
    id UUID DEFAULT generateUUIDv4(),
    timestamp DateTime64(9) DEFAULT now64(),
    armor_type String,
    armor_thickness Float64,
    impact_velocity Float64,
    arrow_mass Float64,
    arrow_head_type String,
    penetration_depth Float64,
    residual_velocity Float64,
    energy_absorbed Float64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (armor_type, timestamp)
TTL timestamp + INTERVAL 1 YEAR;

CREATE MATERIALIZED VIEW IF NOT EXISTS sensor_data_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (device_id, toStartOfHour(timestamp))
AS SELECT
    device_id,
    toStartOfHour(timestamp) as timestamp,
    count() as count,
    sum(bowstring_tension) as sum_tension,
    sum(arm_deformation) as sum_deformation,
    sum(arrow_initial_velocity) as sum_velocity,
    sum(penetration_depth) as sum_penetration,
    max(bowstring_tension) as max_tension,
    max(arm_deformation) as max_deformation
FROM sensor_data
GROUP BY device_id, toStartOfHour(timestamp);
