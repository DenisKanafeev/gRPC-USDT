package server

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"

	"gRPC-USDT/internal/config"
)

// Создаем mock для net.Listener
type MockListener struct {
	mock.Mock
}

func (m *MockListener) Accept() (net.Conn, error) {
	args := m.Called()
	return args.Get(0).(net.Conn), args.Error(1)
}

func (m *MockListener) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockListener) Addr() net.Addr {
	args := m.Called()
	return args.Get(0).(net.Addr)
}

// Mock для net.Addr
type MockAddr struct {
	mock.Mock
}

func (m *MockAddr) Network() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAddr) String() string {
	args := m.Called()
	return args.String(0)
}

// Интерфейс для подмены функции net.Listen
type ListenerFactory func(network, address string) (net.Listener, error)

// TestServerRun проверяет успешный запуск сервера
func TestServerRun(t *testing.T) {
	// Настраиваем логгер
	logger := zaptest.NewLogger(t)

	// Создаем конфигурацию
	cfg := &config.Config{
		GRPCPort: 50051,
		Env:      "test",
	}

	// Создаем реальный gRPC сервер
	grpcServer := grpc.NewServer()

	// Создаем экземпляр тестируемого сервера
	srv := &Server{
		grpcServer: grpcServer,
		logger:     logger,
	}

	// Запускаем сервер в горутине
	go func() {
		err := srv.Run(cfg)
		// При успешном запуске ошибка не должна возникать при штатной остановке
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("Server.Run() error = %v", err)
		}
	}()

	// Даем серверу время на запуск
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что сервер слушает на нужном порту
	conn, err := net.Dial("tcp", "localhost:50051")
	if err == nil {
		conn.Close()
	} else {
		t.Fatalf("Failed to connect to server: %v", err)
	}

	// Останавливаем сервер
	srv.Stop()
}
