package companycreated

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

// Handler runs COMPANY_CREATED chain in one binary (deliverable → progress → workspace).
type Handler struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "COMPANY_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	companyID, err := payload.Int64Field(data, "company_id")
	if err != nil {
		log.Printf("[company_created] missing company_id: %v", err)
		return domain.DispatchPermanent, err
	}
	creatorID := payload.StrField(data, "creator_id")
	if creatorID == "" {
		if n, err := payload.Int64Field(data, "creator_id"); err == nil {
			creatorID = fmt.Sprint(n)
		} else {
			log.Printf("[company_created] missing creator_id: %v", err)
			return domain.DispatchPermanent, err
		}
	}
	companyName := payload.StrField(data, "company_name")
	log.Printf("[company_created] processing COMPANY_CREATED company_id=%d creator_id=%s company_name=%q", companyID, creatorID, companyName)

	err = h.Repo.HandleCompanyCreated(companyID, creatorID, companyName, func(workspaceID string, workspaceName string, createdAt time.Time) error {
		out := map[string]interface{}{
			"workspace_id":   workspaceID,
			"workspace_name": workspaceName,
			"company_id":     fmt.Sprint(companyID),
			"is_default":     true,
			"created_by":     creatorID,
			"created_at":     createdAt.Format("2006-01-02T15:04:05.999999"),
		}
		return h.Publisher.PublishEvent(ctx, "WORKSPACE_CREATED", out, workspaceID)
	})
	if err != nil {
		log.Printf("[company_created] chain error: %v", err)
		if isPermanent(err) {
			return domain.DispatchPermanent, err
		}
		return domain.DispatchRetryable, err
	}
	log.Printf("[company_created] completed COMPANY_CREATED for company %d", companyID)
	return domain.DispatchSuccess, nil
}

func isPermanent(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "not found") || strings.Contains(msg, "missing")
}
