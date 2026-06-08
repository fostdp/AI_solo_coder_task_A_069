package api

import (
	"net/http"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/service"

	"github.com/gin-gonic/gin"
)

type ExportHandler struct {
	svc *service.ExportService
}

func NewExportHandler(svc *service.ExportService) *ExportHandler {
	return &ExportHandler{svc: svc}
}

func (h *ExportHandler) ExportScene(c *gin.Context) {
	sceneID := c.Param("id")
	var req struct {
		Format string `json:"format" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task *model.ExportTask
	var err error

	switch model.ExportFormat(req.Format) {
	case model.ExportFormatOpenSCENARIO:
		task, err = h.svc.ExportOpenSCENARIO(c.Request.Context(), sceneID)
	case model.ExportFormatROSBag:
		task, err = h.svc.ExportROSBag(c.Request.Context(), sceneID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format, use 'openscenario' or 'rosbag'"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *ExportHandler) GetExportStatus(c *gin.Context) {
	id := c.Param("id")
	task, err := h.svc.GetExportStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *ExportHandler) DownloadExport(c *gin.Context) {
	id := c.Param("id")
	task, err := h.svc.GetExportStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "export task not found"})
		return
	}
	if task.Status != model.ExportStatusCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "export not completed", "status": task.Status})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"file_path": task.FilePath,
		"format":    task.Format,
		"scene_id":  task.SceneID,
	})
}
