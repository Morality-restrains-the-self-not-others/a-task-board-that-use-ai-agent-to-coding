package saastest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestIntentMuxConventionWorkspaceAccess pins the convention routing of the
// taskProjectService mock: action prefix first, then tenant_id/{tid} as a
// key=value path param (matching gatewayauth.ParseConventionPath). It also
// pins the real service's method restrictions: workspace-permissions is
// GET-only, set-permission is POST-only.
func TestIntentMuxConventionWorkspaceAccess(t *testing.T) {
	mux := NewIntentMux()
	srv := mux.Server()
	defer srv.Close()

	// POST workspace-permissions must be rejected (GET-only in real service)
	postList := func() int {
		resp, err := http.Post(srv.URL+"/api/projects/workspace-access/workspace-permissions/tenant_id/200/",
			"application/json", bytes.NewBufferString(`{}`))
		if err != nil {
			t.Fatalf("POST workspace-permissions: %v", err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}
	if code := postList(); code != 405 {
		t.Fatalf("expected 405 for POST workspace-permissions, got %d", code)
	}

	// GET workspace-permissions returns empty list
	resp, err := http.Get(srv.URL + "/api/projects/workspace-access/workspace-permissions/tenant_id/200/?workspace_id=ws_1")
	if err != nil {
		t.Fatalf("GET workspace-permissions: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for GET workspace-permissions, got %d", resp.StatusCode)
	}

	// POST set-permission creates the access row
	body, _ := json.Marshal(map[string]string{
		"workspace_id": "ws_1",
		"user_id":      "42",
		"permission":   "admin",
	})
	resp, err = http.Post(srv.URL+"/api/projects/workspace-access/set-permission/tenant_id/200/",
		"application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST set-permission: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for POST set-permission, got %d", resp.StatusCode)
	}

	// GET workspace-permissions now returns the created row
	resp, err = http.Get(srv.URL + "/api/projects/workspace-access/workspace-permissions/tenant_id/200/?workspace_id=ws_1")
	if err != nil {
		t.Fatalf("GET workspace-permissions: %v", err)
	}
	defer resp.Body.Close()
	var entries []WorkspaceAccessEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("decode access list: %v", err)
	}
	if len(entries) != 1 || entries[0].UserID != "42" || entries[0].Permission != "admin" || entries[0].WorkspaceID != "ws_1" {
		t.Fatalf("unexpected access rows: %+v", entries)
	}

	// set-permission is idempotent per (workspace, user): re-POST updates, no dup
	resp, err = http.Post(srv.URL+"/api/projects/workspace-access/set-permission/tenant_id/200/",
		"application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST set-permission (2nd): %v", err)
	}
	resp.Body.Close()
	if len(mux.WorkspaceAccesses) != 1 {
		t.Fatalf("expected 1 access row after upsert, got %d", len(mux.WorkspaceAccesses))
	}
}

// TestIntentMuxConventionOtherResources pins convention routing for the
// deliverable-systems / settings / workspaces shapes used by taskEvents.
func TestIntentMuxConventionOtherResources(t *testing.T) {
	mux := NewIntentMux()
	srv := mux.Server()
	defer srv.Close()

	// deliverable-systems list (tenant_id at the end of the path)
	resp, err := http.Get(srv.URL + "/api/projects/deliverable-systems/tenant_id/200/")
	if err != nil {
		t.Fatalf("GET deliverable-systems: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for GET deliverable-systems, got %d", resp.StatusCode)
	}

	// deliverable-systems set-default
	resp, err = http.Post(srv.URL+"/api/projects/deliverable-systems/ds_default_global/set-default/tenant_id/200/",
		"application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("POST deliverable-systems set-default: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || !mux.DeliverableTenants["200"] {
		t.Fatalf("expected set-default to record tenant 200 (status %d)", resp.StatusCode)
	}

	// settings default-progress-system
	resp, err = http.Get(srv.URL + "/api/projects/settings/default-progress-system/tenant_id/200/")
	if err != nil {
		t.Fatalf("GET settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for GET settings, got %d", resp.StatusCode)
	}
}

// TestIntentMuxConventionTenantIDFirst pins the 2026-08-07 fix: key=value pairs
// (tenant_id/{tid}) may appear at the START of a convention path, with the
// action segments AFTER them. Before the fix, serveTaskProjectConvention set
// action = parts[:kvStart] (empty), so tenant_id-first paths silently fell into
// the project-list branch — GET returned 200 [] instead of the resource, POST
// returned 404 — mirroring the real-service 405 (positional pid swallowed).
func TestIntentMuxConventionTenantIDFirst(t *testing.T) {
	mux := NewIntentMux()
	srv := mux.Server()
	defer srv.Close()

	// GET workspace-permissions with tenant_id FIRST must return the access
	// list (empty JSON array from the handler), not the project list.
	resp, err := http.Get(srv.URL + "/api/projects/tenant_id/200/workspace-access/workspace-permissions/")
	if err != nil {
		t.Fatalf("GET tenant_id-first workspace-permissions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for tenant_id-first GET, got %d", resp.StatusCode)
	}
	var entries []WorkspaceAccessEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("decode access list: %v", err)
	}
	if entries == nil {
		t.Fatal("expected JSON array from workspace-permissions handler, not project list")
	}

	// POST set-permission with tenant_id first must create the access row
	// (positional suffix preserved → routed to set-permission handler).
	body, _ := json.Marshal(map[string]string{
		"workspace_id": "ws_1",
		"user_id":      "42",
		"permission":   "admin",
	})
	resp, err = http.Post(srv.URL+"/api/projects/tenant_id/200/workspace-access/set-permission/",
		"application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST tenant_id-first set-permission: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for tenant_id-first POST, got %d", resp.StatusCode)
	}
	if len(mux.WorkspaceAccesses) != 1 {
		t.Fatalf("expected 1 access row after tenant_id-first POST, got %d", len(mux.WorkspaceAccesses))
	}
}
