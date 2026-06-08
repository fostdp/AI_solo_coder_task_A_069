package api

import (
	"net/http"
	"strings"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/service"

	"github.com/gin-gonic/gin"
)

type SceneHandler struct {
	svc *service.SceneService
}

func NewSceneHandler(svc *service.SceneService) *SceneHandler {
	return &SceneHandler{svc: svc}
}

func (h *SceneHandler) UploadScene(c *gin.Context) {
	name := c.PostForm("name")
	desc := c.PostForm("description")
	sceneType := c.PostForm("scene_type")
	if sceneType == "" {
		sceneType = "highway"
	}
	tagsRaw := c.PostForm("tags")
	var tags []string
	if tagsRaw != "" {
		tags = strings.Split(tagsRaw, ",")
	}

	videoFile, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video file is required"})
		return
	}
	vf, err := videoFile.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open video file"})
		return
	}
	defer vf.Close()

	var canFile interface{ Read(p []byte) (n int, err error) }
	var canSize int64
	cfHeader, err := c.FormFile("can_log")
	if err == nil {
		cf, err := cfHeader.Open()
		if err == nil {
			canFile = cf
			canSize = cfHeader.Size
			defer cf.Close()
		}
	}

	scene, err := h.svc.UploadScene(
		c.Request.Context(), name, desc, sceneType, tags,
		vf, videoFile.Size,
		canFile, canSize,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, scene)
}

func (h *SceneHandler) GetScene(c *gin.Context) {
	id := c.Param("id")
	scene, err := h.svc.GetScene(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scene not found"})
		return
	}
	c.JSON(http.StatusOK, scene)
}

func (h *SceneHandler) ListScenes(c *gin.Context) {
	var filter model.SceneFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	scenes, err := h.svc.ListScenes(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if scenes == nil {
		scenes = []*model.Scene{}
	}
	c.JSON(http.StatusOK, gin.H{"data": scenes})
}

func (h *SceneHandler) DeleteScene(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteScene(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *SceneHandler) GetSceneStats(c *gin.Context) {
	stats, err := h.svc.GetSceneStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
