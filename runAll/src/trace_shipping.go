package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

const (
	traceShippingPendingIssue   = "尚未纳入上次验证，请点击「立即验证」"
	traceShippingStatusPending  = "pending"
	traceShippingStatusVerified = "verified"
	traceShippingStatusStale    = "stale"
	traceShippingDefaultWait    = 15 * time.Second
	traceShippingPromtailSettle = 2 * time.Second
	traceShippingLokiPollEvery  = 500 * time.Millisecond
)

var verifyTraceShippingLoki = defaultVerifyTraceShippingLoki

func defaultVerifyTraceShippingLoki(r *Runner, ctx context.Context, probeID string) (bool, string, map[string]int, error) {
	lokiClient := infrastructure.NewHTTPLokiQueryClient(r.cfg.Observability.LokiURL)
	if err := lokiClient.Ready(ctx); err != nil {
		return false, err.Error(), nil, nil
	}
	counts, err := lokiClient.CountLogsByJob(ctx, probeID, 15*time.Minute)
	if err != nil {
		return false, err.Error(), nil, nil
	}
	return true, "", counts, nil
}

func (r *Runner) configuredServicesForTraceShipping() []Service {
	if r != nil && strings.TrimSpace(r.cfgPath) != "" {
		if cfg, err := loadRuntimeConfig(r.cfgPath); err == nil && cfg != nil {
			return cfg.Flatten()
		}
	}
	if r != nil && r.cfg != nil {
		return r.cfg.Flatten()
	}
	return nil
}

func (r *Runner) regeneratePromtailConfigForTraceShipping(ctx context.Context) string {
	if r == nil {
		return "skipped"
	}
	root, err := r.monorepoRoot()
	if err != nil {
		log.Printf("[runAll] trace shipping: promtail config regen skipped: %v", err)
		return "skipped"
	}
	dummy := Service{Name: "trace-shipping-promtail", WorkingDir: root}
	if err := r.runLifecycleCommand(ctx, &dummy, "bash runAll/scripts/generate-promtail-config.sh"); err != nil {
		log.Printf("[runAll] trace shipping: promtail config regen failed: %v", err)
		return "failed"
	}
	return "ok"
}

var reloadPromtailForTraceShippingCmd = `set -euo pipefail
if docker ps --filter "name=^aimonitor-promtail$" --format '{{.Names}}' 2>/dev/null | grep -q .; then
  docker restart aimonitor-promtail >/dev/null
  echo ok
  exit 0
fi
if docker ps --filter "name=^aimonitor-promtail-local$" --format '{{.Names}}' 2>/dev/null | grep -q .; then
  docker restart aimonitor-promtail-local >/dev/null
  echo ok
  exit 0
fi
if [[ -f AiMonitor/docker-compose.yaml ]]; then
  (
    cd AiMonitor
    if docker compose version >/dev/null 2>&1; then
      docker compose up -d promtail >/dev/null
    else
      docker-compose up -d promtail >/dev/null
    fi
  )
  if docker ps --filter "name=^aimonitor-promtail$" --format '{{.Names}}' 2>/dev/null | grep -q .; then
    echo ok
    exit 0
  fi
fi
echo not_running`

// reloadPromtailShellRun executes the promtail reload shell. Extracted as a var so
// unit tests can assert a test-rooted Runner never reaches a real docker command
// (OPT-20260903-004).
var reloadPromtailShellRun = func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

// promtailLogRoot returns the log root this Runner hands to promtail (via
// RUNALL_LOG_ROOT) and whether it is safe to touch the shared aimonitor-promtail
// container. Unit tests construct Runners whose Logging.FileRoot is a Go
// t.TempDir() (e.g. /tmp/TestAPIObservabilityClearAll3904641115/001); letting
// those execute the real reload would repoint the shared container's
// /var/log/runall mount at an empty temp dir and starve Loki (OPT-20260903-004).
func promtailLogRoot(r *Runner) (string, bool) {
	root := ""
	if r != nil && r.cfg != nil {
		root = strings.TrimSpace(r.cfg.Logging.FileRoot)
	}
	if root == "" {
		root = strings.TrimSpace(os.Getenv("RUNALL_LOG_ROOT"))
	}
	if root == "" {
		return "", false
	}
	return root, !isGoTestTempRoot(root)
}

