package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gatewaycors"
)

type routeMatch struct {
	Scope  scope
	Action string
}

func parseContainerComputePath(path string) (routeMatch, bool) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	// 旧形态: api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/{action}
	if len(parts) >= 10 && parts[0] == "api" && parts[1] == "tenant" && parts[3] == "workspace" && parts[5] == "task" && parts[7] == "cloud" && parts[8] == "compute" {
		action := strings.TrimSuffix(parts[9], "/")
		if action == "" || !strings.HasPrefix(action, "container-") {
			return routeMatch{}, false
		}
		// 防御：task_id 值不应含 "container-"（残缺形态 funcName 粘尾，见下方新形态注释）
		if strings.Contains(parts[6], "container-") {
			return routeMatch{}, false
		}
		sc := scope{
			TenantID:    parts[2],
			WorkspaceID: parts[4],
			TaskID:      parts[6],
		}
		if len(parts) >= 12 && parts[10] == "comment_id" {
			sc.CommentID = parts[11]
		}
		return routeMatch{Scope: sc, Action: action}, true
	}
	// 新形态（kv-last 约定，2026-08-04 前端约定迁移）:
	//   api/cloud/compute/{action}/tenant_id/{t}/workspace_id/{w}/task_id/{task}/comment_id/{cid}/
	//   api/cloud/compute/tenant_id/{t}/workspace_id/{w}/task_id/{task}/comment_id/{cid}/{action}/
	// comment_id 为评论级 CSC 分片键，放 path 而非 query（ADR-0010）。
	// kv 对可出现在任意位置：剔除 kv 段后，剩余段中的 container-* 段即 action。
	// 防御：task_id 值不应含 "container-" —— 前端残缺形态 `task_id/{值}{funcName}`
	// （funcName 缺 / 分隔粘在 task_id 值后）会把粘尾当 task_id 值误接受，须拒绝。
	if len(parts) >= 4 && parts[0] == "api" && parts[1] == "cloud" && parts[2] == "compute" {
		var sc scope
		action := ""
		for i := 3; i < len(parts); i++ {
			switch parts[i] {
			case "tenant_id", "workspace_id", "task_id", "comment_id":
				if i+1 >= len(parts) {
					return routeMatch{}, false
				}
				switch parts[i] {
				case "tenant_id":
					sc.TenantID = parts[i+1]
				case "workspace_id":
					sc.WorkspaceID = parts[i+1]
				case "task_id":
					if strings.Contains(parts[i+1], "container-") {
						return routeMatch{}, false
					}
					sc.TaskID = parts[i+1]
				case "comment_id":
					sc.CommentID = parts[i+1]
				}
				i++
			default:
				if parts[i] == "" {
					continue
				}
				if strings.HasPrefix(parts[i], "container-") {
					if action != "" {
						return routeMatch{}, false
					}
					action = parts[i]
				} else {
					return routeMatch{}, false
				}
			}
		}
		if action == "" || sc.TenantID == "" || sc.WorkspaceID == "" || sc.TaskID == "" {
			return routeMatch{}, false
		}
		return routeMatch{Scope: sc, Action: action}, true
	}
	return routeMatch{}, false
}

func containerPageURLFromBody(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	v, _ := body["container_page_url"].(string)
	return strings.TrimSpace(v)
}

func commentIDFromBody(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return ""
	}
	v, _ := body["comment_id"].(string)
	return strings.TrimSpace(v)
}

func containerPageURLFromRequest(r *http.Request, raw []byte) string {
	if u := containerPageURLFromBody(raw); u != "" {
		return u
	}
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("container_page_url"))
}

func commentIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "comment_id" {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func commentIDFromRequest(r *http.Request, raw []byte) string {
	if r != nil {
		if p := commentIDFromPath(r.URL.Path); p != "" {
			return p
		}
		if q := strings.TrimSpace(r.URL.Query().Get("comment_id")); q != "" {
			return q
		}
	}
	return commentIDFromBody(raw)
}

// lookupCommentID prefers the scoped path comment_id (ADR-0010) over query.
func lookupCommentID(r *http.Request, sc scope) string {
	if cid := strings.TrimSpace(sc.CommentID); cid != "" {
		return cid
	}
	return commentIDFromRequest(r, nil)
}

func buildGitCommitForward(baseURL string, sc scope, rawBody []byte) (upstreamURL string, forwardBody []byte, errDetail string) {
	var body map[string]any
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return "", nil, "invalid json body"
	}
	layerID := strings.TrimSpace(fmt.Sprintf("%v", body["layer_id"]))
	if layerID == "" || layerID == "<nil>" {
		return "", nil, "layer_id required"
	}
	upstreamURL = scopedAPIRoot(baseURL, sc) + "/layers/" + url.PathEscape(layerID) + "/git/commit"
	out := map[string]any{}
	if msg, ok := body["message"].(string); ok && strings.TrimSpace(msg) != "" {
		out["message"] = strings.TrimSpace(msg)
	}
	if sa, ok := body["stage_all"]; ok {
		out["stage_all"] = sa
	}
	forwardBody, err := json.Marshal(out)
	if err != nil {
		return "", nil, "failed to encode forward body"
	}
	return upstreamURL, forwardBody, ""
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskContainerGateway"})
}

func handleContainerCompute(w http.ResponseWriter, r *http.Request) {
	match, ok := parseContainerComputePath(r.URL.Path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"detail": "unknown container compute path",
		})
		return
	}
	ctx := r.Context()

	session, status, body := authorizeContainerRequest(ctx, r, match.Scope)
	if status != http.StatusOK {
		writeRawJSON(w, status, body)
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "failed to read body"})
		return
	}

	// auth-context 无需 resolve 容器目标
	if match.Action == "container-layer-git-push-auth-context" {
		handleContainerLayerGitPushAuthContext(w, r, match.Scope, session)
		return
	}

	commentID := strings.TrimSpace(match.Scope.CommentID)
	if commentID == "" {
		commentID = commentIDFromRequest(r, rawBody)
	}
	target, status, body := cloudResolveTarget(ctx, match.Scope, containerPageURLFromRequest(r, rawBody), commentID)
	if status != http.StatusOK {
		writeRawJSON(w, status, body)
		return
	}

	switch match.Action {
	case "container-layer-git-push":
		handleContainerLayerGitPush(w, r, match.Scope, target, session, rawBody)
	case "container-layer-git-repo-identities-sync":
		handleContainerLayerGitRepoIdentitiesSync(w, r, match.Scope, target, session, rawBody)
	case "container-layer-git-commit":
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use POST"})
			return
		}
		upstreamURL, forwardBody, errDetail := buildGitCommitForward(target.BaseURL, match.Scope, rawBody)
		if errDetail != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": errDetail})
			return
		}
		upStatus, upBody, _ := forwardToContainerService(ctx, match.Scope, http.MethodPost, upstreamURL, target.AccessToken, forwardBody)
		maybeStartJobStreams(ctx, match.Scope, target, match.Action, upStatus, upBody)
		writeRawJSON(w, upStatus, upBody)
	case "container-layer-graph":
		handleContainerLayerGraph(w, r, match.Scope, target)
	case "container-job-execution-log":
		handleContainerJobExecutionLog(w, r, match.Scope, target)
	case "container-job-edit-run":
		handleContainerJobEditRun(w, r, match.Scope, target, rawBody)
	default:
		spec, inRegistry := lookupL0Action(match.Action)
		if !inRegistry {
			writeJSON(w, http.StatusNotImplemented, map[string]string{"detail": "unsupported action: " + match.Action})
			return
		}
		if !methodAllowed(spec, r.Method) {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed for action: " + match.Action})
			return
		}
		plan := spec.Build(forwardRequest{
			BaseURL: target.BaseURL,
			Scope:   match.Scope,
			Method:  r.Method,
			Query:   r.URL.Query(),
			Body:    rawBody,
		})
		if plan.ErrDetail != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": plan.ErrDetail})
			return
		}
		upStatus, upBody, _ := forwardToContainerService(ctx, match.Scope, plan.Method, plan.URL, target.AccessToken, plan.Body)
		maybeStartJobStreams(ctx, match.Scope, target, match.Action, upStatus, upBody)
		writeRawJSON(w, upStatus, upBody)
	}

}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func checkOnlineServiceReady(ctx context.Context, baseURL, accessToken string) (bool, string) {
	if baseURL == "" {
		return false, "container base url not configured"
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return false, "invalid container base url"
	}
	addr := u.Host
	if !strings.Contains(addr, ":") {
		addr = addr + ":8765"
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false, fmt.Sprintf("container online service unreachable: %v", err)
	}
	conn.Close()
	return true, ""
}

