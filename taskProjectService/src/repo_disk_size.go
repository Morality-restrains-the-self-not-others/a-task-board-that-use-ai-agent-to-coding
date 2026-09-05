package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const internalGitLabProviderKey = "gitlab:gitlab-local"

// Short TTL cache for GitLab repository_size to avoid repeated statistics calls
// when users refresh project detail with multiple internal repos.
const diskSizeCacheTTL = 60 * time.Second

type diskSizeCacheEntry struct {
	size      int64
	expiresAt time.Time
}

var (
	diskSizeCacheMu sync.Mutex
	diskSizeCache   = map[string]diskSizeCacheEntry{}
	diskSizeNow     = time.Now
)

func clearDiskSizeCache() {
	diskSizeCacheMu.Lock()
	diskSizeCache = map[string]diskSizeCacheEntry{}
	diskSizeCacheMu.Unlock()
}

func getCachedDiskSize(apiURL string) (int64, bool) {
	key := strings.TrimSpace(apiURL)
	if key == "" {
		return 0, false
	}
	now := diskSizeNow()
	diskSizeCacheMu.Lock()
	defer diskSizeCacheMu.Unlock()
	ent, ok := diskSizeCache[key]
	if !ok || !ent.expiresAt.After(now) {
		if ok {
			delete(diskSizeCache, key)
		}
		return 0, false
	}
	return ent.size, true
}

func putCachedDiskSize(apiURL string, size int64) {
	key := strings.TrimSpace(apiURL)
	if key == "" || size < 0 {
		return
	}
	diskSizeCacheMu.Lock()
	diskSizeCache[key] = diskSizeCacheEntry{
		size:      size,
		expiresAt: diskSizeNow().Add(diskSizeCacheTTL),
	}
	diskSizeCacheMu.Unlock()
}

// IsInternalRepo reports whether repoURL belongs to the platform gitService (gitlab-local).
// Matches by host/netloc against gitlab-local entries only — must not depend on
// matchProvider's first-hit winner when multiple providers share the same website host.
func (r *ProviderResolver) IsInternalRepo(repoURL string) bool {
	if r == nil {
		return false
	}
	host, netloc := extractHostNetloc(repoURL)
	if host == "" {
		return false
	}
	for i := range r.entries {
		e := &r.entries[i]
		if e.ProviderKey != internalGitLabProviderKey {
			continue
		}
		if e.Netloc != "" && netloc != "" && e.Netloc == netloc {
			return true
		}
		if e.Host != "" && e.Host == host {
			return true
		}
	}
	return false
}

