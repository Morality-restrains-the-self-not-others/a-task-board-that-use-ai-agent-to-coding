package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

type chatRequest struct {
	Model  string          `json:"model"`
	Stream bool            `json:"stream"`
	Raw    json.RawMessage `json:"-"`
}

func parseChatRequest(body []byte) (chatRequest, map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return chatRequest{}, nil, err
	}
	cr := chatRequest{}
	if model, ok := m["model"].(string); ok {
		cr.Model = model
	}
	if stream, ok := m["stream"].(bool); ok {
		cr.Stream = stream
	}
	return cr, m, nil
}

func injectStreamUsageOption(body map[string]any) []byte {
	if body == nil {
		body = map[string]any{}
	}
	streamOpts, _ := body["stream_options"].(map[string]any)
	if streamOpts == nil {
		streamOpts = map[string]any{}
	}
	streamOpts["include_usage"] = true
	body["stream_options"] = streamOpts
	raw, _ := json.Marshal(body)
	return raw
}

func forwardUpstream(w http.ResponseWriter, r *http.Request, upstreamURL, apiKey string, body []byte, stream bool) (inputTokens, outputTokens int) {
	timeout := time.Duration(cfg.UpstreamTimeoutSec * float64(time.Second))
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": err.Error()})
		return 0, 0
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	tracelog.ApplyOutboundHeaders(req, r.Context())

	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": fmt.Sprintf("upstream error: %v", err)})
		return 0, 0
	}
	defer resp.Body.Close()

	if stream {
		return pipeSSE(w, resp)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": "failed to read upstream"})
		return 0, 0
	}
	for k, vals := range resp.Header {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
	inTok, outTok := parseUsageFromJSON(respBody)
	return inTok, outTok
}

func pipeSSE(w http.ResponseWriter, resp *http.Response) (int, int) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "streaming not supported"})
		return 0, 0
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)

	scanner := bufio.NewScanner(resp.Body)
	var inputTokens, outputTokens int
	for scanner.Scan() {
		line := scanner.Text()
		_, _ = w.Write([]byte(line + "\n"))
		flusher.Flush()
		in, out := parseUsageFromSSELine(line)
		if in > 0 {
			inputTokens = in
		}
		if out > 0 {
			outputTokens = out
		}
	}
	return inputTokens, outputTokens
}

func parseUsageFromSSELine(line string) (int, int) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "data:") {
		return 0, 0
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "" || data == "[DONE]" {
		return 0, 0
	}
	return parseUsageFromJSON([]byte(data))
}

func parseUsageFromJSON(raw []byte) (int, int) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0, 0
	}
	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		return 0, 0
	}
	inTok := intFromAny(usage["prompt_tokens"])
	if inTok == 0 {
		inTok = intFromAny(usage["input_tokens"])
	}
	outTok := intFromAny(usage["completion_tokens"])
	if outTok == 0 {
		outTok = intFromAny(usage["output_tokens"])
	}
	return inTok, outTok
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case json.Number:
		i, _ := t.Int64()
		return int(i)
	default:
		return 0
	}
}
