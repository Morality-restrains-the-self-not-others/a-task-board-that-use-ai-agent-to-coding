package main

import (
	"net/http"
	"strings"
)

func handleInternalExtractAutoRunSteps(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	imageURL := strField(body, "image_url")
	filePath := strField(body, "path")
	if filePath == "" {
		filePath = defaultAutoRunStepsPath
	}
	if strings.TrimSpace(imageURL) == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "image_url is required")
		return
	}
	result := extractAutoRunStepsFromImage(imageURL, filePath)
	statusCode := http.StatusOK
	if result.Status != "ok" {
		statusCode = http.StatusUnprocessableEntity
	}
	writeErrorMapJSON(w, r, statusCode, map[string]interface{}{
		"markdown": result.Markdown,
		"digest":   result.Digest,
		"status":   result.Status,
		"detail":   result.Detail,
		"path":     normalizeInImagePath(filePath),
	})
}
