package tracelog

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Prometheus metric names: [a-zA-Z_:][a-zA-Z0-9_:]*
var prometheusMetricNameRe = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)

func TestNormalizeMetricsPath_PrefixedResourceIDs(t *testing.T) {
	got := normalizeMetricsPath("/api/cloud/compute/start-vm/tenant_id/850256677331562496/workspace_id/ws_-2309487803472456748")
	want := "/api/cloud/compute/start-vm/tenant_id/:hex/workspace_id/:ws"
	if got != want {
		t.Fatalf("start-vm path: got %q want %q", got, want)
	}
	got = normalizeMetricsPath("/api/cloud/server-container-token/heartbeat/tenant_id/abc/workspace_id/ws_1/task_id/task_880716211791360000/comment_id/cmt_880716216098910208")
	if !strings.Contains(got, "/task_id/:task/") || !strings.Contains(got, "/comment_id/:cmt") {
		t.Fatalf("heartbeat path not folded: %q", got)
	}
	got = normalizeMetricsPath("/api/cloud/cloud-platform/cpa_-2305033758844803750/available-instances/tenant_id/850256677331562496")
	if !strings.Contains(got, "/cloud-platform/:cpa/") || !strings.Contains(got, "tenant_id/:hex") {
		t.Fatalf("available-instances path not folded: %q", got)
	}
}

func TestSanitizePrometheusMetricName_HyphenatedService(t *testing.T) {
	got := sanitizePrometheusMetricName("task-auth")
	if got != "task_auth" {
		t.Fatalf("task-auth → %q, want task_auth", got)
	}
	if !prometheusMetricNameRe.MatchString(got) {
		t.Fatalf("sanitized name %q is not a valid Prometheus metric name prefix", got)
	}
	got = sanitizePrometheusMetricName("task-cloud-service")
	if got != "task_cloud_service" {
		t.Fatalf("task-cloud-service → %q, want task_cloud_service", got)
	}
}

func TestMetricsExpositionUsesSanitizedPrefix(t *testing.T) {
	resetHTTPMetricsForTest()
	orig := metricsServiceName
	t.Cleanup(func() {
		metricsServiceName = orig
		resetHTTPMetricsForTest()
	})
	metricsServiceName = "task-auth"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := MetricsMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	if strings.Contains(body, "task-auth_http") {
		t.Fatalf("exposition still contains hyphenated metric names:\n%s", body)
	}
	if !strings.Contains(body, "# TYPE task_auth_http_requests_total counter") {
		t.Fatalf("missing sanitized TYPE line, body=\n%s", body)
	}
	if !strings.Contains(body, "# TYPE task_auth_http_request_duration_seconds histogram") {
		t.Fatalf("missing sanitized histogram TYPE, body=\n%s", body)
	}
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "# TYPE ") {
			continue
		}
		// # TYPE <name> <type>
		fields := strings.Fields(line)
		if len(fields) < 4 {
			t.Fatalf("malformed TYPE line: %q", line)
		}
		if !prometheusMetricNameRe.MatchString(fields[2]) {
			t.Fatalf("invalid Prometheus metric name in TYPE line: %q", line)
		}
	}
}

func TestDurationHistogramIsCumulativeAndMonotonic(t *testing.T) {
	resetHTTPMetricsForTest()
	orig := metricsServiceName
	t.Cleanup(func() {
		metricsServiceName = orig
		resetHTTPMetricsForTest()
	})
	metricsServiceName = "go_service"

	mux := http.NewServeMux()
	mux.HandleFunc("/api/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(40 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	handler := MetricsMiddleware(mux)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/slow", nil))

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	buckets := parseDurationBuckets(t, body, "GET", "/api/slow")
	if len(buckets) == 0 {
		t.Fatalf("no duration buckets for /api/slow, body=\n%s", body)
	}

	// 40ms observation must NOT land in le=0.005 / 0.01 / 0.025.
	for _, le := range []string{"0.005", "0.01", "0.025"} {
		if buckets[le] != 0 {
			t.Fatalf("le=%s count=%d want 0 (inverted histogram would increment small buckets)", le, buckets[le])
		}
	}
	if buckets["0.05"] < 1 {
		t.Fatalf("le=0.05 count=%d want >=1 after 40ms sleep, buckets=%v", buckets["0.05"], buckets)
	}

	// Cumulative: each larger le >= previous.
	order := []string{"0.005", "0.01", "0.025", "0.05", "0.1", "0.25", "0.5", "1", "2.5", "5", "10", "+Inf"}
	var prev int64
	for _, le := range order {
		v := buckets[le]
		if v < prev {
			t.Fatalf("histogram not monotonic: le=%s count=%d < previous %d; buckets=%v", le, v, prev, buckets)
		}
		prev = v
	}
	if buckets["+Inf"] < 1 {
		t.Fatalf("missing +Inf bucket count, buckets=%v body=\n%s", buckets, body)
	}

	sum := parseDurationSum(t, body, "GET", "/api/slow")
	if sum < 0.03 {
		t.Fatalf("duration sum=%v want >= 0.03 (was hardcoded 0 before fix)", sum)
	}
	count := parseDurationCount(t, body, "GET", "/api/slow")
	if count != 1 {
		t.Fatalf("duration count=%d want 1", count)
	}
}

func parseDurationBuckets(t *testing.T, body, method, path string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	prefix := `http_request_duration_seconds_bucket{method="` + method + `",path="` + path + `",le="`
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, prefix) {
			continue
		}
		// ...le="<le>"} <value>
		idx := strings.Index(line, `le="`)
		if idx < 0 {
			continue
		}
		rest := line[idx+4:]
		end := strings.Index(rest, `"`)
		if end < 0 {
			continue
		}
		le := rest[:end]
		fields := strings.Fields(line)
		v, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
		if err != nil {
			t.Fatalf("parse bucket value %q: %v", line, err)
		}
		out[le] = v
	}
	return out
}

func parseDurationSum(t *testing.T, body, method, path string) float64 {
	t.Helper()
	needle := `http_request_duration_seconds_sum{method="` + method + `",path="` + path + `"} `
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		fields := strings.Fields(line)
		v, err := strconv.ParseFloat(fields[len(fields)-1], 64)
		if err != nil {
			t.Fatalf("parse sum %q: %v", line, err)
		}
		return v
	}
	t.Fatalf("sum line not found for %s %s", method, path)
	return 0
}

func parseDurationCount(t *testing.T, body, method, path string) int64 {
	t.Helper()
	needle := `http_request_duration_seconds_count{method="` + method + `",path="` + path + `"} `
	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, needle) {
			continue
		}
		fields := strings.Fields(line)
		v, err := strconv.ParseInt(fields[len(fields)-1], 10, 64)
		if err != nil {
			t.Fatalf("parse count %q: %v", line, err)
		}
		return v
	}
	t.Fatalf("count line not found for %s %s", method, path)
	return 0
}
