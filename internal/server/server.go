package gophprofile

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/KalessinD/gophprofile/internal/broker/kafka"
	"github.com/KalessinD/gophprofile/internal/common"
	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/handlers"
	"github.com/KalessinD/gophprofile/internal/logger"
	mw "github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/KalessinD/gophprofile/internal/repositories/postgres"
	"github.com/KalessinD/gophprofile/internal/repositories/s3"
	"github.com/KalessinD/gophprofile/internal/services"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"

	"github.com/go-chi/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func GetBaseRouter(cfg *config.ServerConfig, log logger.Logger) *chi.Mux {
	router := chi.NewRouter()

	// Standard middlewares
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(mw.Logger(log))
	router.Use(middleware.Timeout(cfg.ProcessingTimeout))

	// CORS middleware
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-User-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	return router
}

// NewRouter initializes dependencies and configures HTTP routes.
func NewRouter(ctx context.Context, cfg *config.ServerConfig, log logger.Logger, pgdb *sql.DB) (http.Handler, error) {
	var err error

	router := GetBaseRouter(cfg, log)

	var fileStorage services.ObjectStorage
	if cfg.S3.ListenAddr != "" {
		s3Client, s3Err := s3.NewS3Storage(ctx, cfg.S3, log)
		if s3Err != nil {
			return nil, fmt.Errorf("initializing s3 storage: %w", s3Err)
		}
		fileStorage = s3Client
	}

	// Initialize Kafka producer
	saramaCfg := kafka.ConvertToSaramaConfig(cfg.Kafka)
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("initializing kafka producer: %w", err)
	}

	avatarRepo := postgres.NewSQLStorage(pgdb)
	avatarService := services.NewAvatarService(avatarRepo, fileStorage, kafkaProducer, cfg.S3.Bucket, log)
	s3URLBuilder := func(key string) string { // Builder generates HTTP URLs for S3 objects based on config
		scheme := "http"
		if cfg.S3.UseSSL {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s/%s/%s", scheme, cfg.S3.ListenAddr, cfg.S3.Bucket, key)
	}
	avatarHandler := handlers.NewAvatarHandler(avatarService, s3URLBuilder)

	// API V1 Routes
	router.Route("/api/v1", func(r chi.Router) {
		// Routes requiring User-ID header
		r.Group(func(r chi.Router) {
			r.Use(mw.UserIDMiddleware)

			r.Post("/avatars", avatarHandler.UploadAvatar)
			r.Delete("/avatars/{avatar_id}", avatarHandler.DeleteAvatar)
			r.Delete("/users/{user_id}/avatar", avatarHandler.DeleteUserAvatar)
		})

		// Free Access Routes
		r.Get("/avatars/{avatar_id}", avatarHandler.GetAvatar)
		r.Get("/avatars/{avatar_id}/metadata", avatarHandler.GetAvatarMetadata)
		r.Get("/users/{user_id}/avatar", avatarHandler.GetUserAvatar)
		r.Get("/users/{user_id}/avatars", avatarHandler.GetUserAvatars)
	})

	healthHandler := handlers.NewHealthHandler(pgdb, fileStorage, kafkaProducer)

	// System routes
	router.Get("/health", healthHandler.CheckHealth)

	// Web Interface routing
	// Serve index.html for any /web/* path to support direct URL access
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}

	staticIndexPath := filepath.Join(workDir, cfg.WebStaticDir, "index.html")
	if _, statErr := os.Stat(staticIndexPath); statErr != nil {
		return nil, fmt.Errorf("can't read index.html file in %s: %w", cfg.WebStaticDir, statErr)
	}

	router.HandleFunc("/web/upload", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", common.TextHTTPContentType)
		http.ServeFile(w, r, staticIndexPath)
	})

	// Wrap the Chi router with OTel HTTP middleware for automatic tracing of all requests
	otelRouter := otelhttp.NewHandler(router, common.OtelHTTPName)

	return otelRouter, nil
}
