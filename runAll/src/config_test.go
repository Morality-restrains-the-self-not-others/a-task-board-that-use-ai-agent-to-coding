package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_ValidYAML(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: infra
    services:
      - name: redis
        command: "redis-server"
        health_check:
          url: "http://localhost:6379"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != "1" {
		t.Errorf("version = %q, want %q", cfg.Version, "1")
	}
	if len(cfg.Groups) != 1 {
		t.Fatalf("groups len = %d, want 1", len(cfg.Groups))
	}
	svc := cfg.Groups[0].Services[0]
	if svc.Name != "redis" {
		t.Errorf("service name = %q, want %q", svc.Name, "redis")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: infra
    services:
      - name: svc
        command: "echo hi"
        health_check:
          url: "http://localhost:8080"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if svc.OnFailure != "exit" {
		t.Errorf("on_failure default = %q, want %q", svc.OnFailure, "exit")
	}
	if svc.HealthCheck.Timeout != 30 {
		t.Errorf("timeout default = %d, want 30", svc.HealthCheck.Timeout)
	}
	if svc.HealthCheck.Retries != 10 {
		t.Errorf("retries default = %d, want 10", svc.HealthCheck.Retries)
	}
	if svc.HealthCheck.Backoff.Initial != 1.0 {
		t.Errorf("backoff.initial default = %f, want 1.0", svc.HealthCheck.Backoff.Initial)
	}
	if svc.HealthCheck.Backoff.Max != 8.0 {
		t.Errorf("backoff.max default = %f, want 8.0", svc.HealthCheck.Backoff.Max)
	}
	if svc.HealthCheck.Backoff.Multiplier != 2.0 {
		t.Errorf("backoff.multiplier default = %f, want 2.0", svc.HealthCheck.Backoff.Multiplier)
	}
	if svc.HealthCheck.CheckInterval != 10 {
		t.Errorf("check_interval default = %d, want 10", svc.HealthCheck.CheckInterval)
	}
	if svc.HealthCheck.UnhealthyThreshold != 2 {
		t.Errorf("unhealthy_threshold default = %d, want 2", svc.HealthCheck.UnhealthyThreshold)
	}
}

func TestHealthCheck_StartupProbeURL(t *testing.T) {
	hc := HealthCheck{
		URL:         "http://127.0.0.1:8001/api/health/",
		LivenessURL: "http://127.0.0.1:8001/api/live/",
	}
	if got := hc.StartupProbeURL(); got != hc.LivenessURL {
		t.Fatalf("StartupProbeURL() = %q, want %q", got, hc.LivenessURL)
	}
	if got := hc.ReadinessProbeURL(); got != hc.URL {
		t.Fatalf("ReadinessProbeURL() = %q, want %q", got, hc.URL)
	}
	startup := hc.StartupProbeConfig()
	if startup.URL != hc.LivenessURL {
		t.Fatalf("StartupProbeConfig().URL = %q, want %q", startup.URL, hc.LivenessURL)
	}

	plain := HealthCheck{URL: "http://127.0.0.1:9000/health"}
	if got := plain.StartupProbeURL(); got != plain.URL {
		t.Fatalf("StartupProbeURL without liveness = %q, want %q", got, plain.URL)
	}
	if plain.HasSplitProbe() {
		t.Fatal("expected no split probe without liveness_url")
	}
	split := HealthCheck{
		LivenessURL: "http://127.0.0.1:8001/api/live/",
		URL:         "http://127.0.0.1:8001/api/health/",
	}
	if !split.HasSplitProbe() {
		t.Fatal("expected split probe")
	}
}

func TestLoadConfig_LoggingFileRoot(t *testing.T) {
	t.Setenv("RUNALL_LOG_ROOT", "")
	yamlContent := `
version: "1"
logging:
  file_root: /tmp/custom-runall-logs
groups:
  - name: infra
    services:
      - name: svc
        command: "echo hi"
        health_check:
          url: "http://localhost:8080"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Logging.FileRoot != "/tmp/custom-runall-logs" {
		t.Errorf("file_root = %q, want /tmp/custom-runall-logs", cfg.Logging.FileRoot)
	}
}

func TestLoadConfig_LoggingFileRootEnvOverride(t *testing.T) {
	t.Setenv("RUNALL_LOG_ROOT", "/tmp/from-env")
	yamlContent := `
version: "1"
logging:
  file_root: /tmp/from-yaml
groups:
  - name: infra
    services:
      - name: svc
        command: "echo hi"
        health_check:
          url: "http://localhost:8080"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Logging.FileRoot != "/tmp/from-env" {
		t.Errorf("file_root = %q, want env override /tmp/from-env", cfg.Logging.FileRoot)
	}
}

func TestLoadConfig_ObservabilityDefaults(t *testing.T) {
	yamlContent := `
version: "1"
groups:
  - name: infra
    services:
      - name: svc
        command: "echo hi"
        health_check:
          url: "http://localhost:8080"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yamlContent), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Observability.GrafanaURL != "http://127.0.0.1:3000" {
		t.Errorf("grafana_url = %q", cfg.Observability.GrafanaURL)
	}
	if cfg.Observability.LokiURL != "http://127.0.0.1:3100" {
		t.Errorf("loki_url = %q", cfg.Observability.LokiURL)
	}
	if cfg.Observability.TraceDashboardUID != "distributed-trace-view" {
		t.Errorf("trace_dashboard_uid = %q", cfg.Observability.TraceDashboardUID)
	}
}

func TestLoadConfig_DuplicateServiceNames(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: dup
        command: "a"
        health_check:
          url: "http://localhost:1"
  - name: g2
    services:
      - name: dup
        command: "b"
        health_check:
          url: "http://localhost:2"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for duplicate service names")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_MissingDependsOn(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "a"
        health_check:
          url: "http://localhost:1"
        depends_on: [nonexistent]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing depends_on reference")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_InvalidOnFailure(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "a"
        health_check:
          url: "http://localhost:1"
        on_failure: panic
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid on_failure")
	}
	if !strings.Contains(err.Error(), "on_failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlattenServices(t *testing.T) {
	cfg := &Config{
		Groups: []Group{
			{Name: "g1", Services: []Service{
				{Name: "a"}, {Name: "b"},
			}},
			{Name: "g2", Services: []Service{
				{Name: "c"},
			}},
		},
	}
	got := cfg.Flatten()
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name}
	expected := []string{"a", "b", "c"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

func TestLoadConfig_MissingCommand(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        health_check:
          url: "http://localhost:1"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_BuildCommand(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "./app"
        build_command: "go build -o app ."
        health_check:
          url: "http://localhost:1"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if svc.BuildCommand != "go build -o app ." {
		t.Errorf("build_command = %q, want %q", svc.BuildCommand, "go build -o app .")
	}
}

func TestLoadConfig_BuildCommandDefault(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "./app"
        health_check:
          url: "http://localhost:1"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if svc.BuildCommand != "" {
		t.Errorf("build_command default = %q, want empty string", svc.BuildCommand)
	}
}

func TestLoadConfig_MissingHealthCheckURL(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "echo hi"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for missing health_check probe")
	}
	if !strings.Contains(err.Error(), "health_check") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_TCPHealthCheck(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: docker-redis
        start_command: "bash run.sh managed"
        stop_command: "bash run.sh stop"
        health_check:
          tcp: "127.0.0.1:6379"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if !svc.HealthCheck.UsesTCP() {
		t.Fatal("expected TCP health check")
	}
	if svc.HealthCheck.DisplayEndpoint() != "tcp://127.0.0.1:6379" {
		t.Fatalf("DisplayEndpoint = %q", svc.HealthCheck.DisplayEndpoint())
	}
}

func TestLoadConfig_RejectsDualHealthProbes(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "echo hi"
        health_check:
          url: "http://127.0.0.1:1"
          tcp: "127.0.0.1:6379"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for dual probes")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_FailsWhenDualConfigMismatch(t *testing.T) {
	primaryYAML := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "echo primary"
        health_check:
          url: "http://localhost:1"
`
	secondaryYAML := `
version: "1"
groups:
  - name: g1
    services:
      - name: svc
        command: "echo secondary"
        health_check:
          url: "http://localhost:1"
`
	dir := t.TempDir()
	primaryPath := filepath.Join(dir, "config.primary.yaml")
	secondaryPath := filepath.Join(dir, "config.secondary.yaml")
	os.WriteFile(primaryPath, []byte(primaryYAML), 0644)
	os.WriteFile(secondaryPath, []byte(secondaryYAML), 0644)

	_, _, err := LoadConfigWithSourceGuard(primaryPath, secondaryPath)
	if err == nil {
		t.Fatal("expected mismatch error when dual config hashes differ")
	}
	if !strings.Contains(err.Error(), "config source mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfig_RegressionMatrixFixtures(t *testing.T) {
	yaml := `
version: "1"
groups:
  - name: regression
    services:
      - name: matrix-port-conflict
        command: "python3 -m http.server 28080"
        health_check:
          url: "http://127.0.0.1:28080/health"
      - name: matrix-runtime-prereq-blocked
        command: "docker compose up"
        health_check:
          url: "http://127.0.0.1:28081/health"
      - name: matrix-prereq-repaired-healthy
        command: "docker compose up"
        health_check:
          url: "http://127.0.0.1:28082/health"
      - name: matrix-non-owner-rejected
        command: "echo worker"
        health_check:
          url: "http://127.0.0.1:28083/health"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte(yaml), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	services := cfg.Flatten()
	if len(services) != 4 {
		t.Fatalf("services len = %d, want 4", len(services))
	}

	gotNames := map[string]bool{}
	for _, svc := range services {
		gotNames[svc.Name] = true
		if svc.HealthCheck.URL == "" {
			t.Fatalf("service %q health_check.url should not be empty", svc.Name)
		}
	}
	for _, name := range []string{
		"matrix-port-conflict",
		"matrix-runtime-prereq-blocked",
		"matrix-prereq-repaired-healthy",
		"matrix-non-owner-rejected",
	} {
		if !gotNames[name] {
			t.Fatalf("service %q not found in loaded config", name)
		}
	}
}

func TestProductionConfig_GitlabRegionsGroup(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}

	var gitlabGroup *Group
	var infraGroup *Group
	for i := range cfg.Groups {
		g := &cfg.Groups[i]
		switch g.Name {
		case "gitlab-regions":
			gitlabGroup = g
		case "infrastructure":
			infraGroup = g
		}
	}
	if gitlabGroup == nil {
		t.Fatal("missing group gitlab-regions")
	}
	if infraGroup == nil {
		t.Fatal("missing group infrastructure")
	}
	for _, svc := range infraGroup.Services {
		if svc.Name == "git-service" || strings.HasPrefix(svc.Name, "git-service-") {
			t.Fatalf("infrastructure must not contain %q; GitLab instances belong in gitlab-regions", svc.Name)
		}
	}

	byName := map[string]Service{}
	for _, svc := range gitlabGroup.Services {
		byName[svc.Name] = svc
	}
	primary, ok := byName["git-service"]
	if !ok {
		t.Fatal("gitlab-regions missing git-service")
	}
	sh1, ok := byName["git-service-tencent-sh-1"]
	if !ok {
		t.Fatal("gitlab-regions missing git-service-tencent-sh-1")
	}
	if primary.EffectiveStartCommand() == sh1.EffectiveStartCommand() {
		t.Fatalf("start_command must differ: both %q", primary.EffectiveStartCommand())
	}
	if !strings.Contains(primary.HealthCheck.URL, ":8012/") {
		t.Fatalf("git-service health url=%q want port 8012", primary.HealthCheck.URL)
	}
	if !strings.Contains(sh1.HealthCheck.URL, ":8014/") {
		t.Fatalf("git-service-tencent-sh-1 health url=%q want port 8014", sh1.HealthCheck.URL)
	}
	if strings.Contains(sh1.HealthCheck.URL, "127.0.0.1") || strings.Contains(sh1.HealthCheck.URL, "10.2.150.68") {
		t.Fatalf("SH-1 health url=%q must target Host sh, not INFRA_HOST (collides with task-container-gateway :8014)", sh1.HealthCheck.URL)
	}
	if primary.HealthCheck.URL == sh1.HealthCheck.URL {
		t.Fatal("health_check.url must differ between GitLab regions")
	}
	if sh1.Env["GITSERVICE_CONF_APP"] != "git-service-tencent-sh-1" {
		t.Fatalf("SH-1 GITSERVICE_CONF_APP=%q", sh1.Env["GITSERVICE_CONF_APP"])
	}
	if sh1.Env["COMPOSE_PROJECT_NAME"] != "gitservice-tencent-sh-1" {
		t.Fatalf("SH-1 COMPOSE_PROJECT_NAME=%q", sh1.Env["COMPOSE_PROJECT_NAME"])
	}
	if !strings.Contains(sh1.EffectiveStartCommand(), "runall_ssh_sh_gitlab") {
		t.Fatalf("SH-1 start_command=%q must SSH via runall_ssh_sh_gitlab.sh", sh1.EffectiveStartCommand())
	}
	if !strings.Contains(sh1.StopCommand, "runall_ssh_sh_gitlab") {
		t.Fatalf("SH-1 stop_command=%q must SSH via runall_ssh_sh_gitlab.sh", sh1.StopCommand)
	}
	if strings.Contains(sh1.EffectiveStartCommand(), "deploy_tencent_sh_1") {
		t.Fatalf("SH-1 start_command=%q must not run deploy on INFRA host", sh1.EffectiveStartCommand())
	}
}

func isGitLabRunAllService(name string) bool {
	return name == "git-service" || strings.HasPrefix(name, "git-service-")
}

// GitLab is pluggable (ADR-0014 / ADR-0048): other services talk to it at runtime
// and are aligned by humans. If they list git-service* in depends_on, start-all
// skips them whenever GitLab is gated off or unhealthy (ADR-0047).
func TestProductionConfig_NoInboundDependsOnGitLab(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}
	for _, svc := range cfg.Flatten() {
		if isGitLabRunAllService(svc.Name) {
			continue
		}
		for _, dep := range svc.DependsOn {
			if isGitLabRunAllService(dep) {
				t.Errorf("service %q depends_on %q: GitLab is pluggable; other services must not wait on it at start (ADR-0048)", svc.Name, dep)
			}
		}
	}
}

func TestProductionConfig_taskFEHasBuildCommand(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}

	var taskFESvc *Service
	for _, svc := range cfg.Flatten() {
		if svc.Name == "taskFE" {
			svcCopy := svc
			taskFESvc = &svcCopy
			break
		}
	}
	if taskFESvc == nil {
		t.Fatal("taskFE service not found in production config")
	}
	if taskFESvc.BuildCommand == "" {
		t.Fatal("taskFE build_command must be configured for UI build action")
	}
	wantBuild := "bash scripts/runall-lifecycle.sh build"
	if taskFESvc.BuildCommand != wantBuild {
		t.Fatalf("build_command = %q, want %q", taskFESvc.BuildCommand, wantBuild)
	}
	if taskFESvc.StopCommand == "" || taskFESvc.EffectiveStartCommand() == "" {
		t.Fatal("taskFE must have explicit start_command and stop_command")
	}
}

func TestProductionConfig_RootRunAllYaml_TaskEventsHaveBuildCommand(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}

	var found int
	for _, svc := range cfg.Flatten() {
		if !strings.HasPrefix(svc.Name, "task-events-") {
			continue
		}
		found++
		if svc.BuildCommand == "" {
			t.Fatalf("%s build_command is empty", svc.Name)
		}
		if !strings.HasPrefix(svc.BuildCommand, "bash run.sh build ") {
			t.Fatalf("%s build_command = %q, want prefix %q", svc.Name, svc.BuildCommand, "bash run.sh build ")
		}
	}
	if found == 0 {
		t.Fatalf("no task-events-* services found in %s", path)
	}
}

// OPT-20260822-002 回归：task_status_changed 释放消费者在全部重启期间对
// task-cloud-service（:8018）connection refused 会进 DLT。声明 depends_on
// task-cloud-service 后，start-all 中该消费者排在 cloud 之后（等其健康），
// stop-all 逆序中先于 cloud 停止，重启空窗不再有消费者裸奔打满 DLT。
func TestProductionConfig_ReleaseConsumerDependsOnCloud(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}

	// 全部重启期间直接调用 task-cloud-service 的 task-events 消费者必须声明
	// depends_on task-cloud-service，否则可能在 cloud 未就绪时消费终态事件进 DLT。
	cloudCallers := []string{
		"task-events-task-status-changed-1-release-servers-on-terminal",
		"task-events-cloud-server-started-1-process-server-start",
		"task-events-cloud-server-stopped-1-process-server-stop",
		"task-events-cloud-server-start-auto-1-process-server-start-auto",
		"task-events-relay-lifecycle-1-append-token-audit",
		"task-events-relay-lifecycle-2-open-runtime-session",
		"task-events-relay-lifecycle-3-clear-reachability",
		"task-events-relay-lifecycle-4-relay-workflow-update",
	}
	byName := make(map[string]*Service)
	for _, svc := range cfg.Flatten() {
		svcCopy := svc
		byName[svc.Name] = &svcCopy
	}
	for _, consumerName := range cloudCallers {
		consumer, ok := byName[consumerName]
		if !ok {
			t.Errorf("%s not found in production config", consumerName)
			continue
		}
		hasCloud := false
		for _, dep := range consumer.DependsOn {
			if dep == "task-cloud-service" {
				hasCloud = true
				break
			}
		}
		if !hasCloud {
			t.Errorf("%s depends_on missing task-cloud-service (got %v)", consumerName, consumer.DependsOn)
		}
	}
}

func TestResolveConfApps_UnifiedInfraHost(t *testing.T) {
	dir := t.TempDir()

	// Create conf/infra/redis/config.yaml
	redisDir := filepath.Join(dir, "conf", "infra", "redis")
	if err := os.MkdirAll(redisDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	redisYAML := "host: 10.10.10.10\nport: 6379\n"
	if err := os.WriteFile(filepath.Join(redisDir, "config.yaml"), []byte(redisYAML), 0644); err != nil {
		t.Fatalf("write redis config: %v", err)
	}

	// Create conf/core/sms/config.yaml (second conf_app, different host)
	djangoDir := filepath.Join(dir, "conf", "core", "sms")
	if err := os.MkdirAll(djangoDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	djangoYAML := "host: 10.20.20.20\nport: 8001\n"
	if err := os.WriteFile(filepath.Join(djangoDir, "config.yaml"), []byte(djangoYAML), 0644); err != nil {
		t.Fatalf("write django config: %v", err)
	}

	// Create runAll config with multiple conf_app services and ${INFRA_HOST} placeholders (now left unresolved)
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
observability:
  grafana_url: "http://${INFRA_HOST}:3000"
  loki_url: "http://${INFRA_HOST}:3100"
groups:
  - name: infrastructure
    services:
      - name: docker-redis
        conf_app: infra/redis
        start_command: "true"
        stop_command: "true"
        health_check:
          tcp: "${INFRA_HOST}:6379"
      - name: saas-backend
        conf_app: core/sms
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}

	cfg, err := LoadConfig(runAllPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// First service (docker-redis) has ${INFRA_HOST} in tcp, should resolve to its own conf_app host.
	redisSvc := cfg.Groups[0].Services[0]
	if redisSvc.HealthCheck.TCP != "10.10.10.10:6379" {
		t.Fatalf("docker-redis tcp = %q, want 10.10.10.10:6379", redisSvc.HealthCheck.TCP)
	}

	// Second service (saas-backend) uses health_path, URL should be built from its own host.
	djangoSvc := cfg.Groups[0].Services[1]
	if djangoSvc.HealthCheck.URL != "http://10.20.20.20:8001/api/health/" {
		t.Fatalf("saas-backend url = %q, want http://10.20.20.20:8001/api/health/", djangoSvc.HealthCheck.URL)
	}

	// Observability URLs use the first conf_app host as ${INFRA_HOST}.
	if cfg.Observability.GrafanaURL != "http://10.10.10.10:3000" {
		t.Fatalf("grafana_url = %q, want http://10.10.10.10:3000", cfg.Observability.GrafanaURL)
	}
	if cfg.Observability.LokiURL != "http://10.10.10.10:3100" {
		t.Fatalf("loki_url = %q, want http://10.10.10.10:3100", cfg.Observability.LokiURL)
	}
}

func TestResolveConfApps_ExecHealthSkipsHealthPath(t *testing.T) {
	// docker-redis style: conf_app for log_file/host mapping + exec probe (no health_path).
	dir := t.TempDir()
	redisDir := filepath.Join(dir, "conf", "infra", "redis")
	if err := os.MkdirAll(redisDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(redisDir, "config.yaml"), []byte("host: 127.0.0.1\nport: 6379\n"), 0644); err != nil {
		t.Fatalf("write redis config: %v", err)
	}
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: infrastructure
    services:
      - name: docker-redis
        conf_app: infra/redis
        start_command: "true"
        stop_command: "true"
        health_check:
          exec: "true"
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll: %v", err)
	}
	cfg, err := LoadConfig(runAllPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if !svc.HealthCheck.UsesExec() {
		t.Fatal("expected exec health check")
	}
	if svc.HealthCheck.URL != "" {
		t.Fatalf("URL should stay empty for exec probe, got %q", svc.HealthCheck.URL)
	}
}

func TestLoadConfig_ProductionDockerKafkaUsesBrokerExecProbe(t *testing.T) {
	// OPT-20260721-007: docker-kafka must probe the broker via exec, not Kafka UI HTTP alone.
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}
	var kafka *Service
	for gi := range cfg.Groups {
		for si := range cfg.Groups[gi].Services {
			if cfg.Groups[gi].Services[si].Name == "docker-kafka" {
				kafka = &cfg.Groups[gi].Services[si]
			}
		}
	}
	if kafka == nil {
		t.Fatal("docker-kafka service not found in conf/runAll.yaml")
	}
	if !kafka.HealthCheck.UsesExec() {
		t.Fatalf("docker-kafka must use exec broker probe, got url=%q tcp=%q exec=%q",
			kafka.HealthCheck.URL, kafka.HealthCheck.TCP, kafka.HealthCheck.Exec)
	}
	if !strings.Contains(kafka.HealthCheck.Exec, "dockerInfra/kafka/health.sh") {
		t.Fatalf("docker-kafka exec = %q, want dockerInfra/kafka/health.sh", kafka.HealthCheck.Exec)
	}
	if strings.Contains(kafka.HealthCheck.URL, "18080") || strings.Contains(kafka.HealthCheck.Exec, "18080") {
		t.Fatalf("docker-kafka must not rely on Kafka UI :18080 alone: url=%q exec=%q",
			kafka.HealthCheck.URL, kafka.HealthCheck.Exec)
	}
}

func TestResolveConfApps_NoHostVariable(t *testing.T) {
	// Load the production config and verify no ${HOST} placeholder remains.
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}

	for _, svc := range cfg.Flatten() {
		if strings.Contains(svc.HealthCheck.URL, "${HOST}") {
			t.Errorf("service %q: health_check.url still contains ${HOST}: %q", svc.Name, svc.HealthCheck.URL)
		}
		if strings.Contains(svc.HealthCheck.TCP, "${HOST}") {
			t.Errorf("service %q: health_check.tcp still contains ${HOST}: %q", svc.Name, svc.HealthCheck.TCP)
		}
		if strings.Contains(svc.HealthCheck.LivenessURL, "${HOST}") {
			t.Errorf("service %q: health_check.liveness_url still contains ${HOST}: %q", svc.Name, svc.HealthCheck.LivenessURL)
		}
		for k, v := range svc.Env {
			if strings.Contains(v, "${HOST}") {
				t.Errorf("service %q: env[%q] still contains ${HOST}: %q", svc.Name, k, v)
			}
		}
	}

	// Also verify observability URLs don't contain ${HOST}.
	if strings.Contains(cfg.Observability.GrafanaURL, "${HOST}") {
		t.Errorf("grafana_url still contains ${HOST}: %q", cfg.Observability.GrafanaURL)
	}
	if strings.Contains(cfg.Observability.LokiURL, "${HOST}") {
		t.Errorf("loki_url still contains ${HOST}: %q", cfg.Observability.LokiURL)
	}
}

func TestResolveConfApps_HealthCheckURL(t *testing.T) {
	dir := t.TempDir()

	// Create conf/core/sms/config.yaml
	confDir := filepath.Join(dir, "conf", "core", "sms")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	djangoYAML := "host: 10.2.150.89\nport: 8001\n"
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(djangoYAML), 0644); err != nil {
		t.Fatalf("write django config: %v", err)
	}

	// Create conf/docker-infra/config.yaml
	infraDir := filepath.Join(dir, "conf", "docker-infra")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	infraYAML := "host: 10.2.150.119\nport: 6379\nportainerPort: 9000\n"
	if err := os.WriteFile(filepath.Join(infraDir, "config.yaml"), []byte(infraYAML), 0644); err != nil {
		t.Fatalf("write docker-infra config: %v", err)
	}

	// Create runAll config at conf/runAll.yaml
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: platform
    services:
      - name: saas-backend
        conf_app: core/sms
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/
          liveness_path: /api/live/
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}

	cfg, err := LoadConfig(runAllPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	// Verify health check URL auto-constructed.
	svc := cfg.Groups[0].Services[0]
	if svc.HealthCheck.URL != "http://10.2.150.89:8001/api/health/" {
		t.Fatalf("health URL = %q", svc.HealthCheck.URL)
	}
	if svc.HealthCheck.LivenessURL != "http://10.2.150.89:8001/api/live/" {
		t.Fatalf("liveness URL = %q", svc.HealthCheck.LivenessURL)
	}
}

func TestResolveConfApps_ConfAppWithoutHealthPath(t *testing.T) {
	dir := t.TempDir()

	// Create conf/task-auth/config.yaml
	confDir := filepath.Join(dir, "conf", "task-auth")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	authYAML := "host: 10.2.150.89\nport: 8003\n"
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(authYAML), 0644); err != nil {
		t.Fatalf("write task-auth config: %v", err)
	}

	// Create conf/docker-infra/config.yaml
	infraDir := filepath.Join(dir, "conf", "docker-infra")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	infraYAML := "host: 10.2.150.119\n"
	if err := os.WriteFile(filepath.Join(infraDir, "config.yaml"), []byte(infraYAML), 0644); err != nil {
		t.Fatalf("write docker-infra config: %v", err)
	}

	// Create runAll config at conf/runAll.yaml
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: platform
    services:
      - name: task-auth
        conf_app: task-auth
        start_command: "true"
        stop_command: "true"
        health_check:
          timeout: 30
          retries: 10
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}

	_, err := LoadConfig(runAllPath)
	if err == nil {
		t.Fatal("expected error for conf_app without health_path")
	}
	if !strings.Contains(err.Error(), "health_path") {
		t.Fatalf("expected health_path error, got: %v", err)
	}
}

func TestProductionConfig_RootRunAllYaml_TaskEventsHaveSplitHealthProbe(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}

	var found int
	for _, svc := range cfg.Flatten() {
		if !strings.HasPrefix(svc.Name, "task-events-") {
			continue
		}
		found++
		hc := svc.HealthCheck
		if !hc.HasSplitProbe() {
			t.Fatalf("%s: expected split health probe (liveness_url + readiness url)", svc.Name)
		}
		if !strings.HasSuffix(hc.LivenessURL, "/api/health/") {
			t.Fatalf("%s: liveness_url = %q, want suffix /api/health/", svc.Name, hc.LivenessURL)
		}
		if !strings.HasSuffix(hc.URL, "/api/health/ready") {
			t.Fatalf("%s: readiness url = %q, want suffix /api/health/ready", svc.Name, hc.URL)
		}
	}
	if found == 0 {
		t.Fatalf("no task-events-* services found in %s", path)
	}
}

func TestLoadConfig_ResolveWorkingDirDotFromRunAllSubdir(t *testing.T) {
	repoRoot := t.TempDir()
	confDir := filepath.Join(repoRoot, "conf")
	runAllDir := filepath.Join(repoRoot, "runAll")
	scriptsDir := filepath.Join(runAllDir, "scripts")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	// Simulate compiled binary at runAll/runAll (must not shadow runAll/scripts/).
	if err := os.WriteFile(filepath.Join(runAllDir, "runAll"), []byte("bin"), 0755); err != nil {
		t.Fatalf("write binary stub: %v", err)
	}

	yaml := `
version: "1"
groups:
  - name: g1
    services:
      - name: promtail-local
        working_dir: .
        start_command: "bash runAll/scripts/runall-local-promtail.sh up"
        stop_command: "bash runAll/scripts/runall-local-promtail.sh down"
        health_check:
          url: "http://127.0.0.1:3100/ready"
`
	configPath := filepath.Join(confDir, "runAll.yaml")
	if err := os.WriteFile(configPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(runAllDir); err != nil {
		t.Fatalf("chdir runAll: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	got := cfg.Groups[0].Services[0].WorkingDir
	if got != repoRoot {
		t.Fatalf("WorkingDir = %q, want project root %q", got, repoRoot)
	}
	if cfg.Groups[0].Services[0].HealthCheck.WorkDir != repoRoot {
		t.Fatalf("HealthCheck.WorkDir = %q, want project root %q", cfg.Groups[0].Services[0].HealthCheck.WorkDir, repoRoot)
	}
}

func writeDomainEventIntentConfig(t *testing.T, root, event, intentKey string, port int, groupID string) {
	t.Helper()
	dir := filepath.Join(root, "conf", "events", "domain-events", event)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	body := "intents:\n"
	body += "  " + intentKey + ":\n"
	body += "    host: 0.0.0.0\n"
	body += "    port: " + fmt.Sprintf("%d", port) + "\n"
	if groupID != "" {
		body += "    groupId: " + groupID + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0644); err != nil {
		t.Fatalf("write domain-event config: %v", err)
	}
}

func TestResolveConfApps_IntentPathBuildsHealthURL(t *testing.T) {
	// OPT-20260816-053: task-events health URLs must be built from the
	// domain-events SSOT via intent_path, not hand-written ports.
	dir := t.TempDir()
	writeDomainEventIntentConfig(t, dir, "billing_transaction_created", "1_process_billing_transaction", 18020,
		"task-events-billing-transaction-created-1-process-billing-transaction")

	// conf_app service provides the ${INFRA_HOST} resolution anchor.
	redisDir := filepath.Join(dir, "conf", "infra", "redis")
	if err := os.MkdirAll(redisDir, 0755); err != nil {
		t.Fatalf("mkdir redis: %v", err)
	}
	if err := os.WriteFile(filepath.Join(redisDir, "config.yaml"), []byte("host: 10.10.10.10\nport: 6379\n"), 0644); err != nil {
		t.Fatalf("write redis config: %v", err)
	}

	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: domain-events-intents
    services:
      - name: task-events-billing-transaction-created-1-process-billing-transaction
        intent_path: events/domain-events/billing_transaction_created/1_process_billing_transaction
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/ready
          liveness_path: /api/health/
          timeout: 60
          retries: 15
      - name: docker-redis
        conf_app: infra/redis
        start_command: "true"
        stop_command: "true"
        health_check:
          tcp: "${INFRA_HOST}:6379"
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}

	cfg, err := LoadConfig(runAllPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	svc := cfg.Groups[0].Services[0]
	if svc.HealthCheck.URL != "http://10.10.10.10:18020/api/health/ready" {
		t.Fatalf("readiness url = %q, want http://10.10.10.10:18020/api/health/ready", svc.HealthCheck.URL)
	}
	if svc.HealthCheck.LivenessURL != "http://10.10.10.10:18020/api/health/" {
		t.Fatalf("liveness url = %q, want http://10.10.10.10:18020/api/health/", svc.HealthCheck.LivenessURL)
	}
}

func TestResolveConfApps_IntentPath_GroupIDMismatch(t *testing.T) {
	// intent_path resolving to a sibling intent's groupId must fail at load time
	// (FE-20260816-EVENTS-FANOUT-HEALTH-PORT copy-paste guard).
	dir := t.TempDir()
	writeDomainEventIntentConfig(t, dir, "relay_lifecycle", "1_append_token_audit", 18039,
		"task-events-relay-lifecycle-1-append-token-audit")
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: domain-events-intents
    services:
      - name: task-events-sse-message-1-send-sse-message
        intent_path: events/domain-events/relay_lifecycle/1_append_token_audit
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/ready
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}
	_, err := LoadConfig(runAllPath)
	if err == nil {
		t.Fatal("expected error for intent_path groupId mismatch")
	}
	if !strings.Contains(err.Error(), "groupId") {
		t.Fatalf("expected groupId mismatch error, got: %v", err)
	}
}

func TestResolveConfApps_IntentPath_MissingIntent(t *testing.T) {
	dir := t.TempDir()
	writeDomainEventIntentConfig(t, dir, "relay_lifecycle", "1_append_token_audit", 18039,
		"task-events-relay-lifecycle-1-append-token-audit")
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: domain-events-intents
    services:
      - name: task-events-relay-lifecycle-1-append-token-audit
        intent_path: events/domain-events/relay_lifecycle/9_no_such_intent
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/ready
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}
	_, err := LoadConfig(runAllPath)
	if err == nil {
		t.Fatal("expected error for unknown intent key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected intent-not-found error, got: %v", err)
	}
}

func TestResolveConfApps_IntentPath_WithoutHealthPath(t *testing.T) {
	dir := t.TempDir()
	writeDomainEventIntentConfig(t, dir, "relay_lifecycle", "1_append_token_audit", 18039,
		"task-events-relay-lifecycle-1-append-token-audit")
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: domain-events-intents
    services:
      - name: task-events-relay-lifecycle-1-append-token-audit
        intent_path: events/domain-events/relay_lifecycle/1_append_token_audit
        start_command: "true"
        stop_command: "true"
        health_check:
          timeout: 30
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}
	_, err := LoadConfig(runAllPath)
	if err == nil {
		t.Fatal("expected error for intent_path without health_path")
	}
	if !strings.Contains(err.Error(), "health_path") {
		t.Fatalf("expected health_path error, got: %v", err)
	}
}

func TestResolveConfApps_IntentPath_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0755); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}
	runAllPath := filepath.Join(dir, "conf", "runAll.yaml")
	runAllYAML := `
version: "1"
groups:
  - name: domain-events-intents
    services:
      - name: task-events-relay-lifecycle-1-append-token-audit
        intent_path: ../../etc/relay_lifecycle/1_append_token_audit
        start_command: "true"
        stop_command: "true"
        health_check:
          health_path: /api/health/ready
`
	if err := os.WriteFile(runAllPath, []byte(runAllYAML), 0644); err != nil {
		t.Fatalf("write runAll config: %v", err)
	}
	_, err := LoadConfig(runAllPath)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected path traversal error, got: %v", err)
	}
}

func TestProductionConfig_RootRunAllYaml_TaskEventsNoHandwrittenPorts(t *testing.T) {
	// OPT-20260816-053: every task-events service must derive its health port from
	// intent_path + domain-events SSOT; hand-written ports are forbidden.
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}
	var found int
	for _, svc := range cfg.Flatten() {
		if !strings.HasPrefix(svc.Name, "task-events-") {
			continue
		}
		found++
		if strings.TrimSpace(svc.IntentPath) == "" {
			t.Fatalf("%s: intent_path is required for task-events services", svc.Name)
		}
		hc := svc.HealthCheck
		if !hc.HasSplitProbe() {
			t.Fatalf("%s: expected split health probe", svc.Name)
		}
		if !strings.HasSuffix(hc.LivenessURL, "/api/health/") {
			t.Fatalf("%s: liveness_url = %q, want suffix /api/health/", svc.Name, hc.LivenessURL)
		}
		if !strings.HasSuffix(hc.URL, "/api/health/ready") {
			t.Fatalf("%s: readiness url = %q, want suffix /api/health/ready", svc.Name, hc.URL)
		}
	}
	if found == 0 {
		t.Fatalf("no task-events-* services found in %s", path)
	}
}
