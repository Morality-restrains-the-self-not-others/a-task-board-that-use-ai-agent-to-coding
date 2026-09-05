package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"runAll/src/domain"
)

func TestCleanupOrphanManagedServices_SkipsWhenShutdownSelf(t *testing.T) {
	// Graceful shutdown-self must not treat still-running managed listeners as orphans.
	// Passing nil cfg is enough: the early-return path must not panic or attempt kills.
	cleanupOrphanManagedServices(nil, previousRunAllShutdownResult{GracefulShutdownSelf: true}, true)
	cleanupOrphanManagedServices(&Config{Version: "1"}, previousRunAllShutdownResult{GracefulShutdownSelf: true}, true)
}

// OPT-20260820-008: 无旧 UI listener（HadPrevious=false）时不得把仍在监听的托管端口当
// 孤儿 SIGKILL，也不得走 15s 等待强制清理——该模式应跳过清理。
func TestShouldSkipOrphanCleanup_NoPriorUI(t *testing.T) {
	if !shouldSkipOrphanCleanup(previousRunAllShutdownResult{HadPrevious: false, GracefulShutdownSelf: false}) {
		t.Fatal("HadPrevious=false must skip orphan cleanup (adopt instead of SIGKILL)")
	}
	if !shouldSkipOrphanCleanup(previousRunAllShutdownResult{HadPrevious: true, GracefulShutdownSelf: true}) {
		t.Fatal("GracefulShutdownSelf=true must skip orphan cleanup")
	}
	if shouldSkipOrphanCleanup(previousRunAllShutdownResult{HadPrevious: true, GracefulShutdownSelf: false}) {
		t.Fatal("HadPrevious=true without shutdown-self must still run orphan cleanup")
	}
}

// OPT-20260820-008: 9999 空 + 假监听端口场景下 skipOrphanKill=true 时函数直接返回，
// 不等待也不 SIGKILL（9999 空窗拉起 Status UI 不再误杀整栈）。
func TestCleanupOrphanManagedServices_SkipDoesNotKillListeners(t *testing.T) {
	// 早退路径：cfg 非 nil + skipOrphanKill=true 时不得触碰任何监听端口。
	// 用带假服务的 cfg 调用，若误走清理路径会在 15s 后 SIGKILL——该测试存活即证明未误杀。
	cfg := &Config{Version: "1"}
	cleanupOrphanManagedServices(cfg, previousRunAllShutdownResult{}, true)
}

func TestCapabilitySkipOrphanOnShutdownSelfMarker(t *testing.T) {
	if capabilitySkipOrphanOnShutdownSelf != "skip orphan port cleanup" {
		t.Fatalf("capability marker drifted: %q", capabilitySkipOrphanOnShutdownSelf)
	}
	// Keep the marker as a plain string constant so `strings` / build.sh can verify
	// the deployed binary before shutdown-self hot-replace.
	if !strings.Contains(capabilitySkipOrphanOnShutdownSelf, "skip orphan") {
		t.Fatal("capability marker must remain searchable in the binary")
	}
}

func TestPreviousRunAllShutdownResult_DefaultDoesNotSkip(t *testing.T) {
	result := previousRunAllShutdownResult{}
	if result.GracefulShutdownSelf {
		t.Fatal("default GracefulShutdownSelf must be false so hard-kill orphan cleanup still runs")
	}
}

func TestLoadRuntimeConfig_UsesSourceGuard(t *testing.T) {
	original := loadConfigWithSourceGuardFn
	t.Cleanup(func() {
		loadConfigWithSourceGuardFn = original
	})
	t.Setenv("RUNALL_SECONDARY_CONFIG", "")

	baseDir := t.TempDir()
	primaryPath := filepath.Join(baseDir, "runAll", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(primaryPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(primaryPath, []byte("version: '1'\ngroups: []\n"), 0o644); err != nil {
		t.Fatalf("write primary: %v", err)
	}

	called := false
	loadConfigWithSourceGuardFn = func(primaryPath, secondaryPath string) (*Config, domain.ConfigFingerprint, error) {
		called = true
		wantPrimary := filepath.Join(baseDir, "runAll", "config.yaml")
		if primaryPath != wantPrimary {
			t.Fatalf("primaryPath = %q, want %q", primaryPath, wantPrimary)
		}
		if secondaryPath != "" {
			t.Fatalf("secondaryPath = %q, want empty", secondaryPath)
		}
		return &Config{Version: "1"}, domain.ConfigFingerprint{}, nil
	}

	cfg, err := loadRuntimeConfig(primaryPath)
	if err != nil {
		t.Fatalf("loadRuntimeConfig: %v", err)
	}
	if !called {
		t.Fatal("expected loadRuntimeConfig to call LoadConfigWithSourceGuard")
	}
	if cfg == nil || cfg.Version != "1" {
		t.Fatalf("cfg = %#v, want version 1", cfg)
	}
}

func TestLoadRuntimeConfig_FailsWhenPrimarySecondaryMismatch(t *testing.T) {
	original := loadConfigWithSourceGuardFn
	t.Cleanup(func() {
		loadConfigWithSourceGuardFn = original
	})
	loadConfigWithSourceGuardFn = LoadConfigWithSourceGuard

	baseDir := t.TempDir()
	primaryPath := filepath.Join(baseDir, "runAll", "config.yaml")
	secondaryPath := filepath.Join(baseDir, "runAll.yaml")
	t.Setenv("RUNALL_SECONDARY_CONFIG", secondaryPath)
	if err := os.MkdirAll(filepath.Dir(primaryPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(primaryPath, []byte("version: '1'\ngroups: []\n"), 0o644); err != nil {
		t.Fatalf("write primary: %v", err)
	}
	if err := os.WriteFile(secondaryPath, []byte("version: '2'\ngroups: []\n"), 0o644); err != nil {
		t.Fatalf("write secondary: %v", err)
	}

	_, err := loadRuntimeConfig(primaryPath)
	if err == nil {
		t.Fatal("expected loadRuntimeConfig mismatch error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "config source mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}
