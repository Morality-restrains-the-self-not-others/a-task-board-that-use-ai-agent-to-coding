package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	defaultAutoRunStepsPath = "/app/autoRunStep.md"
	maxLayerBytesForExtract = 80 << 20 // 80 MiB per layer blob
	maxMarkdownBytes        = 1 << 20  // 1 MiB markdown
)

var extractHTTP = &http.Client{Timeout: 55 * time.Second}

type autoRunStepsExtractResult struct {
	Markdown string
	Digest   string
	Status   string // ok | not_found | auth_failed | failed
	Detail   string
}

func normalizeInImagePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return defaultAutoRunStepsPath
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}

func tarEntryMatchesWanted(name, wanted string) bool {
	n := path.Clean("/" + strings.TrimPrefix(strings.TrimSpace(name), "./"))
	w := normalizeInImagePath(wanted)
	return n == w || strings.TrimPrefix(n, "/") == strings.TrimPrefix(w, "/")
}

func readFileFromTarStream(r io.Reader, wantedPath string) (string, bool, error) {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return "", false, nil
		}
		if err != nil {
			return "", false, err
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		if !tarEntryMatchesWanted(hdr.Name, wantedPath) {
			continue
		}
		limited := io.LimitReader(tr, maxMarkdownBytes+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			return "", false, err
		}
		if int64(len(data)) > maxMarkdownBytes {
			return "", false, fmt.Errorf("autoRunStep.md too large")
		}
		return string(data), true, nil
	}
}

func openLayerBlobReader(raw []byte) (io.ReadCloser, error) {
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		gr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		return gr, nil
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

func findAutoRunStepsInLayerBlob(blob []byte, wantedPath string) (string, bool, error) {
	rc, err := openLayerBlobReader(blob)
	if err != nil {
		return "", false, err
	}
	defer rc.Close()
	return readFileFromTarStream(rc, wantedPath)
}

func fetchRegistryBlob(registry, repository, digest, authHeader string) ([]byte, error) {
	blobURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)
	req, err := http.NewRequest(http.MethodGet, blobURL, nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := extractHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("auth_failed")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("blob HTTP %d %s", resp.StatusCode, string(body))
	}
	limited := io.LimitReader(resp.Body, maxLayerBytesForExtract+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxLayerBytesForExtract {
		return nil, fmt.Errorf("layer blob too large")
	}
	return data, nil
}

func layerDigestsFromManifest(payload map[string]interface{}) []string {
	out := []string{}
	layers, _ := payload["layers"].([]interface{})
	for _, item := range layers {
		layer, _ := item.(map[string]interface{})
		d := strField(layer, "digest")
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

func resolveSingleImageManifest(imageURL string) (registry, repository, digest string, manifest map[string]interface{}, authHeader string, err error) {
	ref := parseImageReference(imageURL)
	registry, repository, err = splitRegistryAndRepository(ref.repo)
	if err != nil {
		return
	}
	payload, authHeader, err := fetchManifestListResponse(registry, repository, ref.reference)
	if err != nil {
		if strings.Contains(err.Error(), "认证") {
			err = fmt.Errorf("auth_failed: %w", err)
		}
		return
	}
	if manifests, ok := payload["manifests"].([]interface{}); ok && len(manifests) > 0 {
		var preferredDigest, fallbackDigest string
		for _, item := range manifests {
			manifestItem, _ := item.(map[string]interface{})
			platform, _ := manifestItem["platform"].(map[string]interface{})
			arch := normalizeArchitecture(strField(platform, "architecture"))
			d := strField(manifestItem, "digest")
			if fallbackDigest == "" && d != "" {
				fallbackDigest = d
			}
			if arch == "x86_64" && d != "" {
				preferredDigest = d
			}
		}
		target := preferredDigest
		if target == "" {
			target = fallbackDigest
		}
		if target == "" {
			err = fmt.Errorf("manifest list has no digest")
			return
		}
		digest = target
		manifest, err = fetchManifestResponse(registry, repository, target, authHeader)
		return
	}
	// Single manifest (tag resolved directly)
	digest = strField(payload, "digest")
	if digest == "" {
		if config, ok := payload["config"].(map[string]interface{}); ok {
			digest = strField(config, "digest")
		}
	}
	if digest == "" {
		digest = ref.reference
	}
	manifest = payload
	return
}

func extractAutoRunStepsFromImage(imageURL, filePath string) autoRunStepsExtractResult {
	wanted := normalizeInImagePath(filePath)
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return autoRunStepsExtractResult{Status: "failed", Detail: "image_url is required"}
	}
	registry, repository, digest, manifest, authHeader, err := resolveSingleImageManifest(imageURL)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "auth_failed") || strings.Contains(msg, "认证") {
			return autoRunStepsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
		}
		return autoRunStepsExtractResult{Status: "failed", Detail: msg, Digest: digest}
	}
	layers := layerDigestsFromManifest(manifest)
	if len(layers) == 0 {
		return autoRunStepsExtractResult{Status: "not_found", Detail: "no layers in manifest", Digest: digest}
	}
	// Upper layers last in OCI; search from top so later layers win.
	for i := len(layers) - 1; i >= 0; i-- {
		blob, err := fetchRegistryBlob(registry, repository, layers[i], authHeader)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "auth_failed") {
				return autoRunStepsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
			}
			// Skip unreadable layer and continue
			continue
		}
		md, found, err := findAutoRunStepsInLayerBlob(blob, wanted)
		if err != nil {
			continue
		}
		if found {
			return autoRunStepsExtractResult{
				Markdown: md,
				Digest:   digest,
				Status:   "ok",
			}
		}
	}
	return autoRunStepsExtractResult{
		Status: "not_found",
		Detail: "autoRunStep.md not found in image layers",
		Digest: digest,
	}
}
