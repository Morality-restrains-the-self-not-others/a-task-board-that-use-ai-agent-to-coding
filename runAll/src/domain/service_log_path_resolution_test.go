package domain

import "testing"

func TestServiceLogPathResolution_EnvOverride(t *testing.T) {
	r := ServiceLogPathResolution{
		ServiceName:    "task-auth",
		EnvOverride:    "/var/log/emergency",
		ConfAppLogFile: "logs/task-auth/auth.log",
		GlobalFileRoot: "../logs",
	}
	result := r.Resolve()
	if result.Value != "/var/log/emergency/task-auth.log" {
		t.Errorf("env override should win, got %q", result.Value)
	}
}

func TestServiceLogPathResolution_ConfAppPriority(t *testing.T) {
	r := ServiceLogPathResolution{
		ServiceName:    "task-auth",
		EnvOverride:    "",
		ConfAppLogFile: "logs/task-auth/auth.log",
		GlobalFileRoot: "../logs",
	}
	result := r.Resolve()
	if result.Value != "logs/task-auth/auth.log" {
		t.Errorf("conf_app should win when no env override, got %q", result.Value)
	}
}

func TestServiceLogPathResolution_Fallback(t *testing.T) {
	r := ServiceLogPathResolution{
		ServiceName:    "task-auth",
		EnvOverride:    "",
		ConfAppLogFile: "",
		GlobalFileRoot: "../logs",
	}
	result := r.Resolve()
	expected := "../logs/task-auth.log"
	if result.Value != expected {
		t.Errorf("fallback = %q, want %q", result.Value, expected)
	}
}

func TestServiceLogPathResolution_EmptyGlobalRoot(t *testing.T) {
	r := ServiceLogPathResolution{
		ServiceName:    "unknown-svc",
		EnvOverride:    "",
		ConfAppLogFile: "",
		GlobalFileRoot: "",
	}
	result := r.Resolve()
	if result.Value != "logs/unknown-svc.log" {
		t.Errorf("empty root fallback = %q, want logs/unknown-svc.log", result.Value)
	}
}

func TestServiceLogPathResolution_WhitespaceTrimmed(t *testing.T) {
	r := ServiceLogPathResolution{
		ServiceName:    "svc",
		EnvOverride:    "  ",
		ConfAppLogFile: "  ",
		GlobalFileRoot: "  ../logs  ",
	}
	result := r.Resolve()
	if result.Value != "../logs/svc.log" {
		t.Errorf("whitespace-trimmed fallback = %q, want ../logs/svc.log", result.Value)
	}
}
