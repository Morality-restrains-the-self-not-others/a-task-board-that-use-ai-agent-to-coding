package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCancelCommentContainerBindingWaitingPrevious(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskCancelWait")

	create := func(commentID string) {
		t.Helper()
		body := `{"comment_id":"` + commentID + `","execution_mode":"wait_previous"}`
		req := httptest.NewRequest(http.MethodPost,
			"/api/tenant/t1/task/taskCancelWait/comment-container-bindings/",
			strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "taskCancelWait", "", "")
		if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
			t.Fatalf("create %s status=%d body=%s", commentID, rec.Code, rec.Body.String())
		}
	}

	create("c1")
	create("c2")
	b2, err := loadCommentContainerBinding("t1", "taskCancelWait", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCommentContainerBindingStatus(b2.ID, ccbStatusWaitingPrevious); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskCancelWait/comment-container-bindings/c2/cancel/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskCancelWait", "", "c2/cancel")
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	binding, _ := resp["binding"].(map[string]interface{})
	if binding == nil || binding["status"] != ccbStatusCancelled {
		t.Fatalf("binding=%v want status=%s", binding, ccbStatusCancelled)
	}

	got, err := loadCommentContainerBinding("t1", "taskCancelWait", "c2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ccbStatusCancelled {
		t.Fatalf("stored status=%s want %s", got.Status, ccbStatusCancelled)
	}

	b1, err := loadCommentContainerBinding("t1", "taskCancelWait", "c1")
	if err != nil {
		t.Fatal(err)
	}
	if err := updateCommentContainerBindingStatus(b1.ID, ccbStatusRunning); err != nil {
		t.Fatal(err)
	}
	bad := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskCancelWait/comment-container-bindings/c1/cancel/", nil)
	badRec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(badRec, bad, "t1", "taskCancelWait", "", "c1/cancel")
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("cancel running status=%d want 400 body=%s", badRec.Code, badRec.Body.String())
	}
}
