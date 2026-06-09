package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connStr string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) InsertDeviceData(ctx context.Context, data *DeviceData) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO device_data (time, device_id, supply_temp, return_temp, flow_rate, power, pressure, cop, cooling_capacity, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		data.Time, data.DeviceID, data.SupplyTemp, data.ReturnTemp, data.FlowRate, data.Power, data.Pressure, data.COP, data.CoolingCapacity, data.Status,
	)
	if err != nil {
		return fmt.Errorf("insert device data: %w", err)
	}
	return nil
}

func (db *DB) GetDeviceData24h(ctx context.Context, deviceID int) ([]DeviceData, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT time, device_id, supply_temp, return_temp, flow_rate, power, pressure, cop, cooling_capacity, status
		 FROM device_data
		 WHERE device_id = $1 AND time >= NOW() - INTERVAL '24 hours'
		 ORDER BY time ASC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("query device data 24h: %w", err)
	}
	defer rows.Close()

	var result []DeviceData
	for rows.Next() {
		var d DeviceData
		if err := rows.Scan(&d.Time, &d.DeviceID, &d.SupplyTemp, &d.ReturnTemp, &d.FlowRate, &d.Power, &d.Pressure, &d.COP, &d.CoolingCapacity, &d.Status); err != nil {
			return nil, fmt.Errorf("scan device data: %w", err)
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (db *DB) InsertPUERecord(ctx context.Context, rec *PUERecord) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO pue_records (time, it_power, cooling_power, distribution_loss, other_infra_power, total_facility_power, pue_value)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		rec.Time, rec.ITPower, rec.CoolingPower, rec.DistributionLoss, rec.OtherInfraPower, rec.TotalFacilityPower, rec.PUEValue,
	)
	if err != nil {
		return fmt.Errorf("insert pue record: %w", err)
	}
	return nil
}

func (db *DB) GetPUERecords(ctx context.Context, hours int) ([]PUERecord, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT time, it_power, cooling_power, distribution_loss, other_infra_power, total_facility_power, pue_value
		 FROM pue_records
		 WHERE time >= NOW() - make_interval(hours => $1)
		 ORDER BY time ASC`,
		hours,
	)
	if err != nil {
		return nil, fmt.Errorf("query pue records: %w", err)
	}
	defer rows.Close()

	var result []PUERecord
	for rows.Next() {
		var r PUERecord
		if err := rows.Scan(&r.Time, &r.ITPower, &r.CoolingPower, &r.DistributionLoss, &r.OtherInfraPower, &r.TotalFacilityPower, &r.PUEValue); err != nil {
			return nil, fmt.Errorf("scan pue record: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (db *DB) InsertAlert(ctx context.Context, alert *Alert) error {
	err := db.pool.QueryRow(ctx,
		`INSERT INTO alerts (time, level, device_id, alert_type, message, value, threshold, acknowledged, dingtalk_sent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		alert.Time, alert.Level, alert.DeviceID, alert.AlertType, alert.Message, alert.Value, alert.Threshold, alert.Acknowledged, alert.DingTalkSent,
	).Scan(&alert.ID)
	if err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	return nil
}

func (db *DB) MarkDingTalkSent(ctx context.Context, alertID int) error {
	_, err := db.pool.Exec(ctx, "UPDATE alerts SET dingtalk_sent = true WHERE id = $1", alertID)
	if err != nil {
		return fmt.Errorf("mark dingtalk sent: %w", err)
	}
	return nil
}

