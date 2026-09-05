package main

import (
	"context"
	"errors"
	"log"
	"strings"
)

type commentContainerAdvanceResult struct {
	Advanced []CommentContainerBinding
	Blocked  []CommentContainerBinding
}

func advanceCommentContainerBindings(companyID, taskID, workspaceID string) (*commentContainerAdvanceResult, error) {
	rows, err := listCommentContainerBindings(companyID, taskID)
	if err != nil {
		return nil, err
	}
	if workspaceID == "" {
		workspaceID = resolveWorkspaceIDForTask(companyID, taskID)
	}

	result := &commentContainerAdvanceResult{
		Advanced: make([]CommentContainerBinding, 0),
		Blocked:  make([]CommentContainerBinding, 0),
	}

	// 纠偏：多个 live binding 误挂同一 csc_id 时 demote（不同 csc_id 可并行）
	if err := ccbEnforceCSCExclusivity(rows); err != nil {
		return nil, err
	}

	for i := range rows {
		b := &rows[i]

		// starting + 已挂 CSC：仅当 server_url/endpoint 就绪时升为 running
		if b.Status == ccbStatusStarting && trim(b.CSCID) != "" {
			promoted, err := ccbTryPromoteStartingToRunning(&rows, i)
			if err != nil {
				return nil, err
			}
			if promoted {
				result.Advanced = append(result.Advanced, rows[i])
				publishCommentContainerBindingAdvanced(companyID, taskID, &rows[i])
			}
			continue
		}

		// starting 且尚未挂接 CSC：ensure 独立 CSC，保持 starting 直至 reachability
		if b.Status == ccbStatusStarting && trim(b.CSCID) == "" {
			claimed, err := ccbTryAttachCommentCSC(&rows, i, workspaceID)
			if err != nil {
				return nil, err
			}
			if claimed {
				// 若 CSC 已可达则可能已升 running；否则仍为 starting+csc
				if rows[i].Status == ccbStatusStarting {
					if promoted, pErr := ccbTryPromoteStartingToRunning(&rows, i); pErr != nil {
						return nil, pErr
					} else if promoted {
						// fall through to publish below
					}
				}
				result.Advanced = append(result.Advanced, rows[i])
				publishCommentContainerBindingAdvanced(companyID, taskID, &rows[i])
			}
			continue
		}

		if !ccbCanScheduleFromStatus(b.Status) {
			continue
		}
		if b.ExecutionMode == ccbExecutionIndependent {
			if err := ccbStartBinding(&rows, i, workspaceID); err != nil {
				return nil, err
			}
			result.Advanced = append(result.Advanced, rows[i])
			publishCommentContainerBindingAdvanced(companyID, taskID, &rows[i])
			continue
		}

		if ccbIsBlockedByDependencies(rows, i) {
			if b.Status != ccbStatusWaitingPrevious {
				if err := updateCommentContainerBindingStatus(b.ID, ccbStatusWaitingPrevious); err != nil {
					return nil, err
				}
				rows[i].Status = ccbStatusWaitingPrevious
				// OPT-20260809-011: 记录「等待前序」阶段事件（仅首次进入）
				logCommentContainerBindingStageBestEffort(&rows[i], ccbStatusWaitingPrevious)
			}
			result.Blocked = append(result.Blocked, rows[i])
			continue
		}

		if err := ccbStartBinding(&rows, i, workspaceID); err != nil {
			return nil, err
		}
		result.Advanced = append(result.Advanced, rows[i])
		publishCommentContainerBindingAdvanced(companyID, taskID, &rows[i])
	}
	return result, nil
}

func ccbCanScheduleFromStatus(status string) bool {
	switch status {
	case ccbStatusPending, ccbStatusWaitingPrevious:
		return true
	default:
		return false
	}
}

