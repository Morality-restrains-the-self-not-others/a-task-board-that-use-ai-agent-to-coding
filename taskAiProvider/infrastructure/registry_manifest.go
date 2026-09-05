package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"registryhost"
	"strings"
	"time"
)

const (
	manifestListAccept   = "application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"
	dockerManifestAccept = "application/vnd.docker.distribution.manifest.v2+json"
	ociManifestAccept    = "application/vnd.oci.image.manifest.v1+json"
	defaultRegistry      = "registry-1.docker.io"
)

// registryScheme is https in production; tests may set http for httptest.
var registryScheme = "https"

func SetRegistrySchemeForTest(scheme string) { registryScheme = scheme }
func ExportRegistrySchemeForTest() string    { return registryScheme }

func registryURL(host, path string) string {
	return registryScheme + "://" + host + path
}

var (
	realmRe   = regexp.MustCompile(`realm="([^"]+)"`)
	serviceRe = regexp.MustCompile(`service="([^"]+)"`)
	scopeRe   = regexp.MustCompile(`scope="([^"]+)"`)
)

// ImageMetadata is the resolve-target-architectures response body.
type ImageMetadata struct {
	TargetArchitectures []string `json:"target_architectures"`
	Size                *int64   `json:"size"`
}

// NormalizeArchitecture aligns with project rule: amd64↔x86_64, arm64↔aarch64.
func NormalizeArchitecture(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	switch normalized {
	case "x86", "x86_64", "amd64":
		return "x86_64"
	case "arm", "arm64", "aarch64":
		return "arm64"
	default:
		return normalized
	}
}

// MergeImageURLWithVersion matches Python merge_image_url_with_version.
func MergeImageURLWithVersion(imageURL, version string) (string, error) {
	normalized := strings.TrimSpace(imageURL)
	if normalized == "" {
		return "", fmt.Errorf("镜像地址不能为空")
	}
	if strings.Contains(normalized, "@") {
		return normalized, nil
	}
	lastSlash := strings.LastIndex(normalized, "/")
	lastColon := strings.LastIndex(normalized, ":")
	if lastColon > lastSlash {
		return normalized, nil
	}
	ver := strings.TrimSpace(version)
	if ver != "" {
		return normalized + ":" + ver, nil
	}
	return normalized, nil
}

func parseImageReference(imageURL string) (repo, reference string) {
	normalized := strings.TrimSpace(imageURL)
	if i := strings.Index(normalized, "@"); i >= 0 {
		return normalized[:i], normalized[i+1:]
	}
	lastSlash := strings.LastIndex(normalized, "/")
	lastColon := strings.LastIndex(normalized, ":")
	if lastColon > lastSlash {
		return normalized[:lastColon], normalized[lastColon+1:]
	}
	return normalized, "latest"
}

func splitRegistryAndRepository(repoWithOptionalRegistry string) (registry, repository string, err error) {
	parts := strings.Split(repoWithOptionalRegistry, "/")
	first := parts[0]
	hasRegistryPrefix := strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost"
	if hasRegistryPrefix {
		registry = first
		repository = strings.Join(parts[1:], "/")
	} else {
		registry = defaultRegistry
		repository = repoWithOptionalRegistry
	}
	if repository == "" {
		return "", "", fmt.Errorf("镜像地址格式错误：缺少仓库名称")
	}
	if registry == defaultRegistry && !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}
	return registry, repository, nil
}

type registryHTTP interface {
	Do(req *http.Request) (*http.Response, error)
}

func defaultRegistryClient() registryHTTP {
	// Registry pulls must not inherit shell HTTP(S)_PROXY / ALL_PROXY.
	// Dev machines often export socks5h://127.0.0.1:1234; when that local
	// proxy is down, Go's default Transport yields:
	//   proxyconnect tcp: dial tcp 127.0.0.1:1234: connect: connection refused
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	return &http.Client{Timeout: 20 * time.Second, Transport: tr}
}

// ResolveContainerImageMetadata fetches OCI/Docker manifest metadata (Python parity).
func ResolveContainerImageMetadata(imageURL string) (*ImageMetadata, error) {
	return resolveContainerImageMetadata(defaultRegistryClient(), imageURL)
}

// ResolveContainerImageMetadataWithClient is for tests with httptest.
func ResolveContainerImageMetadataWithClient(client registryHTTP, imageURL string) (*ImageMetadata, error) {
	return resolveContainerImageMetadata(client, imageURL)
}

