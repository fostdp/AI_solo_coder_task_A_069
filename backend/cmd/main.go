package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"dc-cooling-optimizer/internal/alarm_notifier"
	"dc-cooling-optimizer/internal/api"
	"dc-cooling-optimizer/internal/cooling_optimizer"
	"dc-cooling-optimizer/internal/config"
	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/modbus_gateway"
	"dc-cooling-optimizer/internal/pue_calculator"
	"dc-cooling-optimizer/internal/ws"
)

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
			data, _ := json.Marshal(evt.Data)
			wsHub.BroadcastDeviceUpdate(json.RawMessage(data))
		}
	}()

	go func() {
		for evt := range pueCalc.Output() {
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
			data, _ := json.Marshal(evt.Alert)
			wsHub.BroadcastAlert(json.RawMessage(data))
		}
	}()

	apiServer := api.New(database, wsHub)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHub.HandleWebSocket)
	mux.Handle("/api/", apiServer)

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
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	handler := corsMiddleware(mux)

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
