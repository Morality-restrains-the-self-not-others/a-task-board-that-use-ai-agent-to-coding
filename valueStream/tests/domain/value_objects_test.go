package domain_test

import (
	"testing"

	"valueStream/domain/value_objects"
)

func TestDomainName_NewDomainName_TrimAndValidate(t *testing.T) {
	domain, err := value_objects.NewDomainName("  auth  ")
	if err != nil {
		t.Fatalf("NewDomainName returned error: %v", err)
	}
	if domain.Value() != "auth" {
		t.Fatalf("domain = %q, want auth", domain.Value())
	}
	if _, err := value_objects.NewDomainName("   "); err == nil {
		t.Fatal("expected empty domain name to fail")
	}
}

func TestDomainOrder_NewDomainOrder_ValidateAndCopy(t *testing.T) {
	order, err := value_objects.NewDomainOrder([]string{"B", "A"})
	if err != nil {
		t.Fatalf("NewDomainOrder returned error: %v", err)
	}
	domains := order.Domains()
	if len(domains) != 2 || domains[0].Value() != "B" || domains[1].Value() != "A" {
		t.Fatalf("unexpected domains: %+v", domains)
	}

	// Ensure Domains() returns a defensive copy.
	domains[0], _ = value_objects.NewDomainName("X")
	after := order.Domains()
	if after[0].Value() != "B" {
		t.Fatalf("order mutated by external change, got %q", after[0].Value())
	}

	if _, err := value_objects.NewDomainOrder([]string{"A", "A"}); err == nil {
		t.Fatal("expected duplicate domains to fail")
	}
	if _, err := value_objects.NewDomainOrder([]string{}); err == nil {
		t.Fatal("expected empty domain order to fail")
	}
}

