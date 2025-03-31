package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// DatabaseConnector представляет абстракцию для работы с базой данных
type DatabaseConnector interface {
	Open(driverName, dataSourceName string) (*sql.DB, error)
	Ping() error
	Close() error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// MigrateConnector представляет абстракцию для работы с миграциями
type MigrateConnector interface {
	New(sourceURL, databaseURL string) (*migrate.Migrate, error)
	Up() error
}

// StorageInterface определяет контракт для работы с хранилищем
type StorageInterface interface {
	CreateDatabase(ctx context.Context, dbName string) error
	Migrate(migrationsPath string) error
	SaveRate(ctx context.Context, ask, bid, askAmount, bidAmount float64, ts time.Time) error
	Close() error
}

// DefaultDatabaseConnector - реализация DatabaseConnector по умолчанию
type DefaultDatabaseConnector struct {
	db *sql.DB
}

func (d *DefaultDatabaseConnector) Open(driverName, dataSourceName string) (*sql.DB, error) {
	if d.db == nil {
		var err error
		d.db, err = sql.Open(driverName, dataSourceName)
		if err != nil {
			return nil, err
		}
	}
	return d.db, nil
}

func (d *DefaultDatabaseConnector) Ping() error {
	if d.db == nil {
		return errors.New("database not initialized")
	}
	return d.db.Ping()
}

func (d *DefaultDatabaseConnector) Close() error {
	if d.db == nil {
		return nil
	}
	return d.db.Close()
}

func (d *DefaultDatabaseConnector) ExecContext(
	ctx context.Context,
	query string,
	args ...interface{},
) (sql.Result, error) {
	if d.db == nil {
		return nil, errors.New("database not initialized")
	}
	return d.db.ExecContext(ctx, query, args...)
}

// DefaultMigrateConnector - реализация MigrateConnector по умолчанию
type DefaultMigrateConnector struct {
	m *migrate.Migrate
}

func (d *DefaultMigrateConnector) New(sourceURL, databaseURL string) (*migrate.Migrate, error) {
	if d.m == nil {
		var err error
		d.m, err = migrate.New(sourceURL, databaseURL)
		if err != nil {
			return nil, err
		}
	}
	return d.m, nil
}

func (d *DefaultMigrateConnector) Up() error {
	if d.m == nil {
		return errors.New("migrate not initialized")
	}
	return d.m.Up()
}

// Storage реализует StorageInterface
type Storage struct {
	db               DatabaseConnector
	migrateConnector MigrateConnector
	dsn              string
}

// NewStorage создает новое соединение с базой данных
func NewStorage(dsn string, dbConnector DatabaseConnector, migrateConnector MigrateConnector) (*Storage, error) {
	_, err := dbConnector.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := dbConnector.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &Storage{
		db:               dbConnector,
		migrateConnector: migrateConnector,
		dsn:              dsn,
	}, nil
}

func (s *Storage) createDatabase(ctx context.Context, dbName string, tempDB DatabaseConnector) error {
	tr := otel.GetTracerProvider().Tracer("storage-postgres")
	ctx, span := tr.Start(ctx, "create-database")
	defer span.End()

	tempDSN := strings.ReplaceAll(s.dsn, fmt.Sprintf("/%s", dbName), "/postgres")

	_, err := tempDB.Open("pgx", tempDSN)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "open temp database failed")
		return fmt.Errorf("failed to open temp database: %w", err)
	}
	defer tempDB.Close()

	if err = tempDB.Ping(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ping temp database failed")
		return fmt.Errorf("temp database ping failed: %w", err)
	}

	query := fmt.Sprintf(`DO $$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '%s') THEN
            CREATE DATABASE %s;
        END IF;
    END $$;`, dbName, dbName)

	_, err = tempDB.ExecContext(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create database failed")
		return fmt.Errorf("failed to create database: %w", err)
	}

	log.Printf("Database %s created successfully\n", dbName)

	return nil
}

func (s *Storage) CreateDatabase(ctx context.Context, dbName string) error {
	tempDB := &DefaultDatabaseConnector{}
	return s.createDatabase(ctx, dbName, tempDB)
}

func (s *Storage) Migrate(migrationsPath string) error {
	if strings.TrimSpace(migrationsPath) == "" {
		return errors.New("migrations path cannot be empty")
	}

	if !strings.HasPrefix(migrationsPath, "/") {
		migrationsPath = "/" + migrationsPath
	}

	migrationDSN := strings.Split(s.dsn, "?")[0]
	migrationDSN += "?x-migrations-table=schema_migrations"

	_, err := s.migrateConnector.New("file://"+migrationsPath, migrationDSN)
	if err != nil {
		return fmt.Errorf("migration init failed: %w", err)
	}

	if err := s.migrateConnector.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up failed: %w", err)
	}

	return nil
}

func (s *Storage) SaveRate(
	ctx context.Context,
	ask, bid, askAmount, bidAmount float64,
	ts time.Time,
) error {
	const query = `INSERT INTO rates(ask, bid, ask_amount, bid_amount, timestamp)
                   VALUES($1, $2, $3, $4, $5)`

	tr := otel.GetTracerProvider().Tracer("storage-postgres")
	ctx, span := tr.Start(ctx, "SaveRate",
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			attribute.String("db.operation", "INSERT"),
			attribute.String("db.statement", query),
		))
	defer span.End()

	_, err := s.db.ExecContext(ctx, query, ask, bid, askAmount, bidAmount, ts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save rate failed")
		return fmt.Errorf("save rate failed: %w", err)
	}

	span.SetAttributes(
		attribute.Float64("ask", ask),
		attribute.Float64("bid", bid),
		attribute.String("timestamp", ts.Format(time.RFC3339)),
	)

	return nil
}

func (s *Storage) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("database close failed: %w", err)
	}
	return nil
}
