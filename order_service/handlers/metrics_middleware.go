package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// MetricsMiddleware automatically records metrics for all HTTP requests
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		route := getRouteName(r)
		status := strconv.Itoa(wrapped.statusCode)

		RecordHTTPMetrics(r.Method, route, status, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// getRouteName extracts the route pattern from the request
func getRouteName(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route != nil {
		if path, err := route.GetPathTemplate(); err == nil {
			return path
		}
	}
	return r.URL.Path
}
