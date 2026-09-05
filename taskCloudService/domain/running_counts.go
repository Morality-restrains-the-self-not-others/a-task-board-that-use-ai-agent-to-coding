package domain

import "strings"

// CommentRuntime is a comment-scoped CSC already classified by machine/container predicates.
type CommentRuntime struct {
	CommentID          string
	MachineStarted     bool
	ContainerReachable bool
}

// RunningCounts is the task-level projection: how many comment machines/containers are running.
type RunningCounts struct {
	Machines   int
	Containers int
}

// ComputeRunningCounts counts comment-level rows only. Empty CommentID rows are ignored.
func ComputeRunningCounts(rows []CommentRuntime) RunningCounts {
	var out RunningCounts
	for _, row := range rows {
		if strings.TrimSpace(row.CommentID) == "" {
			continue
		}
		if row.MachineStarted {
			out.Machines++
		}
		if row.ContainerReachable {
			out.Containers++
		}
	}
	return out
}