func parseGitLabRepositorySizeBytes(body []byte) (int64, bool) {
	var payload struct {
		Statistics *struct {
			RepositorySize int64 `json:"repository_size"`
		} `json:"statistics"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Statistics == nil {
		return 0, false
	}
	if payload.Statistics.RepositorySize < 0 {
		return 0, false
	}
	return payload.Statistics.RepositorySize, true
}

func fetchGitLabRepositorySizeBytes(repoURL, token, sessionCookie string) (int64, bool) {
	parts, err := parseGitLabProjectParts(repoURL)
	if err != nil {
		return 0, false
	}
	apiURL := parts.APIBase + "?statistics=true"
	resp, err := gitlabRESTGet(apiURL, token, sessionCookie, defaultGitLabHost())
	if err != nil {
		log.Printf("[taskProjectService] disk-size gitlab request failed repo=%s err=%v", redactRepoURLForLog(repoURL), err)
		return 0, false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskProjectService] disk-size gitlab status=%d repo=%s", resp.StatusCode, redactRepoURLForLog(repoURL))
		return 0, false
	}
	size, ok := parseGitLabRepositorySizeBytes(body)
	if !ok {
		log.Printf("[taskProjectService] disk-size missing statistics repo=%s", redactRepoURLForLog(repoURL))
		return 0, false
	}
	return size, true
}

func redactRepoURLForLog(repoURL string) string {
	host, _ := extractHostNetloc(repoURL)
	if host == "" {
		return "(invalid)"
	}
	return host
}

// enrichProjectGitRepoDiskSizes sets is_internal on git_repo_entries and, for internal
// repos only, disk_size_bytes when GitLab statistics are available for the current user.
func enrichProjectGitRepoDiskSizes(detail map[string]interface{}, userID string, traceHeaders ...map[string]string) {
	trace := outboundTrace(traceHeaders...)
	if detail == nil {
		return
	}
	entries := projectGitRepoEntryMaps(detail)
	if len(entries) == 0 {
		return
	}

	type job struct {
		idx     int
		apiURL  string
		match   providerMatch
		hasSize bool
		size    int64
	}
	jobs := make([]job, 0, len(entries))
	for i, entry := range entries {
		url := strings.TrimSpace(fmt.Sprintf("%v", entry["url"]))
		if url == "" {
			url = strings.TrimSpace(fmt.Sprintf("%v", entry["repo_url"]))
		}
		entry["is_internal"] = false
		delete(entry, "disk_size_bytes")
		if url == "" {
			continue
		}
		apiURL := url
		if normalized, ok := normalizeGitRepoURLForBranchLookup(url); ok {
			apiURL = normalized
		}
		internal := providerResolver != nil && providerResolver.IsInternalRepo(apiURL)
		entry["is_internal"] = internal
		if !internal {
			continue
		}
		match := providerMatch{}
		if providerResolver != nil {
			match = providerResolver.matchProvider(apiURL)
		}
		jobs = append(jobs, job{idx: i, apiURL: apiURL, match: match})
	}

	if len(jobs) == 0 || strings.TrimSpace(userID) == "" {
		detail["git_repo_entries"] = entries
		return
	}

	uid := parseUserIDInt(userID)
	var wg sync.WaitGroup
	for j := range jobs {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			var token string
			if uid > 0 {
				token, _ = fetchGitAccessToken(uid, jobs[j].apiURL, jobs[j].match.GitoauthBase, trace)
			}
			if size, ok := getCachedDiskSize(jobs[j].apiURL); ok {
				jobs[j].hasSize = true
				jobs[j].size = size
				return
			}
			if token == "" {
				log.Printf("[taskProjectService] disk-size skip no token user=%s repo=%s", userID, redactRepoURLForLog(jobs[j].apiURL))
				return
			}
			size, ok := fetchGitLabRepositorySizeBytes(jobs[j].apiURL, token, "")
			if !ok {
				return
			}
			putCachedDiskSize(jobs[j].apiURL, size)
			jobs[j].hasSize = true
			jobs[j].size = size
		}(j)
	}
	wg.Wait()

	for _, j := range jobs {
		if !j.hasSize {
			continue
		}
		entries[j.idx]["disk_size_bytes"] = j.size
	}
	detail["git_repo_entries"] = entries
	log.Printf("[taskProjectService] disk-size enrich user=%s entries=%d internal_jobs=%d", userID, len(entries), len(jobs))
}

func handleProjectGitRepoDiskSizes(w http.ResponseWriter, r *http.Request, tenantID, projectID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthUser(w, r)
	if !ok {
		return
	}
	detail, err := loadProjectDetail(projectID)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if rejectIfProjectNotInTenant(w, r, detail, tenantID) {
		return
	}
	enrichProjectGitRepoDiskSizes(detail, userID, traceHeadersFromRequest(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"git_repo_entries": projectGitRepoEntryMaps(detail),
	})
}

func projectGitRepoEntryMaps(detail map[string]interface{}) []map[string]interface{} {
	raw, ok := detail["git_repo_entries"]
	if ok && raw != nil {
		switch entries := raw.(type) {
		case []map[string]interface{}:
			out := make([]map[string]interface{}, len(entries))
			for i, e := range entries {
				if e == nil {
					out[i] = map[string]interface{}{}
					continue
				}
				cp := make(map[string]interface{}, len(e)+2)
				for k, v := range e {
					cp[k] = v
				}
				out[i] = cp
			}
			return out
		case []interface{}:
			out := make([]map[string]interface{}, 0, len(entries))
			for _, item := range entries {
				m, ok := item.(map[string]interface{})
				if !ok || m == nil {
					continue
				}
				cp := make(map[string]interface{}, len(m)+2)
				for k, v := range m {
					cp[k] = v
				}
				out = append(out, cp)
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	urls := collectProjectGitRepoURLs(detail)
	out := make([]map[string]interface{}, 0, len(urls))
	for _, u := range urls {
		out = append(out, map[string]interface{}{
			"url":         u,
			"clone_alias": "",
		})
	}
	return out
}
