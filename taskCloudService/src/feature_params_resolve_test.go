package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"snowflake"
	"strings"
	"testing"
)

func stubTaskBinding(t *testing.T, source, personalID, ownerID string) {
	t.Helper()
	prev := fetchTaskFeatureParamsBindingFn
	fetchTaskFeatureParamsBindingFn = func(tenantID, taskID string) (*taskFeatureParamsBinding, error) {
		return &taskFeatureParamsBinding{
			Source:           source,
			PersonalConfigID: personalID,
			OwnerID:          ownerID,
			WorkspaceID:      "w1",
			TenantID:         tenantID,
		}, nil
	}
	t.Cleanup(func() { fetchTaskFeatureParamsBindingFn = prev })
}

func stubAllowPersonal(t *testing.T, allowed bool) {
	t.Helper()
	prev := fetchWorkspaceAllowPersonalFn
	fetchWorkspaceAllowPersonalFn = func(_ context.Context, workspaceID string) (bool, error) {
		return allowed, nil
	}
	t.Cleanup(func() { fetchWorkspaceAllowPersonalFn = prev })
}

func TestFeatureParamsEnvWorkspaceOverride(t *testing.T) {
	store := setupBudgetTestDB(t)
	seedCompanyFP(t, "c1", "company-model")
	stubTaskBinding(t, "workspace", "", "")
	stubAllowPersonal(t, false)
	store.putWorkspace("w1", false, &featureParamsConfig{
		ID: "ws-fp-1", ProvidersJSON: "[]", AgentModel: "ws-model", AgentProvider: "wp", AgentMaxSteps: 50, ExtraEnvJSON: "[]",
	})

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-ws", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env[envAgentModel] != "ws-model" {
		t.Fatalf("env=%v", out.Env)
	}
	if out.Env[envScope] != "workspace" {
		t.Fatalf("scope=%s", out.Env[envScope])
	}
	if out.Source != "workspace" {
		t.Fatalf("source=%s", out.Source)
	}
}

func TestFeatureParamsEnvPersonal(t *testing.T) {
	store := setupBudgetTestDB(t)
	seedCompanyFP(t, "c1", "company-model")
	stubTaskBinding(t, "personal", "pc-1", "user-1")
	stubAllowPersonal(t, true)
	store.putPersonal(&featureParamsConfig{
		ID: "pc-1", ProvidersJSON: "[]", AgentModel: "personal-model", AgentProvider: "pp",
		AgentMaxSteps: 30, ExtraEnvJSON: "[]", DisplayName: "个人配置:日常",
	}, "user-1")

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-p", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env[envAgentModel] != "personal-model" {
		t.Fatalf("env=%v", out.Env)
	}
	if out.Env[envScope] != "personal" {
		t.Fatalf("scope=%s", out.Env[envScope])
	}
	if !strings.Contains(out.Env[envConfigName], "日常") {
		t.Fatalf("config_name=%s", out.Env[envConfigName])
	}
}

func TestFeatureParamsEnvSnapshotHitStillReResolves(t *testing.T) {
	setupBudgetTestDB(t)
	seedCompanyFP(t, "c1", "live-model")
	stubTaskBinding(t, "company", "", "")
	stubAllowPersonal(t, false)
	_, _ = db.Exec(`
		INSERT INTO cloud_task_feature_params_snapshot (
			id, task_id, workspace_id, tenant_id, source, resolved_env, providers_summary,
			agent_model, agent_max_steps, created_at
		) VALUES (?, ?, ?, ?, 'company', '{}', '[]', 'stale-model', 200, ?)`,
		snowflake.GenerateIDString(), "task-snap", "w1", "c1", fpNowUTC(),
	)

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-snap", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env[envAgentModel] != "live-model" {
		t.Fatalf("should re-resolve live config, got env=%v", out.Env)
	}
	if countSnapshots(t, "task-snap") != 2 {
		t.Fatalf("expected append-only second snapshot, count=%d", countSnapshots(t, "task-snap"))
	}
}

func TestFeatureParamsEnvPersonalBlockedFallsBackCompany(t *testing.T) {
	store := setupBudgetTestDB(t)
	seedCompanyFP(t, "c1", "company-model")
	stubTaskBinding(t, "personal", "pc-1", "user-1")
	stubAllowPersonal(t, false)
	store.putPersonal(&featureParamsConfig{
		ID: "pc-1", ProvidersJSON: "[]", AgentModel: "personal-model", AgentProvider: "pp",
		AgentMaxSteps: 30, ExtraEnvJSON: "[]", DisplayName: "个人配置:日常",
	}, "user-1")

	out, err := resolveFeatureParamsEnvLocal(context.Background(), getBudgetDB(), "c1", "w1", "task-block", "c1", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Env[envAgentModel] != "company-model" {
		t.Fatalf("env=%v", out.Env)
	}
	if out.Env[envScope] != "company" {
		t.Fatalf("scope=%s", out.Env[envScope])
	}
}

func TestHandleFeatureParamsEnvNoDjangoFallback(t *testing.T) {
	setupBudgetTestDB(t)
	seedCompanyFP(t, "t1", "m1")
	stubTaskBinding(t, "company", "", "")
	stubAllowPersonal(t, false)

	rec := httptest.NewRecorder()
	handleFeatureParamsEnv(
		rec, nil,
		&CloudServerConfig{CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		map[string]any{},
		"t1", "w1", "task1", "tok-proxy",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"TASK_AGENT_MODEL":"m1"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
	if countSnapshots(t, "task1") < 1 {
		t.Fatal("expected snapshot")
	}
}

func TestResolveOwnerUserIDViaCompanyMember(t *testing.T) {
	store := setupBudgetTestDB(t)
	store.putMember("c1", "uid-9", "mem-1", false)
	got := resolveOwnerUserID("c1", "mem-1")
	if got != "uid-9" {
		t.Fatalf("got=%q", got)
	}
}
