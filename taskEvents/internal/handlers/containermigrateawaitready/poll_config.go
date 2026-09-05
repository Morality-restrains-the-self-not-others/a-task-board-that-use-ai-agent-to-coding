package containermigrateawaitready

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Env keys for await poll tuning (defaults: MaxAttempts=36, RetryDelay=5s).
const (
	EnvMaxAttempts = "CONTAINER_MIGRATE_AWAIT_MAX_ATTEMPTS"
	EnvRetryDelay  = "CONTAINER_MIGRATE_AWAIT_RETRY_DELAY"
)

// PollConfig holds await compensation poll parameters.
type PollConfig struct {
	MaxAttempts int
	RetryDelay  time.Duration
}

// DefaultPollConfig returns built-in defaults (5s × 36 ≈ 3min).
func DefaultPollConfig() PollConfig {
	return PollConfig{MaxAttempts: MaxAttempts, RetryDelay: RetryDelay}
}

// LoadPollConfigFromEnv reads optional overrides from the environment.
// CONTAINER_MIGRATE_AWAIT_MAX_ATTEMPTS: positive int
// CONTAINER_MIGRATE_AWAIT_RETRY_DELAY: Go duration ("5s") or integer seconds ("5")
func LoadPollConfigFromEnv() PollConfig {
	cfg := DefaultPollConfig()
	if v := strings.TrimSpace(os.Getenv(EnvMaxAttempts)); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			cfg.MaxAttempts = n
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvRetryDelay)); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d >= 0 {
			cfg.RetryDelay = d
		} else if sec, err := strconv.Atoi(v); err == nil && sec >= 0 {
			cfg.RetryDelay = time.Duration(sec) * time.Second
		}
	}
	return cfg
}
