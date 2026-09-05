package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

// OPT-20260901-019: clone-run 时 cwd=envs/current/taskGitOauth，FindMonorepoRoot
// 必须尊重 CONF_ROOT 指向部署根，否则 conf-local 机密漏叠（HTTP 200、密钥空、
// 厂商 AUTH 失败）。不再做 cwd 上溯的 db/registry.yaml / conf/base.yaml 硬走。
func TestFindMonorepoRootHonorsCONF_ROOT(t *testing.T) {
	deploy := t.TempDir()
	confDir := filepath.Join(deploy, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONF_ROOT", confDir)
	t.Setenv("DEPLOY_ROOT", "")

	unrelated := t.TempDir() // 模拟 envs/current/taskGitOauth（无 conf/base.yaml 标记）
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(unrelated); err != nil {
		t.Fatal(err)
	}

	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot: %v", err)
	}
	if root != deploy {
		t.Fatalf("FindMonorepoRoot=%q, want CONF_ROOT deploy %q", root, deploy)
	}
}
