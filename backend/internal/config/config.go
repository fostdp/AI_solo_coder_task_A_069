package config

import (
	"encoding/json"
	"os"
)

type DatabaseConfig struct {
	ConnectionString string `json:"connection_string"`
	PoolMaxConns     int    `json:"pool_max_conns"`
	PoolMinConns     int    `json:"pool_min_conns"`
}

type ModbusGatewayConfig struct {
	Address                string `json:"address"`
	CollectIntervalSeconds int    `json:"collect_interval_seconds"`
	DialTimeoutSeconds     int    `json:"dial_timeout_seconds"`
	ReadTimeoutSeconds     int    `json:"read_timeout_seconds"`
	WriteTimeoutSeconds    int    `json:"write_timeout_seconds"`
	IdleTimeoutSeconds     int    `json:"idle_timeout_seconds"`
	MaxConnAgeMinutes      int    `json:"max_conn_age_minutes"`
	ReconnectDelaySeconds  int    `json:"reconnect_delay_seconds"`
	RetryCount             int    `json:"retry_count"`
	DataChannelBuffer      int    `json:"data_channel_buffer"`
}

type PUECalculatorConfig struct {
	ITPowerKW                  int     `json:"it_power_kw"`
	DistributionLossKW         int     `json:"distribution_loss_kw"`
	OtherInfraPowerKW          int     `json:"other_infra_power_kw"`
	CalculationIntervalMinutes int     `json:"calculation_interval_minutes"`
	HighPUEThreshold           float64 `json:"high_pue_threshold"`
	HistoryHours               int     `json:"history_hours"`
}

type CoolingOptimizerConfig struct {
	OptimizationIntervalMinutes int     `json:"optimization_interval_minutes"`
	ChillerCapacityUtilization  float64 `json:"chiller_capacity_utilization"`
	SafetyMargin                float64 `json:"safety_margin"`
	OvercoolThresholdDelta      float64 `json:"overcool_threshold_delta"`
	UndercoolThresholdDelta     float64 `json:"undercool_threshold_delta"`
	LowCOPThreshold             float64 `json:"low_cop_threshold"`
	HistoryHours                int     `json:"history_hours"`
}

type DeviceThresholds struct {
	MinCOP             float64 `json:"min_cop"`
	MaxSupplyTempDelta float64 `json:"max_supply_temp_delta"`
	MaxPowerRatio      float64 `json:"max_power_ratio"`
}

type AlarmNotifierConfig struct {
	CheckIntervalMinutes            int              `json:"check_interval_minutes"`
	Level1ViolationDurationMinutes  int              `json:"level1_violation_duration_minutes"`
	Level2PUEThreshold              float64          `json:"level2_pue_threshold"`
	Level2ViolationDurationMinutes  int              `json:"level2_violation_duration_minutes"`
	DingtalkWebhook                 string           `json:"dingtalk_webhook"`
	DingtalkRetryCount              int              `json:"dingtalk_retry_count"`
	DingtalkRetryDelaySeconds       int              `json:"dingtalk_retry_delay_seconds"`
	DeviceThresholds                DeviceThresholds `json:"device_thresholds"`
}

type ServerConfig struct {
	HTTPPort              string `json:"http_port"`
	StaticDir             string `json:"static_dir"`
	WSPingIntervalSeconds int    `json:"ws_ping_interval_seconds"`
	WSSendBuffer          int    `json:"ws_send_buffer"`
}

type Config struct {
	Database         DatabaseConfig         `json:"database"`
	ModbusGateway    ModbusGatewayConfig    `json:"modbus_gateway"`
	PUECalculator    PUECalculatorConfig    `json:"pue_calculator"`
	CoolingOptimizer CoolingOptimizerConfig `json:"cooling_optimizer"`
	AlarmNotifier    AlarmNotifierConfig    `json:"alarm_notifier"`
	Server           ServerConfig           `json:"server"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadOrDefault(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		return DefaultConfig()
	}
	return cfg
}

func DefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			ConnectionString: "postgres://postgres:postgres@localhost:5432/dccooling?sslmode=disable",
			PoolMaxConns:     20,
			PoolMinConns:     4,
		},
		ModbusGateway: ModbusGatewayConfig{
			Address:                "localhost:5020",
			CollectIntervalSeconds: 30,
			DialTimeoutSeconds:     5,
			ReadTimeoutSeconds:     5,
			WriteTimeoutSeconds:    5,
			IdleTimeoutSeconds:     60,
			MaxConnAgeMinutes:      30,
			ReconnectDelaySeconds:  2,
			RetryCount:             1,
			DataChannelBuffer:      500,
		},
		PUECalculator: PUECalculatorConfig{
			ITPowerKW:                  1700,
			DistributionLossKW:         250,
			OtherInfraPowerKW:          50,
			CalculationIntervalMinutes: 5,
			HighPUEThreshold:           1.4,
			HistoryHours:               1,
		},
		CoolingOptimizer: CoolingOptimizerConfig{
			OptimizationIntervalMinutes: 10,
			ChillerCapacityUtilization:  0.9,
			SafetyMargin:                1.1,
			OvercoolThresholdDelta:      2.0,
			UndercoolThresholdDelta:     2.0,
			LowCOPThreshold:             4.0,
			HistoryHours:                1,
		},
		AlarmNotifier: AlarmNotifierConfig{
			CheckIntervalMinutes:           1,
			Level1ViolationDurationMinutes: 10,
			Level2PUEThreshold:             1.5,
			Level2ViolationDurationMinutes: 30,
			DingtalkWebhook:                "",
			DingtalkRetryCount:             3,
			DingtalkRetryDelaySeconds:      5,
			DeviceThresholds: DeviceThresholds{
				MinCOP:             3.0,
				MaxSupplyTempDelta: 5.0,
				MaxPowerRatio:      1.2,
			},
		},
		Server: ServerConfig{
			HTTPPort:              ":8080",
			StaticDir:             "../frontend/dist",
			WSPingIntervalSeconds: 30,
			WSSendBuffer:          256,
		},
	}
}
