package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-migrate/migrate/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDatabaseConnector реализует DatabaseConnector
type MockDatabaseConnector struct {
	mock.Mock
}

func (m *MockDatabaseConnector) Open(driverName, dataSourceName string) (*sql.DB, error) {
	args := m.Called(driverName, dataSourceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sql.DB), args.Error(1)
}

func (m *MockDatabaseConnector) Ping() error {
	return m.Called().Error(0)
}

func (m *MockDatabaseConnector) Close() error {
	return m.Called().Error(0)
}

func (m *MockDatabaseConnector) Exec(query string, args ...interface{}) (sql.Result, error) {
	callArgs := m.Called(query, args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(sql.Result), callArgs.Error(1)
}

// MockMigrateConnector реализует MigrateConnector
type MockMigrateConnector struct {
	mock.Mock
}

func (m *MockMigrateConnector) New(sourceURL, databaseURL string) (*migrate.Migrate, error) {
	args := m.Called(sourceURL, databaseURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*migrate.Migrate), args.Error(1)
}

func (m *MockMigrateConnector) Up() error {
	return m.Called().Error(0)
}

func TestNewStorage(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		setupMockDB func(*MockDatabaseConnector)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful initialization",
			dsn:  "valid_dsn",
			setupMockDB: func(m *MockDatabaseConnector) {
				m.On("Open", "pgx", "valid_dsn").Return(&sql.DB{}, nil)
				m.On("Ping").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "connection error",
			dsn:  "invalid_dsn",
			setupMockDB: func(m *MockDatabaseConnector) {
				m.On("Open", "pgx", "invalid_dsn").Return(nil, errors.New("connection failed"))
			},
			wantErr:     true,
			errContains: "failed to open database",
		},
		{
			name: "ping error",
			dsn:  "valid_dsn",
			setupMockDB: func(m *MockDatabaseConnector) {
				m.On("Open", "pgx", "valid_dsn").Return(&sql.DB{}, nil)
				m.On("Ping").Return(errors.New("ping failed"))
			},
			wantErr:     true,
			errContains: "database ping failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabaseConnector)
			tt.setupMockDB(mockDB)

			_, err := NewStorage(tt.dsn, mockDB, new(MockMigrateConnector))

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestStorage_Migrate(t *testing.T) {
	tests := []struct {
		name           string
		migrationsPath string
		setupMock      func(*MockMigrateConnector)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful migration",
			migrationsPath: "/migrations",
			setupMock: func(m *MockMigrateConnector) {
				m.On("New", "file:///migrations", "postgres://mock?x-migrations-table=schema_migrations").
					Return(&migrate.Migrate{}, nil)
				m.On("Up").Return(nil)
			},
			wantErr: false,
		},
		{
			name:           "empty path",
			migrationsPath: "",
			setupMock:      func(m *MockMigrateConnector) {},
			wantErr:        true,
			errContains:    "migrations path cannot be empty",
		},
		{
			name:           "migration init error",
			migrationsPath: "/migrations",
			setupMock: func(m *MockMigrateConnector) {
				m.On("New", mock.Anything, mock.Anything).
					Return(nil, errors.New("init error"))
			},
			wantErr:     true,
			errContains: "migration init failed",
		},
		{
			name:           "migration up error",
			migrationsPath: "/migrations",
			setupMock: func(m *MockMigrateConnector) {
				m.On("New", mock.Anything, mock.Anything).Return(&migrate.Migrate{}, nil)
				m.On("Up").Return(errors.New("up error"))
			},
			wantErr:     true,
			errContains: "migration up failed",
		},
		{
			name:           "no changes needed",
			migrationsPath: "/migrations",
			setupMock: func(m *MockMigrateConnector) {
				m.On("New", mock.Anything, mock.Anything).Return(&migrate.Migrate{}, nil)
				m.On("Up").Return(migrate.ErrNoChange)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMigrate := new(MockMigrateConnector)
			tt.setupMock(mockMigrate)

			s := &Storage{
				migrateConnector: mockMigrate,
				dsn:              "postgres://mock",
			}

			err := s.Migrate(tt.migrationsPath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			mockMigrate.AssertExpectations(t)
		})
	}
}

func TestStorage_SaveRate(t *testing.T) {
	tests := []struct {
		name        string
		ask         float64
		bid         float64
		askAmount   float64
		bidAmount   float64
		setupMock   func(sqlmock.Sqlmock)
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful save",
			ask:       1.1,
			bid:       1.2,
			askAmount: 100.0,
			bidAmount: 150.0,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`^INSERT INTO rates`).
					WithArgs(1.1, 1.2, 100.0, 150.0, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:      "exec error",
			ask:       1.1,
			bid:       1.2,
			askAmount: 100.0,
			bidAmount: 150.0,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`^INSERT INTO rates`).
					WillReturnError(errors.New("exec error"))
			},
			wantErr:     true,
			errContains: "save rate failed",
		},
		{
			name:      "zero values",
			ask:       0,
			bid:       0,
			askAmount: 0,
			bidAmount: 0,
			setupMock: func(m sqlmock.Sqlmock) {
				m.ExpectExec(`^INSERT INTO rates`).
					WithArgs(0.0, 0.0, 0.0, 0.0, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			s := &Storage{
				db: &DefaultDatabaseConnector{db: db},
			}

			err = s.SaveRate(tt.ask, tt.bid, tt.askAmount, tt.bidAmount, time.Now())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorage_Close(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockDatabaseConnector)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful close",
			setupMock: func(m *MockDatabaseConnector) {
				m.On("Close").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "close error",
			setupMock: func(m *MockDatabaseConnector) {
				m.On("Close").Return(errors.New("close error"))
			},
			wantErr:     true,
			errContains: "database close failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDatabaseConnector)
			tt.setupMock(mockDB)

			s := &Storage{db: mockDB}
			err := s.Close()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestStorage_createDatabase(t *testing.T) {
	tests := []struct {
		name        string
		dbName      string
		setupMock   func(sqlmock.Sqlmock)
		wantErr     bool
		errContains string
	}{
		{
			name:   "success",
			dbName: "test_db",
			setupMock: func(tempMock sqlmock.Sqlmock) {
				tempMock.ExpectPing()
				tempMock.ExpectExec(`DO \$\$.*`).WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:   "temp connection error",
			dbName: "test_db",
			setupMock: func(tempMock sqlmock.Sqlmock) {
				tempMock.ExpectPing().WillReturnError(errors.New("ping error"))
			},
			wantErr:     true,
			errContains: "temp database ping failed",
		},
		{
			name:   "database exists",
			dbName: "test_db",
			setupMock: func(tempMock sqlmock.Sqlmock) {
				tempMock.ExpectPing()
				tempMock.ExpectExec(`DO \$\$`).WillReturnError(fmt.Errorf("already exists"))
			},
			wantErr:     true,
			errContains: "failed to create database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDB, tempMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			defer tempDB.Close()

			// Создаем основной мок, но не настраиваем ожидания для него
			mainDB, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err)
			defer mainDB.Close()

			tt.setupMock(tempMock)

			s := &Storage{
				db:  &DefaultDatabaseConnector{db: mainDB},
				dsn: "postgres://user:pass@localhost/test_db",
			}

			err = s.createDatabase(tt.dbName, &DefaultDatabaseConnector{db: tempDB})

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, tempMock.ExpectationsWereMet())
		})
	}
}

func TestDefaultDatabaseConnector(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (*DefaultDatabaseConnector, sqlmock.Sqlmock)
		testFunc    func(*DefaultDatabaseConnector) error
		wantErr     bool
		errContains string
	}{
		{
			name: "Open success",
			setup: func() (*DefaultDatabaseConnector, sqlmock.Sqlmock) {
				return &DefaultDatabaseConnector{}, nil
			},
			testFunc: func(c *DefaultDatabaseConnector) error {
				_, err := c.Open("pgx", "dsn")
				return err
			},
			wantErr: false,
		},
		{
			name: "Ping error - not initialized",
			setup: func() (*DefaultDatabaseConnector, sqlmock.Sqlmock) {
				return &DefaultDatabaseConnector{}, nil
			},
			testFunc: func(c *DefaultDatabaseConnector) error {
				return c.Ping()
			},
			wantErr:     true,
			errContains: "database not initialized",
		},
		{
			name: "Exec error - not initialized",
			setup: func() (*DefaultDatabaseConnector, sqlmock.Sqlmock) {
				return &DefaultDatabaseConnector{}, nil
			},
			testFunc: func(c *DefaultDatabaseConnector) error {
				_, err := c.Exec("SELECT 1")
				return err
			},
			wantErr:     true,
			errContains: "database not initialized",
		},
		{
			name: "Ping success",
			setup: func() (*DefaultDatabaseConnector, sqlmock.Sqlmock) {
				db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
				require.NoError(t, err)
				mock.ExpectPing()
				return &DefaultDatabaseConnector{db: db}, mock
			},
			testFunc: func(c *DefaultDatabaseConnector) error {
				return c.Ping()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, mock := tt.setup()
			err := tt.testFunc(c)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}

			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
				// Явно ожидаем Close для мока
				if c.db != nil {
					mock.ExpectClose()
					require.NoError(t, c.db.Close())
					assert.NoError(t, mock.ExpectationsWereMet())
				}
			}
		})
	}
}

func TestDefaultMigrateConnector(t *testing.T) {
	tests := []struct {
		name        string
		testFunc    func(*DefaultMigrateConnector) error
		wantErr     bool
		errContains string
	}{
		{
			name: "Up error - not initialized",
			testFunc: func(c *DefaultMigrateConnector) error {
				return c.Up()
			},
			wantErr:     true,
			errContains: "migrate not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DefaultMigrateConnector{}
			err := tt.testFunc(c)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultDatabaseConnector_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("SELECT 1").WillReturnError(errors.New("exec error"))

	connector := &DefaultDatabaseConnector{db: db}
	_, err = connector.Exec("SELECT 1")
	assert.Error(t, err)
}

func TestDefaultDatabaseConnector_NotInitialized(t *testing.T) {
	connector := &DefaultDatabaseConnector{} // db == nil

	err := connector.Ping()
	assert.Error(t, err)

	_, err = connector.Exec("SELECT 1")
	assert.Error(t, err)
}

func TestDefaultMigrateConnector_NewError(t *testing.T) {
	// Для этого теста нужно создать невалидную конфигурацию миграций
	connector := &DefaultMigrateConnector{}
	_, err := connector.New("invalid://", "invalid://")
	assert.Error(t, err)
}

func TestDefaultDatabaseConnector_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		db, err := sql.Open("pgx", "mock_dsn")
		require.NoError(t, err)
		connector := &DefaultDatabaseConnector{db: db}
		err = connector.Close()
		assert.NoError(t, err)
	})

	t.Run("close nil db", func(t *testing.T) {
		connector := &DefaultDatabaseConnector{}
		err := connector.Close()
		assert.NoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		// Мокаем ошибку при закрытии
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		mock.ExpectClose().WillReturnError(errors.New("close error"))
		connector := &DefaultDatabaseConnector{db: db}
		err = connector.Close()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "close error")
	})
}
