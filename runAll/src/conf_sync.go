package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

const confSyncLifecycleLabel = "conf-sync"

// SyncMonorepoConf runs runAll/scripts/conf-sync-all.sh to generate conf replica artifacts (local).
func (r *Runner) SyncMonorepoConf(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("runner is required")
	}
	if r.devToolLogRecorder != nil {
		r.devToolLogRecorder.Append(domain.ToolConfSync, "开始生成配置副本...")
	}
	root, err := r.monorepoRoot()
	if err != nil {
		if r.devToolLogRecorder != nil {
			r.devToolLogRecorder.Append(domain.ToolConfSync, fmt.Sprintf("失败: %v", err))
		}
		return err
	}
	script := filepath.Join(root, "runAll", "scripts", "conf-sync-all.sh")
	if _, err := os.Stat(script); err != nil {
		if r.devToolLogRecorder != nil {
			r.devToolLogRecorder.Append(domain.ToolConfSync, fmt.Sprintf("失败: %v", err))
		}
		return fmt.Errorf("conf sync script not found at %s: %w", script, err)
	}
	dummy := Service{Name: confSyncLifecycleLabel, WorkingDir: root}
	err = r.runLifecycleCommand(ctx, &dummy, "bash runAll/scripts/conf-sync-all.sh")
	if r.devToolLogRecorder != nil {
		if err != nil {
			r.devToolLogRecorder.Append(domain.ToolConfSync, fmt.Sprintf("失败: %v", err))
		} else {
			r.devToolLogRecorder.Append(domain.ToolConfSync, "配置副本生成完成")
		}
	}
	return err
}

func (r *Runner) monorepoRoot() (string, error) {
	for _, svc := range r.cfg.Flatten() {
		if strings.TrimSpace(svc.WorkingDir) == "" {
			continue
		}
		root, err := infrastructure.ResolveMonorepoRoot(svc.WorkingDir)
		if err == nil {
			return root, nil
		}
	}
	return infrastructure.ResolveMonorepoRoot("")
}
