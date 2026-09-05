// Package metrics provides a lightweight Prometheus HTTP middleware and
// a promhttp-based metrics handler for Go microservices using net/http.
//
// Exported API:
//
//	Middleware(next http.Handler) http.Handler   — wraps handler, records RED metrics
//	Handler() http.Handler                        — promhttp handler for /api/metrics
//
// Metrics exposed (all prefixed with the service name from OTEL_SERVICE_NAME env):
//
//	<service>_http_requests_total{method, path, status}       — counter
//	<service>_http_request_duration_seconds{method, path}     — histogram
//	<service>_http_requests_in_flight                         — gauge
package metrics

import (
	"net/http"
	"regexp"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var serviceName string

func init() {
	serviceName = os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "unknown_service"
	}
}

// --- Path normalization (low cardinality) ---

var (
	uuidRe   = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	hexRe    = regexp.MustCompile(`^[0-9a-fA-F]{16,}$`)
	numRe    = regexp.MustCompile(`^\d+$`)
)

// normalizePath replaces high-cardinality segments with placeholders.
func normalizePath(path string) string {
	parts := splitPath(path)
	for i, p := range parts {
		switch {
		case uuidRe.MatchString(p):
			parts[i] = ":uuid"
		case hexRe.MatchString(p):
			parts[i] = ":hex"
		case numRe.MatchString(p):
			parts[i] = ":id"
		}
	}
	return joinPath(parts)
}

func splitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{"/"}
	}
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	if len(parts) == 0 {
		return []string{"/"}
	}
	return parts
}

func joinPath(parts []string) string {
	if len(parts) == 1 && parts[0] == "/" {
		return "/"
	}
	result := ""
	for _, p := range parts {
		result += "/" + p
	}
	return result
}

// --- Prometheus collectors ---

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: serviceName + "_http_requests_total",
			Help: "Total number of HTTP requests handled.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    serviceName + "_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets, // .005 .01 .025 .05 .1 .25 .5 1 2.5 5 10
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: serviceName + "_http_requests_in_flight",
			Help: "Current number of HTTP requests being handled.",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpRequestsInFlight)
}

// --- Middleware ---

// statusRecorder captures the HTTP status code written.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// Middleware returns an http.Handler that records Prometheus RED metrics
// (Rate, Errors, Duration) for every request. Place it inside the trace-logging
// middleware so that tracing is already set up.
//
// Typical usage in main.go:
//
//	mux := http.NewServeMux()
//	mountRoutes(mux)
//	handler := tracelog.Middleware(corsMiddleware(metrics.Middleware(mux)))
//	http.ListenAndServe(addr, handler)
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := normalizePath(r.URL.Path)

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.status)

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// Handler returns an http.Handler that exposes Prometheus metrics via promhttp.
// Use this to replace the hand-written /api/metrics endpoints.
//
// Typical usage:
//
//	mux.Handle("GET /api/metrics", metrics.Handler())
func Handler() http.Handler {
	return promhttp.Handler()
}
