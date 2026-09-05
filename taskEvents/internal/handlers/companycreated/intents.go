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

// DeliverableIntent handles COMPANY_CREATED intent 1 (v4 D2 fan-out).
type DeliverableIntent struct {
	Repo *saas.Repository
}

func (h *DeliverableIntent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	companyID, err := parseCompanyCreated(cmd)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	if err := h.Repo.IntentSetDefaultDeliverable(companyID); err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	log.Printf("[company_created/1_deliverable] ok company=%d", companyID)
	return domain.DispatchSuccess, nil
}

// ProgressIntent handles COMPANY_CREATED intent 2.
type ProgressIntent struct {
	Repo *saas.Repository
}

func (h *ProgressIntent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	companyID, err := parseCompanyCreated(cmd)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	if err := h.Repo.IntentSetDefaultProgress(companyID); err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	log.Printf("[company_created/2_progress] ok company=%d", companyID)
	return domain.DispatchSuccess, nil
}

// WorkspaceIntent handles COMPANY_CREATED intent 3.
type WorkspaceIntent struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
}

func (h *WorkspaceIntent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	data, companyID, creatorID, err := parseCompanyCreatedData(cmd)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	companyName := payload.StrField(data, "company_name")
	// Ensure prerequisites (idempotent — no-ops when intents 1/2 already ran)
	if err := h.Repo.IntentSetDefaultDeliverable(companyID); err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	if err := h.Repo.IntentSetDefaultProgress(companyID); err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	err = h.Repo.IntentCreateDefaultWorkspace(companyID, creatorID, companyName, func(workspaceID string, workspaceName string, createdAt time.Time) error {
		out := map[string]interface{}{
			"workspace_id":   workspaceID,
			"workspace_name": workspaceName,
			"company_id":     fmt.Sprint(companyID),
			"is_default":     true,
			"created_by":     creatorID,
			"created_at":     createdAt.Format("2006-01-02T15:04:05.999999"),
		}
		return h.Publisher.PublishEvent(ctx, "WORKSPACE_CREATED", out, fmt.Sprint(workspaceID))
	})
	if err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	log.Printf("[company_created/3_workspace] ok company=%d data=%v", companyID, data)
	return domain.DispatchSuccess, nil
}

func parseCompanyCreated(cmd domain.DomainCommand) (int64, error) {
	_, companyID, _, err := parseCompanyCreatedData(cmd)
	return companyID, err
}

func parseCompanyCreatedData(cmd domain.DomainCommand) (map[string]interface{}, int64, string, error) {
	if cmd.EventType != "COMPANY_CREATED" {
		return nil, 0, "", fmt.Errorf("unsupported event %s", cmd.EventType)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(cmd.Envelope.Data, &data); err != nil {
		return nil, 0, "", err
	}
	companyID, err := payload.Int64Field(data, "company_id")
	if err != nil {
		return nil, 0, "", err
	}
	creatorID := payload.StrField(data, "creator_id")
	if creatorID == "" {
		n, err := payload.Int64Field(data, "creator_id")
		if err != nil {
			return nil, 0, "", err
		}
		creatorID = fmt.Sprint(n)
	}
	return data, companyID, creatorID, nil
}

func isRetryableCompany(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no default deliverable")
}

// FeatureParamsIntent handles COMPANY_CREATED intent 5 — init tenant feature params (idempotent).
type FeatureParamsIntent struct {
	Repo *saas.Repository
}

func (h *FeatureParamsIntent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	companyID, err := parseCompanyCreated(cmd)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	result, err := h.Repo.IntentInitTenantFeatureParams(fmt.Sprint(companyID))
	if err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	already, _ := result["already_existed"].(bool)
	fpid, _ := result["feature_params_id"].(string)
	log.Printf("[company_created/5_feature_params] ok company=%d already_existed=%v feature_params_id=%s", companyID, already, fpid)
	return domain.DispatchSuccess, nil
}

// ResourceGrantIntent handles COMPANY_CREATED intent 6 — grant initial resources for new company.
type ResourceGrantIntent struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
}

func (h *ResourceGrantIntent) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	_, companyID, creatorID, err := parseCompanyCreatedData(cmd)
	if err != nil {
		return domain.DispatchPermanent, err
	}
	result, err := h.Repo.IntentGrantInitialResources(fmt.Sprint(companyID), creatorID)
	if err != nil {
		if isRetryableCompany(err) {
			return domain.DispatchRetryable, err
		}
		return domain.DispatchPermanent, err
	}
	quotaAfter, _ := result["task_post_quota_after"].(float64)
	log.Printf("[company_created/6_grant] ok company=%d quota_after=%v", companyID, quotaAfter)

	// Publish SSE welcome notification
	ssePayload := map[string]interface{}{
		"task_id": fmt.Sprintf("company:%d", companyID),
		"status_data": map[string]interface{}{
			"event_name":       "new_user_gift",
			"message":          fmt.Sprintf("🎉 欢迎！系统已赠送您 100 个任务帖（有效期 6 个月），当前可用配额：%d 个", int(quotaAfter)),
			"company_id":       fmt.Sprint(companyID),
			"user_id":          creatorID,
			"gift_task_posts":  100,
		},
	}
	_ = h.Publisher.PublishEvent(ctx, "SSE_MESSAGE", ssePayload, fmt.Sprint(companyID))

	return domain.DispatchSuccess, nil
}
