package domain

import (
	"path/filepath"
	"strings"
)

// ServiceLogPathResolution encapsulates the three-level priority for resolving
// a service's log file path:
//
//	1. RUNALL_LOG_ROOT env var (global override)
//	2. conf/<app>/config.yaml → logging.log_file (per-service)
//	3. conf/runAll.yaml → logging.file_root + <service_name>.log (global fallback)
type ServiceLogPathResolution struct {
	ServiceName    string
	EnvOverride    string // RUNALL_LOG_ROOT, empty if not set
	ConfAppLogFile string // from conf_app config, empty if not configured
	GlobalFileRoot string // from runAll.yaml logging.file_root
}

// Resolve returns the final log file path for the service.
// Priority: EnvOverride > ConfAppLogFile > GlobalFileRoot/<service_name>.log
func (r ServiceLogPathResolution) Resolve() LogFilePath {
	// Priority 1: RUNALL_LOG_ROOT env var
	if env := strings.TrimSpace(r.EnvOverride); env != "" {
		return LogFilePath{Value: filepath.Join(env, r.ServiceName+".log")}
	}

	// Priority 2: per-service conf_app logging.log_file
	if cf := strings.TrimSpace(r.ConfAppLogFile); cf != "" {
		return LogFilePath{Value: cf}
	}

	// Priority 3: global file_root fallback
	root := strings.TrimSpace(r.GlobalFileRoot)
	if root == "" {
		root = "logs"
	}
	return LogFilePath{Value: filepath.Join(root, r.ServiceName+".log")}
}
