package config

import "time"

func (c *ModbusGatewayConfig) CollectInterval() time.Duration {
	return time.Duration(c.CollectIntervalSeconds) * time.Second
}

func (c *ModbusGatewayConfig) DialTimeout() time.Duration {
	return time.Duration(c.DialTimeoutSeconds) * time.Second
}

func (c *ModbusGatewayConfig) ReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutSeconds) * time.Second
}

func (c *ModbusGatewayConfig) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutSeconds) * time.Second
}

func (c *ModbusGatewayConfig) IdleTimeout() time.Duration {
	return time.Duration(c.IdleTimeoutSeconds) * time.Second
}

func (c *ModbusGatewayConfig) MaxConnAge() time.Duration {
	return time.Duration(c.MaxConnAgeMinutes) * time.Minute
}

func (c *ModbusGatewayConfig) ReconnectDelay() time.Duration {
	return time.Duration(c.ReconnectDelaySeconds) * time.Second
}

func (c *PUECalculatorConfig) CalculationInterval() time.Duration {
	return time.Duration(c.CalculationIntervalMinutes) * time.Minute
}

func (c *PUECalculatorConfig) History() time.Duration {
	return time.Duration(c.HistoryHours) * time.Hour
}

func (c *CoolingOptimizerConfig) OptimizationInterval() time.Duration {
	return time.Duration(c.OptimizationIntervalMinutes) * time.Minute
}

func (c *CoolingOptimizerConfig) History() time.Duration {
	return time.Duration(c.HistoryHours) * time.Hour
}

func (c *AlarmNotifierConfig) CheckInterval() time.Duration {
	return time.Duration(c.CheckIntervalMinutes) * time.Minute
}

func (c *AlarmNotifierConfig) Level1ViolationDuration() time.Duration {
	return time.Duration(c.Level1ViolationDurationMinutes) * time.Minute
}

func (c *AlarmNotifierConfig) Level2ViolationDuration() time.Duration {
	return time.Duration(c.Level2ViolationDurationMinutes) * time.Minute
}

func (c *AlarmNotifierConfig) DingtalkRetryDelay() time.Duration {
	return time.Duration(c.DingtalkRetryDelaySeconds) * time.Second
}

func (c *ServerConfig) WSPingInterval() time.Duration {
	return time.Duration(c.WSPingIntervalSeconds) * time.Second
}
