package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"runAll/src/domain"
	"runAll/src/infrastructure"
)

func (r *Runner) configFileUnchanged() bool {
	if r == nil || strings.TrimSpace(r.cfgPath) == "" {
		return false
	}
	st, err := os.Stat(r.cfgPath)
	if err != nil {
		return false
	}
	r.cfgReloadMu.Lock()
	defer r.cfgReloadMu.Unlock()
	if r.cfgReloadSize == 0 && r.cfgReloadModTime.IsZero() {
		return false
	}
	return st.Size() == r.cfgReloadSize && st.ModTime().Equal(r.cfgReloadModTime)
}

func (r *Runner) rememberConfigFileStamp() {
	if r == nil || strings.TrimSpace(r.cfgPath) == "" {
		return
	}
	st, err := os.Stat(r.cfgPath)
	if err != nil {
		return
	}
	r.cfgReloadMu.Lock()
	r.cfgReloadSize = st.Size()
	r.cfgReloadModTime = st.ModTime()
	r.cfgReloadMu.Unlock()
}

// reloadConfigFromDisk 从 r.cfgPath 重新加载 runAll.yaml，使 YAML 新增服务
// 可被精准编译重启解析。cfgPath 为空时 no-op（单测内存配置）。
// 文件 mtime+size 未变时短路，避免 /api/status 轮询反复 applyDeployLayout。
// 加载或 DAG 失败时返回 error，调用方必须保留原内存配置。
func (r *Runner) reloadConfigFromDisk() ([]string, error) {
	if r == nil || strings.TrimSpace(r.cfgPath) == "" {
		return nil, nil
	}
	if r.configFileUnchanged() {
		return nil, nil
	}
	fresh, err := LoadConfig(r.cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config %s: %w", r.cfgPath, err)
	}
	levels, err := BuildDAG(fresh.Flatten())
	if err != nil {
		return nil, fmt.Errorf("build DAG from reloaded config: %w", err)
	}

	oldNames := make(map[string]struct{})
	if r.cfg != nil {
		for _, svc := range r.cfg.Flatten() {
			if n := strings.TrimSpace(svc.Name); n != "" {
				oldNames[n] = struct{}{}
			}
		}
	}
	var added []string
	for _, svc := range fresh.Flatten() {
		n := strings.TrimSpace(svc.Name)
		if n == "" {
			continue
		}
		if _, ok := oldNames[n]; !ok {
			added = append(added, n)
		}
	}
	sort.Strings(added)

	r.cfg = fresh
	r.levels = levels
	r.adoptReloadedConfig(fresh)
	r.rememberConfigFileStamp()
	return added, nil
}

// adoptReloadedConfig 把热加载配置同步到 StatusStore / 日志路径。
// 只 Ensure 缺失服务名，不 Init，以免把已运行服务打回 pending。
func (r *Runner) adoptReloadedConfig(cfg *Config) {
	if r == nil || cfg == nil || r.store == nil {
		return
	}
	services := cfg.Flatten()
	names := make([]string, 0, len(services))
	for _, svc := range services {
		if n := strings.TrimSpace(svc.Name); n != "" {
			names = append(names, n)
		}
	}
	r.store.EnsureNames(names)

	for _, svc := range services {
		cmd := strings.TrimSpace(svc.EffectiveStartCommand())
		r.store.SetCommand(svc.Name, cmd)
		r.store.SetURL(svc.Name, svc.HealthCheck.DisplayEndpoint())
		healthPort := domain.ResolveHealthPort(svc.HealthCheck.URL)
		if healthPort == "" {
			healthPort = domain.ResolveTCPPort(svc.HealthCheck.TCP)
		}
		r.store.SetHealthPort(svc.Name, healthPort)
		r.store.SetCommandPort(svc.Name, domain.ResolveCommandPort(cmd))
		deps := make([]DepStatus, len(svc.DependsOn))
		for i, depName := range svc.DependsOn {
			deps[i] = DepStatus{Name: depName, Status: StatusPending}
		}
		r.store.SetDependsOn(svc.Name, deps)
	}
	for _, group := range cfg.Groups {
		for _, svc := range group.Services {
			r.store.SetGroup(svc.Name, group.Name)
		}
	}

	if fileSink, ok := r.fileLogSink.(*infrastructure.FileServiceLogSink); ok {
		for _, svc := range services {
			logFile := resolveServiceLogFile(cfg, svc.Name, svc.LogFile)
			if logFile != "" {
				fileSink.SetServicePath(svc.Name, logFile)
			}
		}
	}
}

func logPreciseRestartReload(added []string, err error) {
	if err != nil {
		log.Printf("[precise-restart] reload runAll config from disk failed (keeping in-memory config): %v", err)
		return
	}
	if len(added) > 0 {
		log.Printf("[precise-restart] reloaded runAll config from disk; new services: %v", added)
	}
}

// reloadRunAllConfigBestEffort 重新加载 runAll.yaml，使进程启动后新增到 YAML 的服务
// 可被 start-all / start-group / 单服务 start 解析。失败保留内存配置（与 PreciseRestart 相同）。
func (r *Runner) reloadRunAllConfigBestEffort() {
	if r == nil {
		return
	}
	added, rerr := r.reloadConfigFromDisk()
	logPreciseRestartReload(added, rerr)
}

// ensureConfigHasServiceBestEffort 单服务 start 前按需重载：仅当服务不在内存配置中才读盘，
// 避免每次启动都做磁盘 I/O + DAG 重建。
func (r *Runner) ensureConfigHasServiceBestEffort(name string) {
	if r == nil || r.cfg == nil {
		return
	}
	if len(r.resolveRegisteredServices(name)) > 0 {
		return
	}
	r.reloadRunAllConfigBestEffort()
}
