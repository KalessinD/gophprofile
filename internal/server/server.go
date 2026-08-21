package gophprofile

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/handlers"
	mw "github.com/KalessinD/gophprofile/internal/middleware"
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
	router.Use(mw.UserIDMiddleware)

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

func NewRouter(ctx context.Context, cfg *config.ServerConfig, log *zap.Logger, pgdb *sql.DB) (http.Handler, error) {
	router := GetBaseRouter(cfg, log)

	var fileStorage services.ObjectStorage
	if cfg.S3ListenAddr != "" {
		s3Client, err := s3.NewS3Storage(ctx, cfg, log)
		if err != nil {
			return nil, fmt.Errorf("initializing s3 storage: %w", err)
		}
		fileStorage = s3Client
	}

	avatraService := services.NewAvatarService(nil, fileStorage, nil, cfg.S3Bucket)
	avatarHandle := handlers.NewAvatarHandler(avatraService)

	// API V1 Роуты
	router.Route("/api/v1", func(r chi.Router) {
		// Avatar routes
		r.Post("/avatars", avatarHandle.UploadAvatar)
		r.Get("/avatars/{avatar_id}", avatarHandle.GetAvatar)
		r.Get("/avatars/{avatar_id}/metadata", avatarHandle.GetAvatarMetadata)
		r.Delete("/avatars/{avatar_id}", avatarHandle.DeleteAvatar)

		// User specific routes
		r.Get("/users/{user_id}/avatar", avatarHandle.GetUserAvatar)
		r.Delete("/users/{user_id}/avatar", avatarHandle.DeleteUserAvatar)
		r.Get("/users/{user_id}/avatars", avatarHandle.GetUserAvatars)
	})

	// System routes
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return router, nil
}
