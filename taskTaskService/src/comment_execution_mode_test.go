package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeCommentExecutionMode(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"", executionModeWaitPrevious, false},
		{"wait_previous", executionModeWaitPrevious, false},
		{"independent", executionModeIndependent, false},
		{"  independent  ", executionModeIndependent, false},
		{"invalid", "", true},
	}
	for _, tc := range tests {
		got, err := normalizeCommentExecutionMode(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("raw=%q expected error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("raw=%q unexpected error: %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("raw=%q got=%q want=%q", tc.raw, got, tc.want)
		}
	}
}

func TestHumanCommentExecutionModeCreateListPatch(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)

	taskID := createTestTaskForComments(t)

	postRec := postComment(t, taskID, `{"content":"hello","execution_mode":"independent"}`)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", postRec.Code, postRec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(postRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	commentID, _ := created["id"].(string)
	if commentID == "" {
		t.Fatal("expected comment id")
	}
	if created["execution_mode"] != executionModeIndependent {
		t.Fatalf("create execution_mode=%v", created["execution_mode"])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/"+taskID+"/comments/", nil)
	listReq.Header.Set("X-Auth-Tenant-Id", "t1")
	listReq.Header.Set("X-Auth-User-Id", "u1")
	listRec := httptest.NewRecorder()
	handleCommentRoutes(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(list))
	}
	if list[0]["execution_mode"] != executionModeIndependent {
		t.Fatalf("list execution_mode=%v", list[0]["execution_mode"])
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/tasks/"+taskID+"/comments/"+commentID+"/", strings.NewReader(`{"execution_mode":"wait_previous"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("X-Auth-Tenant-Id", "t1")
	patchReq.Header.Set("X-Auth-User-Id", "u1")
	patchRec := httptest.NewRecorder()
	handleCommentRoutes(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("patch: expected 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	var patched map[string]interface{}
	if err := json.NewDecoder(patchRec.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if patched["execution_mode"] != executionModeWaitPrevious {
		t.Fatalf("patch execution_mode=%v", patched["execution_mode"])
	}

	mode, err := loadHumanCommentExecutionMode(commentID)
	if err != nil {
		t.Fatalf("load mode: %v", err)
	}
	if mode != executionModeWaitPrevious {
		t.Fatalf("db execution_mode=%q", mode)
	}
}

func TestHumanCommentExecutionModeCreateDefault(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)

	taskID := createTestTaskForComments(t)
	postRec := postComment(t, taskID, `{"content":"default mode"}`)
	if postRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", postRec.Code, postRec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(postRec.Body).Decode(&created)
	if created["execution_mode"] != executionModeWaitPrevious {
		t.Fatalf("default execution_mode=%v", created["execution_mode"])
	}
}

func TestHumanCommentExecutionModePatchInvalid(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)

	taskID := createTestTaskForComments(t)
	postRec := postComment(t, taskID, `{"content":"x"}`)
	var created map[string]interface{}
	json.NewDecoder(postRec.Body).Decode(&created)
	commentID, _ := created["id"].(string)

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/tasks/"+taskID+"/comments/"+commentID+"/", strings.NewReader(`{"execution_mode":"bogus"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("X-Auth-Tenant-Id", "t1")
	patchReq.Header.Set("X-Auth-User-Id", "u1")
	patchRec := httptest.NewRecorder()
	handleCommentRoutes(patchRec, patchReq)
	if patchRec.Code != http.StatusBadRequest {
		t.Fatalf("patch invalid: expected 400, got %d", patchRec.Code)
	}
}
