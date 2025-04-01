package server

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"gRPC-USDT/internal/config"

	"go.uber.org/zap"
)

type Server struct {
	grpcServer *grpc.Server
	logger     *zap.Logger
}

func (s *Server) Run(cfg *config.Config) error {
	// Формирование адреса прослушивания
	addr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Info("Starting gRPC server",
		zap.String("address", addr),
		zap.String("environment", cfg.Env),
	)

	// Запуск сервера с обработкой ошибок
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("grpc server failed: %w", err)
	}
	return nil
}

func (s *Server) Stop() {
	s.logger.Info("Shutting down gRPC server")
	s.grpcServer.GracefulStop()
}
