package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// imageIdentity holds local image Id (config hash) and registry RepoDigest when available.
type imageIdentity struct {
	ID     string // e.g. sha256:abc…
	Digest string // e.g. registry.example/app@sha256:def… (first RepoDigest)
}

// inspectImageIdentity runs `docker image inspect` for Id + RepoDigests after pull.
func inspectImageIdentity(image string) (imageIdentity, error) {
	ref := strings.TrimSpace(image)
	if ref == "" {
		return imageIdentity{}, fmt.Errorf("empty image ref")
	}
	out, errText, code, err := runDockerCLI(
		"image", "inspect", "--format", "{{.Id}}\t{{json .RepoDigests}}", ref,
	)
	if err != nil {
		return imageIdentity{}, err
	}
	if code != 0 {
		msg := strings.TrimSpace(errText)
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		return imageIdentity{}, fmt.Errorf("docker image inspect failed (code=%d): %s", code, msg)
	}
	return parseImageInspectIdentity(out)
}

func parseImageInspectIdentity(raw string) (imageIdentity, error) {
	line := strings.TrimSpace(raw)
	if line == "" {
		return imageIdentity{}, fmt.Errorf("empty inspect output")
	}
	// Prefer first line only (inspect may print one row).
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	idPart, digestsJSON, ok := strings.Cut(line, "\t")
	if !ok {
		// Fallback: whole line is Id.
		id := strings.TrimSpace(line)
		if id == "" {
			return imageIdentity{}, fmt.Errorf("inspect missing Id")
		}
		return imageIdentity{ID: id}, nil
	}
	id := strings.TrimSpace(idPart)
	var digests []string
	_ = json.Unmarshal([]byte(strings.TrimSpace(digestsJSON)), &digests)
	digest := ""
	for _, d := range digests {
		d = strings.TrimSpace(d)
		if d != "" {
			digest = d
			break
		}
	}
	if id == "" && digest == "" {
		return imageIdentity{}, fmt.Errorf("inspect missing Id and RepoDigests")
	}
	return imageIdentity{ID: id, Digest: digest}, nil
}

func formatImageIdentityLog(image string, ident imageIdentity) string {
	parts := []string{fmt.Sprintf("image=%s", strings.TrimSpace(image))}
	if ident.ID != "" {
		parts = append(parts, fmt.Sprintf("id=%s", ident.ID))
	}
	if ident.Digest != "" {
		parts = append(parts, fmt.Sprintf("digest=%s", ident.Digest))
	}
	return "[relayToTrae] docker pull ok: " + strings.Join(parts, " ")
}
