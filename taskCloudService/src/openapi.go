package main

import (
	_ "embed"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openAPISpecYAML []byte

//go:embed openapi-internal.yaml
var openAPIInternalSpecYAML []byte

var (
	openAPIJSONCache []byte
	openAPIJSONOnce  sync.Once
	openAPIJSONErr   error

	openAPIInternalJSONCache []byte
	openAPIInternalJSONOnce  sync.Once
	openAPIInternalJSONErr   error
)

func gatewayPublicBase() string {
	if v := strings.TrimSpace(os.Getenv("TASK_GATEWAY_PUBLIC_BASE")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://127.0.0.1:8443"
}

func buildOpenAPIJSONFrom(spec []byte, serverURL, serverDesc string) ([]byte, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		return nil, err
	}
	doc["servers"] = []map[string]interface{}{
		{"url": serverURL, "description": serverDesc},
	}
	return json.Marshal(doc)
}

func buildOpenAPIJSON() ([]byte, error) {
	return buildOpenAPIJSONFrom(openAPISpecYAML, gatewayPublicBase(), "taskGateway")
}

func buildOpenAPIInternalJSON() ([]byte, error) {
	base := "http://127.0.0.1:8018"
	if cfg.Port > 0 {
		host := strings.TrimSpace(cfg.Host)
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		base = "http://" + host + ":" + strconv.Itoa(cfg.Port)
	}
	return buildOpenAPIJSONFrom(openAPIInternalSpecYAML, base, "taskCloudService direct (ops)")
}

func openAPIJSON() ([]byte, error) {
	openAPIJSONOnce.Do(func() {
		openAPIJSONCache, openAPIJSONErr = buildOpenAPIJSON()
		if openAPIJSONErr != nil {
			log.Printf("[taskCloudService] openapi build: %v", openAPIJSONErr)
		}
	})
	return openAPIJSONCache, openAPIJSONErr
}

func openAPIInternalJSON() ([]byte, error) {
	openAPIInternalJSONOnce.Do(func() {
		openAPIInternalJSONCache, openAPIInternalJSONErr = buildOpenAPIInternalJSON()
		if openAPIInternalJSONErr != nil {
			log.Printf("[taskCloudService] openapi-internal build: %v", openAPIInternalJSONErr)
		}
	})
	return openAPIInternalJSONCache, openAPIInternalJSONErr
}

func handleOpenAPISchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := openAPIJSON()
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, "schema unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func handleOpenAPIInternalSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := openAPIInternalJSON()
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, "schema unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
