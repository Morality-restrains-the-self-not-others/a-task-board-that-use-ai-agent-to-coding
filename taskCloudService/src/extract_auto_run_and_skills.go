package main

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

// autoRunAndSkillsExtractResult carries both files matched in a single OCI
// manifest-resolve + layer-blob pass (OPT-20260821-018). Previously
// ensureInstalledImageAutoRunSteps and ensureInstalledImageSkills each walked
// the same layers separately on install-time re-extract fallback.
type autoRunAndSkillsExtractResult struct {
	AutoRun autoRunStepsExtractResult
	Skills  imageSkillsExtractResult
}

// extractAutoRunAndSkillsFn is a test seam mirroring the single-file seams.
var extractAutoRunAndSkillsFn = extractAutoRunAndSkillsFromImage

// extractAutoRunAndSkillsFromImage resolves the manifest once and walks the
// layer blobs once, matching both /app/autoRunStep.md and /app/imageSkills.yaml
// in the same pass. Per-file results mirror the two single-file walks (newest
// layer wins).
func extractAutoRunAndSkillsFromImage(imageURL string) autoRunAndSkillsExtractResult {
	autoWanted := normalizeInImagePath(defaultAutoRunStepsPath)
	skillsWanted := normalizeInImagePath(defaultImageSkillsPath)
	imageURL = strings.TrimSpace(imageURL)
	out := autoRunAndSkillsExtractResult{}
	if imageURL == "" {
		out.AutoRun = autoRunStepsExtractResult{Status: "failed", Detail: "image_url is required"}
		out.Skills = imageSkillsExtractResult{Status: "failed", Detail: "image_url is required"}
		return out
	}
	registry, repository, digest, manifest, authHeader, err := resolveSingleImageManifest(imageURL)
	if err != nil {
		msg := err.Error()
		status := "failed"
		if strings.Contains(msg, "auth_failed") || strings.Contains(msg, "认证") {
			status = "auth_failed"
		}
		out.AutoRun = autoRunStepsExtractResult{Status: status, Detail: msg, Digest: digest}
		out.Skills = imageSkillsExtractResult{Status: status, Detail: msg, Digest: digest}
		return out
	}
	layers := layerDigestsFromManifest(manifest)
	if len(layers) == 0 {
		out.AutoRun = autoRunStepsExtractResult{Status: "not_found", Detail: "no layers in manifest", Digest: digest}
		out.Skills = imageSkillsExtractResult{Status: "not_found", Detail: "no layers in manifest", Digest: digest}
		return out
	}
	// Upper layers last in OCI; search from top so later layers win.
	found := map[string]string{}
	for i := len(layers) - 1; i >= 0; i-- {
		blob, err := fetchRegistryBlob(registry, repository, layers[i], authHeader)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "auth_failed") {
				out.AutoRun = autoRunStepsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
				out.Skills = imageSkillsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
				return out
			}
			continue
		}
		batch, err := findImageFilesInLayerBlob(blob, []string{autoWanted, skillsWanted})
		if err != nil {
			continue
		}
		for k, v := range batch {
			if _, ok := found[k]; !ok {
				found[k] = v
			}
		}
		if len(found) == 2 {
			break
		}
	}
	out.AutoRun = buildAutoRunFromFound(found, autoWanted, digest)
	out.Skills = buildSkillsFromFound(found, skillsWanted, digest)
	return out
}

func buildAutoRunFromFound(found map[string]string, wanted, digest string) autoRunStepsExtractResult {
	md, ok := found[wanted]
	if !ok {
		return autoRunStepsExtractResult{Status: "not_found", Detail: "autoRunStep.md not found in image layers", Digest: digest}
	}
	return autoRunStepsExtractResult{Markdown: md, Digest: digest, Status: "ok"}
}

func buildSkillsFromFound(found map[string]string, wanted, digest string) imageSkillsExtractResult {
	raw, ok := found[wanted]
	if !ok {
		return imageSkillsExtractResult{Status: "not_found", Detail: "imageSkills.yaml not found in image layers", Digest: digest}
	}
	list, err := parseImageSkillsYAML(raw)
	if err != nil {
		return imageSkillsExtractResult{Status: "failed", Detail: err.Error(), Digest: digest}
	}
	body, err := json.Marshal(list)
	if err != nil {
		return imageSkillsExtractResult{Status: "failed", Detail: err.Error(), Digest: digest}
	}
	return imageSkillsExtractResult{List: list, JSON: string(body), Digest: digest, Status: "ok"}
}

// findImageFilesInLayerBlob scans a single layer tar for several in-image paths
// in one pass, returning the content of each file that is present.
func findImageFilesInLayerBlob(blob []byte, wantedPaths []string) (map[string]string, error) {
	rc, err := openLayerBlobReader(blob)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return readFilesFromTarStream(rc, wantedPaths)
}

func readFilesFromTarStream(r io.Reader, wantedPaths []string) (map[string]string, error) {
	wanted := make([]string, 0, len(wantedPaths))
	wantedSet := make(map[string]bool, len(wantedPaths))
	for _, p := range wantedPaths {
		w := normalizeInImagePath(p)
		if !wantedSet[w] {
			wantedSet[w] = true
			wanted = append(wanted, w)
		}
	}
	out := map[string]string{}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		w, ok := tarEntryWanted(hdr.Name, wantedSet)
		if !ok {
			continue
		}
		if _, seen := out[w]; seen {
			continue
		}
		limited := io.LimitReader(tr, maxMarkdownBytes+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > maxMarkdownBytes {
			return nil, fmt.Errorf("file too large: %s", w)
		}
		out[w] = string(data)
		if len(out) == len(wanted) {
			break
		}
	}
	return out, nil
}

// tarEntryWanted normalizes a tar entry name and reports whether it matches any
// wanted in-image path, returning the matched wanted key (same matching rule as
// tarEntryMatchesWanted).
func tarEntryWanted(name string, wantedSet map[string]bool) (string, bool) {
	n := path.Clean("/" + strings.TrimPrefix(strings.TrimSpace(name), "./"))
	if wantedSet[n] {
		return n, true
	}
	base := strings.TrimPrefix(n, "/")
	for w := range wantedSet {
		if strings.TrimPrefix(w, "/") == base {
			return w, true
		}
	}
	return "", false
}
