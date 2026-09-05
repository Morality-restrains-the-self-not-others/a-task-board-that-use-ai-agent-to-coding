package registrationinvite

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"taskEvents/domain"
)

var supportedEvents = map[string]struct{}{
	"REGISTRATION_INVITE_POLICY_UPDATED": {},
	"REGISTRATION_INVITE_CODE_ISSUED":    {},
	"REGISTRATION_INVITE_CODE_REDEEMED":  {},
}

// Handler records registration-invite domain events as structured observability logs.
// SSOT data remains in taskAuth; this consumer exists for Loki/audit tracing only.
type Handler struct{}

// LocalHandler returns the observability handler.
func LocalHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if _, ok := supportedEvents[cmd.EventType]; !ok {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if len(cmd.Envelope.Data) > 0 {
		if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
			log.Printf("[registration_invite] event=%s invalid json: %v", cmd.EventType, err)
			return domain.DispatchPermanent, err
		}
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	log.Printf(
		"[registration_invite] level=info event=%s code=%s code_id=%s issuer_user_id=%s redeemed_by_user_id=%s issued_day=%s enabled=%v daily_quota=%v updated_by=%s",
		cmd.EventType,
		fieldString(data, "code"),
		fieldString(data, "code_id"),
		fieldString(data, "issuer_user_id"),
		fieldString(data, "redeemed_by_user_id"),
		fieldString(data, "issued_day"),
		data["enabled"],
		data["daily_quota"],
		fieldString(data, "updated_by"),
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
