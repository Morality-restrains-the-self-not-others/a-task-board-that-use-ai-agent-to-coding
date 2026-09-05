package main

import (
	"snowflake"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	budgetSubjectMember    = "member"
	budgetSubjectGroup     = "group"
	budgetSubjectOwnerRole = "owner_role"
)

type tenantBudgetPermissionRow struct {
	ID                 string
	CompanyID          string
	SubjectType        string
	SubjectID          string
	CanRaiseTaskBudget bool
}

func listTenantBudgetPermissions(conn *sql.DB, companyID string) ([]tenantBudgetPermissionRow, error) {
	if conn == nil {
		return nil, fmt.Errorf("budget db not open")
	}
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("company_id required")
	}
	rows, err := conn.Query(`
		SELECT id, company_id, subject_type, subject_id, COALESCE(can_raise_task_budget, 0)
		FROM cloud_tenant_budget_permission
		WHERE company_id = ?
		ORDER BY subject_type, subject_id
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]tenantBudgetPermissionRow, 0)
	for rows.Next() {
		var r tenantBudgetPermissionRow
		var canRaise int
		if err := rows.Scan(&r.ID, &r.CompanyID, &r.SubjectType, &r.SubjectID, &canRaise); err != nil {
			return nil, err
		}
		r.CanRaiseTaskBudget = canRaise != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func upsertTenantBudgetPermission(conn *sql.DB, row tenantBudgetPermissionRow) (tenantBudgetPermissionRow, error) {
	if conn == nil {
		return row, fmt.Errorf("budget db not open")
	}
	companyID := strings.TrimSpace(row.CompanyID)
	subjectType := strings.TrimSpace(row.SubjectType)
	subjectID := strings.TrimSpace(row.SubjectID)
	if companyID == "" || subjectType == "" || subjectID == "" {
		return row, fmt.Errorf("company_id, subject_type, subject_id required")
	}
	if !validBudgetSubjectType(subjectType) {
		return row, fmt.Errorf("subject_type invalid")
	}
	canRaise := 0
	if row.CanRaiseTaskBudget {
		canRaise = 1
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	var existingID string
	err := conn.QueryRow(`
		SELECT id FROM cloud_tenant_budget_permission
		WHERE company_id = ? AND subject_type = ? AND subject_id = ?
		LIMIT 1
	`, companyID, subjectType, subjectID).Scan(&existingID)
	if err == nil && existingID != "" {
		_, err = conn.Exec(`
			UPDATE cloud_tenant_budget_permission
			SET can_raise_task_budget=?, updated_at=?
			WHERE id=?
		`, canRaise, now, existingID)
		if err != nil {
			return row, err
		}
		row.ID = existingID
	} else if err != nil && err != sql.ErrNoRows {
		return row, err
	} else {
		id := snowflake.GenerateIDString()
		_, err = conn.Exec(`
			INSERT INTO cloud_tenant_budget_permission
			(id, company_id, subject_type, subject_id, can_raise_task_budget, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, id, companyID, subjectType, subjectID, canRaise, now, now)
		if err != nil {
			return row, err
		}
		row.ID = id
	}
	row.CompanyID = companyID
	row.SubjectType = subjectType
	row.SubjectID = subjectID
	return row, nil
}

func upsertTenantBudgetPermissions(conn *sql.DB, companyID string, items []tenantBudgetPermissionRow) ([]tenantBudgetPermissionRow, error) {
	for _, item := range items {
		item.CompanyID = companyID
		if _, err := upsertTenantBudgetPermission(conn, item); err != nil {
			return nil, err
		}
	}
	return listTenantBudgetPermissions(conn, companyID)
}

func validBudgetSubjectType(t string) bool {
	switch t {
	case budgetSubjectMember, budgetSubjectGroup, budgetSubjectOwnerRole:
		return true
	default:
		return false
	}
}

func tenantBudgetPermissionToMap(r tenantBudgetPermissionRow) map[string]any {
	return map[string]any{
		"subject_type":          r.SubjectType,
		"subject_id":            r.SubjectID,
		"can_raise_task_budget": r.CanRaiseTaskBudget,
	}
}

// evaluateUserCanRaiseTaskBudget mirrors Django user_can_raise_task_budget.
func evaluateUserCanRaiseTaskBudget(
	conn *sql.DB,
	companyID, userID, memberID, taskOwnerMemberID string,
	isTenantAdmin, isWorkspaceAdmin bool,
) (bool, error) {
	if isTenantAdmin || isWorkspaceAdmin {
		return true, nil
	}
	if conn == nil {
		return false, fmt.Errorf("budget db not open")
	}
	companyID = strings.TrimSpace(companyID)
	userID = strings.TrimSpace(userID)
	memberID = strings.TrimSpace(memberID)
	if companyID == "" || memberID == "" {
		return false, nil
	}
	perms, err := listTenantBudgetPermissions(conn, companyID)
	if err != nil {
		return false, err
	}
	taskOwnerMemberID = strings.TrimSpace(taskOwnerMemberID)
	for _, perm := range perms {
		if !perm.CanRaiseTaskBudget {
			continue
		}
		switch perm.SubjectType {
		case budgetSubjectMember:
			if perm.SubjectID == memberID {
				return true, nil
			}
		case budgetSubjectOwnerRole:
			if taskOwnerMemberID != "" && taskOwnerMemberID == memberID {
				return true, nil
			}
		case budgetSubjectGroup:
			ok, err := userInCompanyGroupHTTP(perm.SubjectID, userID)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

// userInCompanyGroupHTTP checks group membership via taskTenantService (no Django dependency).
// OPT-20260729-024 #10: migrated from Django /api/internal/taskproject/user-in-group/ to Go direct.
func userInCompanyGroupHTTP(groupID, userID string) (bool, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" || userID == "" {
		return false, nil
	}
	if strings.TrimSpace(cfg.TaskTenantURL) == "" {
		return false, nil
	}
	return tenantUserInGroup(groupID, userID)
}

// lookupIsCompanyCreator checks if user is the company creator via taskTenantService.
// Migrated from Django resolve-user-member is_creator field (OPT-20260729-024 #8).
func lookupIsCompanyCreator(tenantID, userID string) bool {
	tenantID = strings.TrimSpace(tenantID)
	userID = strings.TrimSpace(userID)
	if tenantID == "" || userID == "" || strings.TrimSpace(cfg.TaskTenantURL) == "" {
		return false
	}
	isCreator, err := tenantIsCompanyCreator(tenantID, userID)
	if err != nil {
		return false
	}
	return isCreator
}
