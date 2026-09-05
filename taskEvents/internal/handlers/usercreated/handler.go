package usercreated

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"taskEvents/domain"
	"taskEvents/internal/handlers/payload"
	"taskEvents/internal/publish"
	"taskEvents/internal/repository/saas"
)

// Handler implements USER_CREATED → create company + publish COMPANY_CREATED.
type Handler struct {
	Repo      *saas.Repository
	Publisher publish.EventPublisher
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
		log.Printf("[user_created] missing user_id: %v", err)
		return domain.DispatchPermanent, err
	}
	// 手机号/微信注册故意不带 username（避免手机号成为公司名）。
	// 空 username 不得永久失败进 DLT：CreateCompanyForUser 会用个人昵称
	// 生成「{user}的公司」，再空则回退 DefaultPersonalCompanyName。
	username := payload.StrField(data, "username")
	if username == "" {
		log.Printf("[user_created] empty username; company name will resolve from nickname or default user_id=%s", userID)
	}
	log.Printf("[user_created] processing USER_CREATED user_id=%s username=%s", userID, username)

	// 平台角色（super_admin/employee）注册后不应自动创建公司：
	// 平台管理员/员工无租户上下文，菜单走「系统管理」入口而非「开始使用」。
	// 查询失败时返回 Retryable 而非放行，避免平台用户被误建公司。
	platformRoles, err := h.Repo.UserPlatformRoles(userID)
	if err != nil {
		log.Printf("[user_created] platform-roles lookup error user_id=%s err=%v (will retry)", userID, err)
		return domain.DispatchRetryable, err
	}
	for _, r := range platformRoles {
		if r == "super_admin" || r == "employee" {
			log.Printf("[user_created] user_id=%s is platform role %s — skipping auto company creation", userID, r)
			return domain.DispatchSuccess, nil
		}
	}

	if existingID, existingName, ok, err := h.Repo.CompanyByCreator(userID); err != nil {
		return domain.DispatchRetryable, err
	} else if ok {
		log.Printf("[user_created] company already exists for user %s (company %d), re-publishing COMPANY_CREATED", userID, existingID)
		// 自愈（OPT-20260806-013）：公司存在但成员行缺失（上次 createCompanyMember
		// 失败后重试走此分支）→ 仅在缺失时插入，且 member_name 取个人昵称，
		// 绝不覆盖已有正确昵称、不把公司名写入成员列。
		if err := h.Repo.EnsureCreatorMember(existingID, userID); err != nil {
			log.Printf("[user_created] ensure member row (self-heal) error company_id=%d user_id=%s err=%v", existingID, userID, err)
			return domain.DispatchRetryable, err
		}
		// Ensure user is marked as tenant even when company already exists.
		if err := h.Repo.IntentMarkUserTenant(userID); err != nil {
			log.Printf("[user_created] mark_user_tenant (existing) error user_id=%s err=%v (non-fatal)", userID, err)
		}
		// Re-publish to ensure downstream chain runs even when the previous
		// attempt failed after DB write but before event publish.
		payloadOut := map[string]interface{}{
			"company_id":   existingID,
			"company_name": existingName,
			"creator_id":   userID,
		}
		if err := h.Publisher.PublishEvent(ctx, "COMPANY_CREATED", payloadOut, fmt.Sprint(existingID)); err != nil {
			log.Printf("[user_created] re-publish COMPANY_CREATED error for company %d: %v", existingID, err)
			return domain.DispatchRetryable, err
		}
		log.Printf("[user_created] re-published COMPANY_CREATED for company %d", existingID)
		return domain.DispatchSuccess, nil
	}

	created, err := h.Repo.CreateCompanyForUser(userID, username)
	if err != nil {
		// Always retry service-level errors — transient failures (network, restart,
		// DNS, etc.) should not permanently drop USER_CREATED events. The previous
		// isPermanent() check on "not found"/"missing" was too aggressive and caused
		// silent event loss when taskTenantService returned transient 404s.
		log.Printf("[user_created] create company error (will retry): %v", err)
		return domain.DispatchRetryable, err
	}
	log.Printf("[user_created] created company %d name=%s for user %s", created.CompanyID, created.Name, userID)

	// Mark user as tenant immediately on company creation (no payment gate).
	// Previously is_tenant=true was only set on billing recharge; now company
	// creation alone qualifies a user as a tenant.
	if err := h.Repo.IntentMarkUserTenant(userID); err != nil {
		log.Printf("[user_created] mark_user_tenant error user_id=%s err=%v (non-fatal, continuing)", userID, err)
	} else {
		log.Printf("[user_created] marked user %s as tenant", userID)
	}

	payloadOut := map[string]interface{}{
		"company_id":   created.CompanyID,
		"company_name": created.Name,
		"creator_id":   created.CreatorID,
		"created_at":   created.CreatedAt.Format("2006-01-02T15:04:05.999999"),
	}
	if err := h.Publisher.PublishEvent(ctx, "COMPANY_CREATED", payloadOut, fmt.Sprint(created.CompanyID)); err != nil {
		log.Printf("[user_created] publish COMPANY_CREATED error: %v", err)
		return domain.DispatchRetryable, err
	}
	log.Printf("[user_created] sent COMPANY_CREATED for company %d", created.CompanyID)
	return domain.DispatchSuccess, nil
}
