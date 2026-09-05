package main

import "strings"

// Cloud ECS / mock runtime statuses used by work-panel started/idle accounting.
const (
	machineRuntimeRunning      = "Running"
	machineRuntimeStarting     = "Starting"
	machineRuntimePending      = "Pending"
	machineRuntimeInitializing = "Initializing"
	machineRuntimeStopped      = "Stopped"
	machineRuntimeStopping     = "Stopping"
	machineRuntimeReleased     = "Released"
	machineRuntimeFailed       = "Failed"
)

func normalizeMachineRuntimeStatus(status string) string {
	return strings.TrimSpace(status)
}

func isMockMachineInstanceID(instanceID string) bool {
	return strings.HasPrefix(strings.TrimSpace(instanceID), "mock-")
}

// machineRuntimeCountsAsStarted: work-panel「已启动」— Running / mock, or legacy empty status.
// Explicit Starting / Stopped / Released must NOT count.
func machineRuntimeCountsAsStarted(instanceID, runtimeStatus string) bool {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return false
	}
	if isMockMachineInstanceID(instanceID) {
		return true
	}
	status := normalizeMachineRuntimeStatus(runtimeStatus)
	if status == "" {
		// Legacy rows before last_runtime_status was populated; reconcile fills this soon.
		return true
	}
	if machineRuntimeCountsAsStarting(instanceID, status) {
		return false
	}
	if machineRuntimeIsDown(status) || strings.EqualFold(status, machineRuntimeReleased) ||
		strings.EqualFold(status, "Terminated") {
		return false
	}
	return strings.EqualFold(status, machineRuntimeRunning)
}

// machineRuntimeCountsAsStarting: transitional cloud statuses — not yet 「已启动」.
func machineRuntimeCountsAsStarting(instanceID, runtimeStatus string) bool {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" || isMockMachineInstanceID(instanceID) {
		return false
	}
	switch strings.ToLower(normalizeMachineRuntimeStatus(runtimeStatus)) {
	case "starting", "pending", "initializing":
		return true
	default:
		return false
	}
}

// machineRuntimeIsReleasedOrMissing: instance gone from cloud — local binding must be cleared.
func machineRuntimeIsReleasedOrMissing(runtimeStatus string, found bool) bool {
	if !found {
		return true
	}
	switch strings.ToLower(normalizeMachineRuntimeStatus(runtimeStatus)) {
	case "released", "terminated":
		return true
	default:
		return false
	}
}

// machineRuntimeIsDown: stopped / stopping — not started for summary/filter.
func machineRuntimeIsDown(runtimeStatus string) bool {
	switch strings.ToLower(normalizeMachineRuntimeStatus(runtimeStatus)) {
	case "stopped", "stopping", "shutting-down", "shuttingdown":
		return true
	default:
		return false
	}
}
