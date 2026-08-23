package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/logger"
	srv "github.com/KalessinD/gophprofile/internal/server"

	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	MigrationsDir = "./migrations/"
)

func main() {
	if err := run(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func runHTTPServer(cfg *config.ServerConfig, logger *zap.Logger) error {
	rootCtx, cancel := context.WithCancel(context.Background())
	notifyCtx, _ := signal.NotifyContext(rootCtx, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer cancel()

	pgdb, err := databaseWorks(rootCtx, cfg, logger)
	if err != nil {
		return err
	}

	router, err := srv.NewRouter(notifyCtx, cfg, logger, pgdb)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		BaseContext:       func(net.Listener) context.Context { return notifyCtx },
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Server started at " + cfg.ListenAddr)
		if cfg.TLSConfig != nil {
			logger.Info("TLS is enabled, starting HTTPS server")
			serverErrors <- server.ListenAndServeTLS(cfg.TLSConfig.CertFile, cfg.TLSConfig.KeyFile)
		} else {
			serverErrors <- server.ListenAndServe()
		}
	}()

	var serverErr error
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			serverErr = fmt.Errorf("server startup error: %w", err)
		}
	case <-notifyCtx.Done():
		logger.Info("Received shutdown signal, shutting down server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(rootCtx, cfg.GracefullShutdownTimeout)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			serverErr = fmt.Errorf("server shutdown error: %w", err)
		}
	}

	if serverErr != nil {
		return serverErr
	}

	logger.Info("Server stopped gracefully")
	return nil
}

func databaseWorks(ctx context.Context, cfg *config.ServerConfig, logger *zap.Logger) (*sql.DB, error) {
	if cfg.PsqlDSN == "" {
		return nil, errors.New("database_dsn is empty")
	}

	pgdb, err := srv.PsqlConnect(ctx, cfg.PsqlDSN, logger)
	if err != nil {
		return nil, err
	}

	if cfg.ApplyMigrations {
		err = applyDatabaseMigrations(ctx, pgdb, logger)
		if err != nil {
			return nil, err
		}
	}

	return pgdb, nil
}

// applyDatabaseMigrations just applies SQL migrations if required.
func applyDatabaseMigrations(ctx context.Context, pgdb *sql.DB, logger *zap.Logger) error {
	migrations := []string{"migrations/000001_init_project.up.sql"}

	err := srv.NewPgMigrator(pgdb).Apply(ctx, MigrationsDir, migrations)
	if err != nil {
		logger.Error("Can't apply migration", zap.Error(err))
		return err
	}

	return nil
}

func run() error {
	zapLogger, err := logger.NewLogger(config.IsProduction())
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}

	defer func() { _ = zapLogger.Sync() }()

	cfg, err := config.NewServerConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return runHTTPServer(cfg, zapLogger)
}
