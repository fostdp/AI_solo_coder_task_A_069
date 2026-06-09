CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    device_code VARCHAR(32) NOT NULL UNIQUE,
    device_name VARCHAR(128) NOT NULL,
    device_type VARCHAR(32) NOT NULL,
    area VARCHAR(64),
    rated_power FLOAT,
    rated_cooling_capacity FLOAT,
    setpoint_temp FLOAT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS device_data (
    time TIMESTAMPTZ NOT NULL,
    device_id INT NOT NULL REFERENCES devices(id),
    supply_temp FLOAT,
    return_temp FLOAT,
    flow_rate FLOAT,
    power FLOAT,
    pressure FLOAT,
    cop FLOAT,
    cooling_capacity FLOAT,
    status INT DEFAULT 1
);

SELECT create_hypertable('device_data', 'time', if_not_exists => TRUE);

CREATE INDEX idx_device_data_device_id ON device_data (device_id, time DESC);

CREATE TABLE IF NOT EXISTS pue_records (
    time TIMESTAMPTZ NOT NULL,
    it_power FLOAT NOT NULL,
    cooling_power FLOAT NOT NULL,
    distribution_loss FLOAT NOT NULL DEFAULT 0,
    other_infra_power FLOAT NOT NULL DEFAULT 0,
    total_facility_power FLOAT NOT NULL,
    pue_value FLOAT NOT NULL
);

SELECT create_hypertable('pue_records', 'time', if_not_exists => TRUE);

CREATE INDEX idx_pue_records_time ON pue_records (time DESC);

CREATE TABLE IF NOT EXISTS cooling_allocation (
    time TIMESTAMPTZ NOT NULL,
    area VARCHAR(64) NOT NULL,
    heat_load FLOAT,
    allocated_cooling FLOAT,
    setpoint_temp FLOAT,
    actual_temp FLOAT,
    optimization_method VARCHAR(32)
);

SELECT create_hypertable('cooling_allocation', 'time', if_not_exists => TRUE);

CREATE INDEX idx_cooling_allocation_time ON cooling_allocation (time DESC);

CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level INT NOT NULL,
    device_id INT REFERENCES devices(id),
    alert_type VARCHAR(64) NOT NULL,
    message TEXT NOT NULL,
    value FLOAT,
    threshold FLOAT,
    acknowledged BOOLEAN DEFAULT FALSE,
    dingtalk_sent BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_alerts_time ON alerts (time DESC);
CREATE INDEX idx_alerts_level ON alerts (level, acknowledged);

