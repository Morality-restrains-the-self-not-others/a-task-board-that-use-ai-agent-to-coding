package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"runAll/src/domain"
)

var deployLayoutLogged sync.Map

func logDeployLayoutOnce(svcName, detail string) {
	key := svcName + "\x00" + detail
	if _, loaded := deployLayoutLogged.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	log.Printf("[runAll] deploy layout: %s %s", svcName, detail)
}

func isDeployFlatBinStart(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	return strings.HasPrefix(cmd, "./bin/") || strings.HasPrefix(cmd, "bin/")
}

// sourceLessFlatBinRoot is a clone-run tree: ELF live in bin/ and there is no
// taskAuth/ source checkout. Last-resort layout rewrite when cutover.env was
// not loaded (tests, or a bare binary with a missing env file). Production
// clone-run must source cutover.env via run.sh / LoadConfig.
func sourceLessFlatBinRoot(root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "runAll")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "taskAuth")); err == nil {
		return false
	}
	return true
}

func deployLayoutActive(projectRoot string) bool {
	if deployModeActive() {
		return true
	}
	return sourceLessFlatBinRoot(projectRoot)
}

func rewriteRelativeStopForDeploy(svc *Service) {
	stop := strings.TrimSpace(svc.StopCommand)
	if !strings.Contains(stop, "scripts/runall-stop.sh") {
		return
	}
	port := domain.ResolveHealthPort(svc.HealthCheck.URL)
	if port == "" {
		port = domain.ResolveHealthPort(svc.HealthCheck.DisplayEndpoint())
	}
	if port == "" {
		return
	}
	svc.StopCommand = "bash -c 'lsof -ti:" + port + " | xargs kill -9 2>/dev/null || true'"
	logDeployLayoutOnce(svc.Name, "stop_command → lsof :"+port+" (no service scripts/)")
}

// applyDeployLayout rewrites ELF starts so a source-less $DEPLOY_ROOT/bin works.
// Recipe starts (dockerInfra/run.sh, taskEvents run.sh) keep their working_dir.
func (c *Config) applyDeployLayout(projectRoot string) {
	if !deployLayoutActive(projectRoot) {
		return
	}
	if !deployModeActive() && sourceLessFlatBinRoot(projectRoot) {
		logDeployLayoutOnce("_root", "inferred source-less "+projectRoot+" (bin/runAll present, taskAuth/ absent)")
	}
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			start := strings.TrimSpace(svc.EffectiveStartCommand())
			if !isDeployFlatBinStart(start) {
				continue
			}
			if svc.WorkingDir != "." && svc.WorkingDir != "" && !filepath.IsAbs(svc.WorkingDir) {
				logDeployLayoutOnce(svc.Name, "working_dir "+svc.WorkingDir+" → . (flat bin)")
				svc.WorkingDir = "."
			}
			rewriteRelativeStopForDeploy(svc)
			if strings.Contains(start, "../conf/") {
				rewritten := strings.ReplaceAll(start, "../conf/", "conf/")
				if svc.StartCommand != "" {
					svc.StartCommand = rewritten
				} else {
					svc.Command = rewritten
				}
				logDeployLayoutOnce(svc.Name, "conf path rewritten for deploy root")
			}
		}
	}
}

func (c *Config) resolveWorkingDirs(configDir string) {
	// Project root is the parent of the config directory (e.g. conf/runAll.yaml → repo root).
	// Service commands use paths like runAll/scripts/... and AiMonitor/run.sh relative to it.
	projectRoot := filepath.Dir(configDir)
	c.applyDeployLayout(projectRoot)

	cwd, err := os.Getwd()
	if err != nil {
		cwd = projectRoot
	}
	for gi := range c.Groups {
		for si := range c.Groups[gi].Services {
			svc := &c.Groups[gi].Services[si]
			if svc.WorkingDir != "" && !filepath.IsAbs(svc.WorkingDir) {
				// "." always means monorepo root, even when runAll is launched from runAll/.
				if svc.WorkingDir == "." {
					svc.WorkingDir = projectRoot
				} else {
					resolved := filepath.Join(cwd, svc.WorkingDir)
					// Fallback: if cwd-resolved path does not exist, try relative to project root.
					if _, err := os.Stat(resolved); os.IsNotExist(err) {
						fallback := filepath.Join(projectRoot, svc.WorkingDir)
						if _, err := os.Stat(fallback); err == nil {
							resolved = fallback
						}
					}
					svc.WorkingDir = resolved
				}
			}
			// Exec health probes must use the same cwd as start_command / stop_command.
			svc.HealthCheck.WorkDir = svc.WorkingDir
		}
	}
}