type layerGraphResponse struct {
	Layers           []any  `json:"layers"`
	Jobs             []any  `json:"jobs"`
	LayersRoot       string `json:"layers_root"`
	BootstrapLayerID string `json:"bootstrap_layer_id"`
}

func handleContainerLayerGraph(w http.ResponseWriter, r *http.Request, sc scope, target containerTarget) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use GET"})
		return
	}
	ctx := r.Context()

	ok, msg := checkOnlineServiceReady(ctx, target.BaseURL, target.AccessToken)
	if !ok {
		// OPT-20260818-008：dial 拒绝即推测性 server_url 失效 → 通知 cloud 清地址并降回 starting
		if strings.Contains(strings.ToLower(msg), "connection refused") {
			notifyCloudContainerUnreachable(ctx, sc)
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"detail":           "容器内在线服务未启动，请先启动容器",
			"container_status": "offline",
			"diagnosis":        msg,
		})
		return
	}

	root := scopedAPIRoot(target.BaseURL, sc)
	layersURL := root + "/layers"
	jobsURL := root + "/jobs"

	type fetchResult struct {
		Body   []byte
		Status int
		Err    string
	}

	fetch := func(url string) fetchResult {
		status, body, _ := forwardToContainerService(ctx, sc, http.MethodGet, url, target.AccessToken, nil)
		if status >= 400 {
			return fetchResult{Status: status, Err: string(body)}
		}
		return fetchResult{Body: body, Status: status}
	}

	layersRes := fetch(layersURL)
	if layersRes.Err != "" || layersRes.Status >= 400 {
		if len(layersRes.Body) > 0 {
			writeRawJSON(w, layersRes.Status, layersRes.Body)
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": "layers fetch failed: " + layersRes.Err})
		return
	}

	var layersBody map[string]any
	if err := json.Unmarshal(layersRes.Body, &layersBody); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": "invalid layers response"})
		return
	}

	layers, _ := layersBody["layers"].([]any)
	if layers == nil {
		layers = []any{}
	}
	lr, _ := layersBody["layers_root"].(string)
	bs := strings.TrimSpace(fmt.Sprintf("%v", layersBody["bootstrap_layer_id"]))
	if bs == "<nil>" || bs == "" {
		bs = ""
	}

	// jobs 可选：旧镜像 /api/jobs 可能含巨量 output 超时；层行仍可凭 layer.job_status 展示
	jobs := []any{}
	jobsRes := fetch(jobsURL)
	if jobsRes.Err == "" && jobsRes.Status < 400 {
		var jobsBody map[string]any
		if err := json.Unmarshal(jobsRes.Body, &jobsBody); err == nil {
			if j, ok := jobsBody["jobs"].([]any); ok && j != nil {
				jobs = j
			}
		}
	}

	writeJSON(w, http.StatusOK, layerGraphResponse{
		Layers:           layers,
		Jobs:             jobs,
		LayersRoot:       strings.TrimSpace(lr),
		BootstrapLayerID: strings.TrimSpace(bs),
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/internal/start-job-stream/", handleStartJobStream)
	mux.HandleFunc("/api/internal/start-job-stream", handleStartJobStream)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handleHealth(w, r)
			return
		}
		if strings.Contains(r.URL.Path, "relay-to-trae") {
			handleRelayToTrae(w, r)
			return
		}
		// kv-last（/cloud/compute/tenant_id/.../container-clone-log）不含
		// "/cloud/compute/container-" 子串；旧 Contains 会落到 net/http 默认 404，
		// 前端克隆日志/任务日志只显示裸「404」。parse 再拒绝非 container-*。
		if strings.Contains(r.URL.Path, "/cloud/compute/") {
			handleContainerCompute(w, r)
			return
		}
		http.NotFound(w, r)
	})
}
