//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

// TestCompanyCreatedFanOutIntents verifies v4 D2: three independent intents on one COMPANY_CREATED event.
func TestCompanyCreatedFanOutIntents(t *testing.T) {
	const userID = "88002"
	const companyID int64 = 99002

	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()
	repo := saas.New("")

	payload, _ := json.Marshal(map[string]interface{}{
		"company_id": float64(companyID),
		"creator_id": userID,
		"name":       "Fanout Co",
	})
	cmd := domain.DomainCommand{
		EventType: "COMPANY_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "COMPANY_CREATED", Data: payload},
	}

	deliverable := &companycreated.DeliverableIntent{Repo: repo}
	out, err := deliverable.Dispatch(context.Background(), cmd)
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("deliverable intent: out=%v err=%v", out, err)
	}

	progress := &companycreated.ProgressIntent{Repo: repo}
	out, err = progress.Dispatch(context.Background(), cmd)
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("progress intent: out=%v err=%v", out, err)
	}

	pub := &chainPublisher{}
	workspace := &companycreated.WorkspaceIntent{Repo: repo, Publisher: pub}
	out, err = workspace.Dispatch(context.Background(), cmd)
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("workspace intent: out=%v err=%v", out, err)
	}
	if len(pub.events) != 1 || pub.events[0].Type != "WORKSPACE_CREATED" {
		t.Fatalf("expected WORKSPACE_CREATED, got %+v", pub.events)
	}

	cid := fmt.Sprint(companyID)
	if !mux.DeliverableTenants[cid] || !mux.ProgressTenants[cid] {
		t.Fatalf("defaults missing deliverable=%v progress=%v", mux.DeliverableTenants, mux.ProgressTenants)
	}
	if _, ok := mux.Workspaces[cid]; !ok {
		t.Fatal("workspace missing")
	}
}
