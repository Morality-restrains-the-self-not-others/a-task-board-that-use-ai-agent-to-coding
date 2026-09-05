package main

import (
	"log"
	"net/http"
)

const (
	errCodeDescendantsNotTerminal = "DESCENDANTS_NOT_TERMINAL"
	errMsgDescendantsNotTerminal  = "存在未完成或未取消的子任务，无法关闭当前任务"
)

type openDescendant struct {
	ID                 string
	Title              string
	Depth              int
	ProgressColumnName string
}

// findOpenDescendants returns unsettled descendants at any depth.
func findOpenDescendants(tenantID, workspaceID, rootID string) ([]openDescendant, error) {
	nodes, err := listDescendantsBFS(tenantID, workspaceID, rootID, 0)
	if err != nil {
		return nil, err
	}
	nodes = enrichSubtreeNodes(tenantID, workspaceID, nodes)
	var open []openDescendant
	for _, n := range nodes {
		if n.Settled {
			continue
		}
		open = append(open, openDescendant{
			ID:                 n.ID,
			Title:              n.Title,
			Depth:              n.Depth,
			ProgressColumnName: n.ProgressColumnName,
		})
	}
	return open, nil
}

func writeDescendantsNotTerminal(w http.ResponseWriter, open []openDescendant) {
	arr := make([]map[string]interface{}, 0, len(open))
	for _, o := range open {
		arr = append(arr, map[string]interface{}{
			"id":                   o.ID,
			"title":                o.Title,
			"depth":                o.Depth,
			"progress_column_name": o.ProgressColumnName,
		})
	}
	writeErrorMap(w, nil, http.StatusConflict, map[string]interface{}{
		"error":            errMsgDescendantsNotTerminal,
		"detail":           errMsgDescendantsNotTerminal,
		"code":             errCodeDescendantsNotTerminal,
		"open_descendants": arr,
	})
}

// enforceDescendantTerminalGate blocks entering a terminal state when open descendants exist.
// Returns true if the handler should abort (response already written).
func enforceDescendantTerminalGate(w http.ResponseWriter, tenantID, workspaceID, taskID, newColumnName string, prevCompleted, completed bool) bool {
	completedBecameTrue := !prevCompleted && completed
	kind := resolveTerminalKind(newColumnName, completedBecameTrue)
	if kind == "" {
		return false
	}
	// Leaving a non-terminal column for a terminal one, or flipping completed → true.
	open, err := findOpenDescendants(tenantID, workspaceID, taskID)
	if err != nil {
		log.Printf("[taskTaskService] event=descendant_gate_error task_id=%s err=%v", taskID, err)
		writeError(w, nil, http.StatusInternalServerError, err.Error())
		return true
	}
	if len(open) == 0 {
		return false
	}
	log.Printf("[taskTaskService] event=descendant_gate_blocked task_id=%s terminal_kind=%s open_count=%d", taskID, kind, len(open))
	writeDescendantsNotTerminal(w, open)
	return true
}
