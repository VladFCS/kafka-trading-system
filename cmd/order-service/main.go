package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	orderv1 "github.com/vladfc/kafka-trading-system/gen/order/v1"
	"github.com/vladfc/kafka-trading-system/internal/order-service/events"
	"github.com/vladfc/kafka-trading-system/internal/order-service/handler"
	"github.com/vladfc/kafka-trading-system/internal/order-service/repository"
	orderservice "github.com/vladfc/kafka-trading-system/internal/order-service/service"
	"google.golang.org/grpc"
)

type config struct {
	DatabaseURL  string
	KafkaBrokers []string
	GRPCAddr     string
	BatchSize    int32
	Interval     time.Duration
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("order service stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	orderRepository := repository.NewPostgresRepository(pool)
	orderService := orderservice.NewOrderService(orderRepository)
	grpcHandler := handler.NewGRPCHandler(orderService, logger)

	kafkaPublisher := events.NewKafkaPublisher(cfg.KafkaBrokers, logger)
	defer func() {
		if err := kafkaPublisher.Close(); err != nil {
			logger.Error("close kafka publisher", "error", err)
		}
	}()

	outboxPublisher := events.NewOutboxPublisher(
		orderRepository,
		kafkaPublisher,
		logger,
		cfg.BatchSize,
		cfg.Interval,
	)

	grpcServer := grpc.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, grpcHandler)

	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	errCh := make(chan error, 2)

	go func() {
		logger.Info("order service grpc server started", "addr", cfg.GRPCAddr)
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("serve grpc: %w", err)
			return
		}
		errCh <- nil
	}()

	go func() {
		logger.Info("order outbox publisher started",
			"brokers", strings.Join(cfg.KafkaBrokers, ","),
			"batch_size", cfg.BatchSize,
			"interval", cfg.Interval,
		)
		err := outboxPublisher.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("run outbox publisher: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			grpcServer.GracefulStop()
			return err
		}
	}

	grpcServer.GracefulStop()
	return nil
}

func loadConfig() (config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is required")
	}

	return config{
		DatabaseURL:  databaseURL,
		KafkaBrokers: splitCSV(getenvDefault("KAFKA_BROKERS", "localhost:9092")),
		GRPCAddr:     getenvDefault("ORDER_SERVICE_GRPC_ADDR", ":50051"),
		BatchSize:    int32FromEnv("OUTBOX_BATCH_SIZE", 100),
		Interval:     durationFromEnv("OUTBOX_INTERVAL", time.Second),
	}, nil
}

func getenvDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func int32FromEnv(key string, fallback int32) int32 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return int32(parsed)
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
