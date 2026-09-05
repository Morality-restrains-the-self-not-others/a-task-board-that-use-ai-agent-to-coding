package taskcompleted

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
)

// Handler logs TASK_COMPLETED (business logic TBD in Python too).
type Handler struct{}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if cmd.EventType != "TASK_COMPLETED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	_ = json.Unmarshal(cmd.Envelope.Data, &data)
	log.Printf("[task_completed] TASK_COMPLETED: %+v", data)
	return domain.DispatchSuccess, nil
}
