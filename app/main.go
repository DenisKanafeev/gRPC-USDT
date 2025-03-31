package main

import (
	"context"
	"flag"
	"gRPC-USDT/internal/optel"
	"gRPC-USDT/internal/utils"
	"os"
	"time"

	"go.uber.org/zap"
)

func main() {
	logger, err := utils.SetupLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Инициализация трассировки
	tp, err := optel.InitTracer("http://localhost:14268/api/traces", "usdt-service")
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	} else {
		logger.Info("Tracer initialized successfully")
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error("Error shutting down tracer provider", zap.Error(err))
		} else {
			logger.Info("Tracer provider shut down successfully")
		}
	}()

	flagSet := flag.NewFlagSet("gRPC-USDT", flag.ContinueOnError)
	flagSet.Parse(os.Args[1:])

	// Загрузка конфигурации с учетом флагов
	cfg := utils.LoadConfig(logger, flagSet)

	store, err := utils.CreateStorage(cfg, logger)
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

	utils.HandleSignals(logger, grpcServer, tp)
}
