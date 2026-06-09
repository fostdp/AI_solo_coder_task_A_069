package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dc-cooling-optimizer/internal/alarm_notifier"
	"dc-cooling-optimizer/internal/api"
	"dc-cooling-optimizer/internal/cooling_optimizer"
	"dc-cooling-optimizer/internal/config"
	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/metrics"
	"dc-cooling-optimizer/internal/modbus_gateway"
	"dc-cooling-optimizer/internal/pue_calculator"
	"dc-cooling-optimizer/internal/ws"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	gw *gzip.Writer
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gw.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		w.gw.Flush()
		f.Flush()
	}
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		if strings.Contains(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := &gzipResponseWriter{ResponseWriter: w, gw: gz}
		next.ServeHTTP(gzw, r)
	})
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configPath := "config.json"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		configPath = v
	}
	cfg := config.LoadOrDefault(configPath)

	if v := os.Getenv("DB_CONN"); v != "" {
		cfg.Database.ConnectionString = v
	}
	if v := os.Getenv("MODBUS_ADDR"); v != "" {
		cfg.ModbusGateway.Address = v
	}
	if v := os.Getenv("DINGTALK_WEBHOOK"); v != "" {
		cfg.AlarmNotifier.DingtalkWebhook = v
	}
	if v := os.Getenv("HTTP_PORT"); v != "" {
		cfg.Server.HTTPPort = v
	}
	if v := os.Getenv("IT_POWER"); v != "" {
		if f, err := strconv.Atoi(v); err == nil {
			cfg.PUECalculator.ITPowerKW = f
		}
	}
	if v := os.Getenv("DISTRIBUTION_LOSS"); v != "" {
		if f, err := strconv.Atoi(v); err == nil {
			cfg.PUECalculator.DistributionLossKW = f
		}
	}
	if v := os.Getenv("OTHER_INFRA_POWER"); v != "" {
		if f, err := strconv.Atoi(v); err == nil {
			cfg.PUECalculator.OtherInfraPowerKW = f
		}
	}

	database, err := db.New(ctx, cfg.Database.ConnectionString)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	wsHub := ws.NewHub()
	go wsHub.Run(ctx)

	gw := modbus_gateway.New(database, &cfg.ModbusGateway, nil)
	go gw.Start(ctx)

	pueCalc := pue_calculator.New(database, &cfg.PUECalculator, gw.Output())
	go pueCalc.Start(ctx)

	coolOpt := cooling_optimizer.New(database, &cfg.CoolingOptimizer, pueCalc.Output())
	go coolOpt.Start(ctx)

	alarmNtf := alarm_notifier.New(database, &cfg.AlarmNotifier, gw.Output(), pueCalc.Output())
	go alarmNtf.Start(ctx)

	go func() {
		for evt := range gw.Output() {
			metrics.DevicesCollected.WithLabelValues(evt.DeviceType, "success").Inc()
			data, _ := json.Marshal(evt.Data)
			wsHub.BroadcastDeviceUpdate(json.RawMessage(data))
		}
	}()

	go func() {
		for evt := range pueCalc.Output() {
			metrics.PUEValue.Set(evt.PUE)
			data, _ := json.Marshal(map[string]float64{
				"pue":                   evt.PUE,
				"it_power":             evt.ITPower,
				"cooling_power":        evt.CoolingPower,
				"distribution_loss":    evt.DistributionLoss,
				"other_infra_power":    evt.OtherInfraPower,
				"total_facility_power": evt.TotalFacilityPower,
			})
			wsHub.BroadcastPUEUpdate(json.RawMessage(data))
		}
	}()

	go func() {
		for evt := range coolOpt.Output() {
			data, _ := json.Marshal(evt)
			wsHub.BroadcastOptimization(json.RawMessage(data))
		}
	}()

	go func() {
		for evt := range alarmNtf.Output() {
			metrics.AlertsTriggered.WithLabelValues(
				strconv.Itoa(evt.Alert.Level),
				evt.Alert.AlertType,
			).Inc()
			data, _ := json.Marshal(evt.Alert)
			wsHub.BroadcastAlert(json.RawMessage(data))
		}
	}()

	apiServer := api.New(database, wsHub)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHub.HandleWebSocket)
	mux.Handle("/api/", apiServer)

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	staticDir := cfg.Server.StaticDir
	if _, err := os.Stat(staticDir); err != nil {
		staticDir = "./frontend/dist"
		if _, err := os.Stat(staticDir); err != nil {
			staticDir = "../frontend/dist"
			if _, err := os.Stat(staticDir); err != nil {
				staticDir = ""
			}
		}
	}
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			if strings.Contains(r.URL.Path, ".js") || strings.Contains(r.URL.Path, ".css") {
				w.Header().Set("Cache-Control", "public, max-age=86400")
			}
			fs.ServeHTTP(w, r)
		}))
	}

	handler := corsMiddleware(gzipMiddleware(metrics.HTTPMiddleware(mux)))

	srv := &http.Server{
		Addr:    cfg.Server.HTTPPort,
		Handler: handler,
	}

	log.Printf("Starting DC Cooling Optimizer on %s", cfg.Server.HTTPPort)
	log.Printf("  DB: %s", cfg.Database.ConnectionString)
	log.Printf("  Modbus: %s", cfg.ModbusGateway.Address)
	log.Printf("  IT Power: %.0f kW", cfg.PUECalculator.ITPowerKW)
	log.Printf("  Distribution Loss: %.0f kW", cfg.PUECalculator.DistributionLossKW)
	log.Printf("  Other Infra Power: %.0f kW", cfg.PUECalculator.OtherInfraPowerKW)
	log.Printf("  Config: %s", configPath)
	log.Printf("  pprof: http://localhost%s/debug/pprof/", cfg.Server.HTTPPort)
	log.Printf("  metrics: http://localhost%s/metrics", cfg.Server.HTTPPort)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
