package tracelog

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Lightweight Prometheus HTTP metrics — zero external dependencies.
// Records RED (Rate / Errors / Duration) metrics for every HTTP request.
// Exposes standard Prometheus text format on a metrics endpoint.
// ---------------------------------------------------------------------------

var (
	metricsMu sync.Mutex

	// http_requests_total{method, path, status}
	httpReqTotal  = map[string]int64{}
	httpReqLabels []string // ordered label sets for stable output

	// http_request_duration_seconds{method, path, le} — cumulative bucket counts
	httpReqDur    = map[string]int64{}
	httpDurLabels []string

	// http_request_duration_seconds_sum{method, path}
	httpReqDurSum = map[string]float64{}

	// http_requests_in_flight
	httpReqInFlight int64

	metricsServiceName string
)

func init() {
	metricsServiceName = os.Getenv("OTEL_SERVICE_NAME")
	if metricsServiceName == "" {
		if name := os.Getenv("SERVICE_NAME"); name != "" {
			metricsServiceName = name
		} else {
			metricsServiceName = "go_service"
		}
	}
}

// sanitizePrometheusMetricName maps OTEL service names (often hyphenated)
// onto Prometheus metric-name charset [a-zA-Z_:][a-zA-Z0-9_:]*.
// Hyphens in names like task-auth make the whole /metrics scrape fail.
func sanitizePrometheusMetricName(name string) string {
	if name == "" {
		return "go_service"
	}
	var b strings.Builder
	b.Grow(len(name))
	for i, r := range name {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == ':' ||
			(r >= '0' && r <= '9' && i > 0)
		if ok {
			b.WriteRune(r)
			continue
		}
		if r >= '0' && r <= '9' && i == 0 {
			b.WriteByte('_')
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	s := b.String()
	if s == "" || strings.Trim(s, "_") == "" {
		return "go_service"
	}
	return s
}

func metricPrefix() string {
	return sanitizePrometheusMetricName(metricsServiceName)
}

func resetHTTPMetricsForTest() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	httpReqTotal = map[string]int64{}
	httpReqLabels = nil
	httpReqDur = map[string]int64{}
	httpDurLabels = nil
	httpReqDurSum = map[string]float64{}
	httpReqInFlight = 0
}

// Default histogram buckets in seconds (Prometheus defaults).
var durationBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

// --- Path normalisation (prevent high cardinality) ---

var (
	metricsUUIDRe       = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	metricsHexRe        = regexp.MustCompile(`^[0-9a-fA-F]{16,}$`)
	metricsNumRe        = regexp.MustCompile(`^\d+$`)
	metricsTokenRe      = regexp.MustCompile(`^[A-Za-z0-9_-]{32,}$`)
	// Prefixed resource IDs: ws_/task_/cmt_/cpa_ + snowflake-ish value.
	// The tail MUST start with a digit (after an optional '-') so literal label
	// segments like "task_id" are not folded into ":task".
	metricsPrefixedIDRe = regexp.MustCompile(`^(ws|task|cmt|cpa)_-?[0-9][0-9A-Za-z]*$`)
)

func normalizeMetricsPath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		switch {
		case metricsUUIDRe.MatchString(p):
			parts[i] = ":uuid"
		case metricsPrefixedIDRe.MatchString(p):
			switch {
			case strings.HasPrefix(p, "ws_"):
				parts[i] = ":ws"
			case strings.HasPrefix(p, "task_"):
				parts[i] = ":task"
			case strings.HasPrefix(p, "cmt_"):
				parts[i] = ":cmt"
			default:
				parts[i] = ":cpa"
			}
		case metricsHexRe.MatchString(p):
			parts[i] = ":hex"
		case metricsNumRe.MatchString(p):
			parts[i] = ":id"
		case metricsTokenRe.MatchString(p):
			parts[i] = ":token"
		}
	}
	return "/" + strings.Join(parts, "/")
}

func labelKey(parts ...string) string {
	return strings.Join(parts, "|")
}

// --- Exported API ---

// WriteMetricsBody writes the Prometheus text-format HTTP metrics body to w.
// Callers are responsible for setting Content-Type and status code first.
// Use this when you need to combine tracelog HTTP metrics with custom
// application-level metrics in a single /metrics response.
func WriteMetricsBody(w io.Writer) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	writeMetricsLocked(w)
}

