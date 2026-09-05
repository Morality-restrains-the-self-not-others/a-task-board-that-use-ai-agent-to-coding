package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"runAll/src/domain"
)

type Config struct {
	Version       string        `yaml:"version"`
	UseProxy      bool          `yaml:"use_proxy"` // default false — services start without proxy
	Logging       Logging       `yaml:"logging"`
	Observability Observability `yaml:"observability"`
	Groups        []Group       `yaml:"groups"`
}

type Logging struct {
	FileRoot string `yaml:"file_root"`
}

type Observability struct {
	GrafanaURL        string `yaml:"grafana_url"`
	LokiURL           string `yaml:"loki_url"`
	TraceDashboardUID string `yaml:"trace_dashboard_uid"`
	LocalPromtail     bool   `yaml:"local_promtail"`
}

type Group struct {
	Name string `yaml:"name"`
	// SkipStartAll excludes this group's services from PlanStartAll / start-all.
	// Panel single-service start and PlanStartGroup still include them (OPT-20260827-022).
	SkipStartAll bool      `yaml:"skip_start_all"`
	Services     []Service `yaml:"services"`
}

type Service struct {
	Name           string            `yaml:"name"`
	ConfApp        string            `yaml:"conf_app"` // optional: conf/<app>/ directory for host/port resolution
	Command        string            `yaml:"command"`  // deprecated: use start_command
	StartCommand   string            `yaml:"start_command"`
	StopCommand    string            `yaml:"stop_command"`
	RestartCommand string            `yaml:"restart_command"` // deprecated: ignored; restart = stop then start
	BuildCommand   string            `yaml:"build_command"`   // optional, runs before restart
	LaunchMode     string            `yaml:"launch_mode"`     // attach | detach
	Language       string            `yaml:"language"`        // optional, shown in UI; auto-detected when empty
	WorkingDir     string            `yaml:"working_dir"`
	Env            map[string]string `yaml:"env"`
	DependsOn      []string          `yaml:"depends_on"`
	OnFailure      string            `yaml:"on_failure"`
	HealthCheck    HealthCheck       `yaml:"health_check"`
	LogFile        string            `yaml:"log_file"` // optional: per-service log file path override
	// IntentPath points to the domain-events SSOT intent (conf/events/domain-events/<event>/<intent-key>).
	// runAll resolves the health port from that YAML instead of hand-writing it in runAll.yaml
	// (OPT-20260816-053). Used by task-events-* services.
	IntentPath string `yaml:"intent_path"`
	// UseProxy overrides the global Config.UseProxy for this service.
	// When nil (unset), the global Config.UseProxy value is used.
	// Set to true to enable proxy, false to explicitly disable proxy for this service.
	UseProxy *bool `yaml:"use_proxy"`
	// SkipStopOnRestart enables hot-replace restart: keep the service serving while
	// the build runs, then reuse the live container instead of stop→start (OPT-20260820-004).
	// Only safe for detach-launch services whose content is served via bind-mount /
	// symlink switch (e.g. taskFE static nginx); when set and the service is currently
	// Healthy, restartService skips the whole stop phase (stop_command + port sweep).
	SkipStopOnRestart bool `yaml:"skip_stop_on_restart"`
	// RestartReloadCommand is an optional command run after a hot-replace restart has
	// reused a live container, so config changes baked into a bind-mount take effect
	// (e.g. `docker exec taskfe-nginx nginx -s reload`). Best-effort: failure is logged,
	// the service stays healthy.
	RestartReloadCommand string `yaml:"restart_reload_command"`
}

type HealthCheck struct {
	URL                string  `yaml:"url"`
	Exec               string  `yaml:"exec"`          // optional; shell command probe (exit 0 = healthy)
	TCP                string  `yaml:"tcp"`           // optional; TCP dial probe (host:port)
	LivenessURL        string  `yaml:"liveness_url"`  // optional; startup probe only (readiness uses url)
	HealthPath         string  `yaml:"health_path"`   // optional; combined with conf host:port to build url
	LivenessPath       string  `yaml:"liveness_path"` // optional; combined with conf host:port to build liveness_url
	Timeout            int     `yaml:"timeout"`
	Retries            int     `yaml:"retries"`
	CheckInterval      int     `yaml:"check_interval"`      // seconds between continuous health pings
	UnhealthyThreshold int     `yaml:"unhealthy_threshold"` // consecutive failures before marking unhealthy
	Backoff            Backoff `yaml:"backoff"`
	// WorkDir is copied from service working_dir so exec probes share start_command cwd.
	WorkDir string `yaml:"-"`
}

