package main

import (
	"testing"
)

func TestCoarsePermsFromGrantKeys_PeopleAccess(t *testing.T) {
	specs := []applyGrantSpec{
		{key: "people.access", effect: "operate"},
		{key: "people.access.save_actions", effect: "operate"},
	}
	perms := coarsePermsFromGrantKeys(specs)
	found := false
	for _, p := range perms {
		if p == "member:manage" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected member:manage in %v", perms)
	}
}

func TestCoarsePermsFromGrantKeys_Billing(t *testing.T) {
	specs := []applyGrantSpec{{key: "billing.overview", effect: "view"}}
	perms := coarsePermsFromGrantKeys(specs)
	found := false
	for _, p := range perms {
		if p == "billing:view" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected billing:view in %v", perms)
	}
}
