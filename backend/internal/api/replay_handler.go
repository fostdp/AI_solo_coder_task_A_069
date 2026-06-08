package api

import (
	"net/http"
	"strconv"

	"sim-scenario-platform/internal/service"

	"github.com/gin-gonic/gin"
)

type ReplayHandler struct {
	svc *service.SceneService
}

func NewReplayHandler(svc *service.SceneService) *ReplayHandler {
	return &ReplayHandler{svc: svc}
}

func (h *ReplayHandler) GetFrame(c *gin.Context) {
	sceneID := c.Param("id")
	frameIndexStr := c.Param("index")
	frameIndex, err := strconv.Atoi(frameIndexStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frame index"})
		return
	}
	url, err := h.svc.GetFrameURL(c.Request.Context(), sceneID, frameIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url, "frame_index": frameIndex})
}

func (h *ReplayHandler) GetCANSignals(c *gin.Context) {
	sceneID := c.Param("id")
	canLog, err := h.svc.GetCANSignals(c.Request.Context(), sceneID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CAN signals not found"})
		return
	}
	c.JSON(http.StatusOK, canLog)
}

func (h *ReplayHandler) GetReplay(c *gin.Context) {
	sceneID := c.Param("id")
	startTimeStr := c.DefaultQuery("start_time", "0")
	endTimeStr := c.DefaultQuery("end_time", "10")

	startTime, err := strconv.ParseFloat(startTimeStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time"})
		return
	}
	endTime, err := strconv.ParseFloat(endTimeStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time"})
		return
	}

	data, err := h.svc.GetReplayData(c.Request.Context(), sceneID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
