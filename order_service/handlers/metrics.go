package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP request counter
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP request duration histogram
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets, // Default buckets: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"method", "endpoint"},
	)

	// Order-specific metrics
	ordersCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total number of orders created",
		},
	)

	ordersFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_failed_total",
			Help: "Total number of failed orders",
		},
	)

	orderAmountTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "order_amount_total",
			Help: "Total amount of all orders",
		},
	)
)

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecordHTTPMetrics records HTTP request metrics
func RecordHTTPMetrics(method, endpoint, status string, duration float64) {
	httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordOrderCreated increments the orders created counter
func RecordOrderCreated(amount float64) {
	ordersCreatedTotal.Inc()
	orderAmountTotal.Add(amount)
}

// RecordOrderFailed increments the orders failed counter
func RecordOrderFailed() {
	ordersFailedTotal.Inc()
}
