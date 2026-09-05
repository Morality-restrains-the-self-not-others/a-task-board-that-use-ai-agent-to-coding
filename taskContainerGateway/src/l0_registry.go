package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type forwardRequest struct {
	BaseURL string
	Scope   scope
	Method  string
	Query   url.Values
	Body    []byte
}

// scopedAPIRoot builds onlineServiceJS upstream root with task scope in the path
// (…/api/tenant/{t}/workspace/{w}/task/{task}), so container access logs retain scope.
func scopedAPIRoot(base string, sc scope) string {
	return baseTrimmed(base) +
		"/api/tenant/" + pathEscape(sc.TenantID) +
		"/workspace/" + pathEscape(sc.WorkspaceID) +
		"/task/" + pathEscape(sc.TaskID)
}

func (req forwardRequest) scopedAPI(suffix string) string {
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return scopedAPIRoot(req.BaseURL, req.Scope) + suffix
}

type forwardPlan struct {
	Method    string
	URL       string
	Body      []byte
	ErrDetail string
}

type l0ActionSpec struct {
	BrowserMethods []string
	Build          func(forwardRequest) forwardPlan
}

func strField(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func queryRequired(q url.Values, key string) (string, string) {
	v := strings.TrimSpace(q.Get(key))
	if v == "" {
		return "", key + " 必填"
	}
	return v, ""
}

func baseTrimmed(base string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/")
}

func pathEscape(seg string) string {
	return url.PathEscape(strings.TrimSpace(seg))
}

// pathEscapeRelPosix 按路径段分别 PathEscape，保留 / 作为分隔符。
// 不可对整段 relPath 使用 url.PathEscape，否则 foo/bar 会变成 foo%2Fbar，
// Express `/files/*` 无法按目录层级解析，容器侧会 404 not found。
func pathEscapeRelPosix(relPath string) (string, string) {
	rel := strings.ReplaceAll(strings.TrimSpace(relPath), "\\", "/")
	rel = strings.Trim(rel, "/")
	if rel == "" {
		return "", "path 必填"
	}
	parts := strings.Split(rel, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if p == "." || p == ".." {
			return "", "path 无效"
		}
		out = append(out, url.PathEscape(p))
	}
	if len(out) == 0 {
		return "", "path 必填"
	}
	return strings.Join(out, "/"), ""
}

func parseJSONBody(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return map[string]any{}
	}
	return body
}

func passthroughBody(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}

