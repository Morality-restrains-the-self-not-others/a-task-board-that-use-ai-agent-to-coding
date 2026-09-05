package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

var containerStreamHTTP = &http.Client{
	Timeout: 0,
	Transport: &http.Transport{
		ResponseHeaderTimeout: 15 * time.Second,
	},
}

type containerJobContext struct {
	ParentJobID              interface{}
	RepoLayerID              interface{}
	CommandKind              string
	AgentAutoIterationCount  interface{}
	AgentModels              interface{}
}

func parseContainerJobContext(raw map[string]interface{}) containerJobContext {
	out := containerJobContext{CommandKind: "trae"}
	if raw == nil {
		return out
	}
	out.ParentJobID = optStr(raw["parent_job_id"])
	out.RepoLayerID = optStr(raw["repo_layer_id"])
	kind := strings.TrimSpace(fmt.Sprintf("%v", raw["command_kind"]))
	if kind == "shell" || kind == "trae" {
		out.CommandKind = kind
	}
	out.AgentAutoIterationCount = raw["agent_auto_iteration_count"]
	out.AgentModels = raw["agent_models"]
	return out
}

func optStr(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" {
		return nil
	}
	return s
}

func traeJobsRequestBody(prompt string, jobCtx containerJobContext) map[string]interface{} {
	body := map[string]interface{}{
		"command":       prompt,
		"command_kind":  jobCtx.CommandKind,
	}
	if jobCtx.ParentJobID != nil {
		body["parent_job_id"] = jobCtx.ParentJobID
	} else if jobCtx.RepoLayerID != nil {
		body["repo_layer_id"] = jobCtx.RepoLayerID
	}
	env := map[string]string{}
	if jobCtx.AgentAutoIterationCount != nil {
		var steps int
		if _, scanErr := fmt.Sscanf(fmt.Sprintf("%v", jobCtx.AgentAutoIterationCount), "%d", &steps); scanErr == nil && steps > 0 {
			env["TASK_AGENT_MAX_STEPS"] = fmt.Sprintf("%d", steps)
		}
	}
	if items, ok := jobCtx.AgentModels.([]interface{}); ok {
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			model := strings.TrimSpace(fmt.Sprintf("%v", m["model"]))
			provider := strings.TrimSpace(fmt.Sprintf("%v", m["provider"]))
			if model != "" && provider != "" {
				env["TASK_AGENT_MODEL"] = model
				env["TASK_AGENT_MODEL_PROVIDER"] = provider
				break
			}
		}
	}
	if len(env) > 0 {
		body["env"] = env
	}
	return body
}

type containerStreamResult struct {
	StatusCode int
	Chunks     <-chan []byte
	Err        error
	Close      func()
}

func streamLegacyInstruct(ctx context.Context, baseURL, tenantID, workspaceID, taskID, instructID, prompt, traceID string) *containerStreamResult {
	url := legacyInstructURL(baseURL, tenantID, workspaceID, taskID)
	payload := map[string]interface{}{
		"prompt":      prompt,
		"task_id":     taskID,
		"instruct_id": instructID,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &containerStreamResult{StatusCode: 500, Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/plain, application/octet-stream, */*")
	tracelog.ApplyOutboundHeaders(req, ctx)
	resp, err := containerStreamHTTP.Do(req)
	if err != nil {
		return &containerStreamResult{StatusCode: 502, Err: err}
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return &containerStreamResult{StatusCode: resp.StatusCode, Err: nil}
	}
	ch := make(chan []byte, 8)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				ch <- chunk
			}
			if readErr != nil {
				return
			}
		}
	}()
	return &containerStreamResult{StatusCode: resp.StatusCode, Chunks: ch, Close: func() { _ = resp.Body.Close() }}
}

func streamTraeInstruct(ctx context.Context, baseURL, token, instructID, prompt, traceID string, jobCtx containerJobContext) *containerStreamResult {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	jobsURL := base + "/api/jobs"
	jobJSON := traeJobsRequestBody(prompt, jobCtx)
	body, _ := json.Marshal(jobJSON)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, jobsURL, bytes.NewReader(body))
	if err != nil {
		return &containerStreamResult{StatusCode: 500, Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Access-Token", token)
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &containerStreamResult{StatusCode: 502, Err: err}
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return &containerStreamResult{StatusCode: resp.StatusCode}
	}
	var createBody map[string]interface{}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if json.Unmarshal(raw, &createBody) != nil {
		return &containerStreamResult{StatusCode: 200, Err: fmt.Errorf("invalid trae jobs json")}
	}
	jobID := strings.TrimSpace(fmt.Sprintf("%v", createBody["id"]))
	if jobID == "" {
		ch := make(chan []byte, 1)
		ch <- []byte("[Trae Online] 创建任务成功但响应缺少 id 字段\n")
		close(ch)
		return &containerStreamResult{StatusCode: 200, Chunks: ch}
	}

	ch := make(chan []byte, 8)
	go func() {
		defer close(ch)
		detailURL := base + "/api/jobs/" + jobID
		pollClient := &http.Client{Timeout: 60 * time.Second}
		lastLen := 0
		deadline := time.Now().Add(3600 * time.Second)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
			if err != nil {
				return
			}
			pollReq.Header.Set("X-Access-Token", token)
			pollReq.Header.Set("Accept", "application/json")
			tracelog.ApplyOutboundHeaders(pollReq, ctx)
			gr, err := pollClient.Do(pollReq)
			if err != nil {
				ch <- []byte("\n[Trae Online 轮询失败] " + err.Error() + "\n")
				return
			}
			graw, _ := io.ReadAll(gr.Body)
			gr.Body.Close()
			if gr.StatusCode >= 400 {
				ch <- []byte(fmt.Sprintf("\n[Trae Online 轮询失败] HTTP %d\n", gr.StatusCode))
				return
			}
			var jd map[string]interface{}
			if json.Unmarshal(graw, &jd) != nil {
				ch <- []byte("\n[Trae Online 轮询失败] 非 JSON 响应\n")
				return
			}
			text := ""
			if out, ok := jd["output"].(string); ok {
				text = out
			}
			if len(text) > lastLen {
				ch <- []byte(text[lastLen:])
				lastLen = len(text)
			}
			st := strings.TrimSpace(fmt.Sprintf("%v", jd["status"]))
			if st == "completed" || st == "failed" || st == "interrupted" {
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	}()
	return &containerStreamResult{StatusCode: 200, Chunks: ch}
}

func openContainerInstructStream(
	ctx context.Context,
	baseURL, token, streamBackend,
	tenantID, workspaceID, taskID, instructID, prompt, traceID string,
	jobCtx containerJobContext,
) *containerStreamResult {
	if streamBackend == "trae" {
		if strings.TrimSpace(token) == "" {
			return &containerStreamResult{StatusCode: 400, Err: fmt.Errorf("trae stream requires access token")}
		}
		return streamTraeInstruct(ctx, baseURL, token, instructID, prompt, traceID, jobCtx)
	}
	return streamLegacyInstruct(ctx, baseURL, tenantID, workspaceID, taskID, instructID, prompt, traceID)
}

// decodeChunkText is used in tests to read full stream.
func readStreamChunks(ch <-chan []byte) string {
	var merged strings.Builder
	for chunk := range ch {
		merged.Write(chunk)
	}
	return merged.String()
}

func drainReader(r io.Reader) string {
	sc := bufio.NewScanner(r)
	var b strings.Builder
	for sc.Scan() {
		b.WriteString(sc.Text())
	}
	return b.String()
}