// UsesExec reports whether readiness is checked via shell command.
func (hc HealthCheck) UsesExec() bool {
	return strings.TrimSpace(hc.Exec) != ""
}

// UsesTCP reports whether readiness is checked via TCP dial instead of HTTP.
func (hc HealthCheck) UsesTCP() bool {
	return strings.TrimSpace(hc.TCP) != ""
}

// DisplayEndpoint returns a UI-friendly probe target.
func (hc HealthCheck) DisplayEndpoint() string {
	if hc.UsesExec() {
		return "exec://" + strings.TrimSpace(hc.Exec)
	}
	if hc.UsesTCP() {
		return "tcp://" + strings.TrimSpace(hc.TCP)
	}
	return hc.URL
}

// StartupProbeURL returns the HTTP URL used while waiting for a service to become ready.
// When liveness_url is set, startup waits only for the process to serve HTTP (fast path).
// Continuous monitoring and dependency semantics still use url (readiness).
func (hc HealthCheck) StartupProbeURL() string {
	if u := strings.TrimSpace(hc.LivenessURL); u != "" {
		return u
	}
	return hc.URL
}

// ReadinessProbeURL returns the URL for ongoing readiness monitoring.
func (hc HealthCheck) ReadinessProbeURL() string {
	return hc.URL
}

// HasSplitProbe reports whether startup (liveness) and monitoring (readiness) use different URLs.
func (hc HealthCheck) HasSplitProbe() bool {
	live := strings.TrimSpace(hc.LivenessURL)
	ready := strings.TrimSpace(hc.ReadinessProbeURL())
	return live != "" && ready != "" && live != ready
}

// StartupProbeConfig returns a HealthCheck copy that probes liveness during startup.
func (hc HealthCheck) StartupProbeConfig() HealthCheck {
	probe := hc
	probe.URL = hc.StartupProbeURL()
	return probe
}

type Backoff struct {
	Initial    float64 `yaml:"initial"`
	Max        float64 `yaml:"max"`
	Multiplier float64 `yaml:"multiplier"`
}

func (c *Config) validate() error {
	names := map[string]bool{}
	for _, g := range c.Groups {
		for _, svc := range g.Services {
			if svc.Name == "" {
				return fmt.Errorf("service name is required")
			}
			if strings.TrimSpace(svc.EffectiveStartCommand()) == "" {
				return fmt.Errorf("service %q: start_command (or command) is required", svc.Name)
			}
			url := strings.TrimSpace(svc.HealthCheck.URL)
			tcp := strings.TrimSpace(svc.HealthCheck.TCP)
			hasConfApp := strings.TrimSpace(svc.ConfApp) != ""
			if url == "" && tcp == "" && !hasConfApp {
				return fmt.Errorf("service %q: health_check.url, health_check.tcp, or conf_app+health_path is required", svc.Name)
			}
			if url != "" && tcp != "" {
				return fmt.Errorf("service %q: health_check.url and health_check.tcp are mutually exclusive", svc.Name)
			}
			if svc.OnFailure != "exit" && svc.OnFailure != "skip" {
				return fmt.Errorf("service %q: on_failure must be 'exit' or 'skip', got %q", svc.Name, svc.OnFailure)
			}
			if names[svc.Name] {
				return fmt.Errorf("duplicate service name: %q", svc.Name)
			}
			names[svc.Name] = true
		}
	}
	// Validate depends_on references
	for _, g := range c.Groups {
		for _, svc := range g.Services {
			for _, dep := range svc.DependsOn {
				if !names[dep] {
					return fmt.Errorf("service %q: depends_on %q does not exist", svc.Name, dep)
				}
			}
		}
	}
	return nil
}

func (c *Config) Flatten() []Service {
	var result []Service
	for _, g := range c.Groups {
		result = append(result, g.Services...)
	}
	return result
}

