package main

import "testing"

func TestAuthzEventTypeConstants(t *testing.T) {
	cases := []struct {
		got, want string
	}{
		{eventTypeRoleChanged, "RoleChanged"},
		{eventTypePlatformRoleChanged, "PlatformRoleChanged"},
		{eventTypeRoleResourceGroupsChanged, "RoleResourceGroupsChanged"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Fatalf("event type = %q, want %q", tc.got, tc.want)
		}
	}
}

func TestPublishRoleResourceGroupsChanged_NoPanicWhenKafkaOff(t *testing.T) {
	prev := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prev })
	t.Setenv("KAFKA_BOOTSTRAP_SERVERS", "")
	t.Setenv("TASKAUTH_KAFKA_BOOTSTRAP_SERVERS", "")
	// Should not panic / block when Kafka is unavailable.
	publishRoleResourceGroupsChanged("c1", "role-1")
}
