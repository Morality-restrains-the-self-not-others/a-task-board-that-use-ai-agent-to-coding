package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseMySQLUTCDateTimeTreatsWallClockAsUTC(t *testing.T) {
	want := time.Date(2026, 8, 13, 15, 30, 59, 0, time.UTC)
	cases := []string{
		"2026-08-13 15:30:59",
		"2026-08-13T15:30:59Z",
		"2026-08-13T15:30:59+08:00",
	}
	for _, in := range cases {
		got := parseMySQLUTCDateTime(in)
		if !got.Equal(want) {
			t.Fatalf("parseMySQLUTCDateTime(%q)=%s want %s", in, got.UTC().Format(time.RFC3339), want.Format(time.RFC3339))
		}
	}
}

func TestFormatMySQLUTCDateTimeUsesUTCDigits(t *testing.T) {
	in := time.Date(2026, 8, 13, 15, 30, 59, 0, time.FixedZone("CST", 8*3600))
	got := formatMySQLUTCDateTime(in)
	if got != "2026-08-13 07:30:59" {
		t.Fatalf("formatMySQLUTCDateTime=%q", got)
	}
}

func TestCommentCreateAndListCreatedAtRFC3339Z(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	createBody := `{"title":"UTCComment","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create task: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatalf("task id missing: %v", created)
	}

	before := time.Now().UTC()
	cmtReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/tasks/"+taskID+"/comments/", strings.NewReader(`{"content":"utc wall"}`))
	cmtReq.Header.Set("Content-Type", "application/json")
	cmtReq.Header.Set("X-Auth-Tenant-Id", "t1")
	cmtReq.Header.Set("X-Auth-User-Id", "u1")
	cmtRec := httptest.NewRecorder()
	handleCommentRoutes(cmtRec, cmtReq)
	if cmtRec.Code != http.StatusCreated {
		t.Fatalf("create comment: %d %s", cmtRec.Code, cmtRec.Body.String())
	}
	var cmt map[string]interface{}
	if err := json.NewDecoder(cmtRec.Body).Decode(&cmt); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	assertRFC3339ZNearNow(t, cmt["created_at"], before)

	listReq := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/"+taskID+"/comments/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listReq.Header.Set("X-Auth-User-Id", "u1")
	listRec := httptest.NewRecorder()
	handleCommentRoutes(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list comments: %d %s", listRec.Code, listRec.Body.String())
	}
	var listed []map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v body=%s", err, listRec.Body.String())
	}
	if len(listed) < 1 {
		t.Fatalf("listed comments empty")
	}
	assertRFC3339ZNearNow(t, listed[0]["created_at"], before)

	var raw string
	if err := db.QueryRow(`SELECT created_at FROM task_comments WHERE task_id=?`, taskID).Scan(&raw); err != nil {
		t.Fatalf("select created_at: %v", err)
	}
	stored := parseMySQLUTCDateTime(raw)
	jsonTime := mustParseRFC3339Z(t, listed[0]["created_at"])
	if d := stored.Unix() - jsonTime.Unix(); d > 1 || d < -1 {
		t.Fatalf("db unix=%d json unix=%d raw=%q json=%v", stored.Unix(), jsonTime.Unix(), raw, listed[0]["created_at"])
	}
}

func assertRFC3339ZNearNow(t *testing.T, raw interface{}, before time.Time) {
	t.Helper()
	ts := mustParseRFC3339Z(t, raw)
	if ts.Location() != time.UTC {
		t.Fatalf("location=%s want UTC", ts.Location())
	}
	after := time.Now().UTC().Add(2 * time.Second)
	if ts.Before(before.Add(-2*time.Second)) || ts.After(after) {
		t.Fatalf("created_at %s not near now [%s,%s]", ts.Format(time.RFC3339Nano), before.Format(time.RFC3339), after.Format(time.RFC3339))
	}
}

func mustParseRFC3339Z(t *testing.T, raw interface{}) time.Time {
	t.Helper()
	s, _ := raw.(string)
	if s == "" || !strings.HasSuffix(s, "Z") {
		t.Fatalf("created_at=%q want RFC3339 ending with Z", raw)
	}
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		ts, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		t.Fatalf("parse created_at %q: %v", s, err)
	}
	return ts.UTC()
}
