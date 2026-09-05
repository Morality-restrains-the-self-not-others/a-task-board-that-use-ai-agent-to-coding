package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// PackageRef is a parsed releases.yaml package coordinate.
type PackageRef struct {
	Scheme   string
	Owner    string
	Repo     string
	Asset    string
	SHA      string
	FilePath string
}

// ParsePackageRef accepts github://owner/repo/asset@sha or file://path.
func ParsePackageRef(raw string) (PackageRef, error) {
	s := strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(s, "github://"):
		rest := strings.TrimPrefix(s, "github://")
		at := strings.LastIndex(rest, "@")
		if at <= 0 || at == len(rest)-1 {
			return PackageRef{}, fmt.Errorf("package %q: missing @sha", raw)
		}
		sha := rest[at+1:]
		parts := strings.Split(rest[:at], "/")
		if len(parts) < 3 {
			return PackageRef{}, fmt.Errorf("package %q: want github://owner/repo/asset@sha", raw)
		}
		return PackageRef{
			Scheme: "github",
			Owner:  parts[0],
			Repo:   parts[1],
			Asset:  strings.Join(parts[2:], "/"),
			SHA:    sha,
		}, nil
	case strings.HasPrefix(s, "file://"):
		p := strings.TrimPrefix(s, "file://")
		if p == "" {
			return PackageRef{}, fmt.Errorf("package %q: empty file path", raw)
		}
		return PackageRef{Scheme: "file", FilePath: p}, nil
	default:
		return PackageRef{}, fmt.Errorf("package %q: unsupported scheme", raw)
	}
}

func directHTTPClient() *http.Client {
	return &http.Client{
		// taskEvents-bin.tar.gz is ~1GB; 10m is not enough on slow links.
		Timeout: 60 * time.Minute,
		Transport: &http.Transport{
			Proxy: nil,
			// GitHub's edge often RSTs Go's HTTP/2 (PROTOCOL_ERROR / stream ID 1).
			ForceAttemptHTTP2: false,
			TLSNextProto:      map[string]func(authority string, c *tls.Conn) http.RoundTripper{},
		},
	}
}

// FileArtifactFetcher copies a local file:// pin.
type FileArtifactFetcher struct{}

func (FileArtifactFetcher) Fetch(pin ArtifactPin) (string, error) {
	ref, err := ParsePackageRef(pin.Package)
	if err != nil {
		return "", err
	}
	if ref.Scheme != "file" {
		return "", fmt.Errorf("file fetcher got scheme %s", ref.Scheme)
	}
	in, err := os.Open(ref.FilePath)
	if err != nil {
		log.Printf("[deploy-sync] file fetch failed path=%s err=%v", ref.FilePath, err)
		return "", err
	}
	defer in.Close()
	tmp, err := os.CreateTemp("", "deploy-pin-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	log.Printf("[deploy-sync] file fetch ok sha=%s", pin.SHA)
	return tmp.Name(), nil
}

type githubRelease struct {
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"assets"`
}

// GitHubReleaseFetcher downloads a release asset. Token is never logged.
type GitHubReleaseFetcher struct {
	APIBase  string
	Token    string
	Download func(assetAPIURL, token string) (string, error)
}

func (f GitHubReleaseFetcher) apiBase() string {
	if strings.TrimSpace(f.APIBase) != "" {
		return strings.TrimRight(f.APIBase, "/")
	}
	return "https://api.github.com"
}

func (f GitHubReleaseFetcher) Fetch(pin ArtifactPin) (string, error) {
	ref, err := ParsePackageRef(pin.Package)
	if err != nil {
		return "", err
	}
	if ref.Scheme != "github" {
		return "", fmt.Errorf("github fetcher got scheme %s", ref.Scheme)
	}
	tag := ref.SHA
	if strings.TrimSpace(tag) == "" {
		return "", fmt.Errorf("github package missing @tag")
	}
	metaURL := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", f.apiBase(), ref.Owner, ref.Repo, tag)
	log.Printf("[deploy-sync] github release lookup owner=%s repo=%s tag=%s asset=%s", ref.Owner, ref.Repo, tag, ref.Asset)
	req, err := http.NewRequest(http.MethodGet, metaURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := strings.TrimSpace(f.Token); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := directHTTPClient().Do(req)
	if err != nil {
		log.Printf("[deploy-sync] github metadata failed tag=%s err=%v", tag, err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		log.Printf("[deploy-sync] github metadata status=%d tag=%s", resp.StatusCode, tag)
		return "", fmt.Errorf("github release %s: HTTP %d", tag, resp.StatusCode)
	}
	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return "", fmt.Errorf("github release json: %w", err)
	}
	var assetURL string
	for _, a := range rel.Assets {
		if a.Name == ref.Asset {
			assetURL = a.URL
			break
		}
	}
	if assetURL == "" {
		return "", fmt.Errorf("github release %s: asset %s not found", tag, ref.Asset)
	}
	dl := f.Download
	if dl == nil {
		dl = downloadGitHubAsset
	}
	return dl(assetURL, f.Token)
}

var (
	githubAssetProgressEveryBytes int64 = 64 << 20
	githubAssetProgressEvery            = 30 * time.Second
)

type githubAssetProgressReader struct {
	r      io.Reader
	n      int64
	lastN  int64
	lastAt time.Time
}

func (p *githubAssetProgressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	now := time.Now()
	if p.lastAt.IsZero() {
		p.lastAt = now
	}
	if p.n-p.lastN >= githubAssetProgressEveryBytes || now.Sub(p.lastAt) >= githubAssetProgressEvery {
		log.Printf("[deploy-sync] github asset progress bytes=%d", p.n)
		p.lastN = p.n
		p.lastAt = now
	}
	return n, err
}

func downloadGitHubAsset(assetAPIURL, token string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, assetAPIURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/octet-stream")
	if tok := strings.TrimSpace(token); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := directHTTPClient().Do(req)
	if err != nil {
		log.Printf("[deploy-sync] github asset download failed err=%v", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("[deploy-sync] github asset status=%d", resp.StatusCode)
		return "", fmt.Errorf("github asset: HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "deploy-gh-*")
	if err != nil {
		return "", err
	}
	progress := &githubAssetProgressReader{r: resp.Body}
	if _, err := io.Copy(tmp, progress); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	log.Printf("[deploy-sync] github asset stored")
	return tmp.Name(), nil
}

// DefaultArtifactFetcher routes file:// and github:// pins.
type DefaultArtifactFetcher struct {
	GitHub GitHubReleaseFetcher
}

func NewDefaultArtifactFetcher() DefaultArtifactFetcher {
	tok := strings.TrimSpace(os.Getenv("GH_PACKAGES_TOKEN"))
	if tok == "" {
		tok = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	return DefaultArtifactFetcher{GitHub: GitHubReleaseFetcher{Token: tok}}
}

func (f DefaultArtifactFetcher) Fetch(pin ArtifactPin) (string, error) {
	ref, err := ParsePackageRef(pin.Package)
	if err != nil {
		return "", err
	}
	switch ref.Scheme {
	case "file":
		return FileArtifactFetcher{}.Fetch(pin)
	case "github":
		return f.GitHub.Fetch(pin)
	default:
		return "", fmt.Errorf("unsupported scheme %s", ref.Scheme)
	}
}
