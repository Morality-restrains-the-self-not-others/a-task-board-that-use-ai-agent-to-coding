package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"runAll/src/domain"
)

// prepareDeploySourceArtifactsFn is replaced in tests.
var prepareDeploySourceArtifactsFn = prepareDeploySourceArtifacts

// restartServiceTestHook, when set, is invoked at the start of restartService (tests).
var restartServiceTestHook func(name string)

func sourceRootEnv() string {
	return resolveSourceRoot("")
}

func deployRootEnv() string {
	return strings.TrimSpace(os.Getenv("DEPLOY_ROOT"))
}

func requireSourceRoot() (string, error) {
	root := sourceRootEnv()
	if root == "" {
		return "", fmt.Errorf("SOURCE_ROOT is required when DEPLOY_MODE is set")
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("SOURCE_ROOT %q is not a directory: %v", root, err)
	}
	return root, nil
}

func requireDeployRoot() (string, error) {
	root := deployRootEnv()
	if root == "" {
		return "", fmt.Errorf("DEPLOY_ROOT is required when DEPLOY_MODE is set")
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return "", fmt.Errorf("DEPLOY_ROOT %q is not a directory: %v", root, err)
	}
	return root, nil
}

func skipOrchestratorRestart(svc *Service) bool {
	if svc == nil {
		return false
	}
	n := strings.ToLower(strings.TrimSpace(svc.Name))
	return n == "runall" || n == "run-all"
}

func envForSourceCompile(sourceRoot string) []string {
	skip := map[string]bool{
		"DEPLOY_MODE": true,
		"META_ROOT":   true,
		"SOURCE_ROOT": true,
	}
	out := make([]string, 0, 32)
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if skip[key] {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "META_ROOT="+sourceRoot, "SOURCE_ROOT="+sourceRoot)
	return out
}

func runSourceCompile(ctx context.Context, sourceRoot string, names []string, all bool) error {
	script := filepath.Join(sourceRoot, "scripts", "precise-compile.sh")
	if st, err := os.Stat(script); err != nil || st.IsDir() {
		return fmt.Errorf("missing precise-compile.sh under SOURCE_ROOT: %v", err)
	}
	args := make([]string, 0, 2+len(names))
	if all {
		args = append(args, "--all")
	} else {
		args = append(args, names...)
	}
	cmd := exec.CommandContext(ctx, script, args...)
	cmd.Dir = sourceRoot
	cmd.Env = envForSourceCompile(sourceRoot)
	log.Printf("[deploy-source] compile start source_root=%s all=%v names=%v", sourceRoot, all, names)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[deploy-source] compile failed: %v", err)
		return fmt.Errorf("source compile: %w: %s", err, bytes.TrimSpace(out))
	}
	log.Printf("[deploy-source] compile ok source_root=%s", sourceRoot)
	return nil
}

func rsyncConfLocal(ctx context.Context, sourceRoot, deployRoot string) error {
	srcDir := filepath.Join(sourceRoot, "conf-local")
	st, err := os.Stat(srcDir)
	if err != nil || !st.IsDir() {
		return fmt.Errorf("SOURCE_ROOT conf-local missing: %v", err)
	}
	destDir := filepath.Join(deployRoot, "conf-local")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir deploy conf-local: %w", err)
	}
	// 原子替换（OPT-20260902-012）：先 rsync 到同父目录的 staging，成功后再
	// rename 交换。rsync 中途失败只留下 staging（defer 清理），dest 树保持失败前
	// 一致，避免 `rsync -a --delete` 直接打目标中断造成的半写入。
	staging := filepath.Join(deployRoot, fmt.Sprintf("conf-local.incoming.%d", os.Getpid()))
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("clean conf-local staging: %w", err)
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return fmt.Errorf("mkdir conf-local staging: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()
	src := srcDir + string(os.PathSeparator)
	stagingDest := staging + string(os.PathSeparator)
	cmd := exec.CommandContext(ctx, "rsync", "-a", "--delete", src, stagingDest)
	log.Printf("[deploy-source] rsync conf-local start staging=%s", staging)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[deploy-source] rsync conf-local failed: %v", err)
		return fmt.Errorf("rsync conf-local: %w: %s", err, bytes.TrimSpace(out))
	}
	if err := swapConfLocalDir(staging, destDir); err != nil {
		log.Printf("[deploy-source] swap conf-local failed: %v", err)
		return fmt.Errorf("swap conf-local: %w", err)
	}
	log.Printf("[deploy-source] rsync conf-local ok")
	return nil
}

// swapConfLocalDir 用 rename 把已完整同步的 staging 交换为 destDir。
// destDir 先改名成备份再让位，交换成功后再删除备份；任一步失败回滚旧树。
func swapConfLocalDir(staging, destDir string) error {
	parent := filepath.Dir(destDir)
	backup := filepath.Join(parent, fmt.Sprintf("conf-local.old.%d", os.Getpid()))
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	hasOld := true
	if err := os.Rename(destDir, backup); err != nil {
		if os.IsNotExist(err) {
			hasOld = false
		} else {
			return err
		}
	}
	if err := os.Rename(staging, destDir); err != nil {
		if hasOld {
			_ = os.Rename(backup, destDir) // 回滚旧树
		}
		return err
	}
	if hasOld {
		return os.RemoveAll(backup)
	}
	return nil
}

