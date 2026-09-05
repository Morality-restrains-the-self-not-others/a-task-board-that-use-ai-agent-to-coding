package main

import (
	"context"
	"net/http"
	"regexp"
	"strings"
)

const maxSystemAdminListEnrichmentScan = 5000

var systemAdminListDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type systemAdminUserListFilters struct {
	id               string
	email            string
	phone            string
	tenantCompany    string
	loginMethod      string
	referrer         string
	dateJoinedFrom   string
	dateJoinedTo     string
	lastLoginFrom    string
	lastLoginTo      string
	isActive         string
	role             string
	hasProfitSharing string
}

func parseSystemAdminUserListFilters(r *http.Request) systemAdminUserListFilters {
	q := r.URL.Query()
	trim := func(key string) string { return strings.TrimSpace(q.Get(key)) }
	f := systemAdminUserListFilters{
		id:            trim("id"),
		email:         trim("email"),
		phone:         trim("phone"),
		tenantCompany: trim("tenant_company"),
		referrer:      trim("referrer"),
	}
	if lm := trim("login_method"); lm == "email" || lm == "phone" || lm == "username" {
		f.loginMethod = lm
	}
	if d := trim("date_joined_from"); systemAdminListDateRe.MatchString(d) {
		f.dateJoinedFrom = d
	}
	if d := trim("date_joined_to"); systemAdminListDateRe.MatchString(d) {
		f.dateJoinedTo = d
	}
	if d := trim("last_login_from"); systemAdminListDateRe.MatchString(d) {
		f.lastLoginFrom = d
	}
	if d := trim("last_login_to"); systemAdminListDateRe.MatchString(d) {
		f.lastLoginTo = d
	}
	if v := trim("is_active"); v == "true" || v == "false" {
		f.isActive = v
	}
	if role := trim("role"); role == "superuser" || role == "staff" || role == "tenant" || role == "user" || role == "tester" {
		f.role = role
	}
	if v := trim("has_profit_sharing"); v == "true" || v == "false" {
		f.hasProfitSharing = v
	}
	return f
}

func (f systemAdminUserListFilters) keys() []string {
	var keys []string
	add := func(name, val string) {
		if val != "" {
			keys = append(keys, name)
		}
	}
	add("id", f.id)
	add("email", f.email)
	add("phone", f.phone)
	add("tenant_company", f.tenantCompany)
	add("login_method", f.loginMethod)
	add("referrer", f.referrer)
	add("date_joined_from", f.dateJoinedFrom)
	add("date_joined_to", f.dateJoinedTo)
	add("last_login_from", f.lastLoginFrom)
	add("last_login_to", f.lastLoginTo)
	add("is_active", f.isActive)
	add("role", f.role)
	add("has_profit_sharing", f.hasProfitSharing)
	return keys
}

func (f systemAdminUserListFilters) needsEnrichment() bool {
	return f.tenantCompany != "" || f.referrer != "" || f.hasProfitSharing != ""
}

func likeContainsPattern(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	q = strings.ReplaceAll(q, "%", "")
	q = strings.ReplaceAll(q, "_", "")
	return "%" + q + "%"
}

func loginMethodExistsClause(methodType, identifierLike string) (string, []interface{}) {
	args := []interface{}{}
	sql := `EXISTS (SELECT 1 FROM auth_login_method lm WHERE lm.object_id = auth_user.id AND lm.binding_voided_at IS NULL`
	if methodType != "" {
		sql += ` AND lm.method_type = ?`
		args = append(args, methodType)
	}
	if identifierLike != "" {
		sql += ` AND LOWER(lm.identifier) LIKE ?`
		args = append(args, identifierLike)
	}
	sql += `)`
	return sql, args
}

