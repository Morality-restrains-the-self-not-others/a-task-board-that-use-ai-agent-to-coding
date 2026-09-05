package infrastructure

import (
	"encoding/json"
	"strings"

	"taskAiProvider/domain"
)

const DefaultImageSkillsPath = domain.DefaultImageSkillsPath

// ImageSkillsExtractResult is the cached skill catalog extracted from an OCI image.
type ImageSkillsExtractResult struct {
	List     domain.ImageSkillList
	JSON     string
	Digest   string
	Status   string // ok | not_found | auth_failed | failed
	Detail   string
}

func ExtractImageSkillsFromImage(imageURL string) ImageSkillsExtractResult {
	return extractImageSkillsFromImage(extractRegistryClient(), imageURL)
}

func extractImageSkillsFromImage(client registryHTTP, imageURL string) ImageSkillsExtractResult {
	file := extractAutoRunStepsFromImage(client, imageURL, DefaultImageSkillsPath)
	out := ImageSkillsExtractResult{Digest: file.Digest, Status: file.Status, Detail: file.Detail}
	if file.Status != "ok" {
		if file.Status == "not_found" {
			out.Detail = "imageSkills.yaml not found in image layers"
		}
		return out
	}
	list, err := ParseImageSkillsYAML(file.Markdown)
	if err != nil {
		out.Status = "failed"
		out.Detail = err.Error()
		return out
	}
	raw, err := json.Marshal(list)
	if err != nil {
		out.Status = "failed"
		out.Detail = err.Error()
		return out
	}
	out.List = list
	out.JSON = string(raw)
	out.Status = "ok"
	out.Detail = ""
	return out
}

func EmptyImageSkillsJSON() string {
	b, _ := json.Marshal(domain.ImageSkillList{Version: 1, Skills: []domain.ImageSkill{}})
	return string(b)
}

func DecodeImageSkillsJSON(raw string) domain.ImageSkillList {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.ImageSkillList{Version: 1, Skills: []domain.ImageSkill{}}
	}
	var list domain.ImageSkillList
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return domain.ImageSkillList{Version: 1, Skills: []domain.ImageSkill{}}
	}
	if list.Skills == nil {
		list.Skills = []domain.ImageSkill{}
	}
	return list
}
