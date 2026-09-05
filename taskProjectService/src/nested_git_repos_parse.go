package main

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const nestedSourceGitmodules = "gitmodules"

type nestedRepoCandidate struct {
	Path   string
	URL    string
	Source string
}

// NestedRepo is the API-facing nested git repo row.
type NestedRepo struct {
	Path         string `json:"path"`
	URL          string `json:"url"`
	Source       string `json:"source"`
	ResolveError string `json:"resolve_error,omitempty"`
}

var (
	gitmodulesPathRe = regexp.MustCompile(`(?m)^\s*path\s*=\s*(.+)$`)
	gitmodulesURLRe  = regexp.MustCompile(`(?m)^\s*url\s*=\s*(.+)$`)
)

func parseGitmodules(content string) []nestedRepoCandidate {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	parts := regexp.MustCompile(`(?m)^\s*\[submodule\s+[^\]]+\]\s*$`).Split(content, -1)
	out := make([]nestedRepoCandidate, 0)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pathM := gitmodulesPathRe.FindStringSubmatch(part)
		urlM := gitmodulesURLRe.FindStringSubmatch(part)
		if pathM == nil {
			continue
		}
		path := strings.TrimSpace(pathM[1])
		path = strings.Trim(path, "/")
		if path == "" {
			continue
		}
		moduleURL := ""
		if urlM != nil {
			moduleURL = strings.TrimSpace(urlM[1])
		}
		out = append(out, nestedRepoCandidate{
			Path:   path,
			URL:    moduleURL,
			Source: nestedSourceGitmodules,
		})
	}
	return out
}

// isAbsoluteGitRemoteURL reports whether u is an absolute Git remote
// (has a scheme, or scp-like user@host:path / host:path).
func isAbsoluteGitRemoteURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	if strings.Contains(u, "://") {
		return true
	}
	// Relative forms start with ./ ../ or are bare path segments.
	if strings.HasPrefix(u, "./") || strings.HasPrefix(u, "../") || strings.HasPrefix(u, "/") {
		return false
	}
	// scp-like: git@host:path or host.xz:foo/bar.git (colon before any /)
	if i := strings.IndexByte(u, ':'); i > 0 && !strings.Contains(u[:i], "/") {
		return true
	}
	return false
}

// resolveSubmoduleURL resolves a .gitmodules url against the superproject remote
// using Git's relative-URL rules (git help submodule): relative URLs are evaluated
// like relative directories, so a sibling of bar.git is written ../foo.git and the
// parent remote is treated as a directory for joining.
func resolveSubmoduleURL(parentRemote, moduleURL string) (resolved string, resolveError string) {
	parentRemote = strings.TrimSpace(parentRemote)
	moduleURL = strings.TrimSpace(moduleURL)
	if moduleURL == "" {
		return "", "submodule url is empty"
	}
	if isAbsoluteGitRemoteURL(moduleURL) {
		return moduleURL, ""
	}
	if parentRemote == "" {
		return "", "missing parent remote for relative submodule url"
	}
	joined, err := joinGitRemoteAsDirectory(parentRemote, moduleURL)
	if err != "" {
		return "", err
	}
	return joined, ""
}

// joinGitRemoteAsDirectory joins rel onto parentRemote as Git does for submodule
// relative URLs: treat parentRemote as a directory (append "/"), then resolve.
func joinGitRemoteAsDirectory(parentRemote, rel string) (string, string) {
	base := strings.TrimRight(parentRemote, "/") + "/"

	if strings.Contains(base, "://") {
		bu, err := url.Parse(base)
		if err != nil {
			return "", "invalid parent repository URL"
		}
		ref, err := url.Parse(rel)
		if err != nil {
			return "", "invalid relative submodule url"
		}
		out := bu.ResolveReference(ref).String()
		// ResolveReference may keep a trailing slash; strip for repo URLs.
		return strings.TrimRight(out, "/"), ""
	}

	// scp-like / opaque remotes: path-style resolution after the last ':'.
	colon := strings.LastIndex(base, ":")
	if colon < 0 {
		return "", "invalid parent repository URL"
	}
	prefix := base[:colon+1]
	pathPart := base[colon+1:]
	joined := pathCleanJoin(pathPart, rel)
	if joined == "" || strings.HasPrefix(joined, "..") {
		return "", "relative submodule url escapes remote root"
	}
	return prefix + joined, ""
}

// pathCleanJoin resolves rel against a slash-separated base path (may end with /).
func pathCleanJoin(basePath, rel string) string {
	basePath = strings.TrimPrefix(basePath, "/")
	parts := make([]string, 0)
	for _, p := range strings.Split(strings.TrimSuffix(basePath, "/"), "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	for _, p := range strings.Split(rel, "/") {
		switch p {
		case "", ".":
			continue
		case "..":
			if len(parts) == 0 {
				return ""
			}
			parts = parts[:len(parts)-1]
		default:
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "/")
}

func mergeNestedRepos(gitmodules []nestedRepoCandidate, parentHTTPS string) []NestedRepo {
	byPath := map[string]NestedRepo{}

	for _, c := range gitmodules {
		moduleURL := strings.TrimSpace(c.URL)
		resolveErr := ""
		resolved := ""
		if moduleURL == "" {
			resolveErr = "submodule url is empty"
		} else {
			resolved, resolveErr = resolveSubmoduleURL(parentHTTPS, moduleURL)
		}
		byPath[c.Path] = NestedRepo{
			Path:         c.Path,
			URL:          resolved,
			Source:       nestedSourceGitmodules,
			ResolveError: resolveErr,
		}
	}

	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]NestedRepo, 0, len(paths))
	for _, p := range paths {
		out = append(out, byPath[p])
	}
	return out
}
