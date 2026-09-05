package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"taskCloudService/domain"
)

// ccbLogGroup identifies one (workspace_id, company_id, task_id, comment_id)
// group of shard log rows missing a COS startup-log pointer row.
type ccbLogGroup struct {
	WorkspaceID string
	CompanyID   string
	TaskID      string
	CommentID   string
}

// ccbStartupLogBackfillResult summarizes one backfill run.
type ccbStartupLogBackfillResult struct {
	TablesScanned   int
	CommentsScanned int
	Archived        int
	Evicted         int
	Skipped         int
	Errors          int
	ErrorMessages   []string
}

// listCCBLogGroupsMissingPointer returns distinct log groups in the given
// shard table that have no cloud_comment_startup_log_object pointer row.
// Used only by the one-time backfill CLI.
func listCCBLogGroupsMissingPointer(table string) ([]ccbLogGroup, error) {
	if db == nil {
		return nil, fmt.Errorf("db not open")
	}
	rows, err := db.Query(
		`SELECT l.workspace_id, l.company_id, l.task_id, l.comment_id
		 FROM ` + table + ` l
		 LEFT JOIN cloud_comment_startup_log_object p
		   ON p.workspace_id = l.workspace_id
		  AND p.task_id = l.task_id
		  AND p.comment_id = l.comment_id
		 WHERE p.id IS NULL AND l.workspace_id <> ''
		 GROUP BY l.workspace_id, l.company_id, l.task_id, l.comment_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ccbLogGroup, 0)
	for rows.Next() {
		var g ccbLogGroup
		if err := rows.Scan(&g.WorkspaceID, &g.CompanyID, &g.TaskID, &g.CommentID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// listCCBLogGroupsWithPointer returns distinct log groups that still have
// shard rows AND a COS pointer (legacy dual-write leftovers).
func listCCBLogGroupsWithPointer(table string) ([]ccbLogGroup, error) {
	if db == nil {
		return nil, fmt.Errorf("db not open")
	}
	rows, err := db.Query(
		`SELECT l.workspace_id, l.company_id, l.task_id, l.comment_id
		 FROM ` + table + ` l
		 INNER JOIN cloud_comment_startup_log_object p
		   ON p.workspace_id = l.workspace_id
		  AND p.task_id = l.task_id
		  AND p.comment_id = l.comment_id
		 WHERE l.workspace_id <> ''
		 GROUP BY l.workspace_id, l.company_id, l.task_id, l.comment_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ccbLogGroup, 0)
	for rows.Next() {
		var g ccbLogGroup
		if err := rows.Scan(&g.WorkspaceID, &g.CompanyID, &g.TaskID, &g.CommentID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// listCCBLogsForComment loads raw shard log rows for one comment, bypassing
// object-store hydration so the backfill always works off the shard truth.
func listCCBLogsForComment(workspaceID, companyID, taskID, commentID string) ([]CommentContainerBindingLog, error) {
	table, err := ccbLogTable(workspaceID)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at
		 FROM `+table+`
		 WHERE workspace_id=? AND company_id=? AND task_id=? AND comment_id=?
		 ORDER BY created_at ASC, id ASC`,
		workspaceID, companyID, taskID, commentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CommentContainerBindingLog, 0)
	for rows.Next() {
		var l CommentContainerBindingLog
		var created string
		if err := rows.Scan(
			&l.ID, &l.WorkspaceID, &l.CompanyID, &l.TaskID, &l.CommentID, &l.BindingID, &l.Stage, &l.Message, &created,
		); err != nil {
			return nil, err
		}
		l.CreatedAt = parseCloudUTCDateTime(created)
		out = append(out, l)
	}
	return out, rows.Err()
}

// backfillCCBStartupLogs is a one-time ops CLI (OPT-20260827-012): it scans all
// shard tables for comments whose startup logs never reached COS (dual-write
// covers new inserts only), assembles one bundle per comment and uploads it,
// then upserts the pointer row and deletes archived shard rows after a successful Put.
// Must NOT run on a business ticker.
func backfillCCBStartupLogs(ctx context.Context) (ccbStartupLogBackfillResult, error) {
	if db == nil {
		return ccbStartupLogBackfillResult{}, fmt.Errorf("db not open")
	}
	startupLogArchiveMu.Lock()
	defer startupLogArchiveMu.Unlock()
	var res ccbStartupLogBackfillResult
	rule := strings.TrimSpace(stepFullCOSCfg.StartupLogsPathRule)
	for _, table := range ccbLogShardTables() {
		res.TablesScanned++
		groups, err := listCCBLogGroupsMissingPointer(table)
		if err != nil {
			res.Errors++
			res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s scan: %v", table, err))
			continue
		}
		for _, g := range groups {
			res.CommentsScanned++
			logs, err := listCCBLogsForComment(g.WorkspaceID, g.CompanyID, g.TaskID, g.CommentID)
			if err != nil {
				res.Errors++
				res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s load %s: %v", table, g.CommentID, err))
				continue
			}
			id := domain.StartupLogIDs{
				WorkspaceID: g.WorkspaceID, TaskID: g.TaskID, CommentID: g.CommentID,
				KeyPrefix: strings.TrimSpace(stepFullCOSCfg.KeyPrefix),
			}
			if err := id.Validate(); err != nil {
				res.Errors++
				res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s validate %s: %v", table, g.CommentID, err))
				continue
			}
			bundle := domain.EmptyStartupLogBundle(id)
			for _, l := range logs {
				entry := domain.StartupLogEntry{
					ID: l.ID, BindingID: l.BindingID, Stage: l.Stage, Message: l.Message,
					CreatedAt:   formatCloudUTCJSON(l.CreatedAt),
					CompanyID:   l.CompanyID,
					WorkspaceID: l.WorkspaceID,
					TaskID:      l.TaskID,
					CommentID:   l.CommentID,
				}
				if strings.TrimSpace(entry.CreatedAt) == "" && !l.CreatedAt.IsZero() {
					entry.CreatedAt = l.CreatedAt.UTC().Format("2006-01-02 15:04:05")
				}
				bundle = domain.MergeStartupLogEntry(bundle, id, entry)
			}
			if len(bundle.Logs) == 0 {
				res.Skipped++
				continue
			}
			raw, err := domain.MarshalStartupLogBundle(bundle)
			if err != nil {
				res.Errors++
				res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s marshal %s: %v", table, g.CommentID, err))
				continue
			}
			key, err := domain.RenderStartupLogObjectKey(rule, id)
			if err != nil {
				res.Errors++
				res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s key %s: %v", table, g.CommentID, err))
				continue
			}
			etag := ""
			source := "local"
			payload := string(raw)
			putOK := false
			if stepFullObjects != nil {
				e, putErr := stepFullObjects.Put(ctx, key, raw)
				if putErr != nil {
					// COS down: preserve the bundle in the pointer's payload_json
					// so the data is not lost (same fallback as the hot path).
					res.Errors++
					res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s put %s: %v", table, g.CommentID, putErr))
				} else {
					putOK = true
					etag = e
					if strings.EqualFold(strings.TrimSpace(stepFullCOSCfg.Backend), "cos") {
						source = "cos"
						payload = ""
					}
				}
			}
			if err := upsertStartupLogObjectRow(startupLogObjectRow{
				CompanyID: g.CompanyID, WorkspaceID: id.WorkspaceID, TaskID: id.TaskID,
				CommentID: id.CommentID, ObjectKey: key, ETag: etag, Bytes: len(raw),
				Source: source, PayloadJSON: payload,
			}); err != nil {
				res.Errors++
				res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s pointer %s: %v", table, g.CommentID, err))
				continue
			}
			if putOK {
				if delErr := deleteCCBLogShardRowsForComment(g.WorkspaceID, g.CompanyID, g.TaskID, g.CommentID); delErr != nil {
					res.Errors++
					res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s shard-delete %s: %v", table, g.CommentID, delErr))
				}
			}
			res.Archived++
			log.Printf("[taskCloudService] event=ccb_startup_log_backfill_ok workspace_id=%s task_id=%s comment_id=%s object_key=%s bytes=%d source=%s",
				id.WorkspaceID, id.TaskID, id.CommentID, key, len(raw), source)
		}
	}
	evictCCBLogShardsAlreadyInObjectStore(ctx, &res)
	return res, nil
}

// evictCCBLogShardsAlreadyInObjectStore deletes leftover shard rows whose
// comment already has a pointer and whose bundle is readable from the object
// store. Does not use payload_json fallback — COS down must keep MySQL.
func evictCCBLogShardsAlreadyInObjectStore(ctx context.Context, res *ccbStartupLogBackfillResult) {
	if res == nil || stepFullObjects == nil {
		return
	}
	for _, table := range ccbLogShardTables() {
		groups, err := listCCBLogGroupsWithPointer(table)
		if err != nil {
			res.Errors++
			res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s evict-scan: %v", table, err))
			continue
		}
		for _, g := range groups {
			row, found, err := getStartupLogObjectRow(g.WorkspaceID, g.TaskID, g.CommentID)
			if err != nil || !found || strings.TrimSpace(row.ObjectKey) == "" {
				continue
			}
			raw, foundObj, getErr := stepFullObjects.Get(ctx, row.ObjectKey)
			if getErr != nil || !foundObj || len(raw) == 0 {
				continue
			}
			if _, perr := domain.ParseStartupLogBundle(raw); perr != nil {
				continue
			}
			if delErr := deleteCCBLogShardRowsForComment(g.WorkspaceID, g.CompanyID, g.TaskID, g.CommentID); delErr != nil {
				res.Errors++
				res.ErrorMessages = append(res.ErrorMessages, fmt.Sprintf("%s evict %s: %v", table, g.CommentID, delErr))
				continue
			}
			res.Evicted++
			log.Printf("[taskCloudService] event=ccb_startup_log_shard_evict_ok workspace_id=%s task_id=%s comment_id=%s",
				g.WorkspaceID, g.TaskID, g.CommentID)
		}
	}
}
