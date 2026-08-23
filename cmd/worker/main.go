package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KalessinD/gophprofile/internal/broker/kafka"
	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/KalessinD/gophprofile/internal/repositories/postgres"
	"github.com/KalessinD/gophprofile/internal/repositories/s3"
	srv "github.com/KalessinD/gophprofile/internal/server"
	"github.com/KalessinD/gophprofile/internal/worker"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run initializes dependencies and starts the background worker.
func run() error {
	cfg, err := config.NewServerConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	zapLogger, err := logger.NewLogger(config.IsProduction())
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	defer func() { _ = zapLogger.Sync() }()

	rootCtx, cancel := context.WithCancel(context.Background())
	notifyCtx, _ := signal.NotifyContext(rootCtx, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer cancel()

	pgdb, err := srv.PsqlConnect(notifyCtx, cfg.PsqlDSN, zapLogger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	s3Client, err := s3.NewS3Storage(notifyCtx, cfg.S3, zapLogger)
	if err != nil {
		return fmt.Errorf("failed to initialize s3 storage: %w", err)
	}

	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID)
	if err != nil {
		return fmt.Errorf("failed to initialize kafka consumer: %w", err)
	}

	avatarRepo := postgres.NewSQLStorage(pgdb)
	imageProcessor := worker.NewImageProcessor(avatarRepo, s3Client, cfg.S3.Bucket, zapLogger)

	zapLogger.Info("Starting avatar processing worker...",
		zap.String("topic", cfg.Kafka.Topic),
		zap.String("group_id", cfg.Kafka.GroupID),
	)

	kafkaConsumer.ConsumeAvatarEvents(notifyCtx, imageProcessor.ProcessAvatar)

	// Block until shutdown signal is received
	<-notifyCtx.Done()
	zapLogger.Info("Shutdown signal received, stopping worker...")

	if err := kafkaConsumer.Close(); err != nil {
		zapLogger.Error("Failed to close kafka consumer gracefully", zap.Error(err))
	}

	zapLogger.Info("Worker stopped successfully")
	return nil
}
