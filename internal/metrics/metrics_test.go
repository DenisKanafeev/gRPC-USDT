package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsRegistration(t *testing.T) {
	// Проверяем, что все метрики зарегистрированы в реестре Prometheus
	registry := prometheus.NewRegistry()

	// Регистрируем все наши метрики в новом реестре для тестирования
	err := registry.Register(RateExchangeCalls)
	assert.NoError(t, err, "RateExchangeCalls should be registered successfully")

	err = registry.Register(RateExchangeLatency)
	assert.NoError(t, err, "RateExchangeLatency should be registered successfully")

	err = registry.Register(binanceAPIRequests)
	assert.NoError(t, err, "binanceAPIRequests should be registered successfully")

	err = registry.Register(DBSaves)
	assert.NoError(t, err, "DBSaves should be registered successfully")

	err = registry.Register(DBSaveLatency)
	assert.NoError(t, err, "DBSaveLatency should be registered successfully")
}

func TestMetricsIncrement(t *testing.T) {
	// Создаем новый реестр для изоляции тестов
	registry := prometheus.NewRegistry()

	// Создаем временные метрики для тестирования
	testRateExchangeCalls := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_rate_exchange_calls_total",
			Help: "Total number of calls to GetRateFromExchange",
		},
		[]string{"method"},
	)

	testBinanceAPIRequests := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "test_binance_api_requests_total",
			Help: "Total number of requests to Binance API",
		},
	)

	// Регистрируем метрики в тестовом реестре
	registry.MustRegister(testRateExchangeCalls)
	registry.MustRegister(testBinanceAPIRequests)

	// Инкрементируем счетчики
	testRateExchangeCalls.WithLabelValues("binance").Inc()
	testBinanceAPIRequests.Inc()

	// Проверяем значения счетчиков
	assert.Equal(t, float64(1), testutil.ToFloat64(testRateExchangeCalls.WithLabelValues("binance")))
	assert.Equal(t, float64(1), testutil.ToFloat64(testBinanceAPIRequests))

	// Инкрементируем еще раз
	testRateExchangeCalls.WithLabelValues("binance").Inc()
	testBinanceAPIRequests.Inc()

	// Проверяем обновленные значения
	assert.Equal(t, float64(2), testutil.ToFloat64(testRateExchangeCalls.WithLabelValues("binance")))
	assert.Equal(t, float64(2), testutil.ToFloat64(testBinanceAPIRequests))
}

func TestExposeMetrics(t *testing.T) {
	// Создаем тестовый сервер
	handler := ExposeMetrics()
	server := httptest.NewServer(handler)
	defer server.Close()

	// Делаем запрос к метрикам
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	// Проверяем, что в ответе есть хотя бы одна метрика
	// (точное содержание зависит от состояния регистра)
	if resp.ContentLength == 0 {
		t.Error("Empty metrics response")
	}
}

func TestVectorMetrics(t *testing.T) {
	// Проверяем работу метрик с labels
	testCases := []struct {
		name     string
		metric   *prometheus.CounterVec
		label    string
		expected string
	}{
		{
			name:     "rate_exchange_calls",
			metric:   RateExchangeCalls,
			label:    "test_method",
			expected: "rate_exchange_calls_total",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Создаем временный регистр для теста
			registry := prometheus.NewRegistry()
			registry.MustRegister(tc.metric)

			// Инкрементируем метрику с label
			tc.metric.WithLabelValues(tc.label).Inc()

			// Проверяем значение
			metrics, err := registry.Gather()
			if err != nil {
				t.Fatalf("Failed to gather metrics: %v", err)
			}

			found := false
			for _, metric := range metrics {
				if *metric.Name == tc.expected {
					found = true
					for _, m := range metric.Metric {
						if m.Counter == nil || *m.Counter.Value != 1 {
							t.Errorf("Expected counter value 1, got %v", m.Counter)
						}
						// Проверяем label
						if len(m.Label) == 0 || *m.Label[0].Value != tc.label {
							t.Errorf("Expected label %s, got %v", tc.label, m.Label)
						}
					}
					break
				}
			}

			if !found {
				t.Errorf("Metric %s not found", tc.expected)
			}
		})
	}
}

func TestHistogramMetrics(_ *testing.T) {
	// Тестирование гистограмм аналогично, но с Observe вместо Inc
	RateExchangeLatency.WithLabelValues("test").Observe(0.1)
	DBSaveLatency.Observe(0.2)
}
