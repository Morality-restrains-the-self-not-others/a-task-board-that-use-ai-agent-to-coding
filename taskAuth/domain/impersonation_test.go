package domain

import (
	"strings"
	"testing"
)

func validStart(d StartImpersonationDecision) StartImpersonationDecision {
	if d.Reason == "" {
		d.Reason = "排查线上工单问题"
	}
	return d
}

func TestValidateStart_OK(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID:            "admin",
		TargetUserID:           "user",
		ActorHasPlatformManage: true,
		TargetActive:           true,
		IdempotencyKey:         "k1",
	}))
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateStart_Self(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID: "u1", TargetUserID: "u1",
		TargetActive: true, IdempotencyKey: "k",
	}))
	if err != ErrSelfImpersonation {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_Nested(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID: "a", TargetUserID: "b",
		ActorAlreadyImpersonating: true,
		TargetActive:              true,
		IdempotencyKey:            "k",
	}))
	if err != ErrNestedImpersonation {
		t.Fatalf("got %v", err)
	}
	if ErrNestedImpersonation.Error() != "already impersonating" {
		t.Fatalf("sentinel Error() must stay English for errors.Is, got %q", ErrNestedImpersonation.Error())
	}
	if NestedImpersonationClientMessage != "已在模拟登录中，请先退出" {
		t.Fatalf("client copy drifted: %q", NestedImpersonationClientMessage)
	}
}

func TestValidateStart_Inactive(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID: "a", TargetUserID: "b",
		TargetActive: false, IdempotencyKey: "k",
	}))
	if err != ErrTargetInactive {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_Privilege(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID:            "emp",
		TargetUserID:           "admin",
		ActorHasPlatformManage: false,
		TargetIsPlatformAdmin:  true,
		TargetActive:           true,
		IdempotencyKey:         "k",
	}))
	if err != ErrPrivilegeEscalation {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_MissingKey(t *testing.T) {
	err := ValidateStart(StartImpersonationDecision{
		ActorUserID: "a", TargetUserID: "b", TargetActive: true, Reason: "排查线上工单问题",
	})
	if err != ErrMissingIdempotencyKey {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_MissingActor(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		TargetUserID: "b", TargetActive: true, IdempotencyKey: "k",
	}))
	if err != ErrActorOrTargetRequired {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_EmployeeCanImpersonateRegular(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID:            "emp",
		TargetUserID:           "user",
		ActorHasPlatformManage: false,
		TargetIsPlatformAdmin:  false,
		TargetActive:           true,
		IdempotencyKey:         "k",
	}))
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateStart_SuperAdminCanImpersonateAdmin(t *testing.T) {
	err := ValidateStart(validStart(StartImpersonationDecision{
		ActorUserID:            "root",
		TargetUserID:           "admin",
		ActorHasPlatformManage: true,
		TargetIsPlatformAdmin:  true,
		TargetActive:           true,
		IdempotencyKey:         "k",
	}))
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateStart_MissingReason(t *testing.T) {
	err := ValidateStart(StartImpersonationDecision{
		ActorUserID: "a", TargetUserID: "b", TargetActive: true, IdempotencyKey: "k",
	})
	if err != ErrMissingReason {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_ShortReason(t *testing.T) {
	err := ValidateStart(StartImpersonationDecision{
		ActorUserID: "a", TargetUserID: "b", TargetActive: true, IdempotencyKey: "k",
		Reason: "短",
	})
	if err != ErrMissingReason {
		t.Fatalf("got %v", err)
	}
}

func TestValidateStart_ReasonTooLong(t *testing.T) {
	err := ValidateStart(StartImpersonationDecision{
		ActorUserID: "a", TargetUserID: "b", TargetActive: true, IdempotencyKey: "k",
		Reason: strings.Repeat("理", 501),
	})
	if err != ErrReasonTooLong {
		t.Fatalf("got %v", err)
	}
}

func TestValidateImpersonationReason_Trims(t *testing.T) {
	if err := ValidateImpersonationReason("  排查线上工单问题  "); err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestValidateImpersonationReason_RejectsModalPrompt(t *testing.T) {
	for _, s := range []string{
		"请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。",
		"例如：排查线上工单 T-12345",
		"  请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。  ",
	} {
		if err := ValidateImpersonationReason(s); err != ErrReasonIsPromptText {
			t.Fatalf("ValidateImpersonationReason(%q) = %v, want ErrReasonIsPromptText", s, err)
		}
	}
}

func TestValidateImpersonationReason_AcceptsRealJustification(t *testing.T) {
	for _, s := range []string{
		"排查线上工单问题，需验证账号绑定关系",
		"例如：排查线上工单 T-12345 的疑似撞库登录", // starts with placeholder but is a real justification
		"请填写本次模拟登录的理由，该理由用于排查",
	} {
		if err := ValidateImpersonationReason(s); err != nil {
			t.Fatalf("ValidateImpersonationReason(%q) = %v, want nil", s, err)
		}
	}
}

func TestImpersonationNoticeBody_DoesNotEmbedReason(t *testing.T) {
	body := ImpersonationNoticeBody("bootstrap-admin")
	if body == "" {
		t.Fatal("empty body")
	}
	if strings.Contains(body, "理由") {
		t.Fatalf("notice body must not embed reason label: %q", body)
	}
	if !strings.Contains(body, "bootstrap-admin") {
		t.Fatalf("missing actor id: %q", body)
	}
}

func TestStripEmbeddedInboxReason_RemovesTrailingDuplicate(t *testing.T) {
	reason := "排查线上工单问题"
	embedded := ImpersonationNoticeBody("admin") + "\n理由：" + reason
	got := StripEmbeddedInboxReason(embedded, reason)
	if strings.Contains(got, "理由") {
		t.Fatalf("still embedded: %q", got)
	}
	if got != ImpersonationNoticeBody("admin") {
		t.Fatalf("got %q", got)
	}
}

func TestStripEmbeddedInboxReason_LeavesUnrelatedBody(t *testing.T) {
	body := ImpersonationNoticeBody("admin") + "\n另附说明"
	got := StripEmbeddedInboxReason(body, "排查线上工单问题")
	if got != body {
		t.Fatalf("got %q", got)
	}
}
