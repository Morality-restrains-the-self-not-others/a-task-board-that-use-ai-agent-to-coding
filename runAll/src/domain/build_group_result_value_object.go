package domain

import (
	"fmt"
)

// BuildGroupResult captures the outcome of a group-wide build operation.
// Each field documents which services were built, skipped, or failed.
type BuildGroupResult struct {
	// Status summarizes the overall result: "ok" (all succeeded),
	// "partial" (some failed), or "none" (nothing was built).
	Status string `json:"status"`

	// Total is the count of buildable services that were eligible for building.
	Total int `json:"total"`

	// Built is the count of services that built successfully.
	Built int `json:"built"`

	// Failed lists the names of services whose build failed.
	Failed []string `json:"failed,omitempty"`

	// Skipped lists the names of services that were skipped because their
	// current status does not allow building (e.g. starting, building).
	Skipped []string `json:"skipped,omitempty"`

	// NoBuild lists the names of services that have no build command configured.
	NoBuild []string `json:"no_build,omitempty"`
}

const (
	BuildGroupStatusOK      = "ok"
	BuildGroupStatusPartial = "partial"
	BuildGroupStatusNone    = "none"
)

// NewBuildGroupResult creates a validated BuildGroupResult.
func NewBuildGroupResult(
	status string,
	total int,
	built int,
	failed []string,
	skipped []string,
	noBuild []string,
) (BuildGroupResult, error) {
	if status != BuildGroupStatusOK && status != BuildGroupStatusPartial && status != BuildGroupStatusNone {
		return BuildGroupResult{}, fmt.Errorf("build group status must be ok, partial, or none, got %q", status)
	}
	if total < 0 {
		return BuildGroupResult{}, fmt.Errorf("total must be >= 0")
	}
	if built < 0 || built > total {
		return BuildGroupResult{}, fmt.Errorf("built must be between 0 and total")
	}
	return BuildGroupResult{
		Status:  status,
		Total:   total,
		Built:   built,
		Failed:  append([]string(nil), failed...),
		Skipped: append([]string(nil), skipped...),
		NoBuild: append([]string(nil), noBuild...),
	}, nil
}

// ComputeBuildGroupStatus derives the status from counts.
func ComputeBuildGroupStatus(total, built int, failed []string) string {
	if total == 0 {
		return BuildGroupStatusNone
	}
	if built == total && len(failed) == 0 {
		return BuildGroupStatusOK
	}
	return BuildGroupStatusPartial
}

// IsTerminalBuildStatus reports whether a service status allows building.
// Only services actively building cannot start another build (concurrent build
// prevention); all other statuses are buildable because compilation is a
// disk-only operation independent of runtime state.
func IsTerminalBuildStatus(status string) bool {
	return status != ServiceStatusBuilding
}
