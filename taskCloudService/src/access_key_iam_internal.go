package main

import (
	"net/http"
	"strings"
)

func handleInternalAccessKeyIAMRoutes(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/access-key-iam-associations/")
	if path == r.URL.Path {
		path = strings.TrimPrefix(r.URL.Path, "/api/internal/access-key-iam-associations")
	}
	path = strings.Trim(path, "/")
	switch {
	case path == "import" && r.Method == http.MethodPost:
		handleInternalImportAccessKeyIAMAssociations(w, r)
	case path == "" && r.Method == http.MethodPost:
		handleInternalCreateAccessKeyIAMAssociation(w, r)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func handleInternalCreateAccessKeyIAMAssociation(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	authID := strField(body, "auth_id")
	if authID == "" {
		authID = strField(body, "cloud_platform_auth_id")
	}
	accessKey := strField(body, "access_key")
	iamID := strField(body, "iam_id")
	if authID == "" || accessKey == "" || iamID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "auth_id, access_key and iam_id required",)
		return
	}
	if err := createAccessKeyIAMAssociation(authID, accessKey, iamID); err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func handleInternalImportAccessKeyIAMAssociations(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rawItems, _ := body["items"].([]interface{})
	if len(rawItems) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": 0})
		return
	}
	rows := make([]map[string]interface{}, 0, len(rawItems))
	for _, item := range rawItems {
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, m)
		}
	}
	count, err := importAccessKeyIAMAssociations(rows)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": count})
}
