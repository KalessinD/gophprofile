package gophprofile

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/KalessinD/gophprofile/internal/broker/kafka"
	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/handlers"
	mw "github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/KalessinD/gophprofile/internal/repositories/postgres"
	"github.com/KalessinD/gophprofile/internal/repositories/s3"
	"github.com/KalessinD/gophprofile/internal/services"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/go-chi/cors"
)

var (
	maxConnectionRetries           = 3
	waitIntervalBetweenConnections = time.Second * 3
)

func PsqlConnect(ctx context.Context, dsn string, log *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Error("Failed to parse DSN", zap.Error(err))
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	var lastErr error

	for attempt := range maxConnectionRetries {
		lastErr = db.PingContext(ctx)
		if lastErr == nil {
			log.Info("Successfully connected to PostgreSQL", zap.Int("attempt", attempt))
			break
		}

		log.Warn("Failed to connect to PostgreSQL, retrying...",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxConnectionRetries),
			zap.Duration("interval", waitIntervalBetweenConnections),
			zap.Error(lastErr),
		)

		if attempt < maxConnectionRetries {
			select {
			case <-ctx.Done():
				log.Warn("Database connection canceled by context during retry")
				db.Close()
				return nil, ctx.Err()
			case <-time.After(waitIntervalBetweenConnections):
				// Время вышло, идем на следующий круг
			}
		}
	}

	if lastErr != nil {
		log.Error("Failed to connect to PostgreSQL after all retries", zap.Error(lastErr))
		db.Close()
		return nil, fmt.Errorf("db connection failed after %d retries: %w", maxConnectionRetries, lastErr)
	}

	go func() {
		<-ctx.Done()
		log.Info("Closing database connection due to context cancellation")
		db.Close()
	}()

	return db, nil
}

func GetBaseRouter(cfg *config.ServerConfig, log *zap.Logger) *chi.Mux {
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
func NewRouter(ctx context.Context, cfg *config.ServerConfig, log *zap.Logger, pgdb *sql.DB) (http.Handler, error) {
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
	avatarService := services.NewAvatarService(avatarRepo, fileStorage, kafkaProducer, cfg.S3.Bucket)
	avatarHandler := handlers.NewAvatarHandler(avatarService)

	// API V1 Роуты
	router.Route("/api/v1", func(r chi.Router) {
		router.Use(mw.UserIDMiddleware)

		// Avatar routes
		r.Post("/avatars", avatarHandler.UploadAvatar)
		r.Get("/avatars/{avatar_id}", avatarHandler.GetAvatar)
		r.Get("/avatars/{avatar_id}/metadata", avatarHandler.GetAvatarMetadata)
		r.Delete("/avatars/{avatar_id}", avatarHandler.DeleteAvatar)

		// User specific routes
		r.Get("/users/{user_id}/avatar", avatarHandler.GetUserAvatar)
		r.Delete("/users/{user_id}/avatar", avatarHandler.DeleteUserAvatar)
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

	staticIndexPath := filepath.Join(workDir, "web", "static", "index.html")
	if _, statErr := os.Stat(staticIndexPath); statErr == nil {
		router.HandleFunc("/web/{path:[a-zA-Z0-9\\-]+}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, r, staticIndexPath)
		})
	}

	return router, nil
}
