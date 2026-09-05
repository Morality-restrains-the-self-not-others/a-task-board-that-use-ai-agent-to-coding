package main

import "testing"

func TestPublishMemberJoinedHelper_NoKafkaNoPanic(t *testing.T) {
	// When KAFKA_BOOTSTRAP_SERVERS / cfg bootstrap empty, publishEvent returns after log.
	publishMemberJoined("m1", "u1", "c1", "Alice", "member", "ws1", "unit_test", map[string]interface{}{
		"invitation_id": "inv1",
	})
}
