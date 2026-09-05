package projectupdated

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
)

// Handler logs PROJECT_UPDATED (business logic TBD in Python too).
type Handler struct{}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if cmd.EventType != "PROJECT_UPDATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	_ = json.Unmarshal(cmd.Envelope.Data, &data)
	log.Printf("[project_updated] PROJECT_UPDATED: %+v", data)
	return domain.DispatchSuccess, nil
}
