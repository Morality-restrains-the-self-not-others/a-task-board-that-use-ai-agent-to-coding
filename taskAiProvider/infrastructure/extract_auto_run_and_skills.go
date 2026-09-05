package infrastructure

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

// AutoRunAndSkillsExtractResult carries both files matched in a single OCI
// manifest-resolve + layer-blob pass (OPT-20260821-018). Previously
// scheduleAutoRunExtract and scheduleImageSkillsExtract each walked the same
// registry layers separately, doubling manifest fetches and blob downloads
// for every vendor image save / catalog backfill.
type AutoRunAndSkillsExtractResult struct {
	AutoRun AutoRunStepsExtractResult
	Skills  ImageSkillsExtractResult
}

// ExtractAutoRunAndSkillsFromImage resolves the manifest once and walks the
// layer blobs once, matching both /app/autoRunStep.md and /app/imageSkills.yaml
// in the same pass. Per-file results mirror what the two single-file walks
// would return (newest layer wins), so callers can persist both fields with a
// single registry pull.
func ExtractAutoRunAndSkillsFromImage(imageURL string) AutoRunAndSkillsExtractResult {
	return extractAutoRunAndSkillsFromImage(extractRegistryClient(), imageURL)
}

func extractAutoRunAndSkillsFromImage(client registryHTTP, imageURL string) AutoRunAndSkillsExtractResult {
	autoWanted := normalizeInImagePath(DefaultAutoRunStepsPath)
	skillsWanted := normalizeInImagePath(DefaultImageSkillsPath)
	imageURL = strings.TrimSpace(imageURL)
	out := AutoRunAndSkillsExtractResult{}
	if imageURL == "" {
		out.AutoRun = AutoRunStepsExtractResult{Status: "failed", Detail: "image_url is required"}
		out.Skills = ImageSkillsExtractResult{Status: "failed", Detail: "image_url is required"}
		return out
	}
	registry, repository, digest, manifest, authHeader, err := resolveExtractManifest(client, imageURL)
	if err != nil {
		msg := err.Error()
		status := "failed"
		if strings.Contains(msg, "auth_failed") || strings.Contains(msg, "认证") {
			status = "auth_failed"
		}
		out.AutoRun = AutoRunStepsExtractResult{Status: status, Detail: msg, Digest: digest}
		out.Skills = ImageSkillsExtractResult{Status: status, Detail: msg, Digest: digest}
		return out
	}
	layers := layerDigestsFromManifest(manifest)
	if len(layers) == 0 {
		out.AutoRun = AutoRunStepsExtractResult{Status: "not_found", Detail: "no layers in manifest", Digest: digest}
		out.Skills = ImageSkillsExtractResult{Status: "not_found", Detail: "no layers in manifest", Digest: digest}
		return out
	}
	found := map[string]string{}
	for i := len(layers) - 1; i >= 0; i-- {
		blob, err := fetchLayerBlob(client, registry, repository, layers[i], authHeader)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "auth_failed") {
				out.AutoRun = AutoRunStepsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
				out.Skills = ImageSkillsExtractResult{Status: "auth_failed", Detail: msg, Digest: digest}
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
	out.AutoRun = buildAutoRunResult(found, autoWanted, digest)
	out.Skills = buildSkillsResult(found, skillsWanted, digest)
	return out
}

func buildAutoRunResult(found map[string]string, wanted, digest string) AutoRunStepsExtractResult {
	md, ok := found[wanted]
	if !ok {
		return AutoRunStepsExtractResult{Status: "not_found", Detail: "autoRunStep.md not found in image layers", Digest: digest}
	}
	return AutoRunStepsExtractResult{Markdown: md, Digest: digest, Status: "ok"}
}

func buildSkillsResult(found map[string]string, wanted, digest string) ImageSkillsExtractResult {
	raw, ok := found[wanted]
	if !ok {
		return ImageSkillsExtractResult{Status: "not_found", Detail: "imageSkills.yaml not found in image layers", Digest: digest}
	}
	list, err := ParseImageSkillsYAML(raw)
	if err != nil {
		return ImageSkillsExtractResult{Status: "failed", Detail: err.Error(), Digest: digest}
	}
	body, err := json.Marshal(list)
	if err != nil {
		return ImageSkillsExtractResult{Status: "failed", Detail: err.Error(), Digest: digest}
	}
	return ImageSkillsExtractResult{List: list, JSON: string(body), Digest: digest, Status: "ok"}
}

// findImageFilesInLayerBlob scans a single layer tar for several in-image
// paths in one pass, returning the content of each file that is present.
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
