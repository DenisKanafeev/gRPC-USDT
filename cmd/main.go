package main

import (
	"context"
	"flag"
	"fmt"
	"gRPC-USDT/internal/metrics"
	"gRPC-USDT/internal/optel"
	"gRPC-USDT/internal/utils"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap"
)

func main() {

	logger, err := utils.SetupLogger()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	// Инициализация конфигурации
	flagSet := flag.NewFlagSet("gRPC-USDT", flag.ContinueOnError)
	err = flagSet.Parse(os.Args[1:])
	if err != nil {
		logger.Error("Error parsing flags", zap.Error(err))
	}

	// Загрузка конфигурации с учетом флагов
	cfg := utils.LoadConfig(logger, flagSet)

	// Инициализация трассировки
	tp, err := optel.InitTracer(cfg.OTLPEndpoint, "usdt-service")
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	} else {
		logger.Info("Tracer initialized successfully")
		color.Green("You can view traces at http://localhost:16686 (have to start Jaeger for that)")
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		} else {
			logger.Info("Tracer provider shut down successfully")
		}
	}()

	store, err := utils.CreateStorage(cfg)
	if err != nil {
		logger.Fatal("Error creating store", zap.Error(err))
	}

	if err := utils.ApplyMigrations(store, cfg, logger); err != nil {
		logger.Fatal("Error applying migrations", zap.Error(err))
	}

	rateService := utils.CreateRateService(store, logger, cfg)

	grpcServer, _, err := utils.StartServer(logger, cfg, rateService)
	if err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}

	time.Sleep(1 * time.Second)

	if err := utils.PerformHealthCheck(logger, cfg); err != nil {
		logger.Fatal("Healthcheck failed", zap.Error(err))
	}

	// Экспозиция метрик Prometheus
	go func() {
		http.Handle("/metrics", metrics.ExposeMetrics())
		logger.Info("Metrics endpoint started on port", zap.Int("port", cfg.MetricsPort))
		color.Green("You can view metrics at http://localhost:9091 (have to start Prometheus for that)")
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.MetricsPort), nil); err != nil {
			logger.Error("Error starting metrics server", zap.Error(err))
		}
	}()

	utils.HandleSignals(logger, grpcServer, tp)
}
