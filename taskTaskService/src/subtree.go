package main

import (
	"database/sql"
	"log"
	"sort"
	"strings"
	"sync"
)

type subtreeNode struct {
	ID                 string
	Title              string
	ParentTaskID       string
	Depth              int
	ProgressColumnID   string
	ProgressColumnName string
	Completed          bool
	TerminalKind       string
	Settled            bool
}

type subtreeSummary struct {
	Total     int `json:"total"`
	Settled   int `json:"settled"`
	Open      int `json:"open"`
	Completed int `json:"completed"`
	Cancelled int `json:"cancelled"`
}

// progressColumnNameByIDFn resolves column id → display name for a workspace.
// Tests may replace this; default uses validateTaskFields per unique id.
var progressColumnNameByIDFn = defaultProgressColumnNames

// progressColumnNameCache avoids repeated HTTP calls to taskProjectService
// for the same column IDs within a single subtree/BFS operation.
var (
	progressColumnNameCache   = map[string]string{} // key: "tenantID/workspaceID/columnID"
	progressColumnNameCacheMu sync.RWMutex
)

func cachedColumnName(tenantID, workspaceID, columnID string) (string, bool) {
	progressColumnNameCacheMu.RLock()
	v, ok := progressColumnNameCache[tenantID+"/"+workspaceID+"/"+columnID]
	progressColumnNameCacheMu.RUnlock()
	return v, ok
}

func cacheColumnName(tenantID, workspaceID, columnID, name string) {
	progressColumnNameCacheMu.Lock()
	progressColumnNameCache[tenantID+"/"+workspaceID+"/"+columnID] = name
	progressColumnNameCacheMu.Unlock()
}

func defaultProgressColumnNames(tenantID, workspaceID string, columnIDs []string) map[string]string {
	out := make(map[string]string, len(columnIDs))
	seen := map[string]struct{}{}
	for _, id := range columnIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}

		// Check local cache first to avoid repeated HTTP calls (OPT-20260719-040).
		if name, ok := cachedColumnName(tenantID, workspaceID, id); ok {
			out[id] = name
			continue
		}

		name, err := validateTaskFields(tenantID, workspaceID, map[string]interface{}{
			"progress_column_id": id,
		})
		if err != nil {
			log.Printf("[taskTaskService] event=progress_column_name_resolve_fail column_id=%s err=%v", id, err)
			continue
		}
		if name != "" {
			out[id] = name
			cacheColumnName(tenantID, workspaceID, id, name)
		}
	}
	return out
}

func listDirectChildren(tenantID, workspaceID, parentID string) ([]taskRecord, error) {
	rows, err := db.Query(`SELECT `+taskSelectCols+` FROM task_tasks WHERE tenant_id=? AND workspace_id=? AND parent_task_id=? ORDER BY order_num, created_at`,
		tenantID, workspaceID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

func scanTaskRows(rows *sql.Rows) ([]taskRecord, error) {
	var out []taskRecord
	for rows.Next() {
		var t taskRecord
		if err := scanTaskValues(rows.Scan, &t); err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// listDescendantsBFS returns all descendants with depth ≥ 1 (root itself excluded).
// If maxDepth > 0, stops after that depth; maxDepth ≤ 0 means unlimited.
func listDescendantsBFS(tenantID, workspaceID, rootID string, maxDepth int) ([]subtreeNode, error) {
	type item struct {
		id    string
		depth int
	}
	queue := []item{{id: rootID, depth: 0}}
	var nodes []subtreeNode
	seen := map[string]struct{}{rootID: {}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if maxDepth > 0 && cur.depth >= maxDepth {
			continue
		}
		children, err := listDirectChildren(tenantID, workspaceID, cur.id)
		if err != nil {
			return nil, err
		}
		for _, ch := range children {
			if _, ok := seen[ch.ID]; ok {
				continue
			}
			seen[ch.ID] = struct{}{}
			d := cur.depth + 1
			nodes = append(nodes, subtreeNode{
				ID:               ch.ID,
				Title:            ch.Title,
				ParentTaskID:     ch.ParentTaskID,
				Depth:            d,
				ProgressColumnID: ch.ProgressColumnID,
				Completed:        ch.Completed,
			})
			queue = append(queue, item{id: ch.ID, depth: d})
		}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Depth != nodes[j].Depth {
			return nodes[i].Depth < nodes[j].Depth
		}
		return nodes[i].ID < nodes[j].ID
	})
	return nodes, nil
}

func enrichSubtreeNodes(tenantID, workspaceID string, nodes []subtreeNode) []subtreeNode {
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ProgressColumnID)
	}
	names := progressColumnNameByIDFn(tenantID, workspaceID, ids)
	for i := range nodes {
		name := names[nodes[i].ProgressColumnID]
		nodes[i].ProgressColumnName = name
		kind := resolveTerminalKind(name, nodes[i].Completed)
		nodes[i].TerminalKind = kind
		nodes[i].Settled = kind != ""
	}
	return nodes
}

func summarizeSubtree(nodes []subtreeNode) subtreeSummary {
	s := subtreeSummary{Total: len(nodes)}
	for _, n := range nodes {
		if !n.Settled {
			s.Open++
			continue
		}
		s.Settled++
		switch n.TerminalKind {
		case "completed":
			s.Completed++
		case "cancelled":
			s.Cancelled++
		}
	}
	return s
}

func subtreeNodeJSON(n subtreeNode) map[string]interface{} {
	parent := interface{}(nil)
	if n.ParentTaskID != "" {
		parent = n.ParentTaskID
	}
	colID := interface{}(nil)
	if n.ProgressColumnID != "" {
		colID = n.ProgressColumnID
	}
	return map[string]interface{}{
		"id":                   n.ID,
		"title":                n.Title,
		"parent_task":          parent,
		"depth":                n.Depth,
		"progress_column_id":   colID,
		"progress_column_name": n.ProgressColumnName,
		"completed":            n.Completed,
		"terminal_kind":        n.TerminalKind,
		"settled":              n.Settled,
	}
}

func buildSubtreePayload(tenantID, workspaceID, rootID string, maxDepth int) (map[string]interface{}, error) {
	if maxDepth <= 0 {
		maxDepth = 2
	}
	nodes, err := listDescendantsBFS(tenantID, workspaceID, rootID, maxDepth)
	if err != nil {
		return nil, err
	}
	nodes = enrichSubtreeNodes(tenantID, workspaceID, nodes)
	sum := summarizeSubtree(nodes)
	arr := make([]map[string]interface{}, 0, len(nodes))
	for _, n := range nodes {
		arr = append(arr, subtreeNodeJSON(n))
	}
	return map[string]interface{}{
		"task_id":   rootID,
		"max_depth": maxDepth,
		"summary": map[string]interface{}{
			"total":     sum.Total,
			"settled":   sum.Settled,
			"open":      sum.Open,
			"completed": sum.Completed,
			"cancelled": sum.Cancelled,
		},
		"nodes": arr,
	}, nil
}
