package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"runAll/src/domain"
)

func LoadConfig(path string) (*Config, error) {
	if err := loadCutoverEnvForConfig(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	baseData := data
	data, err = overlayConfLocal(path, data)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	if err := checkConfLocalDoesNotExtendBase(baseData, &cfg, path); err != nil {
		return nil, err
	}

	if env := strings.TrimSpace(os.Getenv("RUNALL_LOG_ROOT")); env != "" {
		cfg.Logging.FileRoot = env
	}

	cfg.fillDefaults()
	if err := cfg.resolveConfApps(path); err != nil {
		return nil, fmt.Errorf("resolve conf apps: %w", err)
	}
	if err := cfg.normalizeServiceLifecycleCommands(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	absDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("resolve config dir: %w", err)
	}
	cfg.resolveWorkingDirs(absDir)

	return &cfg, nil
}

func LoadConfigWithSourceGuard(primaryPath, secondaryPath string) (*Config, domain.ConfigFingerprint, error) {
	emptyFingerprint := domain.ConfigFingerprint{}

	primaryHash, err := fileSHA256(primaryPath)
	if err != nil {
		return nil, emptyFingerprint, fmt.Errorf("hash primary config: %w", err)
	}

	if strings.TrimSpace(secondaryPath) != "" {
		secondaryHash, err := fileSHA256(secondaryPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, emptyFingerprint, fmt.Errorf("hash secondary config: %w", err)
		}
		if err == nil && secondaryHash != primaryHash {
			return nil, emptyFingerprint, fmt.Errorf("config source mismatch: primary=%s secondary=%s", primaryPath, secondaryPath)
		}
	}

	fingerprint, err := domain.NewConfigFingerprint(primaryPath, primaryHash)
	if err != nil {
		return nil, emptyFingerprint, fmt.Errorf("create config fingerprint: %w", err)
	}

	cfg, err := LoadConfig(primaryPath)
	if err != nil {
		return nil, emptyFingerprint, err
	}

	return cfg, fingerprint, nil
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (c *Config) fillDefaults() {
	c.Logging.FileRoot = resolveLogFileRoot(c.Logging.FileRoot)
	c.Observability = resolveObservability(c.Observability)
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			if svc.OnFailure == "" {
				svc.OnFailure = "exit"
			}
			if svc.HealthCheck.Timeout == 0 {
				svc.HealthCheck.Timeout = 30
			}
			if svc.HealthCheck.Retries == 0 {
				svc.HealthCheck.Retries = 10
			}
			if svc.HealthCheck.Backoff.Initial == 0 {
				svc.HealthCheck.Backoff.Initial = 1.0
			}
			if svc.HealthCheck.Backoff.Max == 0 {
				svc.HealthCheck.Backoff.Max = 8.0
			}
			if svc.HealthCheck.Backoff.Multiplier == 0 {
				svc.HealthCheck.Backoff.Multiplier = 2.0
			}
			if svc.HealthCheck.CheckInterval == 0 {
				svc.HealthCheck.CheckInterval = 10
			}
			if svc.HealthCheck.UnhealthyThreshold == 0 {
				svc.HealthCheck.UnhealthyThreshold = 2
			}
		}
	}
}
