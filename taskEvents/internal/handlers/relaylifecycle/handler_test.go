package relaylifecycle

import "testing"

func TestMapRelayBusToAuditEventType(t *testing.T) {
	audit, ok := mapRelayBusToAuditEventType("RELAY_START_ACCEPTED")
	if !ok || audit != "relay_start_accepted" {
		t.Fatalf("got %q ok=%v", audit, ok)
	}
	if _, ok := mapRelayBusToAuditEventType("UNKNOWN"); ok {
		t.Fatal("expected unknown to fail")
	}
}