// isGoTestTempRoot reports whether root is under a Go test temp dir (t.TempDir()).
// Go names the per-test directory "Test<TestName><random>" directly under
// os.TempDir(); t.TempDir() may append "/NNN" for repeated calls in one test.
// Real roots such as /tmp/ram-work/logs or the deploy root do not match.
func isGoTestTempRoot(root string) bool {
	clean := filepath.Clean(root)
	tmp := filepath.Clean(os.TempDir())
	rel, err := filepath.Rel(tmp, clean)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	first := strings.SplitN(rel, string(filepath.Separator), 2)[0]
	return strings.HasPrefix(first, "Test")
}

func (r *Runner) reloadPromtailForTraceShipping(ctx context.Context) string {
	if r == nil {
		return "skipped"
	}
	if _, prodRoot := promtailLogRoot(r); !prodRoot {
		log.Printf("[runAll] trace shipping: skip shared promtail reload for non-production log root (OPT-20260903-004)")
		return "skipped"
	}
	root, err := r.monorepoRoot()
	if err != nil {
		log.Printf("[runAll] trace shipping: promtail reload skipped: %v", err)
		return "skipped"
	}
	dummy := Service{Name: "trace-shipping-promtail-reload", WorkingDir: root}
	cmd := exec.CommandContext(ctx, getBashPath(), "-c", reloadPromtailForTraceShippingCmd)
	cmd.Dir = root
	cmd.Env = r.buildServiceEnv(dummy)
	out, err := reloadPromtailShellRun(ctx, cmd)
	trimmed := strings.TrimSpace(string(out))
	if trimmed != "" {
		log.Printf("[runAll] trace shipping: promtail reload output: %s", trimmed)
	}
	if err != nil {
		log.Printf("[runAll] trace shipping: promtail reload failed: %v", err)
		return "failed"
	}
	switch trimmed {
	case "ok":
		return "ok"
	case "not_running":
		return "not_running"
	default:
		if trimmed == "" {
			return "not_running"
		}
		return "failed"
	}
}

func (r *Runner) preparePromtailForTraceShipping(ctx context.Context) string {
	r.regeneratePromtailConfigForTraceShipping(ctx)
	reload := r.reloadPromtailForTraceShipping(ctx)
	if reload == "ok" {
		select {
		case <-ctx.Done():
		case <-time.After(traceShippingPromtailSettle):
		}
	}
	return reload
}

func pollLokiForProbeCounts(ctx context.Context, r *Runner, probeID string, serviceNames []string, maxWait time.Duration) (bool, string, map[string]int) {
	query := func() (bool, string, map[string]int) {
		ready, errStr, counts, _ := verifyTraceShippingLoki(r, ctx, probeID)
		return ready, errStr, counts
	}
	lokiReady, lokiErr, counts := query()
	if maxWait <= 0 {
		return lokiReady, lokiErr, counts
	}
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if lokiReady && allServicesInLokiCounts(counts, serviceNames) {
			return lokiReady, lokiErr, counts
		}
		select {
		case <-ctx.Done():
			return lokiReady, lokiErr, counts
		case <-time.After(traceShippingLokiPollEvery):
			lokiReady, lokiErr, counts = query()
		}
	}
	return lokiReady, lokiErr, counts
}

func allServicesInLokiCounts(counts map[string]int, serviceNames []string) bool {
	if len(serviceNames) == 0 || counts == nil {
		return false
	}
	for _, name := range serviceNames {
		if counts[name] <= 0 {
			return false
		}
	}
	return true
}

func parseTraceShippingWait(raw string, defaultWait time.Duration) time.Duration {
	if defaultWait <= 0 {
		defaultWait = traceShippingDefaultWait
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultWait
	}
	sec, err := strconv.Atoi(raw)
	if err != nil || sec < 0 || sec > 30 {
		return defaultWait
	}
	return time.Duration(sec) * time.Second
}

// TraceShippingReportForUI returns the cached report aligned with the current runAll.yaml service list.
func (r *Runner) TraceShippingReportForUI() *domain.TraceShippingReport {
	if r == nil {
		return nil
	}
	r.traceShippingMu.RLock()
	cached := r.traceShippingLast
	r.traceShippingMu.RUnlock()
	return alignTraceShippingReportWithConfig(r, cached)
}

