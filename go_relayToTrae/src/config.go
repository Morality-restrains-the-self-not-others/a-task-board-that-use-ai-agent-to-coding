package main

import (
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"confload"
)

// Timeout configuration values (parsed from env or conf/ YAML).
var (
	resetTimeoutSec     float64
	pushTimeoutSec      float64
	procTermTimeoutSec  float64
	procKillTimeoutSec  float64
	portProbeTimeoutSec float64

	secretToken string
)

func timeoutFromEnv(key string, defaultVal, minVal, maxVal float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return defaultVal
	}
	return math.Max(minVal, math.Min(maxVal, val))
}

func initTimeouts() {
	resetTimeoutSec = timeoutFromEnv("RELAY_TO_TRAE_RESET_TIMEOUT_SEC", 45.0, 0.1, 120.0)
	pushTimeoutSec = timeoutFromEnv("RELAY_TO_TRAE_PUSH_TIMEOUT_SEC", 20.0, 0.1, 60.0)
	procTermTimeoutSec = timeoutFromEnv("RELAY_TO_TRAE_PROC_TERM_TIMEOUT_SEC", 12.0, 0.1, 60.0)
	procKillTimeoutSec = timeoutFromEnv("RELAY_TO_TRAE_PROC_KILL_TIMEOUT_SEC", 5.0, 0.1, 30.0)
	portProbeTimeoutSec = timeoutFromEnv("RELAY_TO_TRAE_PORT_PROBE_TIMEOUT_SEC", 0.35, 0.1, 5.0)
	secretToken = strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_SECRET"))
}

func applyPortConfig(repoRoot string) error {
	var block struct {
		Host               string  `yaml:"host"`
		Port               int     `yaml:"port"`
		Secret             *string `yaml:"secret"`
		PublicIP           string  `yaml:"publicIp"`
		BusinessApiOrigin  string  `yaml:"businessApiOrigin"`
		Timeouts           struct {
			ResetSec     *float64 `yaml:"resetSec"`
			PushSec      *float64 `yaml:"pushSec"`
			ProcTermSec  *float64 `yaml:"procTermSec"`
			ProcKillSec  *float64 `yaml:"procKillSec"`
			PortProbeSec *float64 `yaml:"portProbeSec"`
		} `yaml:"timeouts"`
	}
	if err := confload.ReadAppConfig(repoRoot, "relay-to-trae", &block); err != nil {
		return nil
	}
	if block.Host != "" {
		os.Setenv("RELAY_TO_TRAE_HOST", block.Host)
	}
	if block.Port != 0 {
		os.Setenv("RELAY_TO_TRAE_PORT", strconv.Itoa(block.Port))
	}
	if block.Secret != nil {
		os.Setenv("RELAY_TO_TRAE_SECRET", *block.Secret)
	}
	if pip := strings.TrimSpace(block.PublicIP); pip != "" {
		host := pip
		if strings.Contains(pip, "://") {
			if u, err := url.Parse(pip); err == nil && strings.TrimSpace(u.Hostname()) != "" {
				host = u.Hostname()
			}
		}
		if strings.TrimSpace(os.Getenv("TRAE_PUBLIC_IP")) == "" {
			os.Setenv("TRAE_PUBLIC_IP", host)
		}
		if strings.TrimSpace(os.Getenv("PUBLIC_IP")) == "" {
			os.Setenv("PUBLIC_IP", host)
		}
		if strings.Contains(pip, "://") && strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PUBLIC_ORIGIN")) == "" {
			os.Setenv("RELAY_TO_TRAE_PUBLIC_ORIGIN", strings.TrimRight(pip, "/"))
		}
	}
	if biz := strings.TrimSpace(block.BusinessApiOrigin); biz != "" {
		if strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_BUSINESS_API_ORIGIN")) == "" {
			os.Setenv("RELAY_TO_TRAE_BUSINESS_API_ORIGIN", biz)
		}
	}

	to := block.Timeouts
	if to.ResetSec != nil && strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_RESET_TIMEOUT_SEC")) == "" {
		os.Setenv("RELAY_TO_TRAE_RESET_TIMEOUT_SEC", strconv.FormatFloat(*to.ResetSec, 'f', -1, 64))
	}
	if to.PushSec != nil && strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PUSH_TIMEOUT_SEC")) == "" {
		os.Setenv("RELAY_TO_TRAE_PUSH_TIMEOUT_SEC", strconv.FormatFloat(*to.PushSec, 'f', -1, 64))
	}
	if to.ProcTermSec != nil && strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PROC_TERM_TIMEOUT_SEC")) == "" {
		os.Setenv("RELAY_TO_TRAE_PROC_TERM_TIMEOUT_SEC", strconv.FormatFloat(*to.ProcTermSec, 'f', -1, 64))
	}
	if to.ProcKillSec != nil && strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PROC_KILL_TIMEOUT_SEC")) == "" {
		os.Setenv("RELAY_TO_TRAE_PROC_KILL_TIMEOUT_SEC", strconv.FormatFloat(*to.ProcKillSec, 'f', -1, 64))
	}
	if to.PortProbeSec != nil && strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_PORT_PROBE_TIMEOUT_SEC")) == "" {
		os.Setenv("RELAY_TO_TRAE_PORT_PROBE_TIMEOUT_SEC", strconv.FormatFloat(*to.PortProbeSec, 'f', -1, 64))
	}
	return nil
}

func onlineServiceRunSh(repoRoot string) string {
	return filepath.Join(repoRoot, "trae-agent", "onlineServiceJS", "run.sh")
}

func pickPort() int {
	raw := strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_ONLINE_PORT"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("PORT"))
	}
	if raw == "" {
		raw = "8765"
	}
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 8765
	}
	return port
}
