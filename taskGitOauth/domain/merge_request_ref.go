package domain

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	MergeProviderGitHub = "github"
	MergeProviderGitLab = "gitlab"
)

// MergeRequestRef is a parsed GitHub PR or GitLab MR URL.
type MergeRequestRef struct {
	Provider    string
	Host        string // hostname only (no port)
	Scheme      string // http or https from html_url
	Port        string // optional explicit port from html_url
	APIOrigin   string // optional provider website origin override (scheme+host[:port])
	ProjectPath string
	Number      int
	HTMLURL     string
}

func (r MergeRequestRef) Origin() string {
	if o := strings.TrimRight(strings.TrimSpace(r.APIOrigin), "/"); o != "" {
		return o
	}
	if r.Host == "" {
		return ""
	}
	scheme := strings.ToLower(strings.TrimSpace(r.Scheme))
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	host := r.Host
	if p := strings.TrimSpace(r.Port); p != "" {
		host = net.JoinHostPort(r.Host, p)
	}
	return scheme + "://" + host
}

// ParseMergeRequestURL parses GitHub /pull/N or GitLab /-/merge_requests/N URLs.
func ParseMergeRequestURL(raw string) (MergeRequestRef, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return MergeRequestRef{}, fmt.Errorf("html_url required")
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return MergeRequestRef{}, fmt.Errorf("invalid html_url")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return MergeRequestRef{}, fmt.Errorf("html_url scheme must be http or https")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return MergeRequestRef{}, fmt.Errorf("invalid html_url host")
	}
	scheme := strings.ToLower(u.Scheme)
	port := u.Port()
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if host == "github.com" || host == "www.github.com" {
		return parseGitHubPull(s, parts)
	}
	return parseGitLabMR(s, scheme, host, port, parts)
}

func parseGitHubPull(raw string, parts []string) (MergeRequestRef, error) {
	// owner/repo/pull/N
	if len(parts) < 4 || parts[2] != "pull" {
		return MergeRequestRef{}, fmt.Errorf("not a github pull request url")
	}
	n, err := strconv.Atoi(parts[3])
	if err != nil || n <= 0 {
		return MergeRequestRef{}, fmt.Errorf("invalid pull number")
	}
	return MergeRequestRef{
		Provider:    MergeProviderGitHub,
		Host:        "github.com",
		Scheme:      "https",
		Port:        "",
		ProjectPath: parts[0] + "/" + parts[1],
		Number:      n,
		HTMLURL:     raw,
	}, nil
}

func parseGitLabMR(raw, scheme, host, port string, parts []string) (MergeRequestRef, error) {
	// group[/sub]/project/-/merge_requests/N
	idx := -1
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "-" && parts[i+1] == "merge_requests" {
			idx = i
			break
		}
	}
	if idx < 1 || idx+2 >= len(parts) {
		return MergeRequestRef{}, fmt.Errorf("not a gitlab merge request url")
	}
	n, err := strconv.Atoi(parts[idx+2])
	if err != nil || n <= 0 {
		return MergeRequestRef{}, fmt.Errorf("invalid merge request iid")
	}
	project := strings.Join(parts[:idx], "/")
	if project == "" {
		return MergeRequestRef{}, fmt.Errorf("missing gitlab project path")
	}
	return MergeRequestRef{
		Provider:    MergeProviderGitLab,
		Host:        host,
		Scheme:      scheme,
		Port:        port,
		ProjectPath: project,
		Number:      n,
		HTMLURL:     raw,
	}, nil
}

func (r MergeRequestRef) EncodedProjectPath() string {
	return url.PathEscape(r.ProjectPath)
}
