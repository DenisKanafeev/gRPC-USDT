package utils

import (
	"gRPC-USDT/internal/config"
	"gRPC-USDT/internal/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestSetupLogger(t *testing.T) {
	logger, err := SetupLogger()
	assert.NoError(t, err)
	assert.NotNil(t, logger)
}

func TestCreateRateService(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}

	mockStorage := new(storage.Storage)
	service := CreateRateService(mockStorage, logger, cfg)

	assert.NotNil(t, service)
}

func TestCreateStorage(t *testing.T) {

	t.Run("invalid config", func(t *testing.T) {
		logger := zap.NewNop()
		cfg := &config.Config{} // Пустая конфигурация

		store, err := CreateStorage(cfg, logger)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
		if store != nil {
			t.Errorf("Expected nil storage, got %v", store)
		}
	})
}