// resolveServiceLogFile returns the log file path for a service using three-level
// priority: RUNALL_LOG_ROOT env → conf_app logging.log_file → file_root/<name>.log
func resolveServiceLogFile(cfg *Config, svcName string, confAppLogFile string) string {
	resolution := domain.ServiceLogPathResolution{
		ServiceName:    svcName,
		EnvOverride:    strings.TrimSpace(os.Getenv("RUNALL_LOG_ROOT")),
		ConfAppLogFile: strings.TrimSpace(confAppLogFile),
	}
	if cfg != nil {
		resolution.GlobalFileRoot = strings.TrimSpace(cfg.Logging.FileRoot)
	}
	return resolution.Resolve().Value
}

func resolveLogFileRoot(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		return configured
	}
	return strings.TrimSpace(os.Getenv("RUNALL_LOG_ROOT"))
}

func resolveObservability(o Observability) Observability {
	if env := strings.TrimSpace(os.Getenv("GRAFANA_URL")); env != "" {
		o.GrafanaURL = env
	}
	if env := strings.TrimSpace(os.Getenv("LOKI_URL")); env != "" {
		o.LokiURL = env
	}
	if strings.TrimSpace(o.GrafanaURL) == "" {
		o.GrafanaURL = "http://127.0.0.1:3000"
	}
	if strings.TrimSpace(o.LokiURL) == "" {
		o.LokiURL = "http://127.0.0.1:3100"
	}
	if strings.TrimSpace(o.TraceDashboardUID) == "" {
		o.TraceDashboardUID = "distributed-trace-view"
	}
	return o
}

// LocalPromtailEnabled reports whether Mac-side Promtail should ship tee logs to Loki.
func (c *Config) LocalPromtailEnabled() bool {
	if c == nil {
		return false
	}
	return c.Observability.LocalPromtail
}

// ShouldUseProxy returns whether proxy should be used for this service.
// Per-service override takes precedence; falls back to global config (default false = no proxy).
func (svc Service) ShouldUseProxy(globalUseProxy bool) bool {
	if svc.UseProxy != nil {
		return *svc.UseProxy
	}
	return globalUseProxy
}

// confAppConfig is a minimal struct to read host/port from conf/<app>/config.yaml.
type confAppConfig struct {
	Host     string         `yaml:"host"`
	Port     int            `yaml:"port"`
	HTTPPort int            `yaml:"httpPort"`
	Logging  confAppLogging `yaml:"logging"`
}

type confAppLogging struct {
	LogFile string `yaml:"log_file"`
}

// domainEventConfig reads conf/events/domain-events/<event>/config.yaml (SSOT for
// task-events intent listen ports and groupIds).
type domainEventConfig struct {
	Intents map[string]domainEventIntent `yaml:"intents"`
}

type domainEventIntent struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	GroupID string `yaml:"groupId"`
}

// effectivePort returns the service port, preferring `port` over `httpPort`.
func (c confAppConfig) effectivePort() int {
	if c.Port > 0 {
		return c.Port
	}
	return c.HTTPPort
}