func resolveContainerImageMetadata(client registryHTTP, imageURL string) (*ImageMetadata, error) {
	if strings.TrimSpace(imageURL) == "" {
		return nil, fmt.Errorf("镜像地址不能为空")
	}
	repoWithReg, reference := parseImageReference(imageURL)
	registry, repository, err := splitRegistryAndRepository(repoWithReg)
	if err != nil {
		return nil, err
	}
	if err := registryhost.RejectPrivateRegistry(registry); err != nil {
		return nil, err
	}
	payload, authHeader, err := fetchManifestList(client, registry, repository, reference)
	if err != nil {
		return nil, registryhost.AnnotateRegistryFetchError(err, registry)
	}

	if manifests, _ := payload["manifests"].([]any); len(manifests) > 0 {
		architectures := make([]string, 0)
		seen := map[string]bool{}
		var preferredDigest, fallbackDigest string
		var fallbackSize *int64
		for _, raw := range manifests {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			platform, _ := item["platform"].(map[string]any)
			rawArch, _ := platform["architecture"].(string)
			arch := NormalizeArchitecture(rawArch)
			if arch != "" && !seen[arch] {
				seen[arch] = true
				architectures = append(architectures, arch)
			}
			digest, _ := item["digest"].(string)
			if fallbackDigest == "" && digest != "" {
				fallbackDigest = digest
				if sz, ok := claimInt64(item["size"]); ok && sz >= 0 {
					v := sz
					fallbackSize = &v
				}
			}
			if arch == "x86_64" && digest != "" {
				preferredDigest = digest
			}
		}
		targetDigest := preferredDigest
		if targetDigest == "" {
			targetDigest = fallbackDigest
		}
		var parsedSize *int64
		if targetDigest != "" {
			if child, err := fetchSingleManifest(client, registry, repository, targetDigest, authHeader); err == nil {
				parsedSize = extractSizeFromSingleManifest(child)
			}
		}
		if parsedSize == nil {
			if tagM, err := fetchSingleManifest(client, registry, repository, reference, authHeader); err == nil {
				parsedSize = extractSizeFromSingleManifest(tagM)
			}
		}
		if parsedSize == nil {
			parsedSize = fallbackSize
		}
		return &ImageMetadata{TargetArchitectures: architectures, Size: parsedSize}, nil
	}

	config, _ := payload["config"].(map[string]any)
	rawArch, _ := config["architecture"].(string)
	if rawArch == "" {
		rawArch, _ = payload["architecture"].(string)
	}
	arch := NormalizeArchitecture(rawArch)
	archList := []string{}
	if arch != "" {
		archList = []string{arch}
	}
	size := extractSizeFromSingleManifest(payload)
	mediaType, _ := payload["mediaType"].(string)
	if arch == "" && mediaType == dockerManifestAccept {
		if digest, _ := config["digest"].(string); digest != "" {
			if blob, err := fetchBlobJSON(client, registry, repository, digest, authHeader); err == nil {
				if ba, _ := blob["architecture"].(string); ba != "" {
					if na := NormalizeArchitecture(ba); na != "" {
						archList = []string{na}
					}
				}
			}
		}
	}
	return &ImageMetadata{TargetArchitectures: archList, Size: size}, nil
}

func fetchManifestList(client registryHTTP, registry, repository, reference string) (map[string]any, string, error) {
	manifestURL := registryURL(registry, fmt.Sprintf("/v2/%s/manifests/%s", repository, reference))
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", manifestListAccept)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	authHeader := ""
	if resp.StatusCode == http.StatusUnauthorized {
		www := resp.Header.Get("WWW-Authenticate")
		_ = resp.Body.Close()
		if !strings.Contains(www, "Bearer") {
			return nil, "", fmt.Errorf("registry unauthorized")
		}
		token, err := getBearerToken(client, www)
		if err != nil {
			return nil, "", err
		}
		authHeader = "Bearer " + token
		req2, _ := http.NewRequest(http.MethodGet, manifestURL, nil)
		req2.Header.Set("Accept", manifestListAccept)
		req2.Header.Set("Authorization", authHeader)
		resp, err = client.Do(req2)
		if err != nil {
			return nil, "", err
		}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("registry status %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", fmt.Errorf("invalid manifest json")
	}
	return payload, authHeader, nil
}

func getBearerToken(client registryHTTP, wwwAuthenticate string) (string, error) {
	realmM := realmRe.FindStringSubmatch(wwwAuthenticate)
	if len(realmM) < 2 {
		return "", fmt.Errorf("无法解析镜像仓库认证信息")
	}
	u, err := url.Parse(realmM[1])
	if err != nil {
		return "", err
	}
	q := u.Query()
	if m := serviceRe.FindStringSubmatch(wwwAuthenticate); len(m) >= 2 {
		q.Set("service", m[1])
	}
	if m := scopeRe.FindStringSubmatch(wwwAuthenticate); len(m) >= 2 {
		q.Set("scope", m[1])
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	var payload map[string]any
	_ = json.Unmarshal(body, &payload)
	token, _ := payload["token"].(string)
	if token == "" {
		token, _ = payload["access_token"].(string)
	}
	if token == "" {
		return "", fmt.Errorf("镜像仓库未返回访问令牌")
	}
	return token, nil
}

func fetchSingleManifest(client registryHTTP, registry, repository, ref, authHeader string) (map[string]any, error) {
	manifestURL := registryURL(registry, fmt.Sprintf("/v2/%s/manifests/%s", repository, ref))
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", dockerManifestAccept+", "+ociManifestAccept)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("manifest status %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func fetchBlobJSON(client registryHTTP, registry, repository, digest, authHeader string) (map[string]any, error) {
	blobURL := registryURL(registry, fmt.Sprintf("/v2/%s/blobs/%s", repository, digest))
	req, err := http.NewRequest(http.MethodGet, blobURL, nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("blob status %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func extractSizeFromSingleManifest(manifest map[string]any) *int64 {
	var total int64
	if layers, ok := manifest["layers"].([]any); ok {
		for _, raw := range layers {
			layer, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if sz, ok := claimInt64(layer["size"]); ok && sz >= 0 {
				total += sz
			}
		}
	}
	if total > 0 {
		return &total
	}
	if config, ok := manifest["config"].(map[string]any); ok {
		if sz, ok := claimInt64(config["size"]); ok && sz >= 0 {
			v := sz
			return &v
		}
	}
	return nil
}
