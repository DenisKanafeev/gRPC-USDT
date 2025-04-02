package utils

import (
	"context"
	"gRPC-USDT/api/proto"
	"net"
	"strconv"
	"testing"
	"time"

	"google.golang.org/grpc/credentials/insecure"

	"gRPC-USDT/internal/config"
	"gRPC-USDT/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health/grpc_health_v1"
)

func TestSetupLogger(t *testing.T) {
	t.Run("successful logger creation", func(t *testing.T) {
		logger, err := SetupLogger()
		require.NoError(t, err)
		assert.NotNil(t, logger)
	})
}

func TestCreateStorage(t *testing.T) {
	t.Run("invalid config", func(t *testing.T) {
		cfg := &config.Config{
			DBUser:     "user",
			DBPassword: "pass",
			DBHost:     "localhost",
			DBPort:     5432,
			DBName:     "db",
		}

		_, err := CreateStorage(cfg)
		assert.Error(t, err) // Должна быть ошибка подключения
	})
}

func TestCreateRateService(t *testing.T) {
	t.Run("create service", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &config.Config{}
		mockStorage := &storage.Storage{}

		service := CreateRateService(mockStorage, logger, cfg)
		assert.NotNil(t, service)
	})
}

func TestStartServer(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{GRPCPort: 0} // 0 для автоматического выбора свободного порта
	mockService := &proto.UnimplementedRateServiceServer{}

	// Запускаем сервер
	srv, lis, err := StartServer(logger, cfg, mockService)
	require.NoError(t, err)

	// Гарантируем очистку ресурсов после теста
	t.Cleanup(func() {
		srv.Stop()
		lis.Close()
	})

	//Небольшая задержка на всякий случай
	time.Sleep(100 * time.Millisecond)

	// Подключаемся к серверу
	conn, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	// Проверяем healthcheck
	healthClient := health.NewHealthClient(conn)
	resp, err := healthClient.Check(context.Background(), &health.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, health.HealthCheckResponse_SERVING, resp.Status)
}

func TestPerformHealthCheck(t *testing.T) {
	t.Run("health check success", func(t *testing.T) {
		// Запускаем тестовый сервер
		srv := grpc.NewServer()
		health.RegisterHealthServer(srv, &HealthService{})

		lis, err := net.Listen("tcp", ":0") // :0 для случайного свободного порта
		require.NoError(t, err)

		go func() {
			if err := srv.Serve(lis); err != nil {
				t.Logf("Server error: %v", err)
			}
		}()
		defer srv.Stop()

		// Даем серверу время запуститься
		time.Sleep(100 * time.Millisecond)

		_, port, _ := net.SplitHostPort(lis.Addr().String())
		logger := zap.NewNop()
		cfg := &config.Config{GRPCPort: mustAtoi(port)}

		err = PerformHealthCheck(logger, cfg)
		assert.NoError(t, err)
	})
}

// Закомментил, потому что сигналы конфликтуют при запуске make test
//func TestHandleSignals(t *testing.T) {
//	t.Run("signal handling", func(t *testing.T) {
//		logger := zap.NewNop()
//		srv := grpc.NewServer()
//		tp := trace.NewTracerProvider()
//
//		// Запускаем обработчик сигналов в отдельной горутине
//		go HandleSignals(logger, srv, tp)
//
//		// Посылаем сигнал
//		proc, err := os.FindProcess(os.Getpid())
//		require.NoError(t, err)
//		_ = proc.Signal(os.Interrupt)
//
//		// Даем время на обработку
//		time.Sleep(100 * time.Millisecond)
//	})
//}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
