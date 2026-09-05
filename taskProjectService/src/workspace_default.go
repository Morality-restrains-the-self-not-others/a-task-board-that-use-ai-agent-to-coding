package main

import (
	"database/sql"
	"fmt"
)

func unsetTenantWorkspaceDefaultsTx(tx *sql.Tx, tenantID string) error {
	_, err := tx.Exec("UPDATE project_workspace_entries SET is_default=0 WHERE company_id=?", tenantID)
	return err
}

// applyWorkspaceDefaultFlag sets is_default for one workspace. When isDefault is true,
// sibling workspaces in the same tenant are cleared so at most one default exists.
func applyWorkspaceDefaultFlag(tenantID, workspaceID string, isDefault bool) error {
	if tenantID == "" || workspaceID == "" {
		return fmt.Errorf("tenant and workspace required")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if isDefault {
		if err := unsetTenantWorkspaceDefaultsTx(tx, tenantID); err != nil {
			return err
		}
	}
	flag := 0
	if isDefault {
		flag = 1
	}
	if _, err := tx.Exec(
		"UPDATE project_workspace_entries SET is_default=? WHERE id=? AND company_id=?",
		flag, workspaceID, tenantID,
	); err != nil {
		return err
	}
	return tx.Commit()
}
