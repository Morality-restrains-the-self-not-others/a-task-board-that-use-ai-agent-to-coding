package main

import (
	"net/http"
	"strings"
	"tracelog"
)

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/api/schema/", handleOpenAPISchema)
	mux.HandleFunc("/api/schema", handleOpenAPISchema)
	mux.HandleFunc("/api/swagger/", handleSwaggerUI)
	mux.HandleFunc("/api/swagger", handleSwaggerUI)
	mux.HandleFunc("/api/schema-internal/", handleOpenAPIInternalSchema)
	mux.HandleFunc("/api/schema-internal", handleOpenAPIInternalSchema)
	mux.HandleFunc("/api/swagger-internal/", handleSwaggerInternalUI)
	mux.HandleFunc("/api/swagger-internal", handleSwaggerInternalUI)

	mux.HandleFunc("/api/internal/taskbill/consume-task-post-quota/", handleInternalConsumeTaskPostQuota)
	mux.HandleFunc("/api/internal/taskbill/consume-task-post-renewal/", handleInternalConsumeTaskPostRenewal)
	mux.HandleFunc("/api/internal/taskbill/charge-server-start/", handleInternalChargeServerStart)
	mux.HandleFunc("/api/internal/taskbill/charge-gitlab-traffic/", handleInternalChargeGitlabTraffic)
	mux.HandleFunc("/api/internal/taskbill/gitlab-traffic-gate/", handleInternalGitlabTrafficGate)
	mux.HandleFunc("/api/internal/taskbill/report-gitlab-disk-usage/", handleInternalReportGitlabDiskUsage)
	mux.HandleFunc("/api/internal/taskbill/gitlab-resources/", handleInternalListGitlabResources)
	mux.HandleFunc("/api/internal/taskbill/sync-gitlab-disk-quotas/", handleInternalSyncGitlabDiskQuotas)
	mux.HandleFunc("/api/internal/taskbill/gitlab-regions/", func(w http.ResponseWriter, r *http.Request) {
		// Route: /gitlab-regions/ → list; /gitlab-regions/{slug}/capacity/ → capacity CRUD
		path := strings.TrimRight(r.URL.Path, "/")
		if strings.HasSuffix(path, "/capacity") {
			handleAdminUpdateRegionCapacity(w, r)
			return
		}
		if r.Method == http.MethodGet {
			handleGitlabRegionsList(w, r)
			return
		}
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	})
	mux.HandleFunc("/api/internal/taskbill/gitlab-regions-admin/", handleSystemAdminListRegionsFull)
	mux.HandleFunc("/api/internal/taskbill/get-or-create-account/", handleInternalGetOrCreateAccount)
	mux.HandleFunc("/api/internal/taskbill/accounts/", handleInternalGetAccount)
	mux.HandleFunc("/api/internal/taskbill/check-server-start-balance/", handleInternalCheckServerStartBalance)
	mux.HandleFunc("/api/internal/taskbill/expire-resource-grants/", handleInternalExpireResourceGrants)
	mux.HandleFunc("/api/internal/taskbill/admin-grant-resources/", handleInternalAdminGrantResources)
	mux.HandleFunc("/api/internal/taskbill/backfill-grant-orders/", handleInternalBackfillGrantOrders)
	mux.HandleFunc("/api/internal/taskbill/referral/consumption-monthly-totals/", handleInternalReferralConsumptionMonthly)
	mux.HandleFunc("/api/internal/taskbill/referral/sync-edge/", handleInternalReferralSyncEdge)
	mux.HandleFunc("/api/internal/taskbill/referral/disable-eligibility/", handleInternalDisableReferralEligibility)
	mux.HandleFunc("/api/internal/taskbill/referral/commission-summary/", handleInternalReferralCommissionSummary)
	mux.HandleFunc("/api/internal/taskbill/referral/commission-rate/", handleInternalReferralCommissionRate)
	mux.HandleFunc("/api/internal/taskbill/referral/settle-due/", handleInternalReferralSettleDue)
	mux.HandleFunc("/api/internal/taskbill/referral/void/", handleInternalReferralVoid)
	mux.HandleFunc("/api/internal/taskbill/profit-sharing/process-pending/", handleInternalProcessPendingProfitSharings)
	mux.HandleFunc("/api/internal/taskbill/profit-sharing/receivers/ensure/", handleInternalEnsureProfitSharingReceiver)
	mux.HandleFunc("/api/internal/taskbill/profit-sharing/receivers/delete/", handleInternalDeleteProfitSharingReceiver)
	mux.HandleFunc("/api/internal/taskbill/referral/config/", handleAdminReferralConfig)
	mux.HandleFunc("/api/internal/taskbill/refund-applications/", handleInternalRefundApplicationsRouter)
	mux.HandleFunc("/api/internal/taskbill/invoice-applications/", handleSystemAdminInvoiceApplicationsRouter)
	mux.HandleFunc("/api/internal/taskbill/users/", handleInternalTaskBillUsersRouter)
	mux.HandleFunc("/api/internal/taskbill/users", handleInternalTaskBillUsersRouter)
	mux.HandleFunc("/api/internal/taskbill/refund-policy/", handleInternalRefundPolicy)
	mux.HandleFunc("/api/internal/taskbill/resource-pricing/", handleInternalResourcePricingRouter)
	mux.HandleFunc("/api/internal/taskbill/pricing-packages/", handleInternalPricingPackages)
	mux.HandleFunc("/api/internal/taskbill/membership/", handleInternalGetMembership)
	mux.HandleFunc("/api/internal/taskbill/ensure-membership/", handleInternalEnsureMembership)
	mux.HandleFunc("/api/billing/wechat/notify/", handleWechatNotify)
	mux.HandleFunc("/api/billing/wechat/fapiao/notify/", handleWechatFapiaoNotify)
	mux.HandleFunc("/api/billing/profitsharing/notify/", handleProfitSharingNotify)
	mux.HandleFunc("/api/billing/profitsharing/change-notify/", handleProfitSharingNotify)
	mux.HandleFunc("/api/billing/profitsharing/change-notify", handleProfitSharingNotify)
	mux.HandleFunc("/api/billing/profit-sharing/referrer-orders/", handleReferrerProfitSharingOrders)
	mux.HandleFunc("/api/billing/profit-sharing/referrer-pending/", handleReferrerProfitSharingPending)

	mux.HandleFunc("/api/internal/taskbill/admin/orders/", handleInternalAdminListAllOrders)
	mux.HandleFunc("/api/internal/taskbill/admin/user-recharge-consumption/", handleInternalUserRechargeConsumption)

	// Legal (license_agreement + privacy_policy) — migrated from Django 2026-07-26
	mux.HandleFunc("/api/system_admin/license-agreement/", handleSystemAdminLicenseAgreementRouter)
	mux.HandleFunc("/api/system_admin/privacy-policy/", handleSystemAdminPrivacyPolicyRouter)
	// Convention alias: system-admin (dash)
	mux.HandleFunc("/api/system-admin/license-agreement/", handleSystemAdminLicenseAgreementRouter)
	mux.HandleFunc("/api/system-admin/privacy-policy/", handleSystemAdminPrivacyPolicyRouter)

	// OPT-049: system-admin orders — migrated from Django 2026-07-30
	mux.HandleFunc("/api/system_admin/orders/{order_id}/comments/", handleSystemAdminOrderComments)
	mux.HandleFunc("/api/system-admin/orders/{order_id}/comments/", handleSystemAdminOrderComments)
	mux.HandleFunc("/api/system_admin/orders/{order_id}/", handleSystemAdminGetOrder)
	mux.HandleFunc("/api/system-admin/orders/{order_id}/", handleSystemAdminGetOrder)
	mux.HandleFunc("/api/system_admin/orders/", handleSystemAdminListOrders)
	// Convention alias: system-admin (dash)
	mux.HandleFunc("/api/system-admin/orders/", handleSystemAdminListOrders)
	mux.HandleFunc("/api/system-admin/orders", handleSystemAdminListOrders)
	mux.HandleFunc("/api/system-admin/tenant-quotas/", handleSystemAdminTenantQuotas)
	mux.HandleFunc("/api/system-admin/tenant-quotas", handleSystemAdminTenantQuotas)
	mux.HandleFunc("/api/system_admin/tenant-quotas/", handleSystemAdminTenantQuotas)
	mux.HandleFunc("/api/system_admin/tenant-quotas", handleSystemAdminTenantQuotas)
	// OPT-20260823-058: 幂等键审计记录（admin_grant 操作者/目标租户追溯）
	mux.HandleFunc("/api/system-admin/idempotency-records/", handleSystemAdminListIdempotencyRecords)
	mux.HandleFunc("/api/system_admin/idempotency-records/", handleSystemAdminListIdempotencyRecords)
	mux.HandleFunc("/api/system-admin/profit-sharing/refresh-wechat/", handleSystemAdminRefreshProfitSharingWechat)
	mux.HandleFunc("/api/system-admin/profit-sharing/refresh-wechat", handleSystemAdminRefreshProfitSharingWechat)
	mux.HandleFunc("/api/system_admin/profit-sharing/refresh-wechat/", handleSystemAdminRefreshProfitSharingWechat)
	mux.HandleFunc("/api/system_admin/profit-sharing/refresh-wechat", handleSystemAdminRefreshProfitSharingWechat)
	// 不用 {id}/share：与 refresh-wechat/ 在 ServeMux 上互相冲突（{id} 可匹配 refresh-wechat）
	mux.HandleFunc("/api/system_admin/profit-sharing/", handleSystemAdminProfitSharingRoutes)
	mux.HandleFunc("/api/system_admin/profit-sharing", handleSystemAdminProfitSharingRoutes)
	mux.HandleFunc("/api/system-admin/profit-sharing/", handleSystemAdminProfitSharingRoutes)
	mux.HandleFunc("/api/system-admin/profit-sharing", handleSystemAdminProfitSharingRoutes)
	mux.HandleFunc("/api/system-admin/order-number/parse/", handleSystemAdminParseOrderNumber)
	mux.HandleFunc("/api/system-admin/order-number/parse", handleSystemAdminParseOrderNumber)

	// OPT-049: system-admin user recharge history — migrated from Django 2026-07-30
	mux.HandleFunc("/api/system_admin/users/{uid}/recharges/", handleSystemAdminUserRecharges)
	// Convention alias: system-admin (dash)
	mux.HandleFunc("/api/system-admin/users/{uid}/recharges/", handleSystemAdminUserRecharges)

	// System-admin pricing/refund/consumption — migrated from Django 2026-07-30 (OPT-040)
	mux.HandleFunc("/api/system_admin/gitlab-regions/", handleSystemAdminGitlabRegions)
	mux.HandleFunc("/api/system_admin/gitlab-regions", handleSystemAdminGitlabRegions)
	// Convention alias: system-admin (dash)
	mux.HandleFunc("/api/system-admin/gitlab-regions/", handleSystemAdminGitlabRegions)
	mux.HandleFunc("/api/system-admin/gitlab-regions", handleSystemAdminGitlabRegions)
	mux.HandleFunc("/api/system-admin/resource-pricing/", handleInternalResourcePricingRouter)
	mux.HandleFunc("/api/system-admin/resource-pricing", handleInternalResourcePricingRouter)
	mux.HandleFunc("/api/system-admin/user-recharge-consumption/", handleInternalUserRechargeConsumption)
	mux.HandleFunc("/api/system-admin/user-recharge-consumption", handleInternalUserRechargeConsumption)
	mux.HandleFunc("/api/system-admin/refund-applications/", handleInternalRefundApplicationsRouter)
	mux.HandleFunc("/api/system-admin/refund-applications", handleInternalRefundApplicationsRouter)
	mux.HandleFunc("/api/system-admin/invoice-applications/", handleSystemAdminInvoiceApplicationsRouter)
	mux.HandleFunc("/api/system-admin/invoice-applications", handleSystemAdminInvoiceApplicationsRouter)
	mux.HandleFunc("/api/system-admin/refund-policy/", handleSystemAdminRefundPolicy)
	mux.HandleFunc("/api/system-admin/refund-policy", handleSystemAdminRefundPolicy)

	mux.HandleFunc("/api/license-agreement/public/current/", handlePublicCurrentLicenseAgreement)
	mux.HandleFunc("/api/license-agreement/consent/", handleLicenseConsent)
	mux.HandleFunc("/api/privacy-policy/public/current/", handlePublicCurrentPrivacyPolicy)
	mux.HandleFunc("/api/privacy-policy/consent/", handlePrivacyConsent)

	// OPT-049: public product pricing — migrated from Django 2026-07-30
	mux.HandleFunc("/api/public/product-pricing/", handlePublicProductPricing)

	mux.HandleFunc("/api/system-admin/feedback-link-groups/", handleSystemAdminFeedbackLinkGroupsRouter)
	mux.HandleFunc("/api/system-admin/feedback-resource-kinds/", handleSystemAdminFeedbackResourceKinds)
	mux.HandleFunc("/api/system-admin/feedback-resource-kinds", handleSystemAdminFeedbackResourceKinds)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := stripTenantIDKV(r.URL.Path)
		switch {
		case strings.HasPrefix(p, "/api/billing/wechat/fapiao/notify"):
			handleWechatFapiaoNotify(w, r)
		case strings.HasPrefix(p, "/api/billing/wechat/notify"):
			handleWechatNotify(w, r)
		case strings.HasPrefix(p, "/api/billing/profitsharing/change-notify"):
			handleProfitSharingNotify(w, r)
		case strings.HasPrefix(p, "/api/billing/profitsharing/notify"):
			handleProfitSharingNotify(w, r)
		case strings.Contains(p, "/billing/accounts/admin_grant_points"):
			handleAdminGrantResources(w, r)
		case strings.Contains(p, "/billing/profit-sharing/referrer-orders/"):
			handleReferrerProfitSharingOrders(w, r)
		case strings.HasSuffix(p, "/billing/feedback-links/"):
			handleTenantFeedbackLinks(w, r)
		case strings.HasSuffix(p, "/billing/refund-applications/"):
			handleTenantRefundApplications(w, r)
		case strings.HasSuffix(p, "/billing/accounts/balance/"):
			handleBalance(w, r)
		case strings.HasSuffix(p, "/billing/gitlab-resources/purchase/"):
			handleGitlabResourcesPurchase(w, r)
		case strings.HasSuffix(p, "/billing/gitlab-resources/provision/"):
			handleAdminProvisionGitlabResource(w, r)
		case strings.HasSuffix(p, "/billing/gitlab-regions/"):
			handleGitlabRegionsList(w, r)
		case strings.HasSuffix(p, "/billing/gitlab-resources/"):
			handleGitlabResources(w, r)
		// 手机号验证（支付前门禁）
		case strings.HasSuffix(p, "/billing/phone-verification-status/"):
			handlePhoneVerificationStatus(w, r)
		case strings.HasSuffix(p, "/billing/verify-phone-code/"):
			handleVerifyPhoneCode(w, r)
		// 资源订单
		case strings.Contains(p, "/billing/orders/") && strings.HasSuffix(p, "/invoice-applications/"):
			handleTenantInvoiceApplications(w, r)
		case strings.Contains(p, "/billing/orders/") && strings.Contains(p, "/invoices/") && strings.HasSuffix(p, "/file/"):
			handleInvoiceFileGET(w, r)
		case strings.Contains(p, "/billing/orders/") && strings.HasSuffix(p, "/invoices/"):
			handleTenantListInvoices(w, r)
		case strings.Contains(p, "/billing/orders/") && strings.HasSuffix(p, "/callback/"):
			handleOrderPaymentCallback(w, r)
		case strings.Contains(p, "/billing/orders/") && strings.HasSuffix(p, "/pay/"):
			handlePayOrder(w, r)
		case strings.Contains(p, "/billing/orders/") && strings.HasSuffix(p, "/cancel/"):
			handleOrderCancel(w, r)
		case strings.Contains(p, "/billing/orders/") && strings.HasSuffix(p, "/comments/"):
			handleTenantOrderComments(w, r)
		case strings.Contains(p, "/billing/orders/") && !strings.HasSuffix(p, "/orders/"):
			handleGetOrder(w, r)
		case strings.HasSuffix(p, "/billing/orders/"):
			if r.Method == http.MethodPost {
				handleCreateOrder(w, r)
			} else {
				handleListOrders(w, r)
			}
		case strings.HasSuffix(p, "/billing/quotas/"):
			handleResourceQuotas(w, r)
		case strings.HasSuffix(p, "/billing/order-pricing/"):
			handleOrderPricingV2(w, r)
		case strings.HasSuffix(p, "/billing/units/"):
			handleUnitsList(w, r)
		case strings.HasSuffix(p, "/billing/statistics/"):
			handleBillingStatistics(w, r)
		case strings.HasSuffix(p, "/billing/membership/"):
			handleMembership(w, r)
		case strings.HasPrefix(p, "/api/public/resource-pricing"):
			handlePublicResourcePricing(w, r)
		case strings.HasSuffix(p, "/billing/transactions/list_filtered/"):
			handleTransactionsList(w, r, true)
		case strings.HasSuffix(p, "/billing/transactions/"):
			handleTransactionsList(w, r, false)
		case strings.HasSuffix(p, "/billing/usages/"):
			handleUsagesList(w, r)
		default:
			writeErrorJSONMap(w, r, http.StatusNotFound, map[string]string{"detail": "not found", "path": p})
		}
	})
}
