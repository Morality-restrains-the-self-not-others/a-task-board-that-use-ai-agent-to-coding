package aiassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/repository/saas"
)

// Handler persists AI assistant reply to projects_todo_ai_comment.
type Handler struct {
	Repo *saas.Repository
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_ = ctx
	if cmd.EventType != "AI_ASSISTANT_REPLY_COMPLETED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	streamOK := boolField(data, "stream_ok")
	clientAbandoned := boolField(data, "client_abandoned")
	commentID, err := payload.Int64Field(data, "ai_comment_id")
	if err != nil {
		log.Printf("[ai_assistant] missing ai_comment_id")
		return domain.DispatchPermanent, err
	}
	text := payload.StrField(data, "assistant_text")
	shouldPersist := streamOK || (clientAbandoned && text != "")
	if !shouldPersist {
		log.Printf("[ai_assistant] skip persist id=%d stream_ok=%v client_abandoned=%v", commentID, streamOK, clientAbandoned)
		return domain.DispatchSuccess, nil
	}
	ok, err := h.Repo.UpdateAssistantResponse(commentID, text)
	if err != nil {
		return domain.DispatchRetryable, err
	}
	if !ok {
		log.Printf("[ai_assistant] comment not found id=%d", commentID)
		return domain.DispatchSuccess, nil
	}
	log.Printf("[ai_assistant] wrote assistant_response id=%d", commentID)
	return domain.DispatchSuccess, nil
}

func boolField(data map[string]interface{}, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true" || b == "1"
	default:
		return fmt.Sprint(v) == "true"
	}
}
