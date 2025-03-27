package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetEnv(t *testing.T) {
	// Очистка env перед тестом
	os.Clearenv()

	// Тест с существующей переменной окружения
	os.Setenv("TEST_KEY", "test_value")
	assert.Equal(t, "test_value", getEnv("TEST_KEY", "default_value"))

	// Тест с отсутствующей переменной окружения
	assert.Equal(t, "default_value", getEnv("NONEXISTENT_KEY", "default_value"))
}

func TestGetEnvAsInt(t *testing.T) {
	// Очистка env перед тестом
	os.Clearenv()

	// Тест с корректным числовым значением
	os.Setenv("TEST_INT", "42")
	assert.Equal(t, 42, getEnvAsInt("TEST_INT", 0))

	// Тест с некорректным числовым значением
	os.Setenv("TEST_INT", "not_a_number")
	assert.Equal(t, 0, getEnvAsInt("TEST_INT", 0))

	// Тест с отсутствующей переменной окружения
	assert.Equal(t, 10, getEnvAsInt("NONEXISTENT_INT", 10))
}

func TestLoadConfig(t *testing.T) {
	// Очистка env перед тестом
	os.Clearenv()

	// Создаем тестовый logger с помощью zap
	logger, err := zap.NewDevelopment()
	assert.NoError(t, err)
	defer logger.Sync()

	// Тест с неполным набором обязательных переменных
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("BINANCE_API_URL", "https://api.binance.com")

	cfg := LoadConfig(logger)

	// Проверяем значения по умолчанию
	assert.Equal(t, "local", cfg.Env)
	assert.Equal(t, "testuser", cfg.DBUser)
	assert.Equal(t, "testpass", cfg.DBPassword)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "testdb", cfg.DBName)
	assert.Equal(t, "../internal/storage/migrations", cfg.MigrationsPath)
	assert.Equal(t, 50051, cfg.GRPCPort)
	assert.Equal(t, "https://api.binance.com", cfg.BinanceAPIURL)

	// Тест с полностью переопределенными переменными
	os.Setenv("ENV", "production")
	os.Setenv("DB_HOST", "remotehost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("GRPC_PORT", "8080")
	os.Setenv("MIGRATIONS_PATH", "./migrations")

	cfg = LoadConfig(logger)

	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "remotehost", cfg.DBHost)
	assert.Equal(t, 5433, cfg.DBPort)
	assert.Equal(t, 8080, cfg.GRPCPort)
	assert.Equal(t, "./migrations", cfg.MigrationsPath)
}
