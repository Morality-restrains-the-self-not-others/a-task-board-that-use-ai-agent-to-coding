package main

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"taskTaskService/src/domain"
)

// containerImageSnapshot 是创建/更新任务时落库的「名↔ID 映射快照」（D2=B）：
// 镜像与技能的名字在保存时定格，镜像目录后续变化不影响任务显示与解析。
// JSON 列 container_image_snapshot；skill 字段缺省省略（任务未绑定技能时）。
type containerImageSnapshot struct {
	ImageID   string `json:"image_id"`
	ImageName string `json:"image_name"`
	SkillID   string `json:"skill_id,omitempty"`
	SkillName string `json:"skill_name,omitempty"`
}

func (s *containerImageSnapshot) encode() (string, error) {
	if s == nil {
		return "", nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decodeContainerImageSnapshot 解析存储快照；空/非法返回 nil。
func decodeContainerImageSnapshot(raw string) *containerImageSnapshot {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var s containerImageSnapshot
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil
	}
	if s.ImageID == "" {
		return nil
	}
	return &s
}

// snapshotToJSON 把存储快照渲染为响应对象（skill 字段仅在有值时出现，省略缺省）；
// 空/非法返回 nil。
func snapshotToJSON(raw string) interface{} {
	s := decodeContainerImageSnapshot(raw)
	if s == nil {
		return nil
	}
	out := map[string]interface{}{"image_id": s.ImageID, "image_name": s.ImageName}
	if s.SkillID != "" {
		out["skill_id"] = s.SkillID
	}
	if s.SkillName != "" {
		out["skill_name"] = s.SkillName
	}
	return out
}

// skillTokenRE 匹配描述中的 /token（与镜像技能名同形，镜像技能名规则见 imageSkillNameRE）。
var skillTokenRE = regexp.MustCompile(`/([a-z0-9][a-z0-9-]{0,62})`)

// firstSkillTokenInDescription 在描述中提取第一个「属于镜像技能列表」的 /token 技能名；
// 未命中返回 ""。只按列表名匹配，避免把 /api、/v1 等普通路径误判为技能引用。
func firstSkillTokenInDescription(description string, list domain.ImageSkillList) string {
	if len(list.Skills) == 0 {
		return ""
	}
	names := map[string]struct{}{}
	for _, s := range list.Skills {
		names[s.Name] = struct{}{}
	}
	for _, m := range skillTokenRE.FindAllStringSubmatch(description, -1) {
		if _, ok := names[m[1]]; ok {
			return m[1]
		}
	}
	return ""
}

// taskImageSkillError 承载技能绑定解析失败的错误（HTTP 状态 + 错误码）。
type taskImageSkillError struct {
	status int
	code   string
	msg    string
}

// resolveTaskImageSkill 解析创建/更新任务请求携带的镜像技能绑定（D4 契约）：
//   - container_image_id 存在但目录解析失败（已卸载/非本租户）→ 400 image_not_found
//   - container_image_skill_id 提供 → 必须属于镜像 image_skills 列表，否则 400 skill_not_in_image
//   - container_image_skill_id 缺失（老前端）→ 描述中按镜像技能名提取首个 /token 反解
//     （尽力而为，非 fail-closed）；描述无列表内技能 token → 不绑定（skill 置空）
//   - skill_id 提供但无 image → 400 skill_id_requires_image
//
// 返回保存时快照（nil = 不绑定）。imageID 为空且 skillID 为空时直接返回 nil。
func resolveTaskImageSkill(tenantID, imageID, skillID, description string) (*containerImageSnapshot, *taskImageSkillError) {
	imageID = strings.TrimSpace(imageID)
	skillID = strings.TrimSpace(skillID)
	if imageID == "" {
		if skillID != "" {
			return nil, &taskImageSkillError{status: 400, code: "skill_id_requires_image",
				msg: "container_image_skill_id requires container_image_id"}
		}
		return nil, nil
	}
	img, err := lookupInstalledImageFn(tenantID, imageID)
	if err != nil {
		if errors.Is(err, ErrInstalledImageNotFound) {
			return nil, &taskImageSkillError{status: 400, code: "image_not_found",
				msg: "container_image not found"}
		}
		return nil, &taskImageSkillError{status: 502, code: "image_lookup_failed",
			msg: err.Error()}
	}
	if img == nil {
		return nil, &taskImageSkillError{status: 400, code: "image_not_found",
			msg: "container_image not found"}
	}
	var skill *domain.ImageSkill
	if skillID != "" {
		for i := range img.ImageSkills.Skills {
			if img.ImageSkills.Skills[i].ID == skillID {
				skill = &img.ImageSkills.Skills[i]
				break
			}
		}
		if skill == nil {
			return nil, &taskImageSkillError{status: 400, code: "skill_not_in_image",
				msg: "container_image_skill_id not in image skills"}
		}
	} else {
		// 老前端兼容：描述含列表内 /token 时按名反解；不命中不绑定（skill_id_required
		// 防御码不启用，避免 /api 等普通路径文本触发 400 回归）。
		if name := firstSkillTokenInDescription(description, img.ImageSkills); name != "" {
			for i := range img.ImageSkills.Skills {
				if img.ImageSkills.Skills[i].Name == name {
					skill = &img.ImageSkills.Skills[i]
					break
				}
			}
		}
	}
	snap := &containerImageSnapshot{ImageID: img.ID, ImageName: img.Name}
	if skill != nil {
		snap.SkillID = skill.ID
		snap.SkillName = skill.Name
	}
	return snap, nil
}
