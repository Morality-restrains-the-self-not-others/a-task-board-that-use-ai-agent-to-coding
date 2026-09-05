package main

import (
	"net/http"

	"authz"
)

func canGrantSuperuser(r *http.Request, actorID string) bool {
	if authz.HasPlatformPerm(r, authz.PermPlatformManage) {
		return true
	}
	names, err := platformRoleNamesFromTable(actorID)
	if err != nil {
		return false
	}
	for _, name := range names {
		if name == "super_admin" {
			return true
		}
	}
	return false
}

// rejectUnauthorizedSuperuserGrant returns true after writing 403 when the
// body tries to set is_superuser without super_admin / platform:manage.
func rejectUnauthorizedSuperuserGrant(w http.ResponseWriter, r *http.Request, body map[string]interface{}) bool {
	if _, ok := body["is_superuser"]; !ok {
		return false
	}
	actorID, _ := resolveUserIDFromRequest(r)
	if actorID != "" && canGrantSuperuser(r, actorID) {
		return false
	}
	writeErrorDetail(w, r, http.StatusForbidden, "仅系统管理员可设置超级用户")
	return true
}
