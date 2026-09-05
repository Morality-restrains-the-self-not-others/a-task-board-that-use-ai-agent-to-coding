package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openAPISpecYAML []byte

var (
	openAPIJSONCache []byte
	openAPIJSONOnce  sync.Once
	openAPIJSONErr   error
)

func gatewayPublicBase() string {
	if v := strings.TrimSpace(os.Getenv("TASK_GATEWAY_PUBLIC_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://127.0.0.1:8443"
}

func buildOpenAPIJSON() ([]byte, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(openAPISpecYAML, &doc); err != nil {
		return nil, err
	}
	base := gatewayPublicBase()
	doc["servers"] = []map[string]interface{}{
		{"url": base, "description": "taskGateway"},
	}
	return json.Marshal(doc)
}

func openAPIJSON() ([]byte, error) {
	openAPIJSONOnce.Do(func() {
		openAPIJSONCache, openAPIJSONErr = buildOpenAPIJSON()
		if openAPIJSONErr != nil {
			log.Printf("[taskProjectService] openapi build: %v", openAPIJSONErr)
		}
	})
	return openAPIJSONCache, openAPIJSONErr
}

func handleOpenAPISchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := openAPIJSON()
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "schema unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
