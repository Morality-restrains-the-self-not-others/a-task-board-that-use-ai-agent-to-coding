package main

import "syscall"

// managedServiceSysProcAttr isolates a managed service from the runAll
// session so orchestrator death cannot SIGHUP the stack (ADR-0035).
// Setsid alone is used: the child becomes session leader (pgid==pid), so
// stopProcess can still syscall.Kill(-pid). Combining Setpgid after Setsid
// fails with EPERM ("cannot change the process group ID of a session leader").
// Setsid does not prevent SIGPIPE: long-lived services must inherit a log
// file FD (see attachManagedServiceStdio), not a parent-owned stdout pipe.
func managedServiceSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
