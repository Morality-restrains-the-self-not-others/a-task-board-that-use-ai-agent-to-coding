package main

import "testing"

func TestTenantMembersActionURLUsesPeopleManage(t *testing.T) {
	got := tenantMembersActionURL("877397588196749312")
	want := "/tenant/877397588196749312/people/manage/"
	if got != want {
		t.Fatalf("tenantMembersActionURL = %q, want %q (settings/members 前端无此路由，会跳回首页)", got, want)
	}
}
