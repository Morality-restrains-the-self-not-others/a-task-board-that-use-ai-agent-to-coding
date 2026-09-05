package cloudplatformauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/handlers/cloudcommon"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

// Handler processes CLOUD_PLATFORM_AUTHORIZATION_CREATED.
type Handler struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
}

func (h *Handler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	if cmd.EventType != "CLOUD_PLATFORM_AUTHORIZATION_CREATED" {
		return domain.DispatchPermanent, fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return domain.DispatchPermanent, err
	}
	// id 为平台 SSOT 字符串授权 ID（cpa_<snowflake>），不可用 Int64Field 解析
	//（strconv.ParseInt 对 "cpa_..." 失败，OPT-20260809-026 同类修复）。
	authID := payload.StrField(data, "id")
	if authID == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing id")
	}
	platformType := payload.StrField(data, "platform_type")
	secretID := payload.StrField(data, "secret_id")
	secretKey := payload.StrField(data, "secret_key")
	if platformType == "" || secretID == "" || secretKey == "" {
		return domain.DispatchPermanent, fmt.Errorf("missing platform credentials fields")
	}
	log.Printf("[cloud_platform_auth] processing authorization id=%s platform=%s", authID, platformType)

	if _, err := h.Repo.CloudAuthorizationByID(authID); err != nil {
		log.Printf("[cloud_platform_auth] authorization lookup: %v", err)
		return domain.DispatchPermanent, err
	}
	iamID, err := aliyun.GetIAMID(secretID, secretKey, "cn-hangzhou")
	if err != nil {
		log.Printf("[cloud_platform_auth] get iam id: %v", err)
		return domain.DispatchRetryable, err
	}
	if iamID != "" {
		if err := h.Repo.CreateAccessKeyIAMAssociation(authID, secretID, iamID); err != nil {
			return domain.DispatchRetryable, err
		}
		log.Printf("[cloud_platform_auth] created IAM association iam_id=%s", iamID)
	}
	sseTaskID := fmt.Sprintf("auth_%s", authID)
	_ = cloudcommon.PublishSSE(ctx, h.Publisher, sseTaskID, map[string]interface{}{
		"status":     "sdk_call",
		"message":    "云厂商SDK调用信息",
		"event_name": "server_status_update",
		"sdk_call": map[string]interface{}{
			"method":  "get_iam_id",
			"request": map[string]interface{}{"platform_type": platformType, "secret_id": secretID},
			"response": map[string]interface{}{"iam_id": iamID},
		},
	})
	return domain.DispatchSuccess, nil
}
