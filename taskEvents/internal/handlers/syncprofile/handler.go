package syncprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/repository/saas"
)

// Handler implements USER_CREATED → upsert auth_user_profile.username.
// Runs as a standalone intent (2_sync_user_profile), independent of company creation.
type Handler struct {
	Repo *saas.Repository
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "USER_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	userID, err := payload.RequiredStrField(data, "user_id")
	if err != nil {
		log.Printf("[sync_profile] missing user_id: %v", err)
		return domain.DispatchPermanent, err
	}
	username := payload.StrField(data, "username")
	if username == "" {
		// 空 username 跳过：手机号注册/微信登录事件不带昵称（个人昵称由
		// taskAuth 登录链路同步直写）。无条件 upsert 空值会抹掉已写入的昵称。
		log.Printf("[sync_profile] skipping empty username for user_id=%s", userID)
		return domain.DispatchSuccess, nil
	}
	log.Printf("[sync_profile] upserting username=%q for user_id=%s", username, userID)

	if err := h.Repo.UpsertUserProfileUsername(userID, username); err != nil {
		log.Printf("[sync_profile] upsert error: %v", err)
		return domain.DispatchRetryable, err
	}
	log.Printf("[sync_profile] synced username for user %s", userID)
	return domain.DispatchSuccess, nil
}