func (c *Config) resolveConfApps(configPath string) error {
	confDir := filepath.Dir(configPath)

	localHost := ""
	serviceHosts := make(map[string]string)
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			app := strings.TrimSpace(svc.ConfApp)
			if app == "" {
				continue
			}
			// Prevent path traversal.
			if strings.Contains(app, "..") {
				return fmt.Errorf("service %q: conf_app %q contains path traversal", svc.Name, app)
			}

			appPath := filepath.Join(confDir, app, "config.yaml")
			data, err := os.ReadFile(appPath)
			if err != nil {
				return fmt.Errorf("service %q: conf_app %q: read config: %w", svc.Name, app, err)
			}
			root := filepath.Dir(confDir)
			rel := filepath.ToSlash(filepath.Join(app, "config.yaml"))
			data, err = overlayConfLocalRel(root, rel, data)
			if err != nil {
				return fmt.Errorf("service %q: conf_app %q: overlay conf-local: %w", svc.Name, app, err)
			}
			var appCfg confAppConfig
			if err := yaml.Unmarshal(data, &appCfg); err != nil {
				return fmt.Errorf("service %q: conf_app %q: parse config: %w", svc.Name, app, err)
			}
			host := strings.TrimSpace(appCfg.Host)
			// Resolve ${INFRA_HOST:-default} bash-style template in host value.
			host = resolveInfraHostTemplate(host)
			if host == "" {
				return fmt.Errorf("service %q: conf_app %q: config has empty host", svc.Name, app)
			}
			// Bind address 0.0.0.0 is not a valid HTTP client target; probe via loopback.
			healthHost := host
			if healthHost == "0.0.0.0" || healthHost == "::" || healthHost == "[::]" {
				healthHost = "127.0.0.1"
			}
			if localHost == "" {
				localHost = healthHost
			}
			serviceHosts[svc.Name] = healthHost

			// Propagate per-service log_file from conf_app if not overridden in runAll.yaml.
			if svc.LogFile == "" && appCfg.Logging.LogFile != "" {
				svc.LogFile = appCfg.Logging.LogFile
			}

			// If the service already has a health check url, tcp, or exec defined
			// directly (e.g. ${INFRA_HOST} placeholders or dockerInfra/*.sh), conf_app
			// is used purely for config mapping. Skip health_path-based URL construction.
			if strings.TrimSpace(svc.HealthCheck.URL) != "" ||
				strings.TrimSpace(svc.HealthCheck.TCP) != "" ||
				svc.HealthCheck.UsesExec() {
				continue
			}

			port := appCfg.effectivePort()
			if port <= 0 {
				return fmt.Errorf("service %q: conf_app %q: config has invalid port (port or httpPort required)", svc.Name, app)
			}

			baseURL := fmt.Sprintf("http://%s:%d", healthHost, port)
			hp := strings.TrimSpace(svc.HealthCheck.HealthPath)
			if hp == "" {
				return fmt.Errorf("service %q: conf_app is set but health_path is empty", svc.Name)
			}
			svc.HealthCheck.URL = baseURL + hp

			if lp := strings.TrimSpace(svc.HealthCheck.LivenessPath); lp != "" {
				svc.HealthCheck.LivenessURL = baseURL + lp
			}
		}
	}

	// Resolve task-events health URLs from the domain-events SSOT via intent_path
	// (OPT-20260816-053). Must run before the ${INFRA_HOST} replacement below so the
	// constructed URLs get the same host resolution as hand-written ones.
	if err := c.resolveDomainEventHealthPaths(confDir); err != nil {
		return err
	}

	// Resolve ${INFRA_HOST} placeholder in url/tcp/liveness_url/env for all services.
	// Services with conf_app use their own host; others fall back to the first conf_app host.
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			host, ok := serviceHosts[svc.Name]
			if !ok {
				host = localHost
			}
			if host == "" {
				continue
			}
			svc.HealthCheck.URL = strings.ReplaceAll(svc.HealthCheck.URL, "${INFRA_HOST}", host)
			svc.HealthCheck.TCP = strings.ReplaceAll(svc.HealthCheck.TCP, "${INFRA_HOST}", host)
			svc.HealthCheck.LivenessURL = strings.ReplaceAll(svc.HealthCheck.LivenessURL, "${INFRA_HOST}", host)
			for k, v := range svc.Env {
				svc.Env[k] = strings.ReplaceAll(v, "${INFRA_HOST}", host)
			}
		}
	}

	// Resolve ${INFRA_HOST} in observability URLs.
	infraHost := localHost
	if infraHost == "" {
		infraHost = "127.0.0.1"
	}
	c.Observability.GrafanaURL = strings.ReplaceAll(c.Observability.GrafanaURL, "${INFRA_HOST}", infraHost)
	c.Observability.LokiURL = strings.ReplaceAll(c.Observability.LokiURL, "${INFRA_HOST}", infraHost)

	return nil
}