CREATE TABLE IF NOT EXISTS optimization_suggestions (
    id SERIAL PRIMARY KEY,
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    suggestion_type VARCHAR(64) NOT NULL,
    content JSONB NOT NULL,
    pue_value FLOAT,
    applied BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_optimization_suggestions_time ON optimization_suggestions (time DESC);

INSERT INTO devices (device_code, device_name, device_type, area, rated_power, rated_cooling_capacity, setpoint_temp) VALUES
('CHU-001', 'Chiller Unit 1', 'chiller', 'A', 500, 2000, 7),
('CHU-002', 'Chiller Unit 2', 'chiller', 'A', 500, 2000, 7),
('CHU-003', 'Chiller Unit 3', 'chiller', 'B', 500, 2000, 7),
('CHU-004', 'Chiller Unit 4', 'chiller', 'B', 500, 2000, 7),
('CHU-005', 'Chiller Unit 5', 'chiller', 'C', 600, 2500, 7),
('CHU-006', 'Chiller Unit 6', 'chiller', 'C', 600, 2500, 7),
('CHU-007', 'Chiller Unit 7', 'chiller', 'D', 600, 2500, 7),
('CHU-008', 'Chiller Unit 8', 'chiller', 'D', 600, 2500, 7),
('CT-001', 'Cooling Tower 1', 'cooling_tower', 'A', 75, 1500, 32),
('CT-002', 'Cooling Tower 2', 'cooling_tower', 'A', 75, 1500, 32),
('CT-003', 'Cooling Tower 3', 'cooling_tower', 'B', 75, 1500, 32),
('CT-004', 'Cooling Tower 4', 'cooling_tower', 'B', 75, 1500, 32),
('CT-005', 'Cooling Tower 5', 'cooling_tower', 'C', 90, 1800, 32),
('CT-006', 'Cooling Tower 6', 'cooling_tower', 'C', 90, 1800, 32),
('CT-007', 'Cooling Tower 7', 'cooling_tower', 'D', 90, 1800, 32),
('CT-008', 'Cooling Tower 8', 'cooling_tower', 'D', 90, 1800, 32),
('CT-009', 'Cooling Tower 9', 'cooling_tower', 'A', 75, 1500, 32),
('CT-010', 'Cooling Tower 10', 'cooling_tower', 'B', 75, 1500, 32),
('CT-011', 'Cooling Tower 11', 'cooling_tower', 'C', 90, 1800, 32),
('CT-012', 'Cooling Tower 12', 'cooling_tower', 'D', 90, 1800, 32),
('PAC-001', 'Precision AC 1', 'precision_ac', 'A', 50, 150, 22),
('PAC-002', 'Precision AC 2', 'precision_ac', 'A', 50, 150, 22),
('PAC-003', 'Precision AC 3', 'precision_ac', 'A', 50, 150, 22),
('PAC-004', 'Precision AC 4', 'precision_ac', 'A', 50, 150, 22),
('PAC-005', 'Precision AC 5', 'precision_ac', 'A', 50, 150, 22),
('PAC-006', 'Precision AC 6', 'precision_ac', 'A', 50, 150, 22),
('PAC-007', 'Precision AC 7', 'precision_ac', 'A', 50, 150, 22),
('PAC-008', 'Precision AC 8', 'precision_ac', 'A', 50, 150, 22),
('PAC-009', 'Precision AC 9', 'precision_ac', 'A', 50, 150, 22),
('PAC-010', 'Precision AC 10', 'precision_ac', 'A', 50, 150, 22),
('PAC-011', 'Precision AC 11', 'precision_ac', 'B', 50, 150, 22),
('PAC-012', 'Precision AC 12', 'precision_ac', 'B', 50, 150, 22),
('PAC-013', 'Precision AC 13', 'precision_ac', 'B', 50, 150, 22),
('PAC-014', 'Precision AC 14', 'precision_ac', 'B', 50, 150, 22),
('PAC-015', 'Precision AC 15', 'precision_ac', 'B', 50, 150, 22),
('PAC-016', 'Precision AC 16', 'precision_ac', 'B', 50, 150, 22),
('PAC-017', 'Precision AC 17', 'precision_ac', 'B', 50, 150, 22),
('PAC-018', 'Precision AC 18', 'precision_ac', 'B', 50, 150, 22),
('PAC-019', 'Precision AC 19', 'precision_ac', 'B', 50, 150, 22),
('PAC-020', 'Precision AC 20', 'precision_ac', 'B', 50, 150, 22),
('PAC-021', 'Precision AC 21', 'precision_ac', 'C', 50, 150, 22),
('PAC-022', 'Precision AC 22', 'precision_ac', 'C', 50, 150, 22),
('PAC-023', 'Precision AC 23', 'precision_ac', 'C', 50, 150, 22),
('PAC-024', 'Precision AC 24', 'precision_ac', 'C', 50, 150, 22),
('PAC-025', 'Precision AC 25', 'precision_ac', 'C', 50, 150, 22),
('PAC-026', 'Precision AC 26', 'precision_ac', 'C', 50, 150, 22),
('PAC-027', 'Precision AC 27', 'precision_ac', 'C', 50, 150, 22),
('PAC-028', 'Precision AC 28', 'precision_ac', 'C', 50, 150, 22),
('PAC-029', 'Precision AC 29', 'precision_ac', 'C', 50, 150, 22),
('PAC-030', 'Precision AC 30', 'precision_ac', 'C', 50, 150, 22),
('PAC-031', 'Precision AC 31', 'precision_ac', 'D', 50, 150, 22),
('PAC-032', 'Precision AC 32', 'precision_ac', 'D', 50, 150, 22),
('PAC-033', 'Precision AC 33', 'precision_ac', 'D', 50, 150, 22),
('PAC-034', 'Precision AC 34', 'precision_ac', 'D', 50, 150, 22),
('PAC-035', 'Precision AC 35', 'precision_ac', 'D', 50, 150, 22),
('PAC-036', 'Precision AC 36', 'precision_ac', 'D', 50, 150, 22),
('PAC-037', 'Precision AC 37', 'precision_ac', 'D', 50, 150, 22),
('PAC-038', 'Precision AC 38', 'precision_ac', 'D', 50, 150, 22),
('PAC-039', 'Precision AC 39', 'precision_ac', 'D', 50, 150, 22),
('PAC-040', 'Precision AC 40', 'precision_ac', 'D', 50, 150, 22),
('PAC-041', 'Precision AC 41', 'precision_ac', 'E', 55, 170, 22),
('PAC-042', 'Precision AC 42', 'precision_ac', 'E', 55, 170, 22),
('PAC-043', 'Precision AC 43', 'precision_ac', 'E', 55, 170, 22),
('PAC-044', 'Precision AC 44', 'precision_ac', 'E', 55, 170, 22),
('PAC-045', 'Precision AC 45', 'precision_ac', 'E', 55, 170, 22),
('PAC-046', 'Precision AC 46', 'precision_ac', 'E', 55, 170, 22),
('PAC-047', 'Precision AC 47', 'precision_ac', 'E', 55, 170, 22),
('PAC-048', 'Precision AC 48', 'precision_ac', 'E', 55, 170, 22),
('PAC-049', 'Precision AC 49', 'precision_ac', 'E', 55, 170, 22),
('PAC-050', 'Precision AC 50', 'precision_ac', 'E', 55, 170, 22),
('PAC-051', 'Precision AC 51', 'precision_ac', 'F', 55, 170, 22),
('PAC-052', 'Precision AC 52', 'precision_ac', 'F', 55, 170, 22),
('PAC-053', 'Precision AC 53', 'precision_ac', 'F', 55, 170, 22),
('PAC-054', 'Precision AC 54', 'precision_ac', 'F', 55, 170, 22),
('PAC-055', 'Precision AC 55', 'precision_ac', 'F', 55, 170, 22),
('PAC-056', 'Precision AC 56', 'precision_ac', 'F', 55, 170, 22),
('PAC-057', 'Precision AC 57', 'precision_ac', 'F', 55, 170, 22),
('PAC-058', 'Precision AC 58', 'precision_ac', 'F', 55, 170, 22),
('PAC-059', 'Precision AC 59', 'precision_ac', 'F', 55, 170, 22),
('PAC-060', 'Precision AC 60', 'precision_ac', 'F', 55, 170, 22),
('PAC-061', 'Precision AC 61', 'precision_ac', 'G', 55, 170, 22),
('PAC-062', 'Precision AC 62', 'precision_ac', 'G', 55, 170, 22),
('PAC-063', 'Precision AC 63', 'precision_ac', 'G', 55, 170, 22),
('PAC-064', 'Precision AC 64', 'precision_ac', 'G', 55, 170, 22),
('PAC-065', 'Precision AC 65', 'precision_ac', 'G', 55, 170, 22),
('PAC-066', 'Precision AC 66', 'precision_ac', 'G', 55, 170, 22),
('PAC-067', 'Precision AC 67', 'precision_ac', 'G', 55, 170, 22),
('PAC-068', 'Precision AC 68', 'precision_ac', 'G', 55, 170, 22),
('PAC-069', 'Precision AC 69', 'precision_ac', 'G', 55, 170, 22),
('PAC-070', 'Precision AC 70', 'precision_ac', 'G', 55, 170, 22),
('PAC-071', 'Precision AC 71', 'precision_ac', 'H', 55, 170, 22),
('PAC-072', 'Precision AC 72', 'precision_ac', 'H', 55, 170, 22),
('PAC-073', 'Precision AC 73', 'precision_ac', 'H', 55, 170, 22),
('PAC-074', 'Precision AC 74', 'precision_ac', 'H', 55, 170, 22),
('PAC-075', 'Precision AC 75', 'precision_ac', 'H', 55, 170, 22),
('PAC-076', 'Precision AC 76', 'precision_ac', 'H', 55, 170, 22),
('PAC-077', 'Precision AC 77', 'precision_ac', 'H', 55, 170, 22),
('PAC-078', 'Precision AC 78', 'precision_ac', 'H', 55, 170, 22),
('PAC-079', 'Precision AC 79', 'precision_ac', 'H', 55, 170, 22),
('PAC-080', 'Precision AC 80', 'precision_ac', 'H', 55, 170, 22),
('CDU-001', 'Liquid CDU 1', 'cdu', 'A', 80, 500, 18),
('CDU-002', 'Liquid CDU 2', 'cdu', 'A', 80, 500, 18),
('CDU-003', 'Liquid CDU 3', 'cdu', 'B', 80, 500, 18),
('CDU-004', 'Liquid CDU 4', 'cdu', 'B', 80, 500, 18),
('CDU-005', 'Liquid CDU 5', 'cdu', 'C', 80, 500, 18),
('CDU-006', 'Liquid CDU 6', 'cdu', 'C', 80, 500, 18),
('CDU-007', 'Liquid CDU 7', 'cdu', 'D', 80, 500, 18),
('CDU-008', 'Liquid CDU 8', 'cdu', 'D', 80, 500, 18),
('CDU-009', 'Liquid CDU 9', 'cdu', 'E', 80, 500, 18),
('CDU-010', 'Liquid CDU 10', 'cdu', 'E', 80, 500, 18),
('CDU-011', 'Liquid CDU 11', 'cdu', 'F', 80, 500, 18),
('CDU-012', 'Liquid CDU 12', 'cdu', 'F', 80, 500, 18),
('CDU-013', 'Liquid CDU 13', 'cdu', 'G', 80, 500, 18),
('CDU-014', 'Liquid CDU 14', 'cdu', 'G', 80, 500, 18),
('CDU-015', 'Liquid CDU 15', 'cdu', 'H', 80, 500, 18),
('CDU-016', 'Liquid CDU 16', 'cdu', 'H', 80, 500, 18),
('CDU-017', 'Liquid CDU 17', 'cdu', 'A', 80, 500, 18),
('CDU-018', 'Liquid CDU 18', 'cdu', 'B', 80, 500, 18),
('CDU-019', 'Liquid CDU 19', 'cdu', 'C', 80, 500, 18),
('CDU-020', 'Liquid CDU 20', 'cdu', 'D', 80, 500, 18);

CREATE OR REPLACE FUNCTION continuous_aggregate_device_5min()
RETURNS VOID AS $$
BEGIN
    CREATE MATERIALIZED VIEW IF NOT EXISTS device_data_5min
    WITH (timescaledb.continuous) AS
    SELECT
        time_bucket('5 minutes', time) AS bucket,
        device_id,
        AVG(supply_temp) AS avg_supply_temp,
        AVG(return_temp) AS avg_return_temp,
        AVG(flow_rate) AS avg_flow_rate,
        AVG(power) AS avg_power,
        AVG(pressure) AS avg_pressure,
        AVG(cop) AS avg_cop,
        AVG(cooling_capacity) AS avg_cooling_capacity
    FROM device_data
    GROUP BY bucket, device_id
    WITH NO DATA;

    PERFORM add_continuous_aggregate_policy('device_data_5min',
        start_offset => INTERVAL '1 hour',
        end_offset => INTERVAL '5 minutes',
        schedule_interval => INTERVAL '5 minutes');
END;
$$ LANGUAGE plpgsql;

SELECT continuous_aggregate_device_5min();

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'pue_records' AND column_name = 'total_power') THEN
        ALTER TABLE pue_records RENAME COLUMN total_power TO total_facility_power;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'pue_records' AND column_name = 'distribution_loss') THEN
        ALTER TABLE pue_records ADD COLUMN distribution_loss FLOAT NOT NULL DEFAULT 0;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'pue_records' AND column_name = 'other_infra_power') THEN
        ALTER TABLE pue_records ADD COLUMN other_infra_power FLOAT NOT NULL DEFAULT 0;
    END IF;
END $$;

CREATE OR REPLACE FUNCTION setup_compression_and_retention()
RETURNS VOID AS $$
BEGIN
    ALTER TABLE device_data SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'device_id',
        timescaledb.compress_orderby = 'time DESC'
    );

    PERFORM add_compression_policy('device_data', INTERVAL '7 days');

    ALTER TABLE pue_records SET (
        timescaledb.compress,
        timescaledb.compress_orderby = 'time DESC'
    );

    PERFORM add_compression_policy('pue_records', INTERVAL '30 days');

    ALTER TABLE cooling_allocation SET (
        timescaledb.compress,
        timescaledb.compress_segmentby = 'area',
        timescaledb.compress_orderby = 'time DESC'
    );

    PERFORM add_compression_policy('cooling_allocation', INTERVAL '30 days');

    PERFORM add_retention_policy('device_data', INTERVAL '90 days');
    PERFORM add_retention_policy('pue_records', INTERVAL '365 days');
    PERFORM add_retention_policy('cooling_allocation', INTERVAL '180 days');
END;
$$ LANGUAGE plpgsql;

SELECT setup_compression_and_retention();
