package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"confload"
	"tracelog"
)

// FanyiAgentConfig mirrors conf/taskProjectService/config.yaml ai_agent_config.fanyi_agent.
// OPT-20260806-057: 从 conf/core/django/config.yaml 迁出（Django 目录退役）。
type FanyiAgentConfig struct {
	APIKey      string  `yaml:"api_key"`
	BaseURL     string  `yaml:"base_url"`
	Model       string  `yaml:"model"`
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
	TopP        float64 `yaml:"top_p"`
}

var fanyiAgentCfg FanyiAgentConfig

// Title translation must finish inside the create-task debounce UX.
// Conf max_tokens is for general chat and can be 4096; a one-line branch
// name must not request that budget (slow/empty completions).
// Live 2026-09-02: successful DeepSeek calls took up to ~11.5s; hung ones
// were cut at ~20s with truncated JSON. 16s covers observed success.
//
// deepseek-v4-flash enables thinking by default; reasoning tokens share the
// same max_tokens budget as the final answer. With a 128-token cap, thinking
// alone returns HTTP 200 + empty content (trace 487b9895… 2026-09-04). Title
// translation therefore always sends thinking.type=disabled.
const fanyiTitleMaxTokens = 128
const fanyiTitleTimeout = 16 * time.Second

// fanyiHTTPClient is DirectClient so business outbound never inherits shell proxies.
var fanyiHTTPClient = tracelog.DirectClient(fanyiTitleTimeout)

func fanyiRequestMaxTokens() int {
	n := fanyiAgentCfg.MaxTokens
	if n <= 0 || n > fanyiTitleMaxTokens {
		return fanyiTitleMaxTokens
	}
	return n
}

func isFanyiTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Client.Timeout") ||
		strings.Contains(msg, "context deadline") ||
		strings.Contains(msg, "i/o timeout")
}

func loadFanyiAgentConfig(repoRoot string) {
	var wrap struct {
		AIAgentConfig struct {
			FanyiAgent FanyiAgentConfig `yaml:"fanyi_agent"`
		} `yaml:"ai_agent_config"`
	}
	if err := confload.ReadAppConfigResolved(repoRoot, "taskProjectService", &wrap); err != nil {
		log.Printf("[taskProjectService] WARN fanyi_agent config: %v", err)
		return
	}
	cfgLoaded := wrap.AIAgentConfig.FanyiAgent
	if cfgLoaded.MaxTokens <= 0 {
		cfgLoaded.MaxTokens = 256
	}
	fanyiAgentCfg = cfgLoaded
	log.Printf("[taskProjectService] fanyi_agent: model=%s base_url=%s configured=%v",
		fanyiAgentCfg.Model, fanyiAgentCfg.BaseURL, fanyiAgentCfg.APIKey != "" && fanyiAgentCfg.BaseURL != "" && fanyiAgentCfg.Model != "")
}

// reloadFanyiAgentConfig 运行时重新加载翻译 agent 配置，无需重启服务。
func reloadFanyiAgentConfig() {
	if monorepoRoot == "" {
		log.Printf("[taskProjectService] WARN reload fanyi_agent: monorepo root unknown, skip")
		return
	}
	loadFanyiAgentConfig(monorepoRoot)
}

func fanyiAgentConfigured() bool {
	return strings.TrimSpace(fanyiAgentCfg.APIKey) != "" &&
		strings.TrimSpace(fanyiAgentCfg.BaseURL) != "" &&
		strings.TrimSpace(fanyiAgentCfg.Model) != ""
}

type fanyiChatRequest struct {
	Model       string             `json:"model"`
	Messages    []fanyiChatMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	TopP        float64            `json:"top_p"`
	Stream      bool               `json:"stream"`
	Thinking    fanyiThinkingMode  `json:"thinking"`
}

// fanyiThinkingMode is DeepSeek Chat Completions thinking control.
// See https://api-docs.deepseek.com/ — default for v4-flash is enabled.
type fanyiThinkingMode struct {
	Type string `json:"type"`
}

type fanyiChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type fanyiChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
}

// translateTitleWithFanyiAgent calls OpenAI-compatible chat/completions and sanitizes the result.
func translateTitleWithFanyiAgent(ctx context.Context, sourceTitle string) (string, error) {
	if !fanyiAgentConfigured() {
		return "", fmt.Errorf("AI_AGENT_CONFIG.fanyi_agent 配置不完整")
	}
	ctx, cancel := context.WithTimeout(ctx, fanyiTitleTimeout)
	defer cancel()
	endpoint := strings.TrimRight(fanyiAgentCfg.BaseURL, "/") + "/chat/completions"
	payload := fanyiChatRequest{
		Model: fanyiAgentCfg.Model,
		Messages: []fanyiChatMessage{
			{
				Role: "system",
				Content: "你是分支命名翻译助手。" +
					"请把输入任务标题翻译成简洁英文，仅返回一行英文短语。" +
					"禁止返回中文、解释、标点句号。",
			},
			{Role: "user", Content: sourceTitle},
		},
		MaxTokens:   fanyiRequestMaxTokens(),
		Temperature: fanyiAgentCfg.Temperature,
		TopP:        fanyiAgentCfg.TopP,
		Stream:      false,
		Thinking:    fanyiThinkingMode{Type: "disabled"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+fanyiAgentCfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := fanyiHTTPClient.Do(req)
	if err != nil {
		if isFanyiTimeoutErr(err) {
			return "", fmt.Errorf("fanyi_agent 请求超时")
		}
		return "", fmt.Errorf("fanyi_agent 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		if isFanyiTimeoutErr(readErr) {
			return "", fmt.Errorf("fanyi_agent 请求超时: status=%d bytes=%d", resp.StatusCode, len(body))
		}
		return "", fmt.Errorf("fanyi_agent 读取响应失败: status=%d bytes=%d: %w", resp.StatusCode, len(body), readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyPreview := string(body)
		if len(bodyPreview) > 500 {
			bodyPreview = bodyPreview[:500]
		}
		return "", fmt.Errorf("fanyi_agent 请求失败: status=%d bytes=%d body=%s", resp.StatusCode, len(body), bodyPreview)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return "", fmt.Errorf("fanyi_agent 响应为空: status=%d bytes=0", resp.StatusCode)
	}
	var parsed fanyiChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		preview := strings.TrimSpace(string(body))
		if len(preview) > 80 {
			preview = preview[:80]
		}
		return "", fmt.Errorf("fanyi_agent 响应无效: status=%d bytes=%d preview=%q", resp.StatusCode, len(body), preview)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		finishReason := ""
		reasoningLen := 0
		if len(parsed.Choices) > 0 {
			finishReason = parsed.Choices[0].FinishReason
			reasoningLen = len(strings.TrimSpace(parsed.Choices[0].Message.ReasoningContent))
		}
		return "", fmt.Errorf(
			"fanyi_agent 返回内容为空: status=%d bytes=%d finish_reason=%s reasoning_len=%d",
			resp.StatusCode, len(body), finishReason, reasoningLen,
		)
	}
	return sanitizeBranchTitleSegment(parsed.Choices[0].Message.Content), nil
}
