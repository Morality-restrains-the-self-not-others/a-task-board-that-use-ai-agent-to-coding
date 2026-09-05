package main

import (
	"os"
	"testing"
)

func TestTimeoutFromEnvUsesDefaultWhenUnset(t *testing.T) {
	os.Unsetenv("TEST_TIMEOUT_KEY")
	got := timeoutFromEnv("TEST_TIMEOUT_KEY", 45.0, 0.1, 120.0)
	if got != 45.0 {
		t.Errorf("expected default 45.0, got %v", got)
	}
}

func TestTimeoutFromEnvParsesValidValue(t *testing.T) {
	os.Setenv("TEST_TIMEOUT_KEY", "30.5")
	defer os.Unsetenv("TEST_TIMEOUT_KEY")
	got := timeoutFromEnv("TEST_TIMEOUT_KEY", 45.0, 0.1, 120.0)
	if got != 30.5 {
		t.Errorf("expected 30.5, got %v", got)
	}
}

func TestTimeoutFromEnvClampsToMin(t *testing.T) {
	os.Setenv("TEST_TIMEOUT_KEY", "0.01")
	defer os.Unsetenv("TEST_TIMEOUT_KEY")
	got := timeoutFromEnv("TEST_TIMEOUT_KEY", 45.0, 0.1, 120.0)
	if got != 0.1 {
		t.Errorf("expected clamped to min 0.1, got %v", got)
	}
}

func TestTimeoutFromEnvClampsToMax(t *testing.T) {
	os.Setenv("TEST_TIMEOUT_KEY", "999.0")
	defer os.Unsetenv("TEST_TIMEOUT_KEY")
	got := timeoutFromEnv("TEST_TIMEOUT_KEY", 45.0, 0.1, 120.0)
	if got != 120.0 {
		t.Errorf("expected clamped to max 120.0, got %v", got)
	}
}

func TestTimeoutFromEnvRejectsInvalidString(t *testing.T) {
	os.Setenv("TEST_TIMEOUT_KEY", "not-a-number")
	defer os.Unsetenv("TEST_TIMEOUT_KEY")
	got := timeoutFromEnv("TEST_TIMEOUT_KEY", 45.0, 0.1, 120.0)
	if got != 45.0 {
		t.Errorf("expected default 45.0 for invalid input, got %v", got)
	}
}

func TestTimeoutFromEnvHandlesWhitespace(t *testing.T) {
	os.Setenv("TEST_TIMEOUT_KEY", "  60.0  ")
	defer os.Unsetenv("TEST_TIMEOUT_KEY")
	got := timeoutFromEnv("TEST_TIMEOUT_KEY", 45.0, 0.1, 120.0)
	if got != 60.0 {
		t.Errorf("expected 60.0, got %v", got)
	}
}

func TestPickPortDefault(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_ONLINE_PORT")
	os.Unsetenv("PORT")
	got := pickPort()
	if got != 8765 {
		t.Errorf("expected default 8765, got %d", got)
	}
}

func TestPickPortFromRelayEnv(t *testing.T) {
	os.Setenv("RELAY_TO_TRAE_ONLINE_PORT", "9999")
	defer os.Unsetenv("RELAY_TO_TRAE_ONLINE_PORT")
	got := pickPort()
	if got != 9999 {
		t.Errorf("expected 9999, got %d", got)
	}
}

func TestPickPortFallsBackToPortEnv(t *testing.T) {
	os.Unsetenv("RELAY_TO_TRAE_ONLINE_PORT")
	os.Setenv("PORT", "8888")
	defer os.Unsetenv("PORT")
	got := pickPort()
	if got != 8888 {
		t.Errorf("expected 8888, got %d", got)
	}
}

func TestPickPortRejectsInvalidValue(t *testing.T) {
	os.Setenv("RELAY_TO_TRAE_ONLINE_PORT", "abc")
	defer os.Unsetenv("RELAY_TO_TRAE_ONLINE_PORT")
	got := pickPort()
	if got != 8765 {
		t.Errorf("expected default 8765 for invalid, got %d", got)
	}
}
