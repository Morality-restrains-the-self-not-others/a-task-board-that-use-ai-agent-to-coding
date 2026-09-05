package saas

import (
	"fmt"
	"time"
)

// HandleCompanyCreated runs the COMPANY_CREATED chain: deliverable → progress → workspace.
func (r *Repository) HandleCompanyCreated(companyID int64, creatorID, companyName string, publish func(workspaceID string, workspaceName string, createdAt time.Time) error) error {
	if err := r.IntentSetDefaultDeliverable(companyID); err != nil {
		return err
	}
	if err := r.IntentSetDefaultProgress(companyID); err != nil {
		return err
	}
	return r.IntentCreateDefaultWorkspace(companyID, creatorID, companyName, publish)
}

// IntentSetDefaultDeliverable is v4 intent 1 for COMPANY_CREATED.
// Calls taskProjectService directly (bypasses Django).
func (r *Repository) IntentSetDefaultDeliverable(companyID int64) error {
	return r.setDefaultDeliverableDirect(fmt.Sprint(companyID))
}

// IntentSetDefaultProgress is v4 intent 2 for COMPANY_CREATED.
// Calls taskProjectService directly (bypasses Django).
func (r *Repository) IntentSetDefaultProgress(companyID int64) error {
	return r.setDefaultProgressDirect(fmt.Sprint(companyID))
}

// IntentInitTenantFeatureParams is v4 intent 5 for COMPANY_CREATED.
// Calls taskCloudService directly (bypasses Django).
func (r *Repository) IntentInitTenantFeatureParams(companyID string) (map[string]interface{}, error) {
	return r.InitTenantFeatureParamsDirect(companyID)
}

// IntentGrantInitialResources is v4 intent 6 for COMPANY_CREATED.
// Calls taskBill directly (bypasses Django).
func (r *Repository) IntentGrantInitialResources(companyID, creatorID string) (map[string]interface{}, error) {
	return r.grantInitialResourcesDirect(companyID, creatorID)
}

// IntentCreateDefaultWorkspace is v4 intent 3 for COMPANY_CREATED.
// Calls taskProjectService directly (bypasses Django).
func (r *Repository) IntentCreateDefaultWorkspace(companyID int64, creatorID, companyName string, publish func(workspaceID string, workspaceName string, createdAt time.Time) error) error {
	ws, err := r.createDefaultWorkspaceDirect(fmt.Sprint(companyID), creatorID, companyName)
	if err != nil {
		return err
	}
	if publish == nil {
		return nil
	}
	return publish(ws.WorkspaceID, ws.WorkspaceName, ws.CreatedAt)
}

// IntentMarkUserTenant sets is_tenant=true on a paying user directly via taskAuth PATCH.
// Formerly called Django intent endpoint which just forwarded to taskAuth.
func (r *Repository) IntentMarkUserTenant(userID string) error {
	if err := r.markUserTenantDirect(userID); err != nil {
		return err
	}
	return nil
}
