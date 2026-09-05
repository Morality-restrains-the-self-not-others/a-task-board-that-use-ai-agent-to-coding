package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"tracelog"
)

type jobStreamConfig struct {
	PollIntervalSec float64
	PollConnectSec  float64
	PollReadSec     float64
	PollDeadlineSec float64
}

var (
	jobStreamCfg = jobStreamConfig{
		PollIntervalSec: 1.0,
		PollConnectSec:  5,
		PollReadSec:     30,
		PollDeadlineSec: 3600,
	}
	jobStreamActive   sync.Map // key taskID+"\x00"+jobID -> struct{}
	jobStreamTriggers = map[string]struct{}{
		"container-layer-command": {},
		"container-job-redo":      {},
		"container-job-continue":  {},
	}
)

func initJobStreamConfig() {
	jobStreamCfg.PollIntervalSec = envFloat("CONTAINER_JOB_STREAM_POLL_INTERVAL", jobStreamCfg.PollIntervalSec)
	jobStreamCfg.PollConnectSec = envFloat("CONTAINER_JOB_STREAM_POLL_CONNECT_TIMEOUT", jobStreamCfg.PollConnectSec)
	jobStreamCfg.PollReadSec = envFloat("CONTAINER_JOB_STREAM_POLL_READ_TIMEOUT", jobStreamCfg.PollReadSec)
	jobStreamCfg.PollDeadlineSec = envFloat("CONTAINER_JOB_STREAM_POLL_DEADLINE", jobStreamCfg.PollDeadlineSec)
}

func envFloat(key string, defaultVal float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return defaultVal
	}
	return v
}

func actionTriggersJobStream(action string) bool {
	_, ok := jobStreamTriggers[action]
	return ok
}

// jobStreamPollEnabled 网关轮询容器 events 的逃生开关。默认关闭：步骤由容器 PUSH → Kafka → SSE。
func jobStreamPollEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CONTAINER_JOB_STREAM_POLL_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

/** onlineServiceJS recordJobEvent 终态；勿依赖 GET /api/jobs/:id（旧镜像仍可能带巨量 output 导致轮询超时）。 */
func isTerminalJobEventPhase(phase string) bool {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "completed", "failed", "interrupted":
		return true
	default:
		return false
	}
}

func extractJobIDsFromResponse(body []byte) []string {
	if len(body) == 0 {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	var ids []string
	if id := jobIDFromMap(parsed); id != "" {
		ids = append(ids, id)
	}
	if jobs, ok := parsed["jobs"].([]any); ok {
		for _, item := range jobs {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if job, ok := row["job"].(map[string]any); ok {
				if id := jobIDFromMap(job); id != "" {
					ids = append(ids, id)
				}
			} else if id := jobIDFromMap(row); id != "" {
				ids = append(ids, id)
			}
		}
	}
	if job, ok := parsed["job"].(map[string]any); ok {
		if id := jobIDFromMap(job); id != "" && len(ids) == 0 {
			ids = append(ids, id)
		}
	}
	return dedupeStrings(ids)
}

func jobIDFromMap(m map[string]any) string {
	if id := strings.TrimSpace(fmt.Sprintf("%v", m["id"])); id != "" && id != "<nil>" {
		return id
	}
	if id := strings.TrimSpace(fmt.Sprintf("%v", m["job_id"])); id != "" && id != "<nil>" {
		return id
	}
	return ""
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func maybeStartJobStreams(ctx context.Context, sc scope, target containerTarget, action string, upStatus int, upBody []byte) {
	if !jobStreamPollEnabled() {
		return
	}
	if upStatus < 200 || upStatus >= 300 {
		return
	}
	if !actionTriggersJobStream(action) {
		return
	}
	for _, jobID := range extractJobIDsFromResponse(upBody) {
		startJobStream(ctx, sc, target, jobID)
	}
}

func startJobStream(parent context.Context, sc scope, target containerTarget, jobID string) {
	taskID := strings.TrimSpace(sc.TaskID)
	jid := strings.TrimSpace(jobID)
	if taskID == "" || jid == "" {
		return
	}
	key := taskID + "\x00" + jid
	if _, loaded := jobStreamActive.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	traceID := traceIDFromContext(parent)
	go func() {
		defer jobStreamActive.Delete(key)
		runJobStreamPoll(parent, sc, target, jid, traceID)
	}()
}

func runJobStreamPoll(parent context.Context, sc scope, target containerTarget, jobID, traceID string) {
	ctx := context.WithoutCancel(parent)
	deadline := time.Now().Add(time.Duration(jobStreamCfg.PollDeadlineSec * float64(time.Second)))
	root := scopedAPIRoot(target.BaseURL, sc)
	eventsURL := root + "/jobs/" + pathEscape(jobID) + "/events"
	jobURL := root + "/jobs/" + pathEscape(jobID)

	publishStart(ctx, sc, jobID, traceID)
	offset := 0
	for time.Now().Before(deadline) {
		eventsBody, err := fetchJobStreamJSON(ctx, http.MethodGet, eventsURL, target.AccessToken, map[string]string{
			"offset": strconv.Itoa(offset),
			"limit":  "500",
		})
		if err != nil {
			jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
				"phase": "error", "message": "轮询失败: " + err.Error(),
			})
			return
		}
		if st, ok := eventsBody["status_code"].(float64); ok && int(st) >= 400 {
			jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
				"phase": "error", "message": fmt.Sprintf("轮询失败: HTTP %d", int(st)),
			})
			return
		}
		body, _ := eventsBody["body"].(map[string]any)
		if body == nil {
			jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
				"phase": "error", "message": "轮询失败: 非 JSON 响应",
			})
			return
		}
		if events, ok := body["events"].([]any); ok {
			for _, item := range events {
				ev, ok := item.(map[string]any)
				if !ok {
					continue
				}
				phase := strings.TrimSpace(fmt.Sprintf("%v", ev["phase"]))
				msg := ev["message"]
				msgStr := ""
				if s, ok := msg.(string); ok {
					msgStr = s
				}
				payload := map[string]any{
					"phase":   phase,
					"message": msgStr,
					"event":   ev,
				}
				if phase == "" {
					payload["phase"] = "chunk"
				}
				jobStreamPublishFunc(ctx, sc, jobID, traceID, payload)
				if isTerminalJobEventPhase(phase) {
					st := strings.ToLower(strings.TrimSpace(phase))
					jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
						"phase": "done", "message": "", "job_status": st,
					})
					return
				}
			}
		}
		if nxt, ok := body["next_offset"].(float64); ok && int(nxt) >= offset {
			offset = int(nxt)
		}

		jobBody, err := fetchJobStreamJSON(ctx, http.MethodGet, jobURL, target.AccessToken, nil)
		if err == nil {
			if jb, ok := jobBody["body"].(map[string]any); ok {
				st := strings.TrimSpace(fmt.Sprintf("%v", jb["status"]))
				if st == "completed" || st == "failed" || st == "interrupted" {
					jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
						"phase": "done", "message": "", "job_status": st,
					})
					return
				}
			}
		}

		time.Sleep(time.Duration(jobStreamCfg.PollIntervalSec * float64(time.Second)))
	}
	jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
		"phase": "error", "message": "轮询超时",
	})
}

