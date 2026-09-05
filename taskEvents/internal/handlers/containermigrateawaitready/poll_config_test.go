package containermigrateawaitready

import (
	"context"
	"os"
	"testing"
	"time"

	"taskEvents/internal/repository/cloudconfig"
)

func TestLoadPollConfigFromEnvDefaults(t *testing.T) {
	_ = os.Unsetenv(EnvMaxAttempts)
	_ = os.Unsetenv(EnvRetryDelay)
	cfg := LoadPollConfigFromEnv()
	if cfg.MaxAttempts != MaxAttempts {
		t.Fatalf("MaxAttempts=%d want %d", cfg.MaxAttempts, MaxAttempts)
	}
	if cfg.RetryDelay != RetryDelay {
		t.Fatalf("RetryDelay=%v want %v", cfg.RetryDelay, RetryDelay)
	}
}

func TestLoadPollConfigFromEnvOverrides(t *testing.T) {
	t.Setenv(EnvMaxAttempts, "12")
	t.Setenv(EnvRetryDelay, "2s")
	cfg := LoadPollConfigFromEnv()
	if cfg.MaxAttempts != 12 {
		t.Fatalf("MaxAttempts=%d want 12", cfg.MaxAttempts)
	}
	if cfg.RetryDelay != 2*time.Second {
		t.Fatalf("RetryDelay=%v want 2s", cfg.RetryDelay)
	}

	t.Setenv(EnvRetryDelay, "7")
	cfg = LoadPollConfigFromEnv()
	if cfg.RetryDelay != 7*time.Second {
		t.Fatalf("RetryDelay=%v want 7s (integer seconds)", cfg.RetryDelay)
	}
}

func TestHandlerUsesConfiguredRetryDelay(t *testing.T) {
	var slept time.Duration
	pub := &recordingPublisher{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{LastRuntimeStatus: "Starting", InstanceID: "i-1"}},
		Publisher:   pub,
		MaxAttempts: 10,
		RetryDelay:  123 * time.Millisecond,
		Sleep:       func(d time.Duration) { slept = d },
	}
	_, err := h.Dispatch(context.Background(), makeCmd(map[string]interface{}{
		"company_id": float64(1), "workspace_id": "ws", "task_id": "t1", "attempt": 1,
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if slept != 123*time.Millisecond {
		t.Fatalf("slept=%v want 123ms", slept)
	}
}
