package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func billServiceBaseURL() string {
	if u := strings.TrimRight(strings.TrimSpace(cfg.BillServiceURL), "/"); u != "" {
		return u
	}
	return "http://127.0.0.1:8004"
}

func cloudServiceBaseURL() string {
	if u := strings.TrimRight(strings.TrimSpace(cfg.CloudServiceURL), "/"); u != "" {
		return u
	}
	return "http://127.0.0.1:8018"
}

func tenantServiceBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(cfg.TenantServiceURL), "/")
}

func fetchBillDeletionBlockers(ctx context.Context, userID string) ([]deletionBlocker, error) {
	url := fmt.Sprintf("%s/api/internal/taskbill/users/%s/account-deletion-blockers/", billServiceBaseURL(), userID)
	body, status, err := internalServiceGet(ctx, url, "X-TaskBill-Internal-Secret", cfg.BillInternalSecret)
	if err != nil {
		return []deletionBlocker{{
			Code: "BILLING_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证账户资金状态，请稍后重试",
		}}, nil
	}
	if status != http.StatusOK {
		return []deletionBlocker{{
			Code: "BILLING_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证账户资金状态，请稍后重试",
		}}, nil
	}
	return decodeBlockersResponse(body)
}

func fetchTenantDeletionBlockers(ctx context.Context, userID string) ([]deletionBlocker, error) {
	base := tenantServiceBaseURL()
	if base == "" {
		return []deletionBlocker{{
			Code: "TENANT_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证租户成员状态，请稍后重试",
		}}, nil
	}
	url := fmt.Sprintf("%s/api/internal/tenant/users/%s/account-deletion-blockers/", base, userID)
	body, status, err := internalServiceGet(ctx, url, "X-Internal-Secret", cfg.InternalSecret)
	if err != nil {
		return []deletionBlocker{{
			Code: "TENANT_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证租户成员状态，请稍后重试",
		}}, nil
	}
	if status != http.StatusOK {
		return []deletionBlocker{{
			Code: "TENANT_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证租户成员状态，请稍后重试",
		}}, nil
	}
	return decodeBlockersResponse(body)
}

func fetchCloudDeletionBlockers(ctx context.Context, userID string) ([]deletionBlocker, error) {
	url := fmt.Sprintf("%s/api/internal/cloud/users/%s/account-deletion-blockers/", cloudServiceBaseURL(), userID)
	body, status, err := internalServiceGet(ctx, url, "X-Internal-Secret", cfg.InternalSecret)
	if err != nil {
		return []deletionBlocker{{
			Code: "CLOUD_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证云资源状态，请稍后重试",
		}}, nil
	}
	if status != http.StatusOK {
		return []deletionBlocker{{
			Code: "CLOUD_CHECK_UNAVAILABLE", Blocking: true,
			Message: "无法验证云资源状态，请稍后重试",
		}}, nil
	}
	return decodeBlockersResponse(body)
}
