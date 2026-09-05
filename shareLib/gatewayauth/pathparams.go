package gatewayauth

import (
	"net/http"
	"strings"
)

// ParseConventionPath extracts key=value params from convention-compliant paths:
//
//	/api/serviceName/funcName/tenant_id/123/workspace_id/456/task_id/789/comment_id/cmt/rest
//
// The key=value pairs may appear at any position in the path. This function:
//  1. Finds the first recognized key (tenant_id, workspace_id, task_id, user_id, comment_id)
//  2. Parses all subsequent key=value pairs, setting headers
//  3. Returns the remaining path — the action/funcName prefix PLUS any
//     positional segments AFTER the last key=value pair (preserved, not swallowed)
//
// Examples:
//
//	"server-images/vpcs/tenant_id/123" → "server-images/vpcs", sets X-Auth-Tenant-Id=123
//	"tenant_id/123/proj_1"             → "proj_1", sets X-Auth-Tenant-Id=123
//	"proj_1/tenant_id/123"             → "proj_1", sets X-Auth-Tenant-Id=123
func ParseConventionPath(r *http.Request, path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	// Find the index where key=value pairs start
	kvStart := -1
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "tenant_id", "workspace_id", "task_id", "user_id", "comment_id":
			kvStart = i
		}
		if kvStart >= 0 {
			break
		}
	}
	if kvStart < 0 {
		return path // No key=value pairs found
	}
	// Parse key=value pairs and set headers; kvEnd marks the first segment
	// AFTER the last consumed pair (positional suffix, preserved below).
	kvEnd := kvStart
	for i := kvStart; i+1 < len(parts); i += 2 {
		key, val := parts[i], parts[i+1]
		switch key {
		case "tenant_id":
			r.Header.Set(HeaderAuthTenantID, val)
			kvEnd = i + 2
		case "workspace_id":
			r.Header.Set("X-Workspace-Id", val)
			kvEnd = i + 2
		case "task_id":
			r.Header.Set("X-Task-Id", val)
			kvEnd = i + 2
		case "user_id":
			r.Header.Set("X-Auth-User-Id", val)
			kvEnd = i + 2
		case "comment_id":
			r.Header.Set(HeaderCommentID, val)
			kvEnd = i + 2
		default:
			// Unrecognized key — stop parsing; the remainder (this key and
			// everything after it) is a positional suffix preserved below.
			i = len(parts) // break outer loop
		}
	}
	// Return the path minus the consumed key=value pairs (prefix + positional suffix).
	rest := append(append([]string{}, parts[:kvStart]...), parts[kvEnd:]...)
	return strings.Join(rest, "/")
}

// ApplyTenantIDFromPath applies a tenant_id embedded as the first segment of path
// by setting X-Auth-Tenant-Id and returning the remaining path.
// This is used for legacy positional tenant paths: /api/tenant/{tid}/...
func ApplyTenantIDFromPath(r *http.Request, path string) string {
	p := strings.Trim(path, "/")
	idx := strings.Index(p, "/")
	if idx < 0 {
		r.Header.Set(HeaderAuthTenantID, p)
		return ""
	}
	r.Header.Set(HeaderAuthTenantID, p[:idx])
	return p[idx+1:]
}