func alignTraceShippingReportWithConfig(r *Runner, report *domain.TraceShippingReport) *domain.TraceShippingReport {
	services := r.configuredServicesForTraceShipping()
	if len(services) == 0 {
		if report == nil {
			return nil
		}
		copy := *report
		return &copy
	}

	configuredTotal := len(services)
	if report == nil {
		out := &domain.TraceShippingReport{
			VerificationStatus: traceShippingStatusPending,
			Summary: domain.TraceShippingSummary{
				Total:           configuredTotal,
				ConfiguredTotal: configuredTotal,
				Failed:          configuredTotal,
			},
			Services: make([]domain.TraceShippingServiceResult, 0, configuredTotal),
		}
		for _, svc := range services {
			entry := domain.TraceShippingServiceResult{
				ServiceName: svc.Name,
				Issue:       traceShippingPendingIssue,
			}
			if st := r.store.Get(svc.Name); st != nil {
				entry.ServiceStatus = string(st.Status)
			}
			out.Services = append(out.Services, entry)
		}
		sortTraceShippingServices(out.Services)
		return out
	}

	byName := make(map[string]domain.TraceShippingServiceResult, len(report.Services))
	for _, svc := range report.Services {
		byName[svc.ServiceName] = svc
	}

	merged := make([]domain.TraceShippingServiceResult, 0, configuredTotal)
	stale := false
	for _, svc := range services {
		entry, ok := byName[svc.Name]
		if !ok {
			entry = domain.TraceShippingServiceResult{
				ServiceName: svc.Name,
				Issue:       traceShippingPendingIssue,
			}
			stale = true
		}
		if st := r.store.Get(svc.Name); st != nil {
			entry.ServiceStatus = string(st.Status)
		}
		if report.ProbeTraceID != "" && entry.GrafanaURL == "" && entry.Issue != traceShippingPendingIssue {
			if link, err := domain.GrafanaTraceLink(
				r.cfg.Observability.GrafanaURL,
				r.cfg.Observability.TraceDashboardUID,
				report.ProbeTraceID,
			); err == nil {
				entry.GrafanaURL = link + "&var-service=" + url.QueryEscape(entry.ServiceName)
			}
		}
		merged = append(merged, entry)
	}

	out := *report
	out.VerificationStatus = traceShippingStatusVerified
	if stale || len(report.Services) != configuredTotal {
		out.VerificationStatus = traceShippingStatusStale
	}
	out.Services = merged
	out.Summary = summarizeTraceShippingServices(merged, configuredTotal)
	sortTraceShippingServices(out.Services)
	return &out
}

func summarizeTraceShippingServices(services []domain.TraceShippingServiceResult, configuredTotal int) domain.TraceShippingSummary {
	summary := domain.TraceShippingSummary{
		Total:           configuredTotal,
		ConfiguredTotal: configuredTotal,
	}
	for i := range services {
		svc := services[i]
		if svc.LocalProbeOK {
			summary.LocalOK++
		}
		if svc.LokiVisible {
			summary.LokiOK++
		}
		if strings.TrimSpace(svc.Issue) != "" {
			summary.Failed++
		}
	}
	return summary
}

func sortTraceShippingServices(services []domain.TraceShippingServiceResult) {
	sort.Slice(services, func(i, j int) bool {
		if services[i].LokiVisible != services[j].LokiVisible {
			return !services[i].LokiVisible && services[j].LokiVisible
		}
		if services[i].LocalProbeOK != services[j].LocalProbeOK {
			return !services[i].LocalProbeOK && services[j].LocalProbeOK
		}
		return services[i].ServiceName < services[j].ServiceName
	})
}

