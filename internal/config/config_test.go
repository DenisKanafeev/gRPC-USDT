package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func setRequiredEnv() {
	_ = os.Setenv("DB_USER", "test-user")
	_ = os.Setenv("DB_PASSWORD", "test-pass")
	_ = os.Setenv("DB_NAME", "test-db")
	_ = os.Setenv("BINANCE_API_URL", "http://test.api")
	_ = os.Setenv("OTLP_ENDPOINT", "http://test-otel:4317")
}

func TestLoadConfig(t *testing.T) {
	// Сохраняем оригинальные env переменные
	originalEnv := map[string]string{
		"ENV":             os.Getenv("ENV"),
		"DB_USER":         os.Getenv("DB_USER"),
		"DB_PASSWORD":     os.Getenv("DB_PASSWORD"),
		"DB_HOST":         os.Getenv("DB_HOST"),
		"DB_PORT":         os.Getenv("DB_PORT"),
		"DB_NAME":         os.Getenv("DB_NAME"),
		"MIGRATIONS_PATH": os.Getenv("MIGRATIONS_PATH"),
		"GRPC_PORT":       os.Getenv("GRPC_PORT"),
		"BINANCE_API_URL": os.Getenv("BINANCE_API_URL"),
		"METRICS_PORT":    os.Getenv("METRICS_PORT"),
		"OTLP_ENDPOINT":   os.Getenv("OTLP_ENDPOINT"),
	}

	// Восстанавливаем env после тестов
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	logger := zap.NewNop()

	tests := []struct {
		name           string
		setupEnv       func()
		setupFlags     func(*flag.FlagSet)
		expectedConfig Config
		expectError    bool
	}{
		{
			name: "default values with required env",
			setupEnv: func() {
				os.Clearenv()
				setRequiredEnv()
			},
			setupFlags: func(f *flag.FlagSet) {},
			expectedConfig: Config{
				Env:            "local",
				DBUser:         "test-user",
				DBPassword:     "test-pass",
				DBHost:         "localhost",
				DBPort:         5432,
				DBName:         "test-db",
				MigrationsPath: "../internal/storage/migrations",
				GRPCPort:       50051,
				BinanceAPIURL:  "http://test.api",
				MetricsPort:    2112,
				OTLPEndpoint:   "http://test-otel:4317",
			},
		},
		{
			name: "env vars override defaults",
			setupEnv: func() {
				os.Clearenv()
				setRequiredEnv()
				_ = os.Setenv("ENV", "test-env")
				_ = os.Setenv("DB_HOST", "test-host")
				_ = os.Setenv("DB_PORT", "1234")
				_ = os.Setenv("MIGRATIONS_PATH", "/custom/migrations")
				_ = os.Setenv("GRPC_PORT", "8080")
				_ = os.Setenv("METRICS_PORT", "9090")
			},
			setupFlags: func(f *flag.FlagSet) {},
			expectedConfig: Config{
				Env:            "test-env",
				DBUser:         "test-user",
				DBPassword:     "test-pass",
				DBHost:         "test-host",
				DBPort:         1234,
				DBName:         "test-db",
				MigrationsPath: "/custom/migrations",
				GRPCPort:       8080,
				BinanceAPIURL:  "http://test.api",
				MetricsPort:    9090,
				OTLPEndpoint:   "http://test-otel:4317",
			},
		},
		{
			name: "explicit flags override everything",
			setupEnv: func() {
				os.Clearenv()
				setRequiredEnv()
				_ = os.Setenv("ENV", "env-value")
				_ = os.Setenv("DB_HOST", "env-host")
			},
			setupFlags: func(f *flag.FlagSet) {
				// Регистрируем флаги с дефолтными значениями
				f.String("env", "default-env", "")
				f.String("db-user", "default-user", "")
				f.String("db-password", "default-pass", "")
				f.String("db-host", "default-host", "")
				f.String("db-port", "0000", "")
				f.String("db-name", "default-db", "")
				f.String("migrations-path", "default/path", "")
				f.String("grpc-port", "0000", "")
				f.String("binance-api-url", "http://default.api", "")
				f.String("metrics-port", "0000", "")
				f.String("otlp-endpoint", "http://default-otel:4317", "")

				// Устанавливаем явные значения флагов
				_ = f.Set("env", "flag-value")
				_ = f.Set("db-user", "flag-user")
				_ = f.Set("db-password", "flag-pass")
				_ = f.Set("db-host", "flag-host")
				_ = f.Set("db-port", "4321")
				_ = f.Set("db-name", "flag-db")
				_ = f.Set("migrations-path", "/flag/migrations")
				_ = f.Set("grpc-port", "8081")
				_ = f.Set("binance-api-url", "http://flag.api")
				_ = f.Set("metrics-port", "9091")
				_ = f.Set("otlp-endpoint", "http://flag-otel:4317")
			},
			expectedConfig: Config{
				Env:            "flag-value",
				DBUser:         "flag-user",
				DBPassword:     "flag-pass",
				DBHost:         "flag-host",
				DBPort:         4321,
				DBName:         "flag-db",
				MigrationsPath: "/flag/migrations",
				GRPCPort:       8081,
				BinanceAPIURL:  "http://flag.api",
				MetricsPort:    9091,
				OTLPEndpoint:   "http://flag-otel:4317",
			},
		},
		{
			name: "default flags are ignored",
			setupEnv: func() {
				os.Clearenv()
				setRequiredEnv()
				_ = os.Setenv("ENV", "env-value")
			},
			setupFlags: func(f *flag.FlagSet) {
				f.String("env", "default-env", "")
				f.String("db-host", "default-host", "")
				// Не устанавливаем значения - оставляем дефолтные
			},
			expectedConfig: Config{
				Env:            "env-value",
				DBUser:         "test-user",
				DBPassword:     "test-pass",
				DBHost:         "localhost",
				DBPort:         5432,
				DBName:         "test-db",
				MigrationsPath: "../internal/storage/migrations",
				GRPCPort:       50051,
				BinanceAPIURL:  "http://test.api",
				MetricsPort:    2112,
				OTLPEndpoint:   "http://test-otel:4317",
			},
		},
		{
			name: "invalid port numbers fall back to defaults",
			setupEnv: func() {
				os.Clearenv()
				setRequiredEnv()
				_ = os.Setenv("DB_PORT", "invalid")
				_ = os.Setenv("GRPC_PORT", "invalid")
				_ = os.Setenv("METRICS_PORT", "invalid")
			},
			setupFlags: func(f *flag.FlagSet) {},
			expectedConfig: Config{
				Env:            "local",
				DBUser:         "test-user",
				DBPassword:     "test-pass",
				DBHost:         "localhost",
				DBPort:         5432,
				DBName:         "test-db",
				MigrationsPath: "../internal/storage/migrations",
				GRPCPort:       50051,
				BinanceAPIURL:  "http://test.api",
				MetricsPort:    2112,
				OTLPEndpoint:   "http://test-otel:4317",
			},
		},

		{
			name: "required fields can be set via flags",
			setupEnv: func() {
				os.Clearenv()
			},
			setupFlags: func(f *flag.FlagSet) {
				f.String("db-user", "", "")
				f.String("db-password", "", "")
				f.String("db-name", "", "")
				f.String("binance-api-url", "", "")
				f.String("otlp-endpoint", "", "")

				_ = f.Set("db-user", "flag-user")
				_ = f.Set("db-password", "flag-pass")
				_ = f.Set("db-name", "flag-db")
				_ = f.Set("binance-api-url", "http://flag.api")
				_ = f.Set("otlp-endpoint", "http://flag-otel:4317")
			},
			expectedConfig: Config{
				Env:            "local",
				DBUser:         "flag-user",
				DBPassword:     "flag-pass",
				DBHost:         "localhost",
				DBPort:         5432,
				DBName:         "flag-db",
				MigrationsPath: "../internal/storage/migrations",
				GRPCPort:       50051,
				BinanceAPIURL:  "http://flag.api",
				MetricsPort:    2112,
				OTLPEndpoint:   "http://flag-otel:4317",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Настраиваем окружение
			tt.setupEnv()

			// Настраиваем флаги
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			tt.setupFlags(flags)

			if tt.expectError {
				assert.Panics(t, func() {
					LoadConfig(logger, flags)
				}, "Expected panic for missing required fields")
				return
			}

			// Загружаем конфиг
			cfg := LoadConfig(logger, flags)

			// Выводим отладочную информацию при неудаче
			if !assert.Equal(t, tt.expectedConfig, cfg) {
				t.Logf("Expected: %+v", tt.expectedConfig)
				t.Logf("Actual:   %+v", cfg)
			}
		})
	}
}
