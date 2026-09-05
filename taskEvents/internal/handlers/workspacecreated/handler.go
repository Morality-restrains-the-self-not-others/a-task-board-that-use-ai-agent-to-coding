package workspacecreated

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/repository/saas"
)

// Handler processes WORKSPACE_CREATED (admins + default system settings).
type Handler struct {
	Repo *saas.Repository
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if cmd.EventType != "WORKSPACE_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	workspaceID := payload.StrField(data, "workspace_id")
	if workspaceID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing workspace_id")
	}
	workspaceName := payload.StrField(data, "workspace_name")
	companyID := payload.StrField(data, "company_id")
	createdBy := payload.StrField(data, "created_by")
	if createdBy == "" {
		if n, err := payload.Int64Field(data, "created_by"); err == nil {
			createdBy = fmt.Sprint(n)
		}
	}
	log.Printf("[workspace_created] WORKSPACE_CREATED id=%s name=%q company=%s", workspaceID, workspaceName, companyID)

	if err := h.Repo.HandleWorkspaceCreated(workspaceID, workspaceName, companyID, createdBy); err != nil {
		log.Printf("[workspace_created] error: %v", err)
		if strings.Contains(err.Error(), "not found") {
			return domain.DispatchPermanent, err
		}
		return domain.DispatchRetryable, err
	}
	return domain.DispatchSuccess, nil
}
