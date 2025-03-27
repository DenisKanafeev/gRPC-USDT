package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"gRPC-USDT/api/proto"
	"gRPC-USDT/internal/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	health "google.golang.org/grpc/health/grpc_health_v1"
)

type MockHealthService struct{}

func (s *MockHealthService) Check(ctx context.Context, req *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}

func (s *MockHealthService) Watch(req *health.HealthCheckRequest, stream health.Health_WatchServer) error {
	return nil
}

func TestServer_Run(t *testing.T) {
	t.Run("successful run", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &config.Config{GRPCPort: 50051}

		grpcServer := grpc.NewServer()
		proto.RegisterRateServiceServer(grpcServer, &proto.UnimplementedRateServiceServer{})
		grpc_health_v1.RegisterHealthServer(grpcServer, &MockHealthService{})

		srv := &Server{grpcServer: grpcServer, logger: logger}

		// Запускаем сервер в отдельной горутине
		go func() {
			if err := srv.Run(cfg); err != nil {
				t.Errorf("Server run failed: %v", err)
			}
		}()

		// Подождать немного, чтобы сервер запустился
		time.Sleep(100 * time.Millisecond)

		// Создание клиента для healthcheck
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}
		conn, err := grpc.Dial("localhost:50051", opts...)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		healthClient := grpc_health_v1.NewHealthClient(conn)
		req := &grpc_health_v1.HealthCheckRequest{}
		resp, err := healthClient.Check(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
			t.Errorf("Server is not serving")
		}

		srv.Stop()
	})

	t.Run("port in use", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &config.Config{GRPCPort: 50051}

		// Создаем сервер, который уже слушает порт
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			t.Fatal(err)
		}
		defer lis.Close()

		grpcServer := grpc.NewServer()
		proto.RegisterRateServiceServer(grpcServer, &proto.UnimplementedRateServiceServer{})
		grpc_health_v1.RegisterHealthServer(grpcServer, &MockHealthService{})

		srv := &Server{grpcServer: grpcServer, logger: logger}

		// Запускаем сервер в отдельной горутине
		go func() {
			if err := srv.Run(cfg); err == nil {
				t.Errorf("Expected error when port is in use")
			}
		}()

		// Подождать немного, чтобы сервер попытался запуститься
		time.Sleep(100 * time.Millisecond)

		srv.Stop()
	})
}
