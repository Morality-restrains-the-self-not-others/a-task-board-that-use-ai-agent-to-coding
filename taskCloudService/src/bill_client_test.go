package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckServerStartBalanceInsufficient(t *testing.T) {
	billSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "insufficient balance", "balance_points": 1, "required_points": 10,
		})
	}))
	defer billSrv.Close()
	cfg.TaskBillURL = billSrv.URL

	err := checkServerStartBalance(context.Background(), "100", "")
	if err == nil {
		t.Fatal("expected insufficient balance error")
	}
	ib, ok := err.(*insufficientBalanceError)
	if !ok || ib.RequiredPoints != 10 {
		t.Fatalf("err=%v", err)
	}
}

func TestCheckServerStartBalanceOK(t *testing.T) {
	billSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/check-server-start-balance/") {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer billSrv.Close()
	cfg.TaskBillURL = billSrv.URL

	if err := checkServerStartBalance(context.Background(), "100", ""); err != nil {
		t.Fatal(err)
	}
}

func TestCheckServerStartBalanceSkippedWhenNoURL(t *testing.T) {
	cfg.TaskBillURL = ""
	if err := checkServerStartBalance(context.Background(), "100", ""); err != nil {
		t.Fatal(err)
	}
}

func TestTaskbillForwardStageMapping(t *testing.T) {
	if stage := taskbillForwardStage("/api/internal/taskbill/charge-server-start/"); stage != "taskbill_charge_server_start" {
		t.Fatalf("stage=%s", stage)
	}
	if stage := taskbillForwardStage("/api/internal/taskbill/check-server-start-balance/"); !strings.Contains(stage, "check") {
		t.Fatalf("stage=%s", stage)
	}
	if stage := taskbillForwardStage("/api/internal/taskbill/charge-gitlab-traffic/"); stage != "taskbill_charge_gitlab_traffic" {
		t.Fatalf("gitlab traffic stage=%s", stage)
	}
}
