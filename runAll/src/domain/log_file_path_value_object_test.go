package domain

import "testing"

func TestNewLogFilePath_Valid(t *testing.T) {
	p, err := NewLogFilePath("logs/task-auth/auth.log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Value != "logs/task-auth/auth.log" {
		t.Errorf("Value = %q", p.Value)
	}
}

func TestNewLogFilePath_Empty(t *testing.T) {
	_, err := NewLogFilePath("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestNewLogFilePath_WhitespaceOnly(t *testing.T) {
	_, err := NewLogFilePath("   ")
	if err == nil {
		t.Error("expected error for whitespace-only path")
	}
}

func TestLogFilePath_IsAbsolute(t *testing.T) {
	abs, _ := NewLogFilePath("/var/log/app.log")
	if !abs.IsAbsolute() {
		t.Error("expected absolute path")
	}

	rel, _ := NewLogFilePath("logs/app.log")
	if rel.IsAbsolute() {
		t.Error("expected relative path")
	}
}

func TestLogFilePath_Dir(t *testing.T) {
	p, _ := NewLogFilePath("logs/task-auth/auth.log")
	if p.Dir() != "logs/task-auth" {
		t.Errorf("Dir = %q, want logs/task-auth", p.Dir())
	}
}
