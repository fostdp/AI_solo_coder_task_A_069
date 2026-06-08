package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dc-cooling-optimizer/internal/db"
	"dc-cooling-optimizer/internal/ws"
)

type APIServer struct {
	db  *db.DB
	hub *ws.Hub
	mux *http.ServeMux
}

func New(database *db.DB, hub *ws.Hub) *APIServer {
	s := &APIServer{
		db:  database,
		hub: hub,
		mux: http.NewServeMux(),
	}
	s.SetupRoutes()
	return s
}

func (s *APIServer) SetupRoutes() {
	s.mux.HandleFunc("/api/devices", s.GetAllDevices)
	s.mux.HandleFunc("/api/devices/", s.handleDevicesRoute)
	s.mux.HandleFunc("/api/pue/history", s.GetPUERecords)
	s.mux.HandleFunc("/api/pue/current", s.GetCurrentPUE)
	s.mux.HandleFunc("/api/cooling/allocation", s.GetCoolingAllocations)
	s.mux.HandleFunc("/api/alerts", s.GetActiveAlerts)
	s.mux.HandleFunc("/api/alerts/", s.handleAlertsRoute)
	s.mux.HandleFunc("/api/optimization/suggestions", s.GetOptimizationSuggestions)
}

func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *APIServer) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *APIServer) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

func (s *APIServer) handleDevicesRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	parts := strings.Split(path, "/")

	switch {
	case path == "status":
		s.GetLatestDeviceStatus(w, r)
	case len(parts) == 2 && parts[1] == "data":
		s.GetDeviceData24h(w, r, parts[0])
	case len(parts) == 1 && parts[0] != "":
		s.GetDevicesByType(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (s *APIServer) handleAlertsRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
	parts := strings.Split(path, "/")

	if len(parts) == 2 && parts[1] == "acknowledge" {
		s.AcknowledgeAlert(w, r, parts[0])
		return
	}

	http.NotFound(w, r)
}

func (s *APIServer) GetAllDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	devices, err := s.db.GetAllDevices(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query devices")
		return
	}

	s.respondJSON(w, http.StatusOK, devices)
}

func (s *APIServer) GetDevicesByType(w http.ResponseWriter, r *http.Request, deviceType string) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	devices, err := s.db.GetDevicesByType(ctx, deviceType)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query devices")
		return
	}

	s.respondJSON(w, http.StatusOK, devices)
}

func (s *APIServer) GetDeviceData24h(w http.ResponseWriter, r *http.Request, deviceIDStr string) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid device id")
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	data, err := s.db.GetDeviceCOPHistory(ctx, deviceID, hours)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query device data")
		return
	}

	s.respondJSON(w, http.StatusOK, data)
}

func (s *APIServer) GetLatestDeviceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	statuses, err := s.db.GetLatestDeviceStatus(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query device status")
		return
	}

	s.respondJSON(w, http.StatusOK, statuses)
}

func (s *APIServer) GetPUERecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	records, err := s.db.GetPUERecords(ctx, hours)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query PUE records")
		return
	}

	s.respondJSON(w, http.StatusOK, records)
}

func (s *APIServer) GetCurrentPUE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	records, err := s.db.GetPUERecords(ctx, 1)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query PUE record")
		return
	}

	if len(records) == 0 {
		s.respondError(w, http.StatusNotFound, "no recent PUE data available")
		return
	}

	s.respondJSON(w, http.StatusOK, records[len(records)-1])
}

func (s *APIServer) GetCoolingAllocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	allocations, err := s.db.GetCoolingAllocations(ctx, hours)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query cooling allocations")
		return
	}

	s.respondJSON(w, http.StatusOK, allocations)
}

func (s *APIServer) GetActiveAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	alerts, err := s.db.GetActiveAlerts(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query alerts")
		return
	}

	s.respondJSON(w, http.StatusOK, alerts)
}

func (s *APIServer) AcknowledgeAlert(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPut {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid alert id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rowsAffected, err := s.db.AcknowledgeAlert(ctx, id)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to acknowledge alert")
		return
	}

	if rowsAffected == 0 {
		s.respondError(w, http.StatusNotFound, "alert not found")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

func (s *APIServer) GetOptimizationSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	suggestions, err := s.db.GetOptimizationSuggestions(ctx)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to query optimization suggestions")
		return
	}

	s.respondJSON(w, http.StatusOK, suggestions)
}
