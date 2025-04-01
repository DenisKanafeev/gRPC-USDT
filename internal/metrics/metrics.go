package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Метрики для сервиса
var (
	RateExchangeCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_exchange_calls_total",
			Help: "Total number of calls to GetRateFromExchange",
		},
		[]string{"method"},
	)

	RateExchangeLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rate_exchange_latency_seconds",
			Help:    "Latency of GetRateFromExchange method",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"method"},
	)

	binanceAPIRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "binance_api_requests_total",
			Help: "Total number of requests to Binance API",
		},
	)

	DBSaves = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "db_saves_total",
			Help: "Total number of successful saves to database",
		},
	)

	DBSaveLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "db_save_latency_seconds",
			Help:    "Latency of saving data to database",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5},
		},
	)
)

func init() {
	prometheus.MustRegister(RateExchangeCalls)
	prometheus.MustRegister(RateExchangeLatency)
	prometheus.MustRegister(binanceAPIRequests)
	prometheus.MustRegister(DBSaves)
	prometheus.MustRegister(DBSaveLatency)
}

// ExposeMetrics - экспозиция метрик через HTTP
func ExposeMetrics() http.Handler {
	return promhttp.Handler()
}
