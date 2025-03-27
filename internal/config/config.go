package config

import (
	"flag"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"os"
	"strconv"
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
}

func LoadConfig(logger *zap.Logger, flags *flag.FlagSet) Config {
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found")
	}

	cfg := Config{
		Env:            getValue(logger, flags, "env", "ENV", "local"),
		DBUser:         getValue(logger, flags, "db-user", "DB_USER", ""),
		DBPassword:     getValue(logger, flags, "db-password", "DB_PASSWORD", ""),
		DBHost:         getValue(logger, flags, "db-host", "DB_HOST", "localhost"),
		DBPort:         getIntValue(logger, flags, "db-port", "DB_PORT", 5432),
		DBName:         getValue(logger, flags, "db-name", "DB_NAME", ""),
		MigrationsPath: getValue(logger, flags, "migrations-path", "MIGRATIONS_PATH", "../internal/storage/migrations"),
		GRPCPort:       getIntValue(logger, flags, "grpc-port", "GRPC_PORT", 50051),
		BinanceAPIURL:  getValue(logger, flags, "binance-api-url", "BINANCE_API_URL", ""),
	}

	validateConfig(logger, cfg)
	logConfig(logger, cfg)
	return cfg
}

func getValue(logger *zap.Logger, flags *flag.FlagSet, flagName, envName, defaultValue string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	if flags.Lookup(flagName) != nil {
		return flags.Lookup(flagName).Value.String()
	}
	return defaultValue
}

func getIntValue(logger *zap.Logger, flags *flag.FlagSet, flagName, envName string, defaultValue int) int {
	if value := os.Getenv(envName); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	if flags.Lookup(flagName) != nil {
		if intVal, err := strconv.Atoi(flags.Lookup(flagName).Value.String()); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func validateConfig(logger *zap.Logger, cfg Config) {
	if cfg.DBUser == "" || cfg.DBPassword == "" || cfg.DBName == "" || cfg.BinanceAPIURL == "" {
		logger.Fatal("Missing required configuration parameters")
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
	)
}
