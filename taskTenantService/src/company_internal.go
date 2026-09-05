package main

import (
	"snowflake"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func handleInternalCompanies(w http.ResponseWriter, r *http.Request) {
	if !checkInternalSecret(r) {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/tenant/companies")
	path = strings.Trim(path, "/")

	switch {
	case (path == "creator" || path == "company-creator") && r.Method == http.MethodGet:
		cid := strings.TrimSpace(r.URL.Query().Get("company_id"))
		if cid == "" {
			writeError(w, r, 400, "company_id required")
			return
		}
		creatorID, found, err := fetchCompanyCreator(cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if !found {
			writeJSON(w, 200, map[string]interface{}{"found": false})
			return
		}
		writeJSON(w, 200, map[string]interface{}{"found": true, "creator_id": creatorID})

	case path == "by-id" && r.Method == http.MethodGet:
		cid := strings.TrimSpace(r.URL.Query().Get("company_id"))
		if cid == "" {
			writeError(w, r, 400, "company_id required")
			return
		}
		c, err := getCompanyByID(cid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if c == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, companyToJSON(c))

	case path == "by-name" && r.Method == http.MethodGet:
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeError(w, r, 400, "name required")
			return
		}
		c, err := getCompanyByName(name)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		if c == nil {
			writeError(w, r, 404, "not found")
			return
		}
		writeJSON(w, 200, companyToJSON(c))

	case path == "by-creator" && r.Method == http.MethodGet:
		uid := strings.TrimSpace(r.URL.Query().Get("creator_id"))
		if uid == "" {
			writeError(w, r, 400, "creator_id required")
			return
		}
		list, err := listCompaniesByCreator(uid)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		items := make([]map[string]interface{}, 0, len(list))
		for i := range list {
			items = append(items, companyToJSON(&list[i]))
		}
		writeJSON(w, 200, items)

	case path == "search" && r.Method == http.MethodGet:
		q := r.URL.Query().Get("q")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		list, err := searchCompanies(q, limit)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		items := make([]map[string]interface{}, 0, len(list))
		for i := range list {
			items = append(items, companyToJSON(&list[i]))
		}
		writeJSON(w, 200, items)

	case path == "name-taken" && r.Method == http.MethodGet:
		taken, err := nameTaken(r.URL.Query().Get("name"), r.URL.Query().Get("exclude_id"))
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]interface{}{"taken": taken})

	case path == "batch" && r.Method == http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid json")
			return
		}
		raw, _ := body["ids"].([]interface{})
		ids := make([]string, 0, len(raw))
		for _, v := range raw {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s == "" || s == "<nil>" {
				continue
			}
			// Avoid scientific notation for large IDs from float64 JSON.
			if f, ok := v.(float64); ok {
				s = strconv.FormatInt(int64(f), 10)
			}
			ids = append(ids, s)
		}
		list, err := listCompaniesByIDs(ids)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		items := make([]map[string]interface{}, 0, len(list))
		for i := range list {
			items = append(items, companyToJSON(&list[i]))
		}
		writeJSON(w, 200, items)

	case path == "upsert" && r.Method == http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid json")
			return
		}
		id := strField(body, "id")
		if id == "" {
			id = strField(body, "company_id")
		}
		if id == "" {
			id = snowflake.GenerateIDString()
		}
		if err := upsertCompanyRow(id, strField(body, "name"), strField(body, "creator_id"), strField(body, "created_at")); err != nil {
			writeError(w, r, 400, err.Error())
			return
		}
		c, _ := getCompanyByID(id)
		writeJSON(w, 200, companyToJSON(c))

	case path == "import" && r.Method == http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, 400, "invalid json")
			return
		}
		raw, _ := body["items"].([]interface{})
		n := 0
		for _, it := range raw {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			id := strField(m, "id")
			if id == "" {
				continue
			}
			if err := upsertCompanyRow(id, strField(m, "name"), strField(m, "creator_id"), strField(m, "created_at")); err == nil {
				n++
			}
		}
		writeJSON(w, 200, map[string]int{"imported": n})

	default:
		writeError(w, r, 404, "not found")
	}
}
