package confload

import (
	"os"
	"path/filepath"
	"testing"
)

func unsetDeliveryRootEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CONF_ROOT", "")
	t.Setenv("DEPLOY_ROOT", "")
}

func writeBaseYAML(t *testing.T, confDir string) {
	t.Helper()
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "base.yaml"), []byte("scheme: https\nbaseDomain: example.com\n"), 0o644); err != nil {
		t.Fatalf("write base.yaml: %v", err)
	}
}

func TestFindConfigRootFromCONF_ROOTConfDirWithoutGitmodules(t *testing.T) {
	unsetDeliveryRootEnv(t)
	deploy := t.TempDir()
	confDir := filepath.Join(deploy, "conf")
	writeBaseYAML(t, confDir)

	t.Setenv("CONF_ROOT", confDir)
	t.Setenv("DEPLOY_ROOT", "")

	// cwd is unrelated and has no markers
	unrelated := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(unrelated); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	root, err := FindConfigRoot()
	if err != nil {
		t.Fatalf("FindConfigRoot: %v", err)
	}
	want, err := filepath.EvalSymlinks(deploy)
	if err != nil {
		want = deploy
	}
	got, err := filepath.EvalSymlinks(root)
	if err != nil {
		got = root
	}
	if got != want {
		t.Fatalf("want deploy root %q, got %q", want, got)
	}
}

func TestFindConfigRootDEPLOY_ROOTPreferredOverCwdWalk(t *testing.T) {
	unsetDeliveryRootEnv(t)

	walkRoot := t.TempDir()
	writeBaseYAML(t, filepath.Join(walkRoot, "conf"))
	sub := filepath.Join(walkRoot, "bin")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	deploy := t.TempDir()
	writeBaseYAML(t, filepath.Join(deploy, "conf"))
	t.Setenv("DEPLOY_ROOT", deploy)
	t.Setenv("CONF_ROOT", "")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	root, err := FindConfigRoot()
	if err != nil {
		t.Fatalf("FindConfigRoot: %v", err)
	}
	want, _ := filepath.EvalSymlinks(deploy)
	got, _ := filepath.EvalSymlinks(root)
	if got != want {
		t.Fatalf("DEPLOY_ROOT must win over cwd walk: want %q got %q", want, got)
	}
}

func TestFindMonorepoRootDelegatesToFindConfigRoot(t *testing.T) {
	unsetDeliveryRootEnv(t)
	deploy := t.TempDir()
	confDir := filepath.Join(deploy, "conf")
	writeBaseYAML(t, confDir)
	t.Setenv("CONF_ROOT", confDir)

	viaConfig, err := FindConfigRoot()
	if err != nil {
		t.Fatal(err)
	}
	viaLegacy, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if viaConfig != viaLegacy {
		t.Fatalf("FindMonorepoRoot must equal FindConfigRoot: %q vs %q", viaLegacy, viaConfig)
	}
}
