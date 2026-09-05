package domain

import "testing"

func TestEvaluatePhoneShareBind_EmptyOthersAllow(t *testing.T) {
	if got := EvaluatePhoneShareBind(false, nil); got != PhoneShareAllow {
		t.Fatalf("empty others: got %s want allow", got)
	}
	if got := EvaluatePhoneShareBind(true, []bool{}); got != PhoneShareAllow {
		t.Fatalf("empty slice: got %s want allow", got)
	}
}

func TestEvaluatePhoneShareBind_CustomerTaken(t *testing.T) {
	if got := EvaluatePhoneShareBind(false, []bool{true}); got != PhoneShareTaken {
		t.Fatalf("customer vs privileged holder: got %s want taken", got)
	}
	if got := EvaluatePhoneShareBind(false, []bool{false}); got != PhoneShareTaken {
		t.Fatalf("customer vs customer: got %s want taken", got)
	}
}

func TestEvaluatePhoneShareBind_PrivilegedVsCustomerTaken(t *testing.T) {
	if got := EvaluatePhoneShareBind(true, []bool{true, false}); got != PhoneShareTaken {
		t.Fatalf("mix includes customer: got %s want taken", got)
	}
}

func TestEvaluatePhoneShareBind_PrivilegedShareUnderLimit(t *testing.T) {
	four := []bool{true, true, true, true}
	if got := EvaluatePhoneShareBind(true, four); got != PhoneShareAllow {
		t.Fatalf("4 privileged others: got %s want allow", got)
	}
}

func TestEvaluatePhoneShareBind_PrivilegedAtLimit(t *testing.T) {
	five := []bool{true, true, true, true, true}
	if got := EvaluatePhoneShareBind(true, five); got != PhoneShareLimit {
		t.Fatalf("5 privileged others: got %s want limit", got)
	}
}

func TestIsPhoneLoginAmbiguous(t *testing.T) {
	if IsPhoneLoginAmbiguous(0) || IsPhoneLoginAmbiguous(1) {
		t.Fatal("0 or 1 match must not be ambiguous")
	}
	if !IsPhoneLoginAmbiguous(2) {
		t.Fatal("2 matches must be ambiguous")
	}
}

func TestMaxSharedPhoneBindings(t *testing.T) {
	if MaxSharedPhoneBindings != 5 {
		t.Fatalf("quota must be 5, got %d", MaxSharedPhoneBindings)
	}
}
