package projectrevisionrecorded

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"taskEvents/domain"
)

type Handler struct{}

func LocalHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if cmd.EventType != "PROJECT_REVISION_RECORDED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if len(cmd.Envelope.Data) > 0 {
		if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
			log.Printf("[project_revision_recorded] event=%s invalid json: %v", cmd.EventType, err)
			return domain.DispatchPermanent, err
		}
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	log.Printf(
		"[project_revision_recorded] level=info event=%s revision_id=%s project_id=%s tenant_id=%s version_num=%v actor_user_id=%s changed_fields=%s",
		cmd.EventType,
		fieldString(data, "revision_id"),
		fieldString(data, "project_id"),
		fieldString(data, "tenant_id"),
		data["version_num"],
		fieldString(data, "actor_user_id"),
		fieldString(data, "changed_fields"),
	)
	return domain.DispatchSuccess, nil
}

func fieldString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

var _ domain.DomainCommandPort = (*Handler)(nil)
