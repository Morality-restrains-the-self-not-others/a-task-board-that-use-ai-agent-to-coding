package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	defaultImageSkillsPath = "/app/imageSkills.yaml"
	maxImageSkills         = 32
)

var imageSkillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type imageSkill struct {
	// ID 是 D1=B（服务端派生）引入的稳定技能 ID：sk_<sha1(name+seed)>[:12]。
	// imageSkills.yaml 不声明 id，由提取/快照/回填路径按镜像 seed 派生；缺省为空。
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

type imageSkillList struct {
	Version      int          `json:"version"`
	DefaultSkill string       `json:"default_skill"`
	Skills       []imageSkill `json:"skills"`
}

func (l imageSkillList) DefaultName() string {
	if strings.TrimSpace(l.DefaultSkill) != "" {
		return l.DefaultSkill
	}
	if len(l.Skills) == 0 {
		return ""
	}
	return l.Skills[0].Name
}

func (l imageSkillList) Has(name string) bool {
	for _, s := range l.Skills {
		if s.Name == name {
			return true
		}
	}
	return false
}

type imageSkillsYAML struct {
	Version int `yaml:"version"`
	Skills  []struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"skills"`
}

func parseImageSkillsYAML(raw string) (imageSkillList, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return imageSkillList{Version: 1, Skills: []imageSkill{}}, nil
	}
	var doc imageSkillsYAML
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return imageSkillList{}, fmt.Errorf("invalid imageSkills.yaml: %w", err)
	}
	if doc.Version == 0 {
		doc.Version = 1
	}
	if doc.Version != 1 {
		return imageSkillList{}, fmt.Errorf("unsupported imageSkills.yaml version %d", doc.Version)
	}
	if len(doc.Skills) > maxImageSkills {
		return imageSkillList{}, fmt.Errorf("too many skills (max %d)", maxImageSkills)
	}
	out := imageSkillList{Version: doc.Version, Skills: make([]imageSkill, 0, len(doc.Skills))}
	seen := map[string]struct{}{}
	for i, s := range doc.Skills {
		name := strings.TrimSpace(s.Name)
		if !imageSkillNameRE.MatchString(name) {
			return imageSkillList{}, fmt.Errorf("invalid skill name %q", s.Name)
		}
		if _, ok := seen[name]; ok {
			return imageSkillList{}, fmt.Errorf("duplicate skill name %q", name)
		}
		seen[name] = struct{}{}
		desc := strings.TrimSpace(s.Description)
		if utf8.RuneCountInString(desc) > 512 {
			return imageSkillList{}, fmt.Errorf("skill %q description too long", name)
		}
		out.Skills = append(out.Skills, imageSkill{Name: name, Description: desc, IsDefault: i == 0})
	}
	if len(out.Skills) > 0 {
		out.DefaultSkill = out.Skills[0].Name
	}
	return out, nil
}

// imageSkillIDSeed 返回技能 ID 派生的种子：优先镜像 external_id，空则回退本服务
// installed id。安装快照、抽取回填与存量迁移三处共用同一规则，保证同一镜像在
// 任何写入路径下派生的技能 ID 一致。
func imageSkillIDSeed(externalID, installedID string) string {
	if externalID != "" {
		return externalID
	}
	return installedID
}

// deriveImageSkillID 按 D1=B（服务端派生）规则生成确定性技能 ID：
// sk_<sha1(name+seed)> 前 12 位 hex。同镜像同技能每次派生一致（重抽/回填稳定）；
// 技能改名即得新 ID（绑定随之失效，行为可预期）。与迁移 SQL 中
// CONCAT('sk_', LEFT(HEX(SHA1(CONCAT(name, seed))), 12)) 保持一致，勿单侧改动。
func deriveImageSkillID(name, seed string) string {
	sum := sha1.Sum([]byte(name + seed))
	return "sk_" + hex.EncodeToString(sum[:])[:12]
}

// assignImageSkillIDs 为缺失 id 的技能派生稳定 ID；已带 id（如厂商目录自带）保持不变。
// seed 为空或技能列表为空时原样返回。幂等：重复调用不改变结果。
func assignImageSkillIDs(list imageSkillList, seed string) imageSkillList {
	if seed == "" || len(list.Skills) == 0 {
		return list
	}
	out := list
	out.Skills = make([]imageSkill, len(list.Skills))
	for i, s := range list.Skills {
		if s.ID == "" {
			s.ID = deriveImageSkillID(s.Name, seed)
		}
		out.Skills[i] = s
	}
	return out
}

func decodeImageSkillsJSON(raw string) imageSkillList {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return imageSkillList{Version: 1, Skills: []imageSkill{}}
	}
	var list imageSkillList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return imageSkillList{Version: 1, Skills: []imageSkill{}}
	}
	if list.Skills == nil {
		list.Skills = []imageSkill{}
	}
	return list
}

func emptyImageSkillsJSON() string {
	b, _ := json.Marshal(imageSkillList{Version: 1, Skills: []imageSkill{}})
	return string(b)
}

// skillsSnapshotFromCatalog 拷贝目录镜像的 image_skills 快照。seed 非空时按 D1=B
// 为缺失 id 的技能派生稳定 ID（已带 id 的技能保持不变），使安装即落库的技能列表
// 携带 id，taskTaskService lookup 无需二次反查。
func skillsSnapshotFromCatalog(imageData map[string]interface{}, seed string) (jsonStr, status, digest string) {
	status = strField(imageData, "image_skills_extract_status")
	digest = strField(imageData, "image_skills_digest")
	switch t := imageData["image_skills"].(type) {
	case string:
		jsonStr = strings.TrimSpace(t)
	case map[string]interface{}:
		b, err := json.Marshal(t)
		if err == nil {
			jsonStr = string(b)
		}
	}
	if jsonStr == "" {
		jsonStr = emptyImageSkillsJSON()
	}
	if seed != "" {
		if raw, err := json.Marshal(assignImageSkillIDs(decodeImageSkillsJSON(jsonStr), seed)); err == nil {
			jsonStr = string(raw)
		}
	}
	return jsonStr, status, digest
}
