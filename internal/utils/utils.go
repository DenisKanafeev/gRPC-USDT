package utils

import (
	"context"
	"flag"
	"fmt"
	"gRPC-USDT/api/proto"
	"gRPC-USDT/internal/config"
	"gRPC-USDT/internal/service"
	"gRPC-USDT/internal/storage"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

type HealthService struct{}

func (s *HealthService) Check(ctx context.Context, req *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}

func (s *HealthService) Watch(req *health.HealthCheckRequest, stream health.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

func SetupLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

func LoadConfig(logger *zap.Logger, flags *flag.FlagSet) *config.Config {
	cfg := config.LoadConfig(logger, flags)
	return &cfg
}

func CreateStorage(cfg *config.Config, logger *zap.Logger) (*storage.Storage, error) {
	dataSourceName := "postgres://" + cfg.DBUser + ":" + cfg.DBPassword + "@" + cfg.DBHost + ":" + strconv.Itoa(cfg.DBPort) + "/" + cfg.DBName + "?sslmode=disable"

	dbConnector := &storage.DefaultDatabaseConnector{}
	migrateConnector := &storage.DefaultMigrateConnector{}

	store, err := storage.NewStorage(dataSourceName, dbConnector, migrateConnector)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func ApplyMigrations(store *storage.Storage, cfg *config.Config, logger *zap.Logger) error {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("error getting current file path")
	}

	// Получаем корневую директорию проекта
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(filename)))

	// Формируем абсолютный путь к миграциям
	migrationsPath := filepath.Join(projectRoot, cfg.MigrationsPath)

	logger.Info("Migrations path", zap.String("path", migrationsPath))

	if err := store.Migrate(migrationsPath); err != nil {
		return err
	}

	return nil
}

func CreateRateService(store *storage.Storage, logger *zap.Logger, cfg *config.Config) proto.RateServiceServer {
	return service.NewRateService(store, logger, cfg, nil)
}

func StartServer(logger *zap.Logger, cfg *config.Config, rateService proto.RateServiceServer) (*grpc.Server, net.Listener, error) {
	grpcServer := grpc.NewServer()
	proto.RegisterRateServiceServer(grpcServer, rateService)
	grpc_health_v1.RegisterHealthServer(grpcServer, &HealthService{})

	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}

	logger.Info("Starting gRPC server", zap.String("address", addr), zap.String("environment", cfg.Env))

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	return grpcServer, lis, nil
}

func PerformHealthCheck(logger *zap.Logger, cfg *config.Config) error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.NewClient("localhost:"+strconv.Itoa(cfg.GRPCPort), opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	healthClient := grpc_health_v1.NewHealthClient(conn)
	req := &grpc_health_v1.HealthCheckRequest{}
	resp, err := healthClient.Check(context.Background(), req)
	if err != nil {
		return err
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("server is not serving")
	}

	logger.Info("Healthcheck passed")
	return nil
}

func HandleSignals(logger *zap.Logger, grpcServer *grpc.Server) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signals
	logger.Info("Received signal, shutting down gracefully...", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		grpcServer.GracefulStop()
	}()

	select {
	case <-ctx.Done():
		logger.Warn("Shutdown timed out, forcing exit")
	case <-time.After(10 * time.Second):
		logger.Info("Server stopped gracefully")
	}

	logger.Info("Server stopped")
}
