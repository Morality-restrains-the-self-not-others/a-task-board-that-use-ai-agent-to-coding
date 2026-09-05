package saas

import (
	"fmt"
	"strings"

	"taskEvents/internal/repository/cloudconfig"
)

// ClearCloudServerConfigAfterStop clears reachability in taskCloudService SSOT.
// Deprecated: use cloudconfig.ClearAfterStop.
func (r *Repository) ClearCloudServerConfigAfterStop(companyID int64, workspaceID, taskID, reason string) error {
	_ = r
	return cloudconfig.ClearAfterStop(strings.TrimSpace(fmt.Sprint(companyID)), workspaceID, taskID, reason, "", "")
}

// RequestClearReachabilityOnTaskCloudService asks Go SSOT to clear reachability natively.
// Deprecated: use cloudconfig.ClearAfterStop.
func RequestClearReachabilityOnTaskCloudService(tenantID, workspaceID, taskID, reason string) error {
	return cloudconfig.ClearAfterStop(tenantID, workspaceID, taskID, reason, "", "")
}
