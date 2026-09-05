package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"confload"
	"taskCloudService/domain"

	"gopkg.in/yaml.v3"
)

// StepFullCOSConfig is the admin-editable COS fragment (secrets only in conf-local).
type StepFullCOSConfig struct {
	Backend             string `yaml:"backend" json:"backend"`
	Bucket              string `yaml:"bucket" json:"bucket"`
	Region              string `yaml:"region" json:"region"`
	PathRule            string `yaml:"pathRule" json:"pathRule"`
	StartupLogsPathRule string `yaml:"startupLogsPathRule" json:"startupLogsPathRule"`
	KeyPrefix           string `yaml:"keyPrefix" json:"keyPrefix"`
	SecretID            string `yaml:"secretId" json:"-"`
	SecretKey           string `yaml:"secretKey" json:"-"`
}

var stepFullCOSCfg = StepFullCOSConfig{
	Backend:             "local",
	PathRule:            domain.DefaultStepFullPathRule,
	StartupLogsPathRule: domain.DefaultStartupLogPathRule,
}

// stepFullCOSWriteDir is the conf/taskCloudService directory; tests may override.
var stepFullCOSWriteDir string

func stepFullCOSDir() string {
	if d := strings.TrimSpace(stepFullCOSWriteDir); d != "" {
		return d
	}
	root, err := findMonorepoRoot()
	if err != nil {
		return ""
	}
	return filepath.Join(root, "conf", "taskCloudService")
}

func loadStepFullCOSConf(repoRoot string) {
	type frag struct {
		Backend             string `yaml:"backend"`
		Bucket              string `yaml:"bucket"`
		Region              string `yaml:"region"`
		PathRule            string `yaml:"pathRule"`
		StartupLogsPathRule string `yaml:"startupLogsPathRule"`
		KeyPrefix           string `yaml:"keyPrefix"`
		SecretID            string `yaml:"secretId"`
		SecretKey           string `yaml:"secretKey"`
	}
	var public frag
	if err := confload.ReadAppFragment(repoRoot, "taskCloudService", "step-full-cos.yaml", &public); err != nil {
		log.Printf("[taskCloudService] step-full-cos.yaml: %v (using local backend defaults)", err)
	}
	out := stepFullCOSCfg
	if s := strings.TrimSpace(public.Backend); s != "" {
		out.Backend = s
	}
	if s := strings.TrimSpace(public.Bucket); s != "" {
		out.Bucket = s
	}
	if s := strings.TrimSpace(public.Region); s != "" {
		out.Region = s
	}
	if s := strings.TrimSpace(public.PathRule); s != "" {
		out.PathRule = s
	}
	if s := strings.TrimSpace(public.StartupLogsPathRule); s != "" {
		out.StartupLogsPathRule = s
	}
	if s := strings.TrimSpace(public.KeyPrefix); s != "" {
		out.KeyPrefix = s
	}
	if s := strings.TrimSpace(public.SecretID); s != "" {
		out.SecretID = s
	}
	if s := strings.TrimSpace(public.SecretKey); s != "" {
		out.SecretKey = s
	}
	if strings.TrimSpace(out.PathRule) == "" {
		out.PathRule = domain.DefaultStepFullPathRule
	}
	if strings.TrimSpace(out.StartupLogsPathRule) == "" {
		out.StartupLogsPathRule = domain.DefaultStartupLogPathRule
	}
	if strings.TrimSpace(out.Backend) == "" {
		out.Backend = "local"
	}
	stepFullCOSCfg = out
}

func writeStepFullCOSFragments(cfg StepFullCOSConfig) error {
	dir := stepFullCOSDir()
	if dir == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	public := map[string]any{
		"backend":             strings.TrimSpace(cfg.Backend),
		"bucket":              strings.TrimSpace(cfg.Bucket),
		"region":              strings.TrimSpace(cfg.Region),
		"pathRule":            strings.TrimSpace(cfg.PathRule),
		"startupLogsPathRule": strings.TrimSpace(cfg.StartupLogsPathRule),
		"keyPrefix":           strings.TrimSpace(cfg.KeyPrefix),
	}
	raw, err := yaml.Marshal(public)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "step-full-cos.yaml"), raw, 0o644); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.SecretID) == "" && strings.TrimSpace(cfg.SecretKey) == "" {
		return nil
	}
	secretDir := filepath.Join(dir, "conf-local")
	if strings.TrimSpace(stepFullCOSWriteDir) == "" {
		root, err := findMonorepoRoot()
		if err != nil {
			return err
		}
		secretDir = filepath.Join(root, "conf-local", "taskCloudService")
	}
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		return err
	}
	secrets := map[string]any{
		"secretId":  cfg.SecretID,
		"secretKey": cfg.SecretKey,
	}
	sraw, err := yaml.Marshal(secrets)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(secretDir, "step-full-cos.yaml"), sraw, 0o600)
}

func applyStepFullCOSConfig(next StepFullCOSConfig) error {
	if strings.TrimSpace(next.PathRule) == "" {
		next.PathRule = domain.DefaultStepFullPathRule
	}
	if err := domain.ValidateStepFullPathRule(next.PathRule); err != nil {
		return err
	}
	if strings.TrimSpace(next.StartupLogsPathRule) == "" {
		next.StartupLogsPathRule = domain.DefaultStartupLogPathRule
	}
	if err := domain.ValidateStartupLogPathRule(next.StartupLogsPathRule); err != nil {
		return err
	}
	backend := strings.ToLower(strings.TrimSpace(next.Backend))
	if backend == "" {
		backend = "local"
	}
	if backend != "local" && backend != "cos" {
		return fmt.Errorf("backend 须为 local 或 cos")
	}
	next.Backend = backend
	if strings.TrimSpace(next.SecretID) == "" {
		next.SecretID = stepFullCOSCfg.SecretID
	}
	if strings.TrimSpace(next.SecretKey) == "" {
		next.SecretKey = stepFullCOSCfg.SecretKey
	}
	if err := writeStepFullCOSFragments(next); err != nil {
		return err
	}
	stepFullCOSCfg = next
	initStepFullObjectStoreFromCfg()
	return nil
}

func stepFullCOSPublicView() map[string]any {
	return map[string]any{
		"backend":             stepFullCOSCfg.Backend,
		"bucket":              stepFullCOSCfg.Bucket,
		"region":              stepFullCOSCfg.Region,
		"pathRule":            stepFullCOSCfg.PathRule,
		"startupLogsPathRule": stepFullCOSCfg.StartupLogsPathRule,
		"keyPrefix":           stepFullCOSCfg.KeyPrefix,
		"secret_configured":   strings.TrimSpace(stepFullCOSCfg.SecretID) != "" && strings.TrimSpace(stepFullCOSCfg.SecretKey) != "",
	}
}
