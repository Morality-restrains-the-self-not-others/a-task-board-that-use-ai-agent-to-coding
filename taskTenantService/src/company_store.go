package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type companyRow struct {
	ID        string
	Name      string
	CreatorID string
	CreatedAt string
	UpdatedAt string
}

func companyToJSON(c *companyRow) map[string]interface{} {
	if c == nil {
		return nil
	}
	return map[string]interface{}{
		"id": c.ID, "name": c.Name, "creator_id": c.CreatorID,
		"created_at": c.CreatedAt, "updated_at": c.UpdatedAt,
	}
}

func scanCompany(row interface {
	Scan(dest ...any) error
}) (*companyRow, error) {
	var c companyRow
	err := row.Scan(&c.ID, &c.Name, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func getCompanyByID(companyID string) (*companyRow, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, nil
	}
	return scanCompany(db.QueryRow(`
		SELECT id, COALESCE(name,''), COALESCE(creator_id,''),
		       COALESCE(created_at,''), COALESCE(updated_at,'')
		FROM tenant_company WHERE id=?`, companyID))
}

func fetchCompanyCreator(companyID string) (creatorID string, found bool, err error) {
	c, err := getCompanyByID(companyID)
	if err != nil {
		return "", false, err
	}
	if c == nil {
		return "", false, nil
	}
	return strings.TrimSpace(c.CreatorID), true, nil
}

func getCompanyByName(name string) (*companyRow, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	return scanCompany(db.QueryRow(`
		SELECT id, COALESCE(name,''), COALESCE(creator_id,''),
		       COALESCE(created_at,''), COALESCE(updated_at,'')
		FROM tenant_company WHERE name=? LIMIT 1`, name))
}

func listCompaniesByCreator(creatorID string) ([]companyRow, error) {
	creatorID = strings.TrimSpace(creatorID)
	if creatorID == "" {
		return nil, nil
	}
	rows, err := db.Query(`
		SELECT id, COALESCE(name,''), COALESCE(creator_id,''),
		       COALESCE(created_at,''), COALESCE(updated_at,'')
		FROM tenant_company WHERE creator_id=? ORDER BY id ASC`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]companyRow, 0)
	for rows.Next() {
		var c companyRow
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func listCompaniesByIDs(ids []string) ([]companyRow, error) {
	out := make([]companyRow, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		c, err := getCompanyByID(id)
		if err != nil {
			return nil, err
		}
		if c != nil {
			out = append(out, *c)
		}
	}
	return out, nil
}

func searchCompanies(query string, limit int) ([]companyRow, error) {
	return searchCompaniesPage(query, limit, 0)
}

func countCompanies(query string) (int, error) {
	query = strings.TrimSpace(query)
	var n int
	var err error
	if query == "" {
		err = db.QueryRow(`SELECT COUNT(*) FROM tenant_company`).Scan(&n)
	} else {
		like := "%" + query + "%"
		err = db.QueryRow(
			`SELECT COUNT(*) FROM tenant_company WHERE name LIKE ? OR id LIKE ?`,
			like, like,
		).Scan(&n)
	}
	return n, err
}

func searchCompaniesPage(query string, limit, offset int) ([]companyRow, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 80
	}
	if limit > adminTenantListSearchCap {
		limit = adminTenantListSearchCap
	}
	query = strings.TrimSpace(query)
	var (
		rows *sql.Rows
		err  error
	)
	if query == "" {
		rows, err = db.Query(`
			SELECT id, COALESCE(name,''), COALESCE(creator_id,''),
			       COALESCE(created_at,''), COALESCE(updated_at,'')
			FROM tenant_company ORDER BY name ASC, id ASC LIMIT ? OFFSET ?`, limit, offset)
	} else {
		like := "%" + query + "%"
		rows, err = db.Query(`
			SELECT id, COALESCE(name,''), COALESCE(creator_id,''),
			       COALESCE(created_at,''), COALESCE(updated_at,'')
			FROM tenant_company
			WHERE name LIKE ? OR id LIKE ?
			ORDER BY name ASC, id ASC LIMIT ? OFFSET ?`, like, like, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]companyRow, 0)
	for rows.Next() {
		var c companyRow
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatorID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// normalizeDatetimeString parses common datetime string formats
// (RFC3339Nano, MySQL DATETIME) and returns a MySQL-safe "YYYY-MM-DD HH:MM:SS"
// string. Empty/unparseable input is returned as-is so COALESCE(NULLIF(?, ”), ...)
// can fall back to CURRENT_TIMESTAMP.
func normalizeDatetimeString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999", s); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	return s
}

func upsertCompanyRow(id, name, creatorID, createdAt string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("empty company id")
	}
	name = strings.TrimSpace(name)
	creatorID = strings.TrimSpace(creatorID)
	createdAt = normalizeDatetimeString(createdAt)
	if createdAt == "" {
		_, err := db.Exec(`
			INSERT INTO tenant_company (id, name, creator_id, created_at, updated_at)
			VALUES (?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
			ON DUPLICATE KEY UPDATE
				name=VALUES(name),
				creator_id=VALUES(creator_id),
				updated_at=CURRENT_TIMESTAMP`,
			id, name, creatorID,
		)
		return err
	}
	_, err := db.Exec(`
		INSERT INTO tenant_company (id, name, creator_id, created_at, updated_at)
		VALUES (?,?,?,?,CURRENT_TIMESTAMP)
		ON DUPLICATE KEY UPDATE
			name=VALUES(name),
			creator_id=VALUES(creator_id),
			updated_at=CURRENT_TIMESTAMP`,
		id, name, creatorID, createdAt,
	)
	return err
}

func updateCompanyName(companyID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	_, err := db.Exec(`UPDATE tenant_company SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		name, companyID)
	return err
}

func nameTaken(name, excludeID string) (bool, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return false, nil
	}
	excludeID = strings.TrimSpace(excludeID)
	var id string
	var err error
	if excludeID == "" {
		err = db.QueryRow(`SELECT id FROM tenant_company WHERE name=? LIMIT 1`, name).Scan(&id)
	} else {
		err = db.QueryRow(`SELECT id FROM tenant_company WHERE name=? AND id!=? LIMIT 1`, name, excludeID).Scan(&id)
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