// ccbEnforceCSCExclusivity 仅纠偏「多个 live binding 共用同一 csc_id」的历史错误；
// 不同 csc_id 允许并行 running（一评论一机器）。
func ccbEnforceCSCExclusivity(rows []CommentContainerBinding) error {
	seen := map[string]int{}
	for i := range rows {
		st := rows[i].Status
		if st != ccbStatusRunning && st != ccbStatusStarting {
			continue
		}
		csc := trim(rows[i].CSCID)
		if csc == "" {
			continue
		}
		if keeper, ok := seen[csc]; ok {
			mockName := resolveCommentMockContainerName(rows[i].TaskID, rows[i].CommentID, rows[i].MockContainerName)
			if err := markCommentContainerBindingStarting(rows[i].ID, mockName); err != nil {
				return err
			}
			rows[i].Status = ccbStatusStarting
			rows[i].CSCID = ""
			rows[i].MockContainerName = mockName
			log.Printf("[taskCloudService] event=comment_container_binding_demote_shared_csc comment_id=%s prev_csc_id=%s keeper_comment_id=%s",
				rows[i].CommentID, csc, rows[keeper].CommentID)
			continue
		}
		seen[csc] = i
	}
	return nil
}

func joinDependsOnCommentIDs(raw interface{}) string {
	ids := make([]string, 0)
	seen := map[string]struct{}{}
	add := func(s string) {
		id := trim(s)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	switch v := raw.(type) {
	case []string:
		for _, s := range v {
			add(s)
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	case string:
		for _, part := range strings.Split(v, ",") {
			add(part)
		}
	}
	return strings.Join(ids, ",")
}

func ccbParseDependsOnCommentIDs(raw string) []string {
	s := trim(raw)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		id := trim(p)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ccbIsBlockedByDependencies:
// - depends 为空：等待全部序位更早的 binding 完成（全部前序）
// - depends 非空：等待所列 comment 的 binding 均完成
func ccbIsBlockedByDependencies(ordered []CommentContainerBinding, idx int) bool {
	if idx < 0 || idx >= len(ordered) {
		return true
	}
	b := &ordered[idx]
	deps := ccbParseDependsOnCommentIDs(b.DependsOnCommentID)
	if len(deps) == 0 {
		for i := 0; i < idx; i++ {
			if ordered[i].Status != ccbStatusCompleted {
				return true
			}
		}
		return false
	}
	byID := make(map[string]*CommentContainerBinding, len(ordered))
	for i := range ordered {
		byID[ordered[i].CommentID] = &ordered[i]
	}
	for _, id := range deps {
		pred := byID[id]
		if pred == nil || pred.Status != ccbStatusCompleted {
			return true
		}
	}
	return false
}

func ccbResolvePredecessor(ordered []CommentContainerBinding, idx int) *CommentContainerBinding {
	b := &ordered[idx]
	deps := ccbParseDependsOnCommentIDs(b.DependsOnCommentID)
	if len(deps) > 0 {
		for i := range ordered {
			if ordered[i].CommentID == deps[0] {
				return &ordered[i]
			}
		}
		return nil
	}
	if idx == 0 {
		return nil
	}
	return &ordered[idx-1]
}

// ccbFailPermanentIfUnconfigured 曾在 mock/空平台首轮 bootstrap 时把 binding 收口 failed
// （OPT-20260812-010）。这会在任务级云平台 / start-vm 尚未落库时误伤：后续阿里云已启动成功，
// 卡片却一直红条「启动失败」。mock 只表示此刻不能启机，不是永久失败。
// 返回 false：调用方继续保持 starting。真正的失败由评论级 start-vm error SSE 收口。
func ccbFailPermanentIfUnconfigured(b *CommentContainerBinding, mockName string, csc *CloudServerConfig) bool {
	if b == nil || csc == nil {
		return false
	}
	log.Printf("[taskCloudService] event=comment_container_binding_keep_starting comment_id=%s csc_id=%s mock=%s reason=await_cloud_platform",
		b.CommentID, csc.ID, mockName)
	return false
}

// ccbStartBinding 启动评论容器绑定：为该 comment 确保独立 CSC 并进入 running。
// independent 不等待前序；wait_previous 仅在依赖满足后调用本函数。
func ccbStartBinding(rows *[]CommentContainerBinding, idx int, workspaceID string) error {
	if rows == nil || idx < 0 || idx >= len(*rows) {
		return nil
	}
	b := &(*rows)[idx]
	if err := updateCommentContainerBindingStatus(b.ID, ccbStatusStarting); err != nil {
		return err
	}
	b.Status = ccbStatusStarting
	// OPT-20260809-011: 记录启动阶段事件
	logCommentContainerBindingStageBestEffort(b, ccbStatusStarting)
	mockName := buildCommentMockContainerName(b.TaskID, b.CommentID)

	csc, err := ensureCommentCloudServerConfig(b.CompanyID, workspaceID, b.TaskID, b.CommentID)
	if err != nil {
		// 无 workspace/模板时保持 starting（勿提前标 running 完成语义）
		log.Printf("[taskCloudService] event=comment_container_binding_start comment_id=%s status=starting csc_id= mock=%s reason=ensure_csc_failed err=%v",
			b.CommentID, mockName, err)
		if err := markCommentContainerBindingStarting(b.ID, mockName); err != nil {
			return err
		}
		b.Status = ccbStatusStarting
		b.MockContainerName = mockName
		b.CSCID = ""
		return nil
	}
	csc, bootErr := bootstrapCommentCSCRuntime(csc)
	if bootErr != nil {
		// mock/空平台：保持 starting，等待任务级云平台或评论级 start-vm；勿收口 failed
		if errors.Is(bootErr, errCommentCSCPlatformUnsupported) && ccbFailPermanentIfUnconfigured(b, mockName, csc) {
			return nil
		}
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_failed comment_id=%s csc_id=%s err=%v",
			b.CommentID, csc.ID, bootErr)
	}
	if commentCSCHasRuntime(csc) {
		if err := markCommentContainerBindingRunning(b.ID, mockName, csc.ID); err != nil {
			return err
		}
		b.Status = ccbStatusRunning
		b.MockContainerName = mockName
		b.CSCID = csc.ID
		// OPT-20260809-011: 实例已就绪 → 直接进入 running（csc_allocated 与 running 同日先写，去重只保留 running 消息）
		logCommentContainerBindingStageBestEffort(b, ccbStageCSCAllocated)
		logCommentContainerBindingStageBestEffort(b, ccbStatusRunning)
		log.Printf("[taskCloudService] event=comment_container_binding_start comment_id=%s status=running csc_id=%s mock=%s parallel=true",
			b.CommentID, csc.ID, mockName)
		return nil
	}
	// 云侧尚未登记 reachability：保持 starting，挂上 csc_id 供后续回填
	if err := markCommentContainerBindingStartingWithCSC(b.ID, mockName, csc.ID); err != nil {
		return err
	}
	b.Status = ccbStatusStarting
	b.MockContainerName = mockName
	b.CSCID = csc.ID
	// OPT-20260809-011: 实例已分配，等待 reachability 注册
	logCommentContainerBindingStageBestEffort(b, ccbStageCSCAllocated)
	log.Printf("[taskCloudService] event=comment_container_binding_start comment_id=%s status=starting csc_id=%s mock=%s awaiting_runtime=true",
		b.CommentID, csc.ID, mockName)
	return nil
}

func ccbTryAttachCommentCSC(rows *[]CommentContainerBinding, idx int, workspaceID string) (bool, error) {
	if rows == nil || idx < 0 || idx >= len(*rows) {
		return false, nil
	}
	b := &(*rows)[idx]
	mockName := resolveCommentMockContainerName(b.TaskID, b.CommentID, b.MockContainerName)
	csc, err := ensureCommentCloudServerConfig(b.CompanyID, workspaceID, b.TaskID, b.CommentID)
	if err != nil {
		return false, nil
	}
	csc, bootErr := bootstrapCommentCSCRuntime(csc)
	if bootErr != nil {
		// mock/空平台：保持 starting，等待云平台或 start-vm；勿收口 failed
		if errors.Is(bootErr, errCommentCSCPlatformUnsupported) && ccbFailPermanentIfUnconfigured(b, mockName, csc) {
			return true, nil
		}
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_failed comment_id=%s csc_id=%s err=%v",
			b.CommentID, csc.ID, bootErr)
	}
	if commentCSCHasRuntime(csc) || cscHasReachableEndpoint(csc) {
		if err := markCommentContainerBindingRunning(b.ID, mockName, csc.ID); err != nil {
			return false, err
		}
		b.Status = ccbStatusRunning
		b.MockContainerName = mockName
		b.CSCID = csc.ID
		// OPT-20260809-011: attach 阶段即已就绪 → 依次记录分配与 running
		logCommentContainerBindingStageBestEffort(b, ccbStageCSCAllocated)
		logCommentContainerBindingStageBestEffort(b, ccbStatusRunning)
		log.Printf("[taskCloudService] event=comment_container_binding_attach_csc comment_id=%s status=running csc_id=%s mock=%s",
			b.CommentID, csc.ID, mockName)
		return true, nil
	}
	if err := markCommentContainerBindingStartingWithCSC(b.ID, mockName, csc.ID); err != nil {
		return false, err
	}
	b.Status = ccbStatusStarting
	b.MockContainerName = mockName
	b.CSCID = csc.ID
	// OPT-20260809-011: 实例已分配，等待 reachability
	logCommentContainerBindingStageBestEffort(b, ccbStageCSCAllocated)
	log.Printf("[taskCloudService] event=comment_container_binding_attach_csc comment_id=%s status=starting csc_id=%s mock=%s awaiting_reachability=true",
		b.CommentID, csc.ID, mockName)
	return true, nil
}

func ccbTryPromoteStartingToRunning(rows *[]CommentContainerBinding, idx int) (bool, error) {
	if rows == nil || idx < 0 || idx >= len(*rows) {
		return false, nil
	}
	b := &(*rows)[idx]
	if b.Status != ccbStatusStarting || trim(b.CSCID) == "" {
		return false, nil
	}
	mockName := resolveCommentMockContainerName(b.TaskID, b.CommentID, b.MockContainerName)
	csc, err := loadCloudServerConfigByID(b.CSCID)
	if err != nil {
		return false, nil
	}
	csc, bootErr := bootstrapCommentCSCRuntime(csc)
	if bootErr != nil {
		// mock/空平台：保持 starting，等待云平台或 start-vm；勿收口 failed
		if errors.Is(bootErr, errCommentCSCPlatformUnsupported) && ccbFailPermanentIfUnconfigured(b, mockName, csc) {
			return true, nil
		}
		log.Printf("[taskCloudService] event=comment_csc_bootstrap_failed comment_id=%s csc_id=%s err=%v",
			b.CommentID, b.CSCID, bootErr)
	}
	if !commentCSCHasRuntime(csc) && !cscHasReachableEndpoint(csc) {
		return false, nil
	}
	if err := markCommentContainerBindingRunning(b.ID, mockName, b.CSCID); err != nil {
		return false, err
	}
	b.Status = ccbStatusRunning
	b.MockContainerName = mockName
	// OPT-20260809-011: 从 starting+csc 晋升 running（csc_allocated 已在上游节点记录）
	logCommentContainerBindingStageBestEffort(b, ccbStatusRunning)
	log.Printf("[taskCloudService] event=comment_container_binding_promote_running comment_id=%s csc_id=%s mock=%s",
		b.CommentID, b.CSCID, mockName)
	return true, nil
}

func publishCommentContainerBindingAdvanced(companyID, taskID string, b *CommentContainerBinding) {
	if b == nil {
		return
	}
	containerName := resolveCommentMockContainerName(b.TaskID, b.CommentID, b.MockContainerName)
	data := map[string]interface{}{
		"company_id":            companyID,
		"task_id":               taskID,
		"comment_id":            b.CommentID,
		"binding_id":            b.ID,
		"execution_mode":        b.ExecutionMode,
		"depends_on_comment_id": b.DependsOnCommentID,
		"status":                b.Status,
		"mock_container_name":   containerName,
		"container_name":        containerName,
		"csc_id":                b.CSCID,
	}
	if err := publishDomainEvent(context.Background(), "CommentContainerBindingAdvanced", data, taskID); err != nil {
		log.Printf("[taskCloudService] CommentContainerBindingAdvanced publish failed task_id=%s comment_id=%s err=%v",
			taskID, b.CommentID, err)
		return
	}
	if strings.TrimSpace(cfg.KafkaBootstrapServers) == "" {
		log.Printf("[taskCloudService] CommentContainerBindingAdvanced task_id=%s comment_id=%s status=%s mock=%s",
			taskID, b.CommentID, b.Status, b.MockContainerName)
	}
}