func publishStart(ctx context.Context, sc scope, jobID, traceID string) {
	jobStreamPublishFunc(ctx, sc, jobID, traceID, map[string]any{
		"phase": "start", "message": "", "event": map[string]any{"phase": "start"},
	})
}

func fetchJobStreamJSON(ctx context.Context, method, rawURL, accessToken string, query map[string]string) (map[string]any, error) {
	u := rawURL
	if len(query) > 0 {
		q := url.Values{}
		for k, v := range query {
			q.Set(k, v)
		}
		u = u + "?" + q.Encode()
	}
	timeout := time.Duration((jobStreamCfg.PollConnectSec + jobStreamCfg.PollReadSec) * float64(time.Second))
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	status, body, err := forwardToOnlineService(reqCtx, method, u, accessToken, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return map[string]any{"status_code": float64(status)}, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("invalid json")
	}
	return map[string]any{"body": parsed}, nil
}

// jobStreamPublishFunc is swapped in tests to assert publish without Kafka.
var jobStreamPublishFunc = publishContainerJobStreamSSE

func publishContainerJobStreamSSE(ctx context.Context, sc scope, jobID, traceID string, fields map[string]any) {
	phase := strings.TrimSpace(fmt.Sprintf("%v", fields["phase"]))
	if phase == "" || phase == "<nil>" {
		phase = "chunk"
	}
	message := ""
	if m, ok := fields["message"].(string); ok {
		message = m
	} else if fields["message"] != nil {
		message = strings.TrimSpace(fmt.Sprintf("%v", fields["message"]))
		if message == "<nil>" {
			message = ""
		}
	}
	statusData := map[string]any{
		"status":     "container_job_stream",
		"event_name": "server_status_update",
		"job_id":     jobID,
		"phase":      phase,
		"message":    message,
	}
	if ev, ok := fields["event"].(map[string]any); ok && len(ev) > 0 {
		statusData["event"] = ev
	}
	if js := strings.TrimSpace(fmt.Sprintf("%v", fields["job_status"])); js != "" && js != "<nil>" {
		statusData["job_status"] = js
	}
	if tid := strings.TrimSpace(traceID); tid != "" {
		statusData["trace_id"] = tid
	}
	start := time.Now()
	err := publishSSEMessage(ctx, sc.TaskID, statusData)
	tracelog.LogForwardStage(ctx, "job_stream_publish", map[string]any{
		"ok":          err == nil,
		"duration_ms": time.Since(start).Milliseconds(),
		"job_id":      jobID,
		"phase":       phase,
		"via":         "kafka_sse",
	})
	if err != nil {
		slog.Warn("container job-stream SSE publish failed", "task_id", sc.TaskID, "job_id", jobID, "error", err)
	}
}
