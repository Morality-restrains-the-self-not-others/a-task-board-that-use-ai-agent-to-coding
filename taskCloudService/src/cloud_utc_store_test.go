package main

import (
	"strings"
	"testing"
	"time"
)

func TestCSCEventBindingCreatedAtUnixAlignedAndJSONZ(t *testing.T) {
	setupCloudTestDB(t)

	before := time.Now().UTC()
	cfg := CloudServerConfig{
		ID:              "cfg-utc-1",
		CompanyID:       "t1",
		WorkspaceID:     "ws1",
		TaskID:          "task-utc-1",
		CommentID:       "cmt-utc-1",
		Platform:        "mock",
		Region:          "cn-test",
		ZoneID:          "cn-test-a",
		AuthorizationID: "auth-utc",
	}
	if err := upsertCloudServerConfig(cfg); err != nil {
		t.Fatalf("upsert CSC: %v", err)
	}
	eventID, _, err := insertPendingStartEvent("t1", "ws1", "task-utc-1", "u1", "cmt-utc-1", map[string]interface{}{"k": "v"})
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	binding, err := insertCommentContainerBinding("t1", "task-utc-1", "cmt-utc-1", "independent", "", "ws1")
	if err != nil {
		t.Fatalf("insert binding: %v", err)
	}

	loaded, err := loadCloudServerConfigByID("cfg-utc-1")
	if err != nil {
		t.Fatalf("load CSC: %v", err)
	}
	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		t.Fatalf("load event: %v", err)
	}

	cscJSON := cloudServerConfigToJSON(loaded)
	assertCloudRFC3339ZNearNow(t, cscJSON["created_at"], before)
	bindJSON := commentContainerBindingToJSON(binding)
	assertCloudRFC3339ZNearNow(t, bindJSON["created_at"], before)

	cscUnix := loaded.CreatedAt.UTC().Unix()
	evUnix := ev.CreatedAt.UTC().Unix()
	bindUnix := binding.CreatedAt.UTC().Unix()
	if d := abs64(cscUnix - evUnix); d >= 2 {
		t.Fatalf("CSC unix=%d event unix=%d diff=%d want < 2", cscUnix, evUnix, d)
	}
	if d := abs64(cscUnix - bindUnix); d >= 2 {
		t.Fatalf("CSC unix=%d binding unix=%d diff=%d want < 2", cscUnix, bindUnix, d)
	}
}

func assertCloudRFC3339ZNearNow(t *testing.T, raw interface{}, before time.Time) {
	t.Helper()
	s, _ := raw.(string)
	if s == "" || !strings.HasSuffix(s, "Z") {
		t.Fatalf("created_at=%q want RFC3339 ending with Z", raw)
	}
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse created_at %q: %v", s, err)
	}
	after := time.Now().UTC().Add(2 * time.Second)
	if ts.Before(before.Add(-2*time.Second)) || ts.After(after) {
		t.Fatalf("created_at %s not near now", s)
	}
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
