package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func nextWorkspaceSeq(tx *sql.Tx, tenantID, workspaceID string) (int, error) {
	if tx == nil {
		return 0, fmt.Errorf("nextWorkspaceSeq: nil tx")
	}
	tenantID = strings.TrimSpace(tenantID)
	workspaceID = strings.TrimSpace(workspaceID)
	if tenantID == "" || workspaceID == "" {
		return 0, fmt.Errorf("nextWorkspaceSeq: tenant_id and workspace_id required")
	}
	if _, err := tx.Exec(
		`INSERT INTO task_workspace_seq(tenant_id, workspace_id, next_val, updated_at)
		 VALUES(?,?,1,UTC_TIMESTAMP())
		 ON DUPLICATE KEY UPDATE tenant_id=tenant_id`,
		tenantID, workspaceID,
	); err != nil {
		return 0, err
	}
	var seq int
	if err := tx.QueryRow(
		`SELECT next_val FROM task_workspace_seq WHERE tenant_id=? AND workspace_id=? FOR UPDATE`,
		tenantID, workspaceID,
	).Scan(&seq); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(
		`UPDATE task_workspace_seq SET next_val=next_val+1, updated_at=UTC_TIMESTAMP()
		 WHERE tenant_id=? AND workspace_id=?`,
		tenantID, workspaceID,
	); err != nil {
		return 0, err
	}
	return seq, nil
}

func insertTaskWithWorkspaceSeq(
	id, tenantID, title, description string,
	completed int,
	priority string,
	orderNum int,
	workspaceID, ownerID, operatorID, deliverableObjID, progressColumnID, parentTaskID, forkFromID, imageID string,
	imageSkillID, containerImageSnapshot string,
	autoRun, autoCommit int,
	fps, pfpc, taskKind, codeLang string,
	agentModel string,
	postExpiresAtVal interface{},
	actorUserID string,
	now time.Time,
) (seq int, revisionID string, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, "", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	seq, err = nextWorkspaceSeq(tx, tenantID, workspaceID)
	if err != nil {
		return 0, "", err
	}
	// JSON 列不能存空字符串（非法 JSON），未绑定技能时写入 NULL。
	var snapVal interface{}
	if containerImageSnapshot != "" {
		snapVal = containerImageSnapshot
	}
	if _, err := tx.Exec(`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,operator_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,image_skill_id,container_image_snapshot,agent_model,auto_run,auto_commit_after_agent_complete,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,post_expires_at,workspace_seq,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, tenantID, title, description, completed, priority,
		orderNum, workspaceID, ownerID, operatorID, deliverableObjID, progressColumnID,
		parentTaskID, forkFromID, imageID,
		imageSkillID, snapVal, agentModel,
		autoRun, autoCommit, fps, pfpc,
		taskKind, codeLang, postExpiresAtVal, seq, now, now); err != nil {
		return 0, "", err
	}
	revID, err := persistTaskRevisionV1(tx, tenantID, workspaceID, id, title, description, actorUserID, now)
	if err != nil {
		return 0, "", err
	}
	if err := tx.Commit(); err != nil {
		return 0, "", err
	}
	committed = true
	return seq, revID, nil
}

func parseNumericSearchSeq(q string) (int, bool) {
	q = strings.TrimSpace(q)
	if q == "" {
		return 0, false
	}
	n, err := strconv.Atoi(q)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
