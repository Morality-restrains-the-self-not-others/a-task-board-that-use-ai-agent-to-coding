package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var envKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const userdataEnvAgentSystemPrompt = "你是容器启动脚本分析助手。" +
	"从给定 UserData 脚本中提取与运行容器有关的环境变量。" +
	"仅返回 JSON，对象键为环境变量名，值为字符串。" +
	"不要返回解释，不要 markdown。"

// extractContainerEnvByAgent mirrors Django _extract_container_env_by_agent.
// Returns env map and parser name ("agent" or "heuristic").
func extractContainerEnvByAgent(userdata string) (map[string]string, string) {
	apiKey := strings.TrimSpace(cfg.UserdataEnvAgentAPIKey)
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.UserdataEnvAgentBaseURL), "/")
	model := strings.TrimSpace(cfg.UserdataEnvAgentModel)
	if apiKey == "" || baseURL == "" || model == "" {
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}

	endpoint := baseURL + "/chat/completions"
	maxTokens := cfg.UserdataEnvAgentMaxTokens
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": userdataEnvAgentSystemPrompt},
			{"role": "user", "content": userdata},
		},
		"max_tokens":  maxTokens,
		"temperature": cfg.UserdataEnvAgentTemperature,
		"top_p":       1,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[taskCloudService] userdata_env_agent request failed: %v", err)
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Printf("[taskCloudService] userdata_env_agent failed: status=%d", resp.StatusCode)
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Choices) == 0 {
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if text == "" {
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	text = stripMarkdownJSONFence(text)
	var obj any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		log.Printf("[taskCloudService] userdata_env_agent returned non-json, fallback heuristic")
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	normalized := normalizeEnvObj(obj)
	if len(normalized) == 0 {
		return heuristicExtractEnvFromUserdata(userdata), "heuristic"
	}
	return normalized, "agent"
}

func stripMarkdownJSONFence(text string) string {
	s := strings.TrimSpace(text)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = regexp.MustCompile(`(?s)^`+"```"+`(?:json)?\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?s)\s*`+"```"+`$`).ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func normalizeEnvObj(raw any) map[string]string {
	m, ok := raw.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for key, value := range m {
		k := strings.TrimSpace(key)
		if k == "" || !envKeyRe.MatchString(k) {
			continue
		}
		if value == nil {
			continue
		}
		out[k] = fmt.Sprintf("%v", value)
	}
	return out
}