// VerifyTraceShipping emits per-service probe logs and checks local tee + Loki visibility.
func (r *Runner) VerifyTraceShipping(ctx context.Context, wait time.Duration) (*domain.TraceShippingReport, error) {
	if r == nil || r.cfg == nil {
		return nil, errRunnerRequired("runner is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if wait < 0 {
		wait = traceShippingDefaultWait
	}

	promtailReload := r.preparePromtailForTraceShipping(ctx)

	configuredServices := r.configuredServicesForTraceShipping()
	if len(configuredServices) == 0 {
		return nil, errRunnerRequired("no services configured in runAll.yaml")
	}

	serviceNames := make([]string, 0, len(configuredServices))
	for _, svc := range configuredServices {
		serviceNames = append(serviceNames, svc.Name)
	}

	probeID := domain.NewProbeTraceID(time.Now().UnixMilli())
	now := time.Now().UTC()

	for _, svc := range configuredServices {
		st := r.store.Get(svc.Name)
		svcStatus := "unknown"
		if st != nil {
			svcStatus = string(st.Status)
		}
		r.appendStructuredLog(svc.Name, "info", "trace_shipping_probe", map[string]interface{}{
			"trace_id":       probeID,
			"service_status": svcStatus,
			"probe_batch":    probeID,
		})
	}

	services := make([]domain.TraceShippingServiceResult, 0, len(configuredServices))
	for _, svc := range configuredServices {
		entry := domain.TraceShippingServiceResult{ServiceName: svc.Name}
		if st := r.store.Get(svc.Name); st != nil {
			entry.ServiceStatus = string(st.Status)
		}
		entry.LocalProbeOK = r.localProbeVisible(svc.Name, probeID)
		services = append(services, entry)
	}

	lokiReady, lokiErr, lokiCounts := pollLokiForProbeCounts(ctx, r, probeID, serviceNames, wait)

	grafanaURL := ""
	if link, err := domain.GrafanaTraceLink(
		r.cfg.Observability.GrafanaURL,
		r.cfg.Observability.TraceDashboardUID,
		probeID,
	); err == nil {
		grafanaURL = link
	}

	configuredTotal := len(configuredServices)
	summary := domain.TraceShippingSummary{
		Total:           configuredTotal,
		ConfiguredTotal: configuredTotal,
	}
	for i := range services {
		svc := &services[i]
		if svc.LocalProbeOK {
			summary.LocalOK++
		}
		if lokiCounts != nil {
			if n, ok := lokiCounts[svc.ServiceName]; ok && n > 0 {
				svc.LokiVisible = true
				svc.LokiLogCount = n
				summary.LokiOK++
			}
		}
		if link, err := domain.GrafanaTraceLink(
			r.cfg.Observability.GrafanaURL,
			r.cfg.Observability.TraceDashboardUID,
			probeID,
		); err == nil {
			svc.GrafanaURL = link + "&var-service=" + url.QueryEscape(svc.ServiceName)
		}
		svc.Issue = traceShippingIssue(svc, lokiReady, lokiErr)
		if svc.Issue != "" {
			summary.Failed++
		}
	}

	sortTraceShippingServices(services)

	report := &domain.TraceShippingReport{
		ProbeTraceID:       probeID,
		VerifiedAt:         now.Format(time.RFC3339),
		VerificationStatus: traceShippingStatusVerified,
		LokiReady:          lokiReady,
		LokiError:          lokiErr,
		GrafanaTraceURL:    grafanaURL,
		PromtailReload:     promtailReload,
		Summary:            summary,
		Services:           services,
	}

	r.traceShippingMu.Lock()
	r.traceShippingLast = report
	r.traceShippingMu.Unlock()
	return report, nil
}

func (r *Runner) LastTraceShippingReport() *domain.TraceShippingReport {
	return r.TraceShippingReportForUI()
}

func traceShippingIssue(svc *domain.TraceShippingServiceResult, lokiReady bool, lokiErr string) string {
	if svc == nil {
		return ""
	}
	switch {
	case !svc.LocalProbeOK:
		return "本地日志文件未采集到 probe（runAll tee 可能中断）"
	case !lokiReady:
		if strings.TrimSpace(lokiErr) != "" {
			return "Loki 不可达: " + lokiErr
		}
		return "Loki 不可达"
	case !svc.LokiVisible:
		return "Loki 中未找到 probe trace（Promtail→Loki 投递异常）"
	default:
		return ""
	}
}

type runnerRequiredError string

func errRunnerRequired(msg string) error { return runnerRequiredError(msg) }

func (e runnerRequiredError) Error() string { return string(e) }