// MetricsHandler returns an http.Handler serving Prometheus text-format HTTP metrics.
func MetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		WriteMetricsBody(w)
	})
}

// --- Internal metrics formatting ---

func writeMetricsLocked(w io.Writer) {
	prefix := metricPrefix()

	// http_requests_total
	fmt.Fprintf(w, "# HELP %s_http_requests_total Total HTTP requests.\n", prefix)
	fmt.Fprintf(w, "# TYPE %s_http_requests_total counter\n", prefix)
	for _, ls := range httpReqLabels {
		if v, ok := httpReqTotal[ls]; ok {
			parts := strings.Split(ls, "|")
			if len(parts) == 3 {
				fmt.Fprintf(w, "%s_http_requests_total{method=\"%s\",path=\"%s\",status=\"%s\"} %d\n",
					prefix, parts[0], parts[1], parts[2], v)
			}
		}
	}

	// http_request_duration_seconds histogram
	fmt.Fprintf(w, "# HELP %s_http_request_duration_seconds HTTP request duration.\n", prefix)
	fmt.Fprintf(w, "# TYPE %s_http_request_duration_seconds histogram\n", prefix)
	for _, ls := range httpDurLabels {
		if v, ok := httpReqDur[ls]; ok {
			parts := strings.Split(ls, "|")
			if len(parts) == 3 {
				fmt.Fprintf(w, "%s_http_request_duration_seconds_bucket{method=\"%s\",path=\"%s\",le=\"%s\"} %d\n",
					prefix, parts[0], parts[1], parts[2], v)
			}
		}
	}
	// histogram _sum and _count (+Inf bucket is the observation count)
	for key, sum := range httpReqDurSum {
		mp := strings.Split(key, "|")
		if len(mp) != 2 {
			continue
		}
		infKey := labelKey(mp[0], mp[1], "+Inf")
		count := httpReqDur[infKey]
		fmt.Fprintf(w, "%s_http_request_duration_seconds_sum{method=\"%s\",path=\"%s\"} %g\n",
			prefix, mp[0], mp[1], sum)
		fmt.Fprintf(w, "%s_http_request_duration_seconds_count{method=\"%s\",path=\"%s\"} %d\n",
			prefix, mp[0], mp[1], count)
	}

	// http_requests_in_flight
	fmt.Fprintf(w, "# HELP %s_http_requests_in_flight Current in-flight requests.\n", prefix)
	fmt.Fprintf(w, "# TYPE %s_http_requests_in_flight gauge\n", prefix)
	fmt.Fprintf(w, "%s_http_requests_in_flight %d\n", prefix, httpReqInFlight)
}

// --- Middleware ---

// metricsRecorder captures the HTTP status code.
type metricsRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *metricsRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *metricsRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// MetricsMiddleware records HTTP RED metrics. Wrap your mux with this.
// Place inside the tracing middleware so trace context is already set up.
//
// Typical usage:
//
//	handler := tracelog.Middleware(corsMiddleware(tracelog.MetricsMiddleware(mux)))
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := normalizeMetricsPath(r.URL.Path)

		httpReqInFlight++
		defer func() { httpReqInFlight-- }()

		rec := &metricsRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		status := strconv.Itoa(rec.status)

		metricsMu.Lock()
		defer metricsMu.Unlock()

		// Counter: http_requests_total
		cKey := labelKey(r.Method, path, status)
		if _, exists := httpReqTotal[cKey]; !exists {
			httpReqLabels = append(httpReqLabels, cKey)
		}
		httpReqTotal[cKey]++

		// Histogram: cumulative buckets — increment every le >= observation.
		dur := time.Since(start).Seconds()
		sumKey := labelKey(r.Method, path)
		httpReqDurSum[sumKey] += dur
		for _, b := range durationBuckets {
			if dur <= b {
				incDurationBucket(r.Method, path, strconv.FormatFloat(b, 'f', -1, 64))
			}
		}
		incDurationBucket(r.Method, path, "+Inf")
	})
}

func incDurationBucket(method, path, le string) {
	dKey := labelKey(method, path, le)
	if _, exists := httpReqDur[dKey]; !exists {
		httpDurLabels = append(httpDurLabels, dKey)
	}
	httpReqDur[dKey]++
}

func init() {
	sort.Float64s(durationBuckets)
}
