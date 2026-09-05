package main

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
	defaultDockerRegistry = "registry-1.docker.io"
	manifestListAccept    = "application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.index.v1+json"
	dockerManifestAccept  = "application/vnd.docker.distribution.manifest.v2+json"
	ociManifestAccept     = "application/vnd.oci.image.manifest.v1+json"
)

var manifestHTTP = &http.Client{Timeout: 20 * time.Second}

var wwwAuthRealmRe = regexp.MustCompile(`realm="([^"]+)"`)
var wwwAuthServiceRe = regexp.MustCompile(`service="([^"]+)"`)
var wwwAuthScopeRe = regexp.MustCompile(`scope="([^"]+)"`)

type imageReference struct {
	repo      string
	reference string
}

func normalizeArchitecture(value string) string {
	n := strings.ToLower(strings.TrimSpace(value))
	switch n {
	case "x86", "x86_64", "amd64":
		return "x86_64"
	case "arm", "arm64", "aarch64":
		return "arm64"
	default:
		return n
	}
}

func parseImageReference(imageURL string) imageReference {
	normalized := strings.TrimSpace(imageURL)
	if at := strings.Index(normalized, "@"); at >= 0 {
		return imageReference{
			repo:      normalized[:at],
			reference: normalized[at+1:],
		}
	}
	lastSlash := strings.LastIndex(normalized, "/")
	lastColon := strings.LastIndex(normalized, ":")
	if lastColon > lastSlash {
		return imageReference{
			repo:      normalized[:lastColon],
			reference: normalized[lastColon+1:],
		}
	}
	return imageReference{repo: normalized, reference: "latest"}
}

func splitRegistryAndRepository(repoWithOptionalRegistry string) (registry, repository string, err error) {
	parts := strings.Split(repoWithOptionalRegistry, "/")
	if len(parts) == 0 {
		return "", "", fmt.Errorf("镜像地址格式错误：缺少仓库名称")
	}
	first := parts[0]
	hasRegistry := strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost"
	if hasRegistry {
		registry = first
		repository = strings.Join(parts[1:], "/")
	} else {
		registry = defaultDockerRegistry
		repository = repoWithOptionalRegistry
	}
	if repository == "" {
		return "", "", fmt.Errorf("镜像地址格式错误：缺少仓库名称")
	}
	if registry == defaultDockerRegistry && !strings.Contains(repository, "/") {
		repository = "library/" + repository
	}
	return registry, repository, nil
}

func parseWWWAuthenticate(authHeader string) (realm, service, scope string) {
	if m := wwwAuthRealmRe.FindStringSubmatch(authHeader); len(m) > 1 {
		realm = m[1]
	}
	if m := wwwAuthServiceRe.FindStringSubmatch(authHeader); len(m) > 1 {
		service = m[1]
	}
	if m := wwwAuthScopeRe.FindStringSubmatch(authHeader); len(m) > 1 {
		scope = m[1]
	}
	return realm, service, scope
}

