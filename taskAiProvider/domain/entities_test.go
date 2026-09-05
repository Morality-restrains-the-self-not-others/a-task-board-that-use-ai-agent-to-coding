package domain

import (
	"testing"
	"time"
)

func TestContainerImageSubmitApproveReject(t *testing.T) {
	img := &ContainerImage{Status: StatusDraft}
	if err := img.Submit(); err != nil {
		t.Fatal(err)
	}
	if img.Status != StatusPendingReview {
		t.Fatalf("got %s", img.Status)
	}
	now := time.Now().UTC()
	if err := img.Approve(1, "ok", now); err != nil {
		t.Fatal(err)
	}
	if img.Status != StatusApproved {
		t.Fatalf("got %s", img.Status)
	}
	img2 := &ContainerImage{Status: StatusPendingReview}
	if err := img2.Reject(1, "bad", now); err != nil {
		t.Fatal(err)
	}
	if img2.Status != StatusRejected {
		t.Fatalf("got %s", img2.Status)
	}
	img3 := &ContainerImage{Status: StatusPendingReview}
	if err := img3.Reject(1, "", now); err == nil {
		t.Fatal("expected reject note required")
	}
}

func TestContainerImageCanDeleteAndVendorWithdraw(t *testing.T) {
	now := time.Now().UTC()
	active := &ContainerImage{Status: StatusApproved, IsActive: true}
	if active.CanDelete() {
		t.Fatal("active approved must not be deletable")
	}
	if err := active.DeleteGuard(); err == nil {
		t.Fatal("expected DeleteGuard error for active")
	}
	inactive := &ContainerImage{Status: StatusApproved, IsActive: false}
	if !inactive.CanDelete() {
		t.Fatal("inactive approved should be deletable")
	}
	pending := &ContainerImage{Status: StatusPendingReview}
	if pending.CanDelete() {
		t.Fatal("pending_review must not be deletable")
	}
	if err := pending.VendorWithdraw(9, "", now); err != nil {
		t.Fatal(err)
	}
	if pending.Status != StatusDraft {
		t.Fatalf("pending withdraw got %s", pending.Status)
	}
	approved := &ContainerImage{Status: StatusApproved, IsActive: true}
	if err := approved.VendorWithdraw(9, "厂商自行下架", now); err != nil {
		t.Fatal(err)
	}
	if approved.Status != StatusDraft || approved.IsActive {
		t.Fatalf("approved withdraw status=%s active=%v", approved.Status, approved.IsActive)
	}
	draft := &ContainerImage{Status: StatusDraft}
	if err := draft.VendorWithdraw(9, "", now); err != nil {
		t.Fatal(err)
	}
	if !draft.CanDelete() {
		t.Fatal("draft should be deletable")
	}
}
