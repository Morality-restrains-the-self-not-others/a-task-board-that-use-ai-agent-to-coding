package domain

import "errors"

var (
	ErrLifecycleStartCommandRequired = errors.New("start_command is required")
	ErrLifecycleStopCommandRequired  = errors.New("stop_command is required")
	ErrLifecycleInvalidLaunchMode    = errors.New("launch_mode must be attach or detach")
)
