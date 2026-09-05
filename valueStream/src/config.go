package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version      string        `yaml:"version"`
	RunallConfig string        `yaml:"runall_config"`
	Runner       RunnerConfig  `yaml:"runner"`
	ValueStreams []ValueStream `yaml:"value_streams"`
	ConfigDir    string        `yaml:"-"`
	ConfigPath   string        `yaml:"-"`
}

type RunnerConfig struct {
	WorkingDir string            `yaml:"working_dir"`
	PytestBin  string            `yaml:"pytest_bin"`
	Env        map[string]string `yaml:"env"`
	PytestArgs []string          `yaml:"pytest_args"`
}

type ValueStream struct {
	Name        string `yaml:"name"`
	Domain      string `yaml:"domain"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

type StepField struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Step struct {
	Name        string      `yaml:"name"`
	Lifecycle   string      `yaml:"status"` // active (default) | planned | deprecated
	TestFile    string      `yaml:"test_file"`
	Fields      []StepField `yaml:"fields"`
	TestFileAbs string      `yaml:"-"`
}

func (s *Step) normalizeLifecycle() {
	if s.Lifecycle == "" {
		s.Lifecycle = "active"
	}
}

func (s *Step) IsActive() bool {
	return s.Lifecycle == "" || s.Lifecycle == "active"
}

func (s *Step) IsPlanned() bool {
	return s.Lifecycle == "planned"
}

func (s *Step) IsDeprecated() bool {
	return s.Lifecycle == "deprecated"
}

func LoadConfig(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	abs, err := filepath.Abs(filepath.Dir(absPath))
	if err != nil {
		return nil, err
	}
	cfg.ConfigDir = abs
	cfg.ConfigPath = absPath
	cfg.fillDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.resolvePaths()
	return &cfg, nil
}

func ReorderValueStreamsByDomain(streams []ValueStream, domainOrder []string) ([]ValueStream, error) {
	if len(streams) == 0 {
		return nil, fmt.Errorf("no value streams configured")
	}
	if len(domainOrder) == 0 {
		return nil, fmt.Errorf("domain_order is required")
	}

	domainToStreams := make(map[string][]ValueStream)
	domainSeen := make(map[string]bool)
	orderedDomains := make([]string, 0)
	for _, vs := range streams {
		domain := strings.TrimSpace(vs.Domain)
		if domain == "" {
			domain = "未分组"
		}
		if !domainSeen[domain] {
			domainSeen[domain] = true
			orderedDomains = append(orderedDomains, domain)
		}
		domainToStreams[domain] = append(domainToStreams[domain], vs)
	}

	requested := make([]string, 0, len(domainOrder))
	requestedSeen := make(map[string]bool, len(domainOrder))
	for _, raw := range domainOrder {
		domain := strings.TrimSpace(raw)
		if domain == "" {
			return nil, fmt.Errorf("domain_order contains empty domain")
		}
		if requestedSeen[domain] {
			return nil, fmt.Errorf("domain_order contains duplicate domain %q", domain)
		}
		if !domainSeen[domain] {
			return nil, fmt.Errorf("domain_order contains unknown domain %q", domain)
		}
		requestedSeen[domain] = true
		requested = append(requested, domain)
	}

	result := make([]ValueStream, 0, len(streams))
	for _, domain := range requested {
		result = append(result, domainToStreams[domain]...)
	}
	for _, domain := range orderedDomains {
		if requestedSeen[domain] {
			continue
		}
		result = append(result, domainToStreams[domain]...)
	}
	if len(result) != len(streams) {
		return nil, fmt.Errorf("internal error: reordered stream count mismatch")
	}
	return result, nil
}

func SaveConfigAtomic(path string, cfg *Config) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("config path is required")
	}
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	cfg.ConfigPath = absPath
	cfg.ConfigDir = filepath.Dir(absPath)
	cfg.fillDefaults()
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("validate config before save: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	mode := os.FileMode(0644)
	if info, statErr := os.Stat(absPath); statErr == nil {
		mode = info.Mode()
	}
	dir := filepath.Dir(absPath)
	tempFile, err := os.CreateTemp(dir, ".value-stream-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tempFile.Chmod(mode); err != nil {
		tempFile.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tempPath, absPath); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	cfg.ConfigPath = absPath
	cfg.ConfigDir = filepath.Dir(absPath)
	cfg.fillDefaults()
	cfg.resolvePaths()
	return nil
}

func (c *Config) fillDefaults() {
	if c.Runner.PytestBin == "" {
		c.Runner.PytestBin = "pytest"
	}
	for i := range c.ValueStreams {
		c.ValueStreams[i].Domain = strings.TrimSpace(c.ValueStreams[i].Domain)
		for j := range c.ValueStreams[i].Steps {
			c.ValueStreams[i].Steps[j].normalizeLifecycle()
		}
	}
}

func (c *Config) validate() error {
	if c.Version != "1" {
		return fmt.Errorf("unsupported version %q, want \"1\"", c.Version)
	}
	if c.Runner.WorkingDir == "" {
		return fmt.Errorf("runner.working_dir is required")
	}
	if len(c.ValueStreams) == 0 {
		return fmt.Errorf("at least one value_streams entry required")
	}

	var runAllNames map[string]bool
	if c.RunallConfig != "" {
		p := filepath.Join(c.ConfigDir, c.RunallConfig)
		var err error
		runAllNames, err = LoadRunAllServiceNames(p)
		if err != nil {
			return fmt.Errorf("runall_config: %w", err)
		}
		// runAll 编排器自身也是一个合法的 field provider（表示 runAll 自身运行时行为）
		runAllNames["runall"] = true
	} else {
		log.Printf("[valueStream] runall_config not set; field provider names not validated against runAll")
	}

	seenStream := map[string]bool{}
	for _, vs := range c.ValueStreams {
		if vs.Name == "" {
			return fmt.Errorf("value stream name is required")
		}
		if strings.TrimSpace(vs.Domain) == "" {
			return fmt.Errorf("stream %q: domain is required", vs.Name)
		}
		if seenStream[vs.Name] {
			return fmt.Errorf("duplicate value stream name %q", vs.Name)
		}
		seenStream[vs.Name] = true
		if len(vs.Steps) == 0 {
			return fmt.Errorf("stream %q: at least one step required", vs.Name)
		}
		seenStep := map[string]bool{}
		for _, st := range vs.Steps {
			if st.Name == "" {
				return fmt.Errorf("stream %q: step name required", vs.Name)
			}
			if seenStep[st.Name] {
				return fmt.Errorf("stream %q: duplicate step name %q", vs.Name, st.Name)
			}
			seenStep[st.Name] = true
			switch st.Lifecycle {
			case "active":
				// active step
			case "planned", "deprecated":
				// ok; test_file not validated for planned/deprecated steps
			default:
				return fmt.Errorf("stream %q step %q: status must be active, planned, or deprecated, got %q", vs.Name, st.Name, st.Lifecycle)
			}
			if st.TestFile == "" {
				return fmt.Errorf("stream %q step %q: test_file required", vs.Name, st.Name)
			}
			seenField := map[string]bool{}
			for _, f := range st.Fields {
				prov, _, _, err := ParseFieldName(f.Name)
				if err != nil {
					return fmt.Errorf("stream %q step %q: %w", vs.Name, st.Name, err)
				}
				if seenField[f.Name] {
					return fmt.Errorf("stream %q step %q: duplicate field %q", vs.Name, st.Name, f.Name)
				}
				seenField[f.Name] = true
				if runAllNames != nil && !runAllNames[prov] {
					// OPT-049: Django saas-backend decommissioned (2026-07-30).
					// Provider name may reference legacy tables that still exist in the DB;
					// skip unknown providers instead of failing startup.
					continue
				}
			}
		}
	}

	wd := c.runnerWorkingDirAbs()
	info, err := os.Stat(wd)
	if err != nil || !info.IsDir() {
		// OPT-049: Django decommissioned (2026-07-30). Pytest runner dir gone; warn not fail.
			log.Printf("[valueStream] runner.working_dir not found: %s (pytests disabled)", wd)
	}
	for i := range c.ValueStreams {
		for j := range c.ValueStreams[i].Steps {
			st := &c.ValueStreams[i].Steps[j]
			if st.IsPlanned() || st.IsDeprecated() {
				continue
			}
			abs := filepath.Join(wd, st.TestFile)
			if _, err := os.Stat(abs); err != nil {
				// Non-critical: skip missing test files (value-stream is observability tooling)
				log.Printf("[valueStream] WARNING: test_file not found (skipped): %s", abs)
			}
		}
	}
	return nil
}

func (c *Config) runnerWorkingDirAbs() string {
	return filepath.Join(c.ConfigDir, c.Runner.WorkingDir)
}

func (c *Config) resolvePaths() {
	wd := c.runnerWorkingDirAbs()
	for i := range c.ValueStreams {
		for j := range c.ValueStreams[i].Steps {
			st := &c.ValueStreams[i].Steps[j]
			if st.IsActive() {
				st.TestFileAbs = filepath.Join(wd, st.TestFile)
			}
		}
	}
}

func (c *Config) StreamByName(name string) (*ValueStream, error) {
	for i := range c.ValueStreams {
		if c.ValueStreams[i].Name == name {
			return &c.ValueStreams[i], nil
		}
	}
	return nil, fmt.Errorf("unknown stream %q", name)
}

func (vs *ValueStream) StepByName(name string) (*Step, error) {
	for i := range vs.Steps {
		if vs.Steps[i].Name == name {
			return &vs.Steps[i], nil
		}
	}
	return nil, fmt.Errorf("unknown step %q", name)
}
