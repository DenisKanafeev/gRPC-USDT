package config

import (
	"flag"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	Env            string
	DBUser         string
	DBPassword     string
	DBHost         string
	DBPort         int
	DBName         string
	MigrationsPath string
	GRPCPort       int
	BinanceAPIURL  string
	MetricsPort    int
	OTLPEndpoint   string
}

func LoadConfig(logger *zap.Logger, flags *flag.FlagSet) Config {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found")
	}

	cfg := Config{
		Env:            getValue(flags, "env", "ENV", "local"),
		DBUser:         getValue(flags, "db-user", "DB_USER", ""),
		DBPassword:     getValue(flags, "db-password", "DB_PASSWORD", ""),
		DBHost:         getValue(flags, "db-host", "DB_HOST", "localhost"),
		DBPort:         getIntValue(flags, "db-port", "DB_PORT", 5432),
		DBName:         getValue(flags, "db-name", "DB_NAME", ""),
		MigrationsPath: getValue(flags, "migrations-path", "MIGRATIONS_PATH", "../internal/storage/migrations"),
		GRPCPort:       getIntValue(flags, "grpc-port", "GRPC_PORT", 50051),
		BinanceAPIURL:  getValue(flags, "binance-api-url", "BINANCE_API_URL", ""),
		MetricsPort:    getIntValue(flags, "metrics-port", "METRICS_PORT", 2112),
		OTLPEndpoint:   getValue(flags, "otlp-endpoint", "OTLP_ENDPOINT", ""),
	}

	validateConfig(logger, cfg)
	logConfig(logger, cfg)
	return cfg
}

func getValue(flags *flag.FlagSet, flagName, envName, defaultValue string) string {
	// 1. Проверяем флаг (только если он был явно установлен)
	if flags != nil {
		if f := flags.Lookup(flagName); f != nil {
			// Если флаг был изменен (значение отличается от дефолтного)
			if f.Value.String() != f.DefValue {
				return f.Value.String()
			}
		}
	}

	// 2. Проверяем переменную окружения
	if value := os.Getenv(envName); value != "" {
		return value
	}

	// 3. Возвращаем значение по умолчанию
	return defaultValue
}

func getIntValue(flags *flag.FlagSet, flagName, envName string, defaultValue int) int {
	// 1. Проверяем флаг (только если он был явно установлен)
	if flags != nil {
		if f := flags.Lookup(flagName); f != nil {
			// Если флаг был изменен (значение отличается от дефолтного)
			if f.Value.String() != f.DefValue {
				if intVal, err := strconv.Atoi(f.Value.String()); err == nil {
					return intVal
				}
			}
		}
	}

	// 2. Проверяем переменную окружения
	if value := os.Getenv(envName); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}

	// 3. Возвращаем значение по умолчанию
	return defaultValue
}

func validateConfig(logger *zap.Logger, cfg Config) {
	if cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" || cfg.BinanceAPIURL == "" || cfg.OTLPEndpoint == "" {
		logger.Fatal("Missing required configuration parameters",
			zap.String("DBUser", cfg.DBUser),
			zap.String("DBName", cfg.DBName),
			zap.String("BinanceAPIURL", cfg.BinanceAPIURL),
			zap.String("OTLPEndpoint", cfg.OTLPEndpoint),
		)
	}
}

func logConfig(logger *zap.Logger, cfg Config) {
	logger.Info("Loaded configuration",
		zap.String("env", cfg.Env),
		zap.String("db_host", cfg.DBHost),
		zap.Int("db_port", cfg.DBPort),
		zap.String("db_name", cfg.DBName),
		zap.String("migrations_path", cfg.MigrationsPath),
		zap.Int("grpc_port", cfg.GRPCPort),
		zap.String("binance_url", cfg.BinanceAPIURL),
		zap.Int("metrics_port", cfg.MetricsPort),
		zap.String("otlp_endpoint", cfg.OTLPEndpoint),
	)
}
