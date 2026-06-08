package api

import (
	"net/http"

	"sim-scenario-platform/internal/model"
	"sim-scenario-platform/internal/service"

	"github.com/gin-gonic/gin"
)

type AnnotationHandler struct {
	svc *service.AnnotationService
}

func NewAnnotationHandler(svc *service.AnnotationService) *AnnotationHandler {
	return &AnnotationHandler{svc: svc}
}

func (h *AnnotationHandler) CreateAnnotation(c *gin.Context) {
	sceneID := c.Param("id")
	var annotation model.Annotation
	if err := c.ShouldBindJSON(&annotation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.CreateAnnotation(c.Request.Context(), sceneID, &annotation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *AnnotationHandler) UpdateAnnotation(c *gin.Context) {
	id := c.Param("id")
	var annotation model.Annotation
	if err := c.ShouldBindJSON(&annotation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.svc.UpdateAnnotation(c.Request.Context(), id, &annotation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *AnnotationHandler) GetAnnotationsByScene(c *gin.Context) {
	sceneID := c.Param("id")
	annotations, err := h.svc.GetAnnotationsByScene(c.Request.Context(), sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if annotations == nil {
		annotations = []*model.Annotation{}
	}
	c.JSON(http.StatusOK, gin.H{"data": annotations})
}

func (h *AnnotationHandler) DeleteAnnotation(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteAnnotation(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
