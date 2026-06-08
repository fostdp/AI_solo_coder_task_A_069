package api

import (
	"sim-scenario-platform/internal/service"

	"github.com/gin-gonic/gin"
)

type Router struct {
	sceneSvc      *service.SceneService
	annotationSvc *service.AnnotationService
	alertSvc      *service.AlertService
	exportSvc     *service.ExportService
}

func NewRouter(sceneSvc *service.SceneService, annotationSvc *service.AnnotationService, alertSvc *service.AlertService, exportSvc *service.ExportService) *Router {
	return &Router{
		sceneSvc:      sceneSvc,
		annotationSvc: annotationSvc,
		alertSvc:      alertSvc,
		exportSvc:     exportSvc,
	}
}

func (r *Router) Setup(engine *gin.Engine) {
	api := engine.Group("/api")
	{
		scenes := api.Group("/scenes")
		{
			sceneHandler := NewSceneHandler(r.sceneSvc)
			scenes.POST("/upload", sceneHandler.UploadScene)
			scenes.GET("", sceneHandler.ListScenes)
			scenes.GET("/stats", sceneHandler.GetSceneStats)
			scenes.GET("/:id", sceneHandler.GetScene)
			scenes.DELETE("/:id", sceneHandler.DeleteScene)

			annotationHandler := NewAnnotationHandler(r.annotationSvc)
			scenes.POST("/:id/annotations", annotationHandler.CreateAnnotation)
			scenes.GET("/:id/annotations", annotationHandler.GetAnnotationsByScene)

			replayHandler := NewReplayHandler(r.sceneSvc)
			scenes.GET("/:id/frames/:index", replayHandler.GetFrame)
			scenes.GET("/:id/can-signals", replayHandler.GetCANSignals)
			scenes.GET("/:id/replay", replayHandler.GetReplay)

			exportHandler := NewExportHandler(r.exportSvc)
			scenes.POST("/:id/export", exportHandler.ExportScene)
		}

		annotations := api.Group("/annotations")
		{
			annotationHandler := NewAnnotationHandler(r.annotationSvc)
			annotations.PUT("/:id", annotationHandler.UpdateAnnotation)
			annotations.DELETE("/:id", annotationHandler.DeleteAnnotation)
		}

		alerts := api.Group("/alerts")
		{
			alertHandler := NewAlertHandler(r.alertSvc)
			alerts.GET("", alertHandler.GetAlerts)
			alerts.PUT("/:id/resolve", alertHandler.ResolveAlert)
		}

		exports := api.Group("/exports")
		{
			exportHandler := NewExportHandler(r.exportSvc)
			exports.GET("/:id", exportHandler.GetExportStatus)
			exports.GET("/:id/download", exportHandler.DownloadExport)
		}
	}
}