func findInstallLocalArtifactsScript(sourceRoot, deployRoot string) (string, error) {
	candidates := []string{
		filepath.Join(sourceRoot, "runAll", "scripts", "install-local-artifacts.sh"),
		filepath.Join(sourceRoot, "scripts", "install-local-artifacts.sh"),
		filepath.Join(deployRoot, "scripts", "install-local-artifacts.sh"),
		filepath.Join(deployRoot, "runAll", "scripts", "install-local-artifacts.sh"),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("install-local-artifacts.sh not found under SOURCE_ROOT or DEPLOY_ROOT")
}

func installArtifacts(ctx context.Context, sourceRoot, deployRoot string) error {
	script, err := findInstallLocalArtifactsScript(sourceRoot, deployRoot)
	if err != nil {
		return err
	}
	src := filepath.Join(sourceRoot, "deploy-binaries")
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		return fmt.Errorf("SOURCE_ROOT deploy-binaries missing: %v", err)
	}
	cmd := exec.CommandContext(ctx, "bash", script, src, deployRoot)
	log.Printf("[deploy-source] install artifacts start")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[deploy-source] install artifacts failed: %v", err)
		return fmt.Errorf("install-local-artifacts: %w: %s", err, bytes.TrimSpace(out))
	}
	log.Printf("[deploy-source] install artifacts ok")
	return nil
}

func prepareDeploySourceArtifacts(ctx context.Context, names []string, all bool) error {
	sourceRoot, err := requireSourceRoot()
	if err != nil {
		return err
	}
	deployRoot, err := requireDeployRoot()
	if err != nil {
		return err
	}
	if err := runSourceCompile(ctx, sourceRoot, names, all); err != nil {
		return err
	}
	if err := rsyncConfLocal(ctx, sourceRoot, deployRoot); err != nil {
		return err
	}
	return installArtifacts(ctx, sourceRoot, deployRoot)
}

func (r *Runner) publishBuildProgress(runID string, ev StartAllProgressEvent) {
	if r == nil || r.progressBroadcaster == nil || runID == "" {
		return
	}
	r.progressBroadcaster.Publish(runID, ev)
}

func (r *Runner) buildAllFromSource(ctx context.Context) (*domain.BuildGroupResult, error) {
	allSvcs := r.cfg.Flatten()
	if len(allSvcs) == 0 {
		return nil, fmt.Errorf("no services configured")
	}

	r.activeBuildAllRunIDMu.Lock()
	runID := r.activeBuildAllRunID
	if runID == "" {
		runID = fmt.Sprintf("build-all-%d", time.Now().UnixNano())
		r.activeBuildAllRunID = runID
	}
	r.activeBuildAllRunIDMu.Unlock()
	defer func() {
		r.activeBuildAllRunIDMu.Lock()
		if r.activeBuildAllRunID == runID {
			r.activeBuildAllRunID = ""
		}
		r.activeBuildAllRunIDMu.Unlock()
	}()

	r.publishBuildProgress(runID, StartAllProgressEvent{
		Total: 1, Remaining: 1, Phase: "starting", Operation: "build", Current: "source-compile",
	})
	if err := prepareDeploySourceArtifactsFn(ctx, nil, true); err != nil {
		log.Printf("[build-all] source compile/install failed: %v", err)
		r.publishBuildProgress(runID, StartAllProgressEvent{
			Total: 1, Failed: 1, Remaining: 0, Done: true, Phase: "error",
			Operation: "build", Error: err.Error(), Errors: []string{err.Error()},
		})
		if r.progressBroadcaster != nil {
			go func() {
				time.Sleep(30 * time.Second)
				r.progressBroadcaster.CloseRun(runID)
			}()
		}
		return nil, err
	}

	if cerr := clearPreciseRestartRegistrationsAfterFullRebuild(r.cfgPath); cerr != nil {
		log.Printf("[build-all] clear precise-restart registrations: %v", cerr)
	} else {
		log.Printf("[build-all] precise-restart registrations cleared")
	}

	r.publishBuildProgress(runID, StartAllProgressEvent{
		Total: 1, Started: 1, Remaining: 0, Done: true, Phase: "done", Operation: "build",
	})
	if r.progressBroadcaster != nil {
		go func() {
			time.Sleep(30 * time.Second)
			r.progressBroadcaster.CloseRun(runID)
		}()
	}
	status := domain.ComputeBuildGroupStatus(1, 1, nil)
	result, err := domain.NewBuildGroupResult(status, 1, 1, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("build all from source: %w", err)
	}
	return &result, nil
}

func shouldSyncMonorepoConf() bool {
	return !deployModeActive()
}
