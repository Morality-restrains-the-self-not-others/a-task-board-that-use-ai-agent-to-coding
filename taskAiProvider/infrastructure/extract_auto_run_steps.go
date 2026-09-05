package infrastructure

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"path"
	"registryhost"
	"strings"
	"time"
)

const (
	DefaultAutoRunStepsPath = "/app/autoRunStep.md"
	maxLayerBytesForExtract = 80 << 20
	maxMarkdownBytes        = 1 << 20
)

// AutoRunStepsExtractResult is the cached preview extracted from an OCI image.
type AutoRunStepsExtractResult struct {
	Markdown string
	Digest   string
	Status   string // ok | not_found | auth_failed | failed
	Detail   string
}

func extractRegistryClient() registryHTTP {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	return &http.Client{Timeout: 55 * time.Second, Transport: tr}
}

func ExtractAutoRunStepsFromImage(imageURL string) AutoRunStepsExtractResult {
	return extractAutoRunStepsFromImage(extractRegistryClient(), imageURL, DefaultAutoRunStepsPath)
}

func extractAutoRunStepsFromImageWithClient(client registryHTTP, imageURL string) AutoRunStepsExtractResult {
	return extractAutoRunStepsFromImage(client, imageURL, DefaultAutoRunStepsPath)
}

func normalizeInImagePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return DefaultAutoRunStepsPath
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

func layerDigestsFromManifest(payload map[string]any) []string {
	out := []string{}
	layers, _ := payload["layers"].([]any)
	for _, item := range layers {
		layer, _ := item.(map[string]any)
		d, _ := layer["digest"].(string)
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

func fetchLayerBlob(client registryHTTP, registry, repository, digest, authHeader string) ([]byte, error) {
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

func resolveExtractManifest(client registryHTTP, imageURL string) (registry, repository, digest string, manifest map[string]any, authHeader string, err error) {
	repoWithReg, reference := parseImageReference(imageURL)
	registry, repository, err = splitRegistryAndRepository(repoWithReg)
	if err != nil {
		return
	}
	if err = registryhost.RejectPrivateRegistry(registry); err != nil {
		return
	}
	payload, authHeader, err := fetchManifestList(client, registry, repository, reference)
	if err != nil {
		err = registryhost.AnnotateRegistryFetchError(err, registry)
		if strings.Contains(err.Error(), "认证") || strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
			err = fmt.Errorf("auth_failed: %w", err)
		}
		return
	}
	if manifests, ok := payload["manifests"].([]any); ok && len(manifests) > 0 {
		var preferredDigest, fallbackDigest string
		for _, item := range manifests {
			manifestItem, _ := item.(map[string]any)
			platform, _ := manifestItem["platform"].(map[string]any)
			rawArch, _ := platform["architecture"].(string)
			arch := NormalizeArchitecture(rawArch)
			d, _ := manifestItem["digest"].(string)
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
		manifest, err = fetchSingleManifest(client, registry, repository, target, authHeader)
		return
	}
	digest, _ = payload["digest"].(string)
	if digest == "" {
		if config, ok := payload["config"].(map[string]any); ok {
			digest, _ = config["digest"].(string)
		}
	}
	if digest == "" {
		digest = reference
	}
	manifest = payload
	return
}

func extractAutoRunStepsFromImage(client registryHTTP, imageURL, filePath string) AutoRunStepsExtractResult {
	wanted := normalizeInImagePath(filePath)
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return AutoRunStepsExtractResult{Status: "failed", Detail: "image_url is required"}
	}
	registry, repository, digest, manifest, authHeader, err := resolveExtractManifest(client, imageURL)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "auth_failed") || strings.Contains(msg, "认证") {
			return AutoRunStepsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
		}
		return AutoRunStepsExtractResult{Status: "failed", Detail: msg, Digest: digest}
	}
	layers := layerDigestsFromManifest(manifest)
	if len(layers) == 0 {
		return AutoRunStepsExtractResult{Status: "not_found", Detail: "no layers in manifest", Digest: digest}
	}
	for i := len(layers) - 1; i >= 0; i-- {
		blob, err := fetchLayerBlob(client, registry, repository, layers[i], authHeader)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "auth_failed") {
				return AutoRunStepsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
			}
			continue
		}
		md, found, err := findAutoRunStepsInLayerBlob(blob, wanted)
		if err != nil {
			continue
		}
		if found {
			return AutoRunStepsExtractResult{Markdown: md, Digest: digest, Status: "ok"}
		}
	}
	return AutoRunStepsExtractResult{
		Status: "not_found",
		Detail: "autoRunStep.md not found in image layers",
		Digest: digest,
	}
}
