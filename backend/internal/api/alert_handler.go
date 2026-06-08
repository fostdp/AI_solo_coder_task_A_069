package api

import (
	"net/http"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/service"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

func (h *AlertHandler) GetAlerts(c *gin.Context) {
	sceneID := c.Query("scene_id")
	alerts, err := h.svc.GetAlerts(c.Request.Context(), sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if alerts == nil {
		alerts = []*model.Alert{}
	}
	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.ResolveAlert(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "resolved"})
}
