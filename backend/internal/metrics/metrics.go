package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	DevicesCollected = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dc_cooling_devices_collected_total",
		Help: "Total number of device data collections",
	}, []string{"device_type", "status"})

	CollectionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dc_cooling_collection_duration_seconds",
		Help:    "Time spent collecting device data",
		Buckets: prometheus.DefBuckets,
	})

	ModbusConnectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dc_cooling_modbus_connections_active",
		Help: "Number of active Modbus TCP connections",
	})

	ModbusErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dc_cooling_modbus_errors_total",
		Help: "Total Modbus TCP errors",
	}, []string{"operation"})

	PUEValue = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dc_cooling_pue_value",
		Help: "Current PUE value",
	})

	PUECalculationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dc_cooling_pue_calculation_duration_seconds",
		Help:    "Time spent calculating PUE",
		Buckets: prometheus.DefBuckets,
	})

	AlertsTriggered = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dc_cooling_alerts_triggered_total",
		Help: "Total alerts triggered",
	}, []string{"level", "type"})

	OptimizationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dc_cooling_optimization_duration_seconds",
		Help:    "Time spent on cooling optimization",
		Buckets: prometheus.DefBuckets,
	})

	WebSocketClients = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dc_cooling_websocket_clients",
		Help: "Number of connected WebSocket clients",
	})

	HTTPRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dc_cooling_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dc_cooling_http_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)

func init() {
	prometheus.MustRegister(
		DevicesCollected,
		CollectionDuration,
		ModbusConnectionsActive,
		ModbusErrors,
		PUEValue,
		PUECalculationDuration,
		AlertsTriggered,
		OptimizationDuration,
		WebSocketClients,
		HTTPRequests,
		HTTPDuration,
	)
}
