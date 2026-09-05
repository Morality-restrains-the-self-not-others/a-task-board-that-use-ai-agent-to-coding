package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

type resolvedOwner struct {
	UserID          string
	CompanyMemberID string
	Username        string
}

// batchResolveTaskOwners resolves owner keys (company_member_id or user_id) to display names.
// Calls taskTenantService directly — no Django dependency (OPT-20260729-024 #9).
// On failure or empty tenant URL, returns an empty map (callers still emit created_by stubs).
func batchResolveTaskOwners(tenantID string, ownerKeys []string) map[string]resolvedOwner {
	out := map[string]resolvedOwner{}
	if strings.TrimSpace(cfg.TaskTenantServiceURL) == "" {
		return out
	}
	keys := make([]string, 0, len(ownerKeys))
	seen := map[string]struct{}{}
	for _, raw := range ownerKeys {
		k := strings.TrimSpace(raw)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return out
	}

	members, err := tenantListMembers(tenantID)
	if err != nil || len(members) == 0 {
		return out
	}

	byMemberID := map[string]map[string]interface{}{}
	byUserID := map[string]map[string]interface{}{}
	for _, m := range members {
		if id := strField(m, "id"); id != "" {
			byMemberID[id] = m
		}
		if uid := strField(m, "user_id"); uid != "" {
			byUserID[uid] = m
		}
	}

	for _, key := range keys {
		raw := byMemberID[key]
		if raw == nil {
			raw = byUserID[key]
		}
		if raw == nil {
			continue
		}
		username := strField(raw, "username")
		if username == "" {
			username = strField(raw, "member_name")
		}
		out[key] = resolvedOwner{
			UserID:          strField(raw, "user_id"),
			CompanyMemberID: strField(raw, "id"),
			Username:        username,
		}
	}
	return out
}

func ownerKeyFromTaskJSON(task map[string]interface{}) string {
	if task == nil {
		return ""
	}
	if v := task["owner"]; v != nil {
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

func createdByObject(ownerKey string, resolved map[string]resolvedOwner) map[string]interface{} {
	cb := map[string]interface{}{
		"id":       nil,
		"username": "",
		"email":    "",
	}
	ownerKey = strings.TrimSpace(ownerKey)
	if ownerKey == "" {
		return cb
	}
	if r, ok := resolved[ownerKey]; ok {
		if r.UserID != "" {
			cb["id"] = r.UserID
		} else {
			cb["id"] = ownerKey
		}
		cb["username"] = r.Username
		return cb
	}
	cb["id"] = ownerKey
	return cb
}

func attachCreatedBy(task map[string]interface{}, resolved map[string]resolvedOwner) {
	if task == nil {
		return
	}
	task["created_by"] = createdByObject(ownerKeyFromTaskJSON(task), resolved)
}

func enrichTasksCreatedBy(tenantID string, tasks []map[string]interface{}) {
	keys := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if k := ownerKeyFromTaskJSON(task); k != "" {
			keys = append(keys, k)
		}
	}
	resolved := batchResolveTaskOwners(tenantID, keys)
	for _, task := range tasks {
		attachCreatedBy(task, resolved)
	}
}

// hydrateTaskContainerImage 把任务绑定的已安装镜像快照（name/version）写入 task JSON。
// 仅在单任务路径（详情/创建/更新）调用，避免列表 N+1 拉取。
// 目录未命中（镜像已卸载/不在本租户已安装表）时置 container_image_removed=true，
// 前端据此展示「已卸载」而非误导为「已绑定」；其余错误静默回退（container_image 保持 nil）。
// D4：container_image.skill = {id, name} — 优先按任务绑定的 image_skill_id 从目录技能列表
// 解析；目录已变（技能改名/列表更新）时回退保存时快照中的 skill_id/skill_name。
// container_image_snapshot（名↔ID 映射快照，D2=B）始终透传，供编辑回填与卸载后展示。
func hydrateTaskContainerImage(out map[string]interface{}, tenantID, imageID, snapshotJSON string) {
	if out == nil {
		return
	}
	if snap := snapshotToJSON(snapshotJSON); snap != nil {
		out["container_image_snapshot"] = snap
	}
	if strings.TrimSpace(imageID) == "" {
		return
	}
	snap := decodeContainerImageSnapshot(snapshotJSON)
	// 技能快照带 image_name：陈旧 installed_image_id 下 Cloud 端按唯一名回退现网 ID。
	var snapImageName string
	if snap != nil {
		snapImageName = snap.ImageName
	}
	img, err := lookupInstalledImageFn(tenantID, imageID, snapImageName)
	if err != nil {
		if errors.Is(err, ErrInstalledImageNotFound) {
			out["container_image_removed"] = true
		} else {
			log.Printf("[taskTask] task_container_image_hydrate_failed tenant_id=%s image_id=%s err=%v", tenantID, imageID, err)
		}
		return
	}
	if img == nil {
		out["container_image_removed"] = true
		return
	}
	ci := map[string]interface{}{
		"id":                img.ID,
		"name":              img.Name,
		"version":           img.Version,
		"external_image_id": img.ExternalImageID,
	}
	if t := tImageSkillJSON(img, tImageSkillIDFromSnapshot(snap)); t != nil {
		ci["skill"] = t
	}
	out["container_image"] = ci
}

// tImageSkillIDFromSnapshot 返回任务绑定的技能 ID；无绑定时返回空。
func tImageSkillIDFromSnapshot(snap *containerImageSnapshot) string {
	if snap == nil {
		return ""
	}
	return snap.SkillID
}

// tImageSkillJSON 按技能 ID 在镜像技能列表中解析 skill = {id, name}；列表中找不到
// （技能改名/目录更新）时回退快照中的技能名。无绑定时返回 nil。
func tImageSkillJSON(img *InstalledImageLookup, skillID string) map[string]interface{} {
	if img == nil {
		return nil
	}
	if skillID != "" {
		for _, s := range img.ImageSkills.Skills {
			if s.ID == skillID {
				return map[string]interface{}{"id": s.ID, "name": s.Name}
			}
		}
	}
	return nil
}

func taskJSONWithCreatedBy(t *taskRecord, tenantID string) map[string]interface{} {
	out := taskToJSON(t, tenantID)
	hydrateTaskContainerImage(out, tenantID, t.InstalledImageID, t.ContainerImageSnapshot)
	enrichTasksCreatedBy(tenantID, []map[string]interface{}{out})
	return out
}

func taskJSONWithCommentsForDetail(t *taskRecord, tenantID string) map[string]interface{} {
	out := taskJSONWithCreatedBy(t, tenantID)
	enrichTaskDetailComments(out, t)
	return out
}