func getBearerToken(wwwAuthenticate string) (string, error) {
	realm, service, scope := parseWWWAuthenticate(wwwAuthenticate)
	if realm == "" {
		return "", fmt.Errorf("无法解析镜像仓库认证信息")
	}
	params := url.Values{}
	if service != "" {
		params.Set("service", service)
	}
	if scope != "" {
		params.Set("scope", scope)
	}
	tokenURL := realm
	if enc := params.Encode(); enc != "" {
		tokenURL += "?" + enc
	}
	resp, err := manifestHTTP.Get(tokenURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("镜像仓库认证失败: HTTP %d", resp.StatusCode)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	token := strField(payload, "token")
	if token == "" {
		token = strField(payload, "access_token")
	}
	if token == "" {
		return "", fmt.Errorf("镜像仓库未返回访问令牌")
	}
	return token, nil
}

func fetchManifestListResponse(registry, repository, reference string) (map[string]interface{}, string, error) {
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, reference)
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", manifestListAccept)
	resp, err := manifestHTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	authHeader := ""
	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if !strings.Contains(wwwAuth, "Bearer") {
			body, _ := io.ReadAll(resp.Body)
			return nil, "", fmt.Errorf("镜像仓库认证失败: HTTP %d %s", resp.StatusCode, string(body))
		}
		token, err := getBearerToken(wwwAuth)
		if err != nil {
			return nil, "", err
		}
		authHeader = "Bearer " + token
		req2, err := http.NewRequest(http.MethodGet, manifestURL, nil)
		if err != nil {
			return nil, "", err
		}
		req2.Header.Set("Accept", manifestListAccept)
		req2.Header.Set("Authorization", authHeader)
		resp2, err := manifestHTTP.Do(req2)
		if err != nil {
			return nil, "", err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode >= 400 {
			body, _ := io.ReadAll(resp2.Body)
			return nil, "", fmt.Errorf("拉取镜像 manifest 失败: HTTP %d %s", resp2.StatusCode, string(body))
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(resp2.Body).Decode(&payload); err != nil {
			return nil, "", err
		}
		return payload, authHeader, nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("拉取镜像 manifest 失败: HTTP %d %s", resp.StatusCode, string(body))
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, "", err
	}
	return payload, authHeader, nil
}

func fetchManifestResponse(registry, repository, referenceOrDigest, authHeader string) (map[string]interface{}, error) {
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, referenceOrDigest)
	req, err := http.NewRequest(http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", dockerManifestAccept+", "+ociManifestAccept)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := manifestHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func extractSizeFromSingleManifest(payload map[string]interface{}) *int64 {
	var total int64
	if layers, ok := payload["layers"].([]interface{}); ok {
		for _, item := range layers {
			layer, _ := item.(map[string]interface{})
			if size, ok := layer["size"].(float64); ok && size >= 0 {
				total += int64(size)
			}
		}
		if total > 0 {
			return &total
		}
	}
	if config, ok := payload["config"].(map[string]interface{}); ok {
		if size, ok := config["size"].(float64); ok && size >= 0 {
			v := int64(size)
			return &v
		}
	}
	return nil
}

func resolveContainerImageMetadata(imageURL string) (map[string]interface{}, error) {
	if strings.TrimSpace(imageURL) == "" {
		return nil, fmt.Errorf("镜像地址不能为空")
	}
	ref := parseImageReference(imageURL)
	registry, repository, err := splitRegistryAndRepository(ref.repo)
	if err != nil {
		return nil, err
	}
	if err := registryhost.RejectPrivateRegistry(registry); err != nil {
		return nil, err
	}
	payload, authHeader, err := fetchManifestListResponse(registry, repository, ref.reference)
	if err != nil {
		return nil, registryhost.AnnotateRegistryFetchError(err, registry)
	}

	if manifests, ok := payload["manifests"].([]interface{}); ok && len(manifests) > 0 {
		architectures := []string{}
		seenArch := map[string]bool{}
		var preferredDigest, fallbackDigest string
		var fallbackSize *int64
		for _, item := range manifests {
			manifest, _ := item.(map[string]interface{})
			platform, _ := manifest["platform"].(map[string]interface{})
			rawArch := strField(platform, "architecture")
			arch := normalizeArchitecture(rawArch)
			if arch != "" && !seenArch[arch] {
				seenArch[arch] = true
				architectures = append(architectures, arch)
			}
			digest := strField(manifest, "digest")
			if fallbackDigest == "" && digest != "" {
				fallbackDigest = digest
				if size, ok := manifest["size"].(float64); ok && size >= 0 {
					v := int64(size)
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
			if child, err := fetchManifestResponse(registry, repository, targetDigest, authHeader); err == nil {
				parsedSize = extractSizeFromSingleManifest(child)
			}
		}
		if parsedSize == nil {
			if tagManifest, err := fetchManifestResponse(registry, repository, ref.reference, authHeader); err == nil {
				parsedSize = extractSizeFromSingleManifest(tagManifest)
			}
		}
		if parsedSize == nil {
			parsedSize = fallbackSize
		}
		return map[string]interface{}{
			"target_architectures": architectures,
			"size":                 parsedSize,
		}, nil
	}

	config, _ := payload["config"].(map[string]interface{})
	rawArch := strField(config, "architecture")
	if rawArch == "" {
		rawArch = strField(payload, "architecture")
	}
	arch := normalizeArchitecture(rawArch)
	archList := []string{}
	if arch != "" {
		archList = append(archList, arch)
	}
	size := extractSizeFromSingleManifest(payload)
	mediaType := strField(payload, "mediaType")
	if len(archList) == 0 && mediaType == dockerManifestAccept {
		if digest := strField(config, "digest"); digest != "" {
			blobURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)
			req, err := http.NewRequest(http.MethodGet, blobURL, nil)
			if err == nil {
				if authHeader != "" {
					req.Header.Set("Authorization", authHeader)
				}
				if resp, err := manifestHTTP.Do(req); err == nil {
					defer resp.Body.Close()
					if resp.StatusCode < 400 {
						var blob map[string]interface{}
						if json.NewDecoder(resp.Body).Decode(&blob) == nil {
							if blobArch := normalizeArchitecture(strField(blob, "architecture")); blobArch != "" {
								archList = []string{blobArch}
							}
						}
					}
				}
			}
		}
	}
	return map[string]interface{}{
		"target_architectures": archList,
		"size":                 size,
	}, nil
}
