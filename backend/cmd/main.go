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

	"dc-cooling-optimizer/internal/alert"
	"dc-cooling-optimizer/internal/api"
	"dc-cooling-optimizer/internal/collector"
	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/optimizer"
	"dc-cooling-optimizer/internal/pue"
	"dc-cooling-optimizer/internal/ws"
)

func main() {
	dbConn := getEnv("DB_CONN", "postgres://postgres:postgres@localhost:5432/dccooling?sslmode=disable")
	modbusAddr := getEnv("MODBUS_ADDR", "localhost:5020")
	dingtalkWebhook := getEnv("DINGTALK_WEBHOOK", "")
	itPowerStr := getEnv("IT_POWER", "2000")
	httpPort := getEnv("HTTP_PORT", ":8080")

	itPower := 2000.0
	if v, err := strconv.ParseFloat(itPowerStr, 64); err == nil {
		itPower = v
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.New(ctx, dbConn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	wsHub := ws.NewHub()
	go wsHub.Run(ctx)

	coll := collector.New(database, modbusAddr)
	coll.DataChannel = func(data *db.DeviceData) {
		raw, _ := json.Marshal(data)
		wsHub.BroadcastDeviceUpdate(json.RawMessage(raw))
	}

	pueCalc := pue.New(database, itPower)
	pueCalc.OnPUEUpdate = func(pueValue, itPwr, coolingPower, totalPower float64) {
		data, _ := json.Marshal(map[string]float64{
			"pue":           pueValue,
			"it_power":      itPwr,
			"cooling_power": coolingPower,
			"total_power":   totalPower,
		})
		wsHub.BroadcastPUEUpdate(json.RawMessage(data))
	}
	pueCalc.OnHighPUE = func(pueValue float64) {
		log.Printf("WARNING: High PUE detected: %.2f", pueValue)
	}

	opt := optimizer.New(database)
	opt.OnOptimization = func(result *optimizer.OptimizationResult) {
		data, _ := json.Marshal(result)
		wsHub.BroadcastOptimization(json.RawMessage(data))
	}

	alertMgr := alert.New(database, dingtalkWebhook)

	go coll.Start(ctx)
	go pueCalc.Start(ctx)
	go opt.Start(ctx)
	go alertMgr.Start(ctx)

	apiServer := api.New(database, wsHub)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHub.HandleWebSocket)
	mux.Handle("/api/", apiServer)

	staticDir := "./frontend/dist"
	if _, err := os.Stat(staticDir); err != nil {
		staticDir = "../frontend/dist"
		if _, err := os.Stat(staticDir); err != nil {
			staticDir = ""
		}
	}
	if staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	srv := &http.Server{
		Addr:    httpPort,
		Handler: corsMiddleware(mux),
	}

	log.Printf("Starting DC Cooling Optimizer on %s", httpPort)
	log.Printf("  DB: %s", dbConn)
	log.Printf("  Modbus: %s", modbusAddr)
	log.Printf("  IT Power: %.0f kW", itPower)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
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

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
