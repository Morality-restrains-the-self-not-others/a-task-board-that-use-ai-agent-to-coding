//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/handlers/companycreated"
	"taskEvents/internal/handlers/usercreated"
	"taskEvents/internal/handlers/workspacecreated"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

type chainPublisher struct {
	events []publishedEvent
}

type publishedEvent struct {
	Type string
	Data map[string]interface{}
}

func (p *chainPublisher) PublishEvent(_ context.Context, eventType string, data map[string]interface{}, _ string) error {
	cp := make(map[string]interface{}, len(data))
	for k, v := range data {
		cp[k] = v
	}
	p.events = append(p.events, publishedEvent{Type: eventType, Data: cp})
	return nil
}

func TestRegistrationChainUserToWorkspace(t *testing.T) {
	const userID = "88001"
	const username = "regchain"

	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()
	repo := saas.New("")

	pub := &chainPublisher{}
	uc := &usercreated.Handler{Repo: repo, Publisher: pub}
	userData, _ := json.Marshal(map[string]interface{}{"user_id": userID, "username": username})
	out, err := uc.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "USER_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "USER_CREATED", Data: userData},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("USER_CREATED: out=%v err=%v", out, err)
	}
	if len(pub.events) != 1 || pub.events[0].Type != "COMPANY_CREATED" {
		t.Fatalf("expected COMPANY_CREATED publish, got %+v", pub.events)
	}

	cc := &companycreated.Handler{Repo: repo, Publisher: pub}
	companyPayload, _ := json.Marshal(pub.events[0].Data)
	out, err = cc.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "COMPANY_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "COMPANY_CREATED", Data: companyPayload},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("COMPANY_CREATED: out=%v err=%v", out, err)
	}
	if len(pub.events) != 2 || pub.events[1].Type != "WORKSPACE_CREATED" {
		t.Fatalf("expected WORKSPACE_CREATED publish, got %+v", pub.events)
	}

	wc := &workspacecreated.Handler{Repo: repo}
	wsPayload, _ := json.Marshal(pub.events[1].Data)
	out, err = wc.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "WORKSPACE_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "WORKSPACE_CREATED", Data: wsPayload},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("WORKSPACE_CREATED: out=%v err=%v", out, err)
	}
	if len(mux.Companies) != 1 || mux.WorkspaceHandled != 1 {
		t.Fatalf("companies=%d workspaceHandled=%d", len(mux.Companies), mux.WorkspaceHandled)
	}
}
