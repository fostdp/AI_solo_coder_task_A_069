package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sim-scenario-platform/internal/api"
	"sim-scenario-platform/internal/config"
	"sim-scenario-platform/internal/service"
	"sim-scenario-platform/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pgStore, err := storage.NewPostgresStore(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgStore.Close()

	if err := pgStore.InitSchema(ctx); err != nil {
		log.Fatalf("Failed to initialize PostgreSQL schema: %v", err)
	}
	fmt.Println("PostgreSQL schema initialized")

	mongoStore, err := storage.NewMongoStore(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoStore.Close(ctx)

	if err := mongoStore.InitCollections(ctx); err != nil {
		log.Fatalf("Failed to initialize MongoDB collections: %v", err)
	}
	fmt.Println("MongoDB collections initialized")

	minioStore, err := storage.NewMinioStore(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MinIO: %v", err)
	}

	if err := minioStore.InitBucket(ctx); err != nil {
		log.Fatalf("Failed to initialize MinIO bucket: %v", err)
	}
	fmt.Println("MinIO bucket initialized")

	sceneSvc := service.NewSceneService(pgStore, mongoStore, minioStore)
	annotationSvc := service.NewAnnotationService(pgStore, mongoStore)
	alertSvc := service.NewAlertService(pgStore)
	exportSvc := service.NewExportService(pgStore, mongoStore)

	router := api.NewRouter(sceneSvc, annotationSvc, alertSvc, exportSvc)

	engine := gin.Default()
	router.Setup(engine)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: engine,
	}

	go func() {
		fmt.Printf("Server starting on port %s\n", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	fmt.Println("Server exited")
}
