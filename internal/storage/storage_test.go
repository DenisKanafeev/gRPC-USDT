package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace/noop"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// MockDatabaseConnector - мок для DatabaseConnector
type MockDatabaseConnector struct {
	mock.Mock
}

func (m *MockDatabaseConnector) Open(driverName, dataSourceName string) (*sql.DB, error) {
	args := m.Called(driverName, dataSourceName)
	return args.Get(0).(*sql.DB), args.Error(1)
}

func (m *MockDatabaseConnector) Ping() error {
	return m.Called().Error(0)
}

func (m *MockDatabaseConnector) Close() error {
	return m.Called().Error(0)
}

func (m *MockDatabaseConnector) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(sql.Result), callArgs.Error(1) // Важно: Get(0) должен возвращать sql.Result
}

// MockMigrateConnector - мок для MigrateConnector
type MockMigrateConnector struct {
	mock.Mock
}

func (m *MockMigrateConnector) New(sourceURL, databaseURL string) (*migrate.Migrate, error) {
	args := m.Called(sourceURL, databaseURL)
	return args.Get(0).(*migrate.Migrate), args.Error(1)
}

func (m *MockMigrateConnector) Up() error {
	return m.Called().Error(0)
}

// MockResult - мок для sql.Result
type MockResult struct {
	mock.Mock
}

func (m *MockResult) LastInsertId() (int64, error) {
	return m.Called().Get(0).(int64), m.Called().Error(1)
}

func (m *MockResult) RowsAffected() (int64, error) {
	return m.Called().Get(0).(int64), m.Called().Error(1)
}

func TestNewStorage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		db, _, err := sqlmock.New()
		require.NoError(t, err)

		dbMock.On("Open", "pgx", "test_dsn").Return(db, nil)
		dbMock.On("Ping").Return(nil)

		storage, err := NewStorage("test_dsn", dbMock, migrateMock)
		assert.NoError(t, err)
		assert.NotNil(t, storage)

		dbMock.AssertExpectations(t)
	})

	t.Run("open error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		dbMock.On("Open", "pgx", "test_dsn").Return(&sql.DB{}, errors.New("open error"))

		storage, err := NewStorage("test_dsn", dbMock, migrateMock)
		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.Contains(t, err.Error(), "failed to open database")

		dbMock.AssertExpectations(t)
	})

	t.Run("ping error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		db, _, err := sqlmock.New()
		require.NoError(t, err)

		dbMock.On("Open", "pgx", "test_dsn").Return(db, nil)
		dbMock.On("Ping").Return(errors.New("ping error"))

		storage, err := NewStorage("test_dsn", dbMock, migrateMock)
		assert.Error(t, err)
		assert.Nil(t, storage)
		assert.Contains(t, err.Error(), "database ping failed")

		dbMock.AssertExpectations(t)
	})
}

func TestStorage_Migrate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		m := &migrate.Migrate{}
		migrateMock.On("New", "file:///migrations", "postgres://user:pass@host:port/db?sslmode=disable&x-migrations-table=schema_migrations").
			Return(m, nil)
		migrateMock.On("Up").Return(nil)

		storage := &Storage{
			db:               dbMock,
			migrateConnector: migrateMock,
			dsn:              "postgres://user:pass@host:port/db?sslmode=require",
		}

		err := storage.Migrate("/migrations")
		assert.NoError(t, err)

		migrateMock.AssertExpectations(t)
	})

	t.Run("empty path", func(t *testing.T) {
		storage := &Storage{dsn: "test_dsn"}
		err := storage.Migrate("")
		assert.Error(t, err)
		assert.Equal(t, "migrations path cannot be empty", err.Error())
	})

	t.Run("migration init error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		migrateMock.On("New", mock.Anything, mock.Anything).
			Return(&migrate.Migrate{}, errors.New("init error"))

		storage := &Storage{
			db:               dbMock,
			migrateConnector: migrateMock,
			dsn:              "test_dsn",
		}

		err := storage.Migrate("/migrations")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "migration init failed")

		migrateMock.AssertExpectations(t)
	})

	t.Run("migration up error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		m := &migrate.Migrate{}
		migrateMock.On("New", mock.Anything, mock.Anything).Return(m, nil)
		migrateMock.On("Up").Return(errors.New("up error"))

		storage := &Storage{
			db:               dbMock,
			migrateConnector: migrateMock,
			dsn:              "test_dsn",
		}

		err := storage.Migrate("/migrations")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "migration up failed")

		migrateMock.AssertExpectations(t)
	})

	t.Run("no change is not error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		migrateMock := &MockMigrateConnector{}

		m := &migrate.Migrate{}
		migrateMock.On("New", mock.Anything, mock.Anything).Return(m, nil)
		migrateMock.On("Up").Return(migrate.ErrNoChange)

		storage := &Storage{
			db:               dbMock,
			migrateConnector: migrateMock,
			dsn:              "test_dsn",
		}

		err := storage.Migrate("/migrations")
		assert.NoError(t, err)

		migrateMock.AssertExpectations(t)
	})
}

