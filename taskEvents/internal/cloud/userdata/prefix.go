package userdata

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func commentPathSegment(commentID string) string {
	cid := strings.TrimSpace(commentID)
	if cid == "" || cid == "-" {
		return ""
	}
	return "/comment/" + cid
}

func buildTaskCloudPrefix(taskAPIEndpoint, tenantID, workspaceID, taskID, commentID string) string {
	origin := strings.TrimRight(strings.TrimSpace(taskAPIEndpoint), "/")
	tid := strings.TrimSpace(tenantID)
	wid := strings.TrimSpace(workspaceID)
	tk := strings.TrimSpace(taskID)
	seg := commentPathSegment(commentID)
	if origin == "" || tid == "" || wid == "" || tk == "" || seg == "" {
		return ""
	}
	return fmt.Sprintf("%s/api/tenant/%s/workspace/%s/task/%s%s/cloud", origin, tid, wid, tk, seg)
}

func applyCloudPrefixRewrite(text string, p ReplaceParams) string {
	cp := buildTaskCloudPrefix(p.TaskAPIEndpoint, p.TenantID, p.WorkspaceID, p.TaskID, p.CommentID)
	bare := strings.TrimRight(strings.TrimSpace(p.TaskAPIEndpoint), "/")
	if text == "" || cp == "" || bare == "" {
		return text
	}
	out := rewriteBareAPIOriginTaskExports(text, cp, bareOriginStrings(bare))
	return rewriteBrowserTaskDetailURL(out, cp, p.TenantID, p.WorkspaceID, p.TaskID)
}

func bareOriginStrings(bare string) []string {
	variants := alternateSchemeOrigins(bare)
	seen := map[string]bool{}
	var out []string
	for _, v := range variants {
		for _, cand := range []string{v, v + "/"} {
			if cand != "" && !seen[cand] {
				seen[cand] = true
				out = append(out, cand)
			}
		}
	}
	return out
}

func alternateSchemeOrigins(origin string) []string {
	raw := strings.TrimRight(strings.TrimSpace(origin), "/")
	if raw == "" {
		return nil
	}
	toParse := raw
	if !strings.Contains(raw, "://") {
		toParse = "https://" + raw
	}
	u, err := url.Parse(toParse)
	if err != nil || u.Host == "" {
		return []string{raw}
	}
	path := u.Path
	httpsU := strings.TrimRight((&url.URL{Scheme: "https", Host: u.Host, Path: path}).String(), "/")
	httpU := strings.TrimRight((&url.URL{Scheme: "http", Host: u.Host, Path: path}).String(), "/")
	seen := map[string]bool{}
	var out []string
	for _, x := range []string{raw, httpsU, httpU} {
		if x != "" && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func rewriteBareAPIOriginTaskExports(text, cloudPrefix string, bareOrigins []string) string {
	if text == "" || cloudPrefix == "" {
		return text
	}
	out := text
	cp := strings.TrimRight(cloudPrefix, "/")
	for _, origin := range bareOrigins {
		if origin == "" || strings.TrimRight(origin, "/") == cp {
			continue
		}
		for _, q := range []string{`"`, `'`, ""} {
			for _, key := range []string{"TASK_API_ENDPOINT", "TaskApiEndPoint"} {
				if q != "" {
					out = strings.ReplaceAll(out, key+"="+q+origin+q, key+"="+q+cp+q)
					out = strings.ReplaceAll(out, "export "+key+"="+q+origin+q, "export "+key+"="+q+cp+q)
					out = strings.ReplaceAll(out, `-e "`+key+"="+origin+`"`, `-e "`+key+"="+cp+`"`)
					out = strings.ReplaceAll(out, "-e '"+key+"="+origin+"'", "-e '"+key+"="+cp+"'")
				} else {
					out = strings.ReplaceAll(out, "-e "+key+"="+origin, "-e "+key+"="+cp)
				}
			}
		}
	}
	return out
}

func rewriteBrowserTaskDetailURL(text, cloudPrefix, tenantID, workspaceID, taskID string) string {
	tid := strings.TrimSpace(tenantID)
	wid := strings.TrimSpace(workspaceID)
	tk := strings.TrimSpace(taskID)
	if text == "" || cloudPrefix == "" || tid == "" || wid == "" || tk == "" {
		return text
	}
	cp := strings.TrimRight(cloudPrefix, "/")
	pat := regexp.MustCompile(
		`(?i)https?://[^\s'"]+/tenant/` + regexp.QuoteMeta(tid) +
			`/workspace/` + regexp.QuoteMeta(wid) +
			`/task-detail/` + regexp.QuoteMeta(tk) + `/?`,
	)
	return pat.ReplaceAllString(text, cp)
}

func rewriteLocalhostVerifyHosts(plain, publicBase string) string {
	b := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	if b == "" || plain == "" {
		return plain
	}
	out := plain
	for _, hostPrefix := range []string{"http://127.0.0.1", "https://127.0.0.1", "http://localhost", "https://localhost"} {
		idx := 0
		for {
			pos := strings.Index(strings.ToLower(out[idx:]), strings.ToLower(hostPrefix))
			if pos < 0 {
				break
			}
			pos += idx
			rest := out[pos+len(hostPrefix):]
			portEnd := 0
			if strings.HasPrefix(rest, ":") {
				for i := 1; i < len(rest); i++ {
					if rest[i] < '0' || rest[i] > '9' {
						portEnd = i
						break
					}
				}
				if portEnd == 0 {
					break
				}
				rest = rest[portEnd:]
			}
			if !strings.HasPrefix(rest, "/api/cloud/server-userdata-verify/") &&
				!strings.Contains(rest, "/cloud/server-userdata-verify/") {
				idx = pos + len(hostPrefix)
				continue
			}
			old := out[pos : pos+len(hostPrefix)+portEnd]
			out = out[:pos] + b + out[pos+len(old):]
			idx = pos + len(b)
		}
	}
	return out
}
