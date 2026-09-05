package main

import "testing"

func TestManagedServiceSysProcAttr_IndependentSession(t *testing.T) {
	attr := managedServiceSysProcAttr()
	if attr == nil {
		t.Fatal("attr is nil")
	}
	if !attr.Setsid {
		t.Fatal("Setsid must be true so the child is not in runAll's session")
	}
	if attr.Setpgid {
		t.Fatal("Setpgid after Setsid is EPERM; session leader already has pgid==pid")
	}
}
