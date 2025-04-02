package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"gRPC-USDT/api/proto"
	"gRPC-USDT/internal/config"
	"gRPC-USDT/internal/metrics"
)

// MockHTTPClient мок для HTTPClient
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// MockRateStorage мок для RateStorage
type MockRateStorage struct {
	mock.Mock
}

func (m *MockRateStorage) SaveRate(ctx context.Context, ask, bid, askAmount, bidAmount float64, ts time.Time) error {
	args := m.Called(ctx, ask, bid, askAmount, bidAmount, ts)
	return args.Error(0)
}

func TestRateService_GetRateFromExchange(t *testing.T) {
	// Сохраняем оригинальные метрики
	originalMetrics := struct {
		RateExchangeCalls   *prometheus.CounterVec
		RateExchangeLatency *prometheus.HistogramVec
	}{
		RateExchangeCalls:   metrics.RateExchangeCalls,
		RateExchangeLatency: metrics.RateExchangeLatency,
	}

	// Восстанавливаем оригинальные метрики после тестов
	defer func() {
		metrics.RateExchangeCalls = originalMetrics.RateExchangeCalls
		metrics.RateExchangeLatency = originalMetrics.RateExchangeLatency
	}()

	// Инициализация тестовых метрик
	metrics.RateExchangeCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_rate_exchange_calls",
			Help: "Number of calls to exchange API (test)",
		},
		[]string{"method"},
	)
	metrics.RateExchangeLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_rate_exchange_latency",
			Help:    "Latency of exchange API calls (test)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Инициализация tracer provider
	otel.SetTracerProvider(noop.NewTracerProvider())

	testConfig := &config.Config{BinanceAPIURL: "https://test-api.com"}
	testLogger := zap.NewNop()

	tests := []struct {
		name           string
		mockHTTPResp   *http.Response
		mockHTTPErr    error
		mockStorageErr error
		wantErr        bool
		wantResp       *proto.GetRateFromExchangeResponse
	}{
		{
			name: "success",
			mockHTTPResp: &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader([]byte(`{
					"asks": [["100.0", "1.0"]],
					"bids": [["99.0", "2.0"]]
				}`))),
			},
			wantResp: &proto.GetRateFromExchangeResponse{
				Success:   true,
				Ask:       100.0,
				Bid:       99.0,
				AskAmount: 2.0,
				BidAmount: 1.0,
			},
		},
		{
			name:        "http client error",
			mockHTTPErr: errors.New("connection refused"),
			wantErr:     true,
		},
		{
			name: "non-200 status code",
			mockHTTPResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "invalid request"}`))),
			},
			wantErr: true,
		},
		{
			name: "invalid JSON response",
			mockHTTPResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`invalid json`))),
			},
			wantErr: true,
			// Не настраиваем mockStorage для этого кейса
		},
		{
			name: "empty order book",
			mockHTTPResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"asks": [], "bids": []}`))),
			},
			wantErr: true,
			// Не настраиваем mockStorage для этого кейса
		},
		{
			name: "malformed order data",
			mockHTTPResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"asks": [["invalid", "data"]], "bids": [["100.0", "1.0"]]}`))),
			},
			wantErr: true,
			// Не настраиваем mockStorage для этого кейса
		},
		{
			name: "storage save error",
			mockHTTPResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"asks": [["100.0", "1.0"]], "bids": [["99.0", "2.0"]]}`))),
			},
			mockStorageErr: errors.New("db connection failed"),
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сбрасываем метрики перед тестом
			metrics.RateExchangeCalls.Reset()
			metrics.RateExchangeLatency.Reset()

			// Мокируем HTTP клиент
			mockHTTP := new(MockHTTPClient)
			mockHTTP.On("Do", mock.Anything).Return(tt.mockHTTPResp, tt.mockHTTPErr)

			// Мокируем хранилище ТОЛЬКО для успешных случаев
			mockStorage := new(MockRateStorage)
			if tt.mockStorageErr != nil ||
				(tt.mockHTTPResp != nil &&
					tt.mockHTTPResp.StatusCode == http.StatusOK &&
					!tt.wantErr) {
				mockStorage.On("SaveRate",
					mock.Anything, // context
					mock.Anything, // ask
					mock.Anything, // bid
					mock.Anything, // askAmount
					mock.Anything, // bidAmount
					mock.Anything, // timestamp
				).Return(tt.mockStorageErr)
			}

			// Создаем сервис с моками
			service := NewRateService(mockStorage, testLogger, testConfig, mockHTTP)

			// Вызываем метод
			resp, err := service.GetRateFromExchange(context.Background(), &proto.GetRateFromExchangeRequest{})

			// Проверки
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantResp.Success, resp.Success)
				assert.Equal(t, tt.wantResp.Ask, resp.Ask)
				assert.Equal(t, tt.wantResp.Bid, resp.Bid)
				assert.Equal(t, tt.wantResp.AskAmount, resp.AskAmount)
				assert.Equal(t, tt.wantResp.BidAmount, resp.BidAmount)
			}

			mockHTTP.AssertExpectations(t)

			// Проверяем мок хранилища только если он должен был вызваться
			if tt.mockStorageErr != nil ||
				(tt.mockHTTPResp != nil &&
					tt.mockHTTPResp.StatusCode == http.StatusOK &&
					!tt.wantErr) {
				mockStorage.AssertExpectations(t)
			}
		})
	}
}

func TestProcessOrder(t *testing.T) {
	tests := []struct {
		name      string
		order     []string
		wantPrice float64
		wantVol   float64
		wantErr   bool
	}{
		{
			name:      "valid order",
			order:     []string{"100.0", "1.0"},
			wantPrice: 100.0,
			wantVol:   1.0,
		},
		{
			name:    "invalid price",
			order:   []string{"invalid", "1.0"},
			wantErr: true,
		},
		{
			name:    "invalid volume",
			order:   []string{"100.0", "invalid"},
			wantErr: true,
		},
		{
			name:    "short slice",
			order:   []string{"100.0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, vol, err := processOrder(tt.order)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantPrice, price)
			assert.Equal(t, tt.wantVol, vol)
		})
	}
}