var l0ActionRegistry = map[string]l0ActionSpec{
	"container-clone-log": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/repos/clone-log/" + pathEscape(layerID))
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-bootstrap-clone-log": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			u := req.scopedAPI("/repos/bootstrap-clone-log")
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-auto-run-steps": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			u := req.scopedAPI("/auto-run-steps")
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layers-empty-root": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			u := req.scopedAPI("/layers/empty-root")
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-file-content": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			relPath, errDetail := queryRequired(req.Query, "path")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			escapedPath, errDetail := pathEscapeRelPosix(relPath)
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/files/" + escapedPath)
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-files": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			maxFiles := 3000
			if raw := strings.TrimSpace(req.Query.Get("max_files")); raw != "" {
				if n, err := strconv.Atoi(raw); err == nil {
					maxFiles = n
				} else {
					return forwardPlan{ErrDetail: "max_files 必须为整数"}
				}
			}
			if maxFiles < 1 {
				maxFiles = 1
			}
			if maxFiles > 5000 {
				maxFiles = 5000
			}
			q := url.Values{}
			q.Set("max_files", strconv.Itoa(maxFiles))
			if p := strings.TrimSpace(req.Query.Get("path")); p != "" {
				q.Set("path", p)
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/files?" + q.Encode())
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-diff-parent-files": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			q := url.Values{}
			if raw := strings.TrimSpace(req.Query.Get("offset")); raw != "" {
				q.Set("offset", raw)
			}
			if raw := strings.TrimSpace(req.Query.Get("limit")); raw != "" {
				q.Set("limit", raw)
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/diff/parent/files")
			if enc := q.Encode(); enc != "" {
				u = u + "?" + enc
			}
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-children": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			q := url.Values{}
			if dir := strings.TrimSpace(req.Query.Get("dir")); dir != "" {
				q.Set("dir", dir)
			}
			if offset := strings.TrimSpace(req.Query.Get("offset")); offset != "" {
				q.Set("offset", offset)
			}
			if limit := strings.TrimSpace(req.Query.Get("limit")); limit != "" {
				q.Set("limit", limit)
			}
			if prefix := strings.TrimSpace(req.Query.Get("prefix")); prefix != "" {
				q.Set("prefix", prefix)
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/children")
			if enc := q.Encode(); enc != "" {
				u = u + "?" + enc
			}
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-git-log": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			q := url.Values{}
			if p := strings.TrimSpace(req.Query.Get("path")); p != "" {
				q.Set("path", p)
			}
			if raw := strings.TrimSpace(req.Query.Get("limit")); raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil {
					return forwardPlan{ErrDetail: "limit 必须为整数"}
				}
				if n < 1 {
					n = 1
				}
				if n > 100 {
					n = 100
				}
				q.Set("limit", strconv.Itoa(n))
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/log")
			if enc := q.Encode(); enc != "" {
				u = u + "?" + enc
			}
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-git-repo-identities": {
		BrowserMethods: []string{http.MethodGet},
		Build: func(req forwardRequest) forwardPlan {
			layerID, errDetail := queryRequired(req.Query, "layer_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/repo-identities")
			return forwardPlan{Method: http.MethodGet, URL: u}
		},
	},
	"container-layer-delete": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			layerID := strField(body, "layer_id")
			if layerID == "" {
				return forwardPlan{ErrDetail: "layer_id 必填"}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID))
			return forwardPlan{Method: http.MethodDelete, URL: u}
		},
	},
	"container-layer-create": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			if strField(body, "parent_layer_id") == "" {
				return forwardPlan{ErrDetail: "parent_layer_id 必填"}
			}
			u := req.scopedAPI("/layers")
			out := map[string]any{"parent_layer_id": strField(body, "parent_layer_id")}
			if v, ok := body["name"]; ok {
				out["name"] = v
			}
			raw, _ := json.Marshal(out)
			return forwardPlan{Method: http.MethodPost, URL: u, Body: raw}
		},
	},
	"container-layer-git-add": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			layerID := strField(body, "layer_id")
			if layerID == "" {
				return forwardPlan{ErrDetail: "layer_id 必填"}
			}
			pathVal := strField(body, "path")
			if pathVal == "" {
				return forwardPlan{ErrDetail: "path 必填且须为字符串"}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/add")
			raw, _ := json.Marshal(map[string]string{"path": pathVal})
			return forwardPlan{Method: http.MethodPost, URL: u, Body: raw}
		},
	},
	"container-layer-git-unstage": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			layerID := strField(body, "layer_id")
			if layerID == "" {
				return forwardPlan{ErrDetail: "layer_id 必填"}
			}
			pathVal := strField(body, "path")
			if pathVal == "" {
				return forwardPlan{ErrDetail: "path 必填且须为字符串"}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/unstage")
			raw, _ := json.Marshal(map[string]string{"path": pathVal})
			return forwardPlan{Method: http.MethodPost, URL: u, Body: raw}
		},
	},
	"container-layer-git-diff-log": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			layerID := strField(body, "layer_id")
			if layerID == "" {
				return forwardPlan{ErrDetail: "layer_id 必填"}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/diff-log")
			out := map[string]any{}
			for _, k := range []string{"path", "staged", "context_lines"} {
				if v, ok := body[k]; ok {
					out[k] = v
				}
			}
			raw, _ := json.Marshal(out)
			return forwardPlan{Method: http.MethodPost, URL: u, Body: raw}
		},
	},
	"container-layer-git-merge": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			layerID := strField(body, "layer_id")
			if layerID == "" {
				return forwardPlan{ErrDetail: "layer_id 必填"}
			}
			targetBranch := strField(body, "target_branch")
			if targetBranch == "" {
				return forwardPlan{ErrDetail: "target_branch 必填"}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/merge")
			out := map[string]any{"target_branch": targetBranch}
			if src := strField(body, "source_ref"); src != "" {
				out["source_ref"] = src
			}
			raw, _ := json.Marshal(out)
			return forwardPlan{Method: http.MethodPost, URL: u, Body: raw}
		},
	},
	// NOTE: browser body is rewritten by handleContainerLayerGitRepoIdentitiesSync
	// (cloud prepare → repo_match_key/user_name/user_email). Registry entry kept for
	// action discovery; Build is unused when the special handler is wired.
	"container-layer-git-repo-identities-sync": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			body := parseJSONBody(req.Body)
			layerID := strField(body, "layer_id")
			if layerID == "" {
				return forwardPlan{ErrDetail: "layer_id 必填"}
			}
			u := req.scopedAPI("/layers/" + pathEscape(layerID) + "/git/repo-identities/sync")
			return forwardPlan{Method: http.MethodPost, URL: u, Body: passthroughBody(req.Body)}
		},
	},
	"container-layer-command": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			u := req.scopedAPI("/jobs")
			return forwardPlan{Method: http.MethodPost, URL: u, Body: passthroughBody(req.Body)}
		},
	},
	"container-job-redo": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			jobID, errDetail := queryRequired(req.Query, "job_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/jobs/" + pathEscape(jobID) + "/redo")
			return forwardPlan{Method: http.MethodPost, URL: u}
		},
	},
	"container-job-interrupt": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			jobID, errDetail := queryRequired(req.Query, "job_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/jobs/" + pathEscape(jobID) + "/interrupt")
			return forwardPlan{Method: http.MethodPost, URL: u}
		},
	},
	"container-task-lifecycle-shutdown": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			u := req.scopedAPI("/task-lifecycle/shutdown")
			return forwardPlan{Method: http.MethodPost, URL: u}
		},
	},
	"container-task-lifecycle-closing-soon": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			u := req.scopedAPI("/task-lifecycle/closing-soon")
			return forwardPlan{Method: http.MethodPost, URL: u}
		},
	},
	"container-job-continue": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			jobID, errDetail := queryRequired(req.Query, "job_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/jobs/" + pathEscape(jobID) + "/continue")
			return forwardPlan{Method: http.MethodPost, URL: u}
		},
	},
	"container-job-delete": {
		BrowserMethods: []string{http.MethodPost},
		Build: func(req forwardRequest) forwardPlan {
			jobID, errDetail := queryRequired(req.Query, "job_id")
			if errDetail != "" {
				return forwardPlan{ErrDetail: errDetail}
			}
			u := req.scopedAPI("/jobs/" + pathEscape(jobID))
			return forwardPlan{Method: http.MethodDelete, URL: u}
		},
	},
}

func methodAllowed(spec l0ActionSpec, method string) bool {
	m := strings.ToUpper(strings.TrimSpace(method))
	for _, allowed := range spec.BrowserMethods {
		if strings.ToUpper(allowed) == m {
			return true
		}
	}
	return false
}

func lookupL0Action(action string) (l0ActionSpec, bool) {
	spec, ok := l0ActionRegistry[action]
	return spec, ok
}