// resolveDomainEventHealthPaths builds health_check URLs for task-events services
// declared with intent_path, reading the listen port from the domain-events SSOT
// (conf/events/domain-events/<event>/config.yaml → intents.<key>.port). This makes
// the port single-sourced: copying a sibling intent's groupId is caught at load time
// by the groupId/name mismatch check.
func (c *Config) resolveDomainEventHealthPaths(confDir string) error {
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			path := strings.TrimSpace(svc.IntentPath)
			if path == "" {
				continue
			}
			if strings.Contains(path, "..") {
				return fmt.Errorf("service %q: intent_path %q contains path traversal", svc.Name, path)
			}
			// An explicit probe wins; intent_path then only serves as SSOT cross-check.
			if strings.TrimSpace(svc.HealthCheck.URL) != "" ||
				strings.TrimSpace(svc.HealthCheck.TCP) != "" ||
				svc.HealthCheck.UsesExec() {
				continue
			}
			parts := strings.Split(path, "/")
			if len(parts) < 2 {
				return fmt.Errorf("service %q: intent_path %q must be <event-dir>/<intent-key>", svc.Name, path)
			}
			intentKey := parts[len(parts)-1]
			eventDir := filepath.Join(parts[:len(parts)-1]...)
			data, err := os.ReadFile(filepath.Join(confDir, eventDir, "config.yaml"))
			if err != nil {
				return fmt.Errorf("service %q: intent_path %q: read config: %w", svc.Name, path, err)
			}
			rel := filepath.ToSlash(filepath.Join(eventDir, "config.yaml"))
			data, err = overlayConfLocalRel(filepath.Dir(confDir), rel, data)
			if err != nil {
				return fmt.Errorf("service %q: intent_path %q: overlay conf-local: %w", svc.Name, path, err)
			}
			var de domainEventConfig
			if err := yaml.Unmarshal(data, &de); err != nil {
				return fmt.Errorf("service %q: intent_path %q: parse config: %w", svc.Name, path, err)
			}
			intent, ok := de.Intents[intentKey]
			if !ok {
				return fmt.Errorf("service %q: intent_path %q: intent key %q not found", svc.Name, path, intentKey)
			}
			if intent.Port <= 0 {
				return fmt.Errorf("service %q: intent_path %q: intent %q has invalid port %d", svc.Name, path, intentKey, intent.Port)
			}
			if intent.GroupID != "" && intent.GroupID != svc.Name {
				return fmt.Errorf("service %q: intent_path %q resolves to groupId %q, name mismatch", svc.Name, path, intent.GroupID)
			}
			hp := strings.TrimSpace(svc.HealthCheck.HealthPath)
			if hp == "" {
				return fmt.Errorf("service %q: intent_path is set but health_check.health_path is empty", svc.Name)
			}
			baseURL := fmt.Sprintf("http://${INFRA_HOST}:%d", intent.Port)
			svc.HealthCheck.URL = baseURL + hp
			if lp := strings.TrimSpace(svc.HealthCheck.LivenessPath); lp != "" {
				svc.HealthCheck.LivenessURL = baseURL + lp
			}
		}
	}
	return nil
}

// resolveInfraHostTemplate resolves bash-style ${INFRA_HOST:-default} templates.
// When INFRA_HOST env var is set, use it; otherwise extract the default value.
func resolveInfraHostTemplate(host string) string {
	if !strings.Contains(host, "${INFRA_HOST") {
		return host
	}
	envVal := strings.TrimSpace(os.Getenv("INFRA_HOST"))
	if envVal != "" {
		return strings.ReplaceAll(host, "${INFRA_HOST:-"+extractInfraDefault(host)+"}", envVal)
	}
	// No env set: extract default value from ${INFRA_HOST:-default} syntax.
	if idx := strings.Index(host, "${INFRA_HOST:-"); idx >= 0 {
		start := idx + len("${INFRA_HOST:-")
		end := strings.Index(host[start:], "}")
		if end >= 0 {
			return host[start : start+end]
		}
	}
	// Bare ${INFRA_HOST} with no default and no env → use loopback.
	return strings.ReplaceAll(host, "${INFRA_HOST}", "127.0.0.1")
}

func extractInfraDefault(host string) string {
	if idx := strings.Index(host, "${INFRA_HOST:-"); idx >= 0 {
		start := idx + len("${INFRA_HOST:-")
		end := strings.Index(host[start:], "}")
		if end >= 0 {
			return host[start : start+end]
		}
	}
	return ""
}
