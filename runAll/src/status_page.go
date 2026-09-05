package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed all:status_ui
var statusUI embed.FS

// statusHTML keeps the historical embed symbol used by tests/helpers.
// It serves the assembled page bytes under the logical name "status.html".
var statusHTML = assembledStatusFS{}

type statusUIManifest struct {
	CSS []string `json:"css"`
	JS  []string `json:"js"`
}

type assembledStatusFS struct{}

func (assembledStatusFS) ReadFile(name string) ([]byte, error) {
	if name != "status.html" {
		return nil, fmt.Errorf("status asset %q not found", name)
	}
	return assembledStatusPage()
}

var (
	assembledStatusOnce sync.Once
	assembledStatusData []byte
	assembledStatusErr  error
)

func assembledStatusPage() ([]byte, error) {
	assembledStatusOnce.Do(func() {
		assembledStatusData, assembledStatusErr = buildStatusPageFromUI()
	})
	return assembledStatusData, assembledStatusErr
}

func buildStatusPageFromUI() ([]byte, error) {
	raw, err := statusUI.ReadFile("status_ui/index.html")
	if err != nil {
		return nil, fmt.Errorf("read status_ui/index.html: %w", err)
	}
	manifestRaw, err := statusUI.ReadFile("status_ui/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read status_ui/manifest.json: %w", err)
	}
	var manifest statusUIManifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return nil, fmt.Errorf("parse status_ui/manifest.json: %w", err)
	}

	css, err := concatStatusUIFiles("status_ui/css", manifest.CSS)
	if err != nil {
		return nil, err
	}
	js, err := concatStatusUIFiles("status_ui/js", manifest.JS)
	if err != nil {
		return nil, err
	}

	page := string(raw)
	page = strings.Replace(page, "/*EMBED_CSS*/", css, 1)
	page = strings.Replace(page, "/*EMBED_JS*/", js, 1)
	if strings.Contains(page, "/*EMBED_CSS*/") || strings.Contains(page, "/*EMBED_JS*/") {
		return nil, fmt.Errorf("status_ui/index.html missing embed markers")
	}
	return []byte(page), nil
}

func concatStatusUIFiles(dir string, names []string) (string, error) {
	if len(names) == 0 {
		return "", fmt.Errorf("%s: empty manifest list", dir)
	}
	var buf bytes.Buffer
	for i, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
			return "", fmt.Errorf("%s: invalid fragment name %q", dir, name)
		}
		data, err := statusUI.ReadFile(dir + "/" + name)
		if err != nil {
			return "", fmt.Errorf("read %s/%s: %w", dir, name, err)
		}
		if i > 0 && buf.Len() > 0 && buf.Bytes()[buf.Len()-1] != '\n' {
			buf.WriteByte('\n')
		}
		buf.Write(data)
	}
	return buf.String(), nil
}
