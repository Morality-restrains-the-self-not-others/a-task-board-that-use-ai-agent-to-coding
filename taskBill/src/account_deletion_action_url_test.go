package main

import "testing"

func TestGitlabResourceActionURLUsesSettingsPage(t *testing.T) {
	got := gitlabResourceActionURL("877397588196749312")
	want := "/tenant/877397588196749312/settings/gitlab-connection/"
	if got != want {
		t.Fatalf("gitlabResourceActionURL = %q, want %q (billing/gitlab-resources 前端无此路由，会跳回首页)", got, want)
	}
}

func TestPendingPaymentActionURLUsesOrderOrDashboard(t *testing.T) {
	if got := pendingPaymentActionURL(877397588196749312, 99); got != "/tenant/877397588196749312/billing/orders/99/" {
		t.Fatalf("with order: got %q", got)
	}
	if got := pendingPaymentActionURL(877397588196749312, 0); got != "/tenant/877397588196749312/billing/" {
		t.Fatalf("recharge pending: got %q", got)
	}
	if got := pendingPaymentActionURL(0, 0); got != "/profile/" {
		t.Fatalf("no tenant fallback: got %q", got)
	}
}

func TestBillingDashboardAndOrdersActionURL(t *testing.T) {
	if got := billingDashboardActionURL("t1"); got != "/tenant/t1/billing/" {
		t.Fatalf("dashboard = %q", got)
	}
	if got := billingOrdersActionURL("t1"); got != "/tenant/t1/billing/orders/" {
		t.Fatalf("orders = %q", got)
	}
}