func (f systemAdminUserListFilters) localWhere(matchedIDs []string, archivedFilter string, identifierLookup bool) (clauses []string, args []interface{}) {
	if len(matchedIDs) > 0 {
		placeholders := make([]string, len(matchedIDs))
		for i, id := range matchedIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		clauses = append(clauses, "id IN ("+strings.Join(placeholders, ",")+")")
	}
	if archivedFilter == "true" {
		clauses = append(clauses, "COALESCE(is_archived, 0) = 1")
	} else if archivedFilter == "false" && !identifierLookup {
		clauses = append(clauses, "COALESCE(is_archived, 0) = 0")
	}
	if f.id != "" {
		clauses = append(clauses, "CAST(id AS CHAR) LIKE ?")
		args = append(args, likeContainsPattern(f.id))
	}
	if f.email != "" {
		sql, a := loginMethodExistsClause("email", likeContainsPattern(f.email))
		clauses = append(clauses, sql)
		args = append(args, a...)
	}
	if f.phone != "" {
		sql, a := loginMethodExistsClause("phone", likeContainsPattern(f.phone))
		clauses = append(clauses, sql)
		args = append(args, a...)
	}
	if f.loginMethod != "" {
		sql, a := loginMethodExistsClause(f.loginMethod, "")
		clauses = append(clauses, sql)
		args = append(args, a...)
	}
	if f.dateJoinedFrom != "" {
		clauses = append(clauses, "date_joined >= ?")
		args = append(args, f.dateJoinedFrom)
	}
	if f.dateJoinedTo != "" {
		clauses = append(clauses, "date_joined < DATE_ADD(?, INTERVAL 1 DAY)")
		args = append(args, f.dateJoinedTo)
	}
	if f.lastLoginFrom != "" {
		clauses = append(clauses, "last_login >= ?")
		args = append(args, f.lastLoginFrom)
	}
	if f.lastLoginTo != "" {
		clauses = append(clauses, "last_login < DATE_ADD(?, INTERVAL 1 DAY)")
		args = append(args, f.lastLoginTo)
	}
	if f.isActive == "true" {
		clauses = append(clauses, "is_active = 1")
	} else if f.isActive == "false" {
		clauses = append(clauses, "is_active = 0")
	}
	switch f.role {
	case "superuser":
		clauses = append(clauses, "is_superuser = 1")
	case "staff":
		clauses = append(clauses, "is_staff = 1 AND is_superuser = 0")
	case "tenant":
		clauses = append(clauses, "COALESCE(is_tenant, 0) = 1 AND COALESCE(is_tester, 0) = 0 AND is_superuser = 0 AND is_staff = 0")
	case "tester":
		clauses = append(clauses, "COALESCE(is_tester, 0) = 1")
	case "user":
		clauses = append(clauses, "is_superuser = 0 AND is_staff = 0 AND COALESCE(is_tenant, 0) = 0 AND COALESCE(is_tester, 0) = 0")
	}
	return clauses, args
}

func querySystemAdminUserIDs(whereSQL string, whereArgs []interface{}, limit int) ([]string, error) {
	q := `SELECT id FROM auth_user ` + whereSQL + ` ORDER BY date_joined DESC LIMIT ?`
	args := append(append([]interface{}{}, whereArgs...), limit)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func paginateIDs(ids []string, offset, limit int) []string {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(ids) {
		return nil
	}
	end := offset + limit
	if end > len(ids) {
		end = len(ids)
	}
	return ids[offset:end]
}

func containsFold(hay, needle string) bool {
	return strings.Contains(strings.ToLower(hay), strings.ToLower(strings.TrimSpace(needle)))
}

func applySystemAdminListEnrichment(ctx context.Context, ids []string, f systemAdminUserListFilters) []string {
	out := ids
	if f.tenantCompany != "" {
		byUser := fetchTenantCompaniesBatch(ctx, out)
		next := make([]string, 0, len(out))
		for _, id := range out {
			for _, co := range byUser[id] {
				if containsFold(co["name"], f.tenantCompany) || containsFold(co["id"], f.tenantCompany) {
					next = append(next, id)
					break
				}
			}
		}
		out = next
	}
	if f.referrer != "" {
		refs := fetchReferrersBatch(ctx, out)
		referrerIDs := make([]string, 0, len(refs))
		seen := map[string]bool{}
		for _, e := range refs {
			rid := strings.TrimSpace(e["referrer_user_id"])
			if rid != "" && !seen[rid] {
				seen[rid] = true
				referrerIDs = append(referrerIDs, rid)
			}
		}
		profiles := batchProfileUsernames(referrerIDs)
		logins := batchLoginMethods(referrerIDs)
		next := make([]string, 0, len(out))
		for _, id := range out {
			e, ok := refs[id]
			if !ok {
				continue
			}
			name := referrerDisplayName(e["referrer_user_id"], profiles, logins)
			if containsFold(name, f.referrer) || containsFold(e["referrer_user_id"], f.referrer) {
				next = append(next, id)
			}
		}
		out = next
	}
	if f.hasProfitSharing != "" {
		want := f.hasProfitSharing == "true"
		quals, ok := fetchQualificationsBatch(ctx, out)
		next := make([]string, 0, len(out))
		for _, id := range out {
			has := ok && quals[id]
			if has == want {
				next = append(next, id)
			}
		}
		out = next
	}
	return out
}