func (db *DB) GetActiveAlerts(ctx context.Context) ([]Alert, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, time, level, device_id, alert_type, message, value, threshold, acknowledged, dingtalk_sent
		 FROM alerts
		 WHERE acknowledged = FALSE
		 ORDER BY time DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active alerts: %w", err)
	}
	defer rows.Close()

	var result []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Time, &a.Level, &a.DeviceID, &a.AlertType, &a.Message, &a.Value, &a.Threshold, &a.Acknowledged, &a.DingTalkSent); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (db *DB) GetDevicesByType(ctx context.Context, deviceType string) ([]Device, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, device_code, device_name, device_type, area, rated_power, rated_cooling_capacity, setpoint_temp, created_at
		 FROM devices
		 WHERE device_type = $1
		 ORDER BY id`,
		deviceType,
	)
	if err != nil {
		return nil, fmt.Errorf("query devices by type: %w", err)
	}
	defer rows.Close()

	var result []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.DeviceCode, &d.DeviceName, &d.DeviceType, &d.Area, &d.RatedPower, &d.RatedCoolingCapacity, &d.SetpointTemp, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (db *DB) GetAllDevices(ctx context.Context) ([]Device, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, device_code, device_name, device_type, area, rated_power, rated_cooling_capacity, setpoint_temp, created_at
		 FROM devices
		 ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query all devices: %w", err)
	}
	defer rows.Close()

	var result []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.ID, &d.DeviceCode, &d.DeviceName, &d.DeviceType, &d.Area, &d.RatedPower, &d.RatedCoolingCapacity, &d.SetpointTemp, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (db *DB) InsertCoolingAllocation(ctx context.Context, alloc *CoolingAllocation) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO cooling_allocation (time, area, heat_load, allocated_cooling, setpoint_temp, actual_temp, optimization_method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		alloc.Time, alloc.Area, alloc.HeatLoad, alloc.AllocatedCooling, alloc.SetpointTemp, alloc.ActualTemp, alloc.OptimizationMethod,
	)
	if err != nil {
		return fmt.Errorf("insert cooling allocation: %w", err)
	}
	return nil
}

func (db *DB) GetCoolingAllocations(ctx context.Context, hours int) ([]CoolingAllocation, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT time, area, heat_load, allocated_cooling, setpoint_temp, actual_temp, optimization_method
		 FROM cooling_allocation
		 WHERE time >= NOW() - make_interval(hours => $1)
		 ORDER BY time ASC`,
		hours,
	)
	if err != nil {
		return nil, fmt.Errorf("query cooling allocations: %w", err)
	}
	defer rows.Close()

	var result []CoolingAllocation
	for rows.Next() {
		var c CoolingAllocation
		if err := rows.Scan(&c.Time, &c.Area, &c.HeatLoad, &c.AllocatedCooling, &c.SetpointTemp, &c.ActualTemp, &c.OptimizationMethod); err != nil {
			return nil, fmt.Errorf("scan cooling allocation: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (db *DB) InsertOptimizationSuggestion(ctx context.Context, sug *OptimizationSuggestion) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO optimization_suggestions (time, suggestion_type, content, pue_value, applied)
		 VALUES ($1, $2, $3, $4, $5)`,
		sug.Time, sug.SuggestionType, sug.Content, sug.PUEValue, sug.Applied,
	)
	if err != nil {
		return fmt.Errorf("insert optimization suggestion: %w", err)
	}
	return nil
}

func (db *DB) GetLatestDeviceStatus(ctx context.Context) ([]DeviceStatus, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT d.id, d.device_code, d.device_name, d.device_type,
		        m.avg_cop,
		        CASE
		            WHEN m.avg_cop >= d.rated_cooling_capacity / d.rated_power * 0.8 THEN 'green'
		            WHEN m.avg_cop >= d.rated_cooling_capacity / d.rated_power * 0.6 THEN 'yellow'
		            ELSE 'red'
		        END AS status,
		        m.avg_power,
		        m.avg_cooling_capacity
		 FROM devices d
		 JOIN LATERAL (
		     SELECT avg_cop, avg_power, avg_cooling_capacity
		     FROM device_data_5min
		     WHERE device_id = d.id
		     ORDER BY bucket DESC
		     LIMIT 1
		 ) m ON true
		 ORDER BY d.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query latest device status: %w", err)
	}
	defer rows.Close()

	var result []DeviceStatus
	for rows.Next() {
		var s DeviceStatus
		if err := rows.Scan(&s.DeviceID, &s.DeviceCode, &s.DeviceName, &s.DeviceType, &s.COP, &s.Status, &s.Power, &s.CoolingCapacity); err != nil {
			return nil, fmt.Errorf("scan device status: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (db *DB) GetDeviceCOPHistory(ctx context.Context, deviceID int, hours int) ([]DeviceData, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT bucket AS time, $1 AS device_id, avg_supply_temp, avg_return_temp, avg_flow_rate, avg_power, avg_pressure, avg_cop AS cop, avg_cooling_capacity AS cooling_capacity, 0 AS status
		 FROM device_data_5min
		 WHERE device_id = $1 AND bucket >= NOW() - make_interval(hours => $2)
		 ORDER BY bucket ASC`,
		deviceID, hours,
	)
	if err != nil {
		return nil, fmt.Errorf("query device cop history: %w", err)
	}
	defer rows.Close()

	var result []DeviceData
	for rows.Next() {
		var d DeviceData
		if err := rows.Scan(&d.Time, &d.DeviceID, &d.SupplyTemp, &d.ReturnTemp, &d.FlowRate, &d.Power, &d.Pressure, &d.COP, &d.CoolingCapacity, &d.Status); err != nil {
			return nil, fmt.Errorf("scan device cop history: %w", err)
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (db *DB) AcknowledgeAlert(ctx context.Context, id int) (int64, error) {
	tag, err := db.pool.Exec(ctx, "UPDATE alerts SET acknowledged = true WHERE id = $1", id)
	if err != nil {
		return 0, fmt.Errorf("acknowledge alert: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (db *DB) GetOptimizationSuggestions(ctx context.Context) ([]OptimizationSuggestion, error) {
	rows, err := db.pool.Query(ctx,
		"SELECT id, time, suggestion_type, content, pue_value, applied FROM optimization_suggestions ORDER BY time DESC LIMIT 50")
	if err != nil {
		return nil, fmt.Errorf("query optimization suggestions: %w", err)
	}
	defer rows.Close()

	var result []OptimizationSuggestion
	for rows.Next() {
		var s OptimizationSuggestion
		if err := rows.Scan(&s.ID, &s.Time, &s.SuggestionType, &s.Content, &s.PUEValue, &s.Applied); err != nil {
			return nil, fmt.Errorf("scan optimization suggestion: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