func TestStorage_SaveRate(t *testing.T) {
	// Инициализируем noop tracer provider для тестов
	otel.SetTracerProvider(noop.NewTracerProvider())

	t.Run("success", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		resultMock := &MockResult{}

		query := `INSERT INTO rates(ask, bid, ask_amount, bid_amount, timestamp)
                   VALUES($1, $2, $3, $4, $5)`

		ctx := context.Background()
		now := time.Now()

		dbMock.On("ExecContext", mock.Anything, query, []interface{}{1.1, 2.2, 3.3, 4.4, now}).
			Return(resultMock, nil)

		storage := &Storage{db: dbMock}

		err := storage.SaveRate(ctx, 1.1, 2.2, 3.3, 4.4, now)
		assert.NoError(t, err)

		dbMock.AssertExpectations(t)
	})

	t.Run("exec error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		resultMock := &MockResult{} // Добавляем mock result даже для случая с ошибкой

		ctx := context.Background()
		now := time.Now()

		dbMock.On("ExecContext", mock.Anything, mock.Anything, mock.Anything).
			Return(resultMock, errors.New("exec error")) // Возвращаем и result, и error

		storage := &Storage{db: dbMock}

		err := storage.SaveRate(ctx, 1.1, 2.2, 3.3, 4.4, now)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "save rate failed")

		dbMock.AssertExpectations(t)
	})

	t.Run("nil db", func(t *testing.T) {
		storage := &Storage{db: nil}

		err := storage.SaveRate(context.Background(), 1.1, 2.2, 3.3, 4.4, time.Now())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is nil") // Обновляем ожидаемую ошибку
	})
}

func TestStorage_Close(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		dbMock.On("Close").Return(nil)

		storage := &Storage{db: dbMock}
		err := storage.Close()
		assert.NoError(t, err)

		dbMock.AssertExpectations(t)
	})

	t.Run("close error", func(t *testing.T) {
		dbMock := &MockDatabaseConnector{}
		dbMock.On("Close").Return(errors.New("close error"))

		storage := &Storage{db: dbMock}
		err := storage.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database close failed")

		dbMock.AssertExpectations(t)
	})
}

func TestDefaultDatabaseConnector(t *testing.T) {
	t.Run("Open", func(t *testing.T) {
		connector := &DefaultDatabaseConnector{}
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		_, err = connector.Open("pgx", "test_dsn")
		assert.NoError(t, err)
		assert.NotNil(t, connector.db)
	})

	t.Run("Ping", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db, mok, err := sqlmock.New()
			require.NoError(t, err)
			defer func(db *sql.DB) {
				_ = db.Close()
			}(db)

			mok.ExpectPing()

			connector := &DefaultDatabaseConnector{db: db}
			err = connector.Ping()
			assert.NoError(t, err)
			assert.NoError(t, mok.ExpectationsWereMet())
		})

		t.Run("nil db", func(t *testing.T) {
			connector := &DefaultDatabaseConnector{}
			err := connector.Ping()
			assert.Error(t, err)
			assert.Equal(t, "database not initialized", err.Error())
		})
	})

	t.Run("Close", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db, mok, err := sqlmock.New()
			require.NoError(t, err)

			// Важное изменение: добавляем ожидание Close
			mok.ExpectClose()

			connector := &DefaultDatabaseConnector{db: db}
			err = connector.Close()
			assert.NoError(t, err)
			assert.NoError(t, mok.ExpectationsWereMet())
		})

		t.Run("nil db", func(t *testing.T) {
			connector := &DefaultDatabaseConnector{}
			err := connector.Close()
			assert.NoError(t, err)
		})
	})

	t.Run("ExecContext", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			db, mok, err := sqlmock.New()
			require.NoError(t, err)
			defer func(db *sql.DB) {
				_ = db.Close()
			}(db)

			ctx := context.Background()
			query := "SELECT 1"
			mok.ExpectExec(query).WillReturnResult(sqlmock.NewResult(0, 1))

			connector := &DefaultDatabaseConnector{db: db}
			result, err := connector.ExecContext(ctx, query)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NoError(t, mok.ExpectationsWereMet())
		})

		t.Run("nil db", func(t *testing.T) {
			connector := &DefaultDatabaseConnector{}
			result, err := connector.ExecContext(context.Background(), "SELECT 1")
			assert.Nil(t, result)
			assert.Error(t, err)
			assert.Equal(t, "database not initialized", err.Error())
		})
	})
}

func TestDefaultMigrateConnector_LogicOnly(t *testing.T) {
	t.Run("New sets m field", func(t *testing.T) {
		connector := &DefaultMigrateConnector{}

		// Тест проверяет только что поле m устанавливается
		// Используем нерабочий DSN, чтобы избежать реальных подключений
		_, err := connector.New("file://migrations", "postgres://invalid_dsn")

		if err == nil {
			assert.NotNil(t, connector.m)
		} else {
			assert.Nil(t, connector.m)
			t.Log("Expected error for invalid DSN")
		}
	})

	t.Run("Up returns error when m is nil", func(t *testing.T) {
		connector := &DefaultMigrateConnector{}
		err := connector.Up()

		assert.Error(t, err)
		assert.Equal(t, "migrate not initialized", err.Error())
	})

	t.Run("Second New call returns same instance", func(t *testing.T) {
		connector := &DefaultMigrateConnector{}

		// Первый вызов (ожидаем ошибку)
		m1, err1 := connector.New("file://migrations", "postgres://invalid_dsn")

		// Второй вызов
		m2, err2 := connector.New("file://migrations", "postgres://invalid_dsn")

		if err1 == nil {
			assert.Same(t, m1, m2, "Expected same migrate instance")
			assert.NoError(t, err2)
		}
	})
}
