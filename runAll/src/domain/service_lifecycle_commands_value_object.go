package domain

import "strings"

// ServiceLifecycleCommands captures explicit shell commands for runAll UI actions.
type ServiceLifecycleCommands struct {
	StartCommand string
	StopCommand  string
	BuildCommand string
	LaunchMode   string // attach | detach
}

func NewServiceLifecycleCommands(start, stop, build, launchMode string) (ServiceLifecycleCommands, error) {
	start = strings.TrimSpace(start)
	stop = strings.TrimSpace(stop)
	if start == "" {
		return ServiceLifecycleCommands{}, ErrLifecycleStartCommandRequired
	}
	if stop == "" {
		return ServiceLifecycleCommands{}, ErrLifecycleStopCommandRequired
	}
	mode := strings.TrimSpace(launchMode)
	if mode == "" {
		mode = "attach"
	}
	if mode != "attach" && mode != "detach" {
		return ServiceLifecycleCommands{}, ErrLifecycleInvalidLaunchMode
	}
	return ServiceLifecycleCommands{
		StartCommand: start,
		StopCommand:  stop,
		BuildCommand: strings.TrimSpace(build),
		LaunchMode:   mode,
	}, nil
}

func (c ServiceLifecycleCommands) IsDetachLaunch() bool {
	return c.LaunchMode == "detach"
}

func (c ServiceLifecycleCommands) HasBuild() bool {
	return strings.TrimSpace(c.BuildCommand) != ""
}
