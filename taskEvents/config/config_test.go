package config

import (
	"os"
	"path/filepath"
	"testing"

	"taskEvents/domain"
)

func TestLoadFromPortConfig(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Skip("monorepo root not found:", err)
	}
	cfg, loadedRoot, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loadedRoot != root {
		t.Fatalf("root mismatch %q vs %q", loadedRoot, root)
	}
	if cfg.Transport != domain.TransportKafka {
		t.Fatalf("expected kafka transport, got %q", cfg.Transport)
	}
	if cfg.RedisPort != 6379 {
		t.Fatalf("redis port: %d", cfg.RedisPort)
	}
	if cfg.RedisHost == "" {
		t.Fatalf("redis host must be set, got empty")
	}
	if cfg.BootstrapServers == "" {
		t.Fatalf("kafka bootstrap servers must be set")
	}
}

func TestLoadIntentCompanyFanOutPorts(t *testing.T) {
	if _, err := FindMonorepoRoot(); err != nil {
		t.Skip("monorepo root not found:", err)
	}
	_, c1, _, err := LoadIntent("company_created", "1_set_default_deliverable_system")
	if err != nil {
		t.Fatal(err)
	}
	_, c2, _, err := LoadIntent("company_created", "2_set_default_progress_system")
	if err != nil {
		t.Fatal(err)
	}
	_, c3, _, err := LoadIntent("company_created", "3_create_default_workspace")
	if err != nil {
		t.Fatal(err)
	}
	if c1.Port != 18027 || c2.Port != 18028 || c3.Port != 18029 {
		t.Fatalf("ports: %d %d %d", c1.Port, c2.Port, c3.Port)
	}
}

func TestLoadIntentWorkPanelFanoutPort(t *testing.T) {
	if _, err := FindMonorepoRoot(); err != nil {
		t.Skip("monorepo root not found:", err)
	}
	_, c, _, err := LoadIntent("task_status_changed", "2_fanout_work_panel_sse")
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != 18048 {
		t.Fatalf("port: %d", c.Port)
	}
	if c.GroupID != "task-events-task-status-changed-2-fanout-work-panel-sse" {
		t.Fatalf("groupId: %s", c.GroupID)
	}
	wantEvents := map[string]bool{
		"TASK_STATUS_CHANGED": true,
		"TASK_CREATED":        true,
		"TASK_DELETED":        true,
	}
	if len(c.Events) != len(wantEvents) {
		t.Fatalf("events: %v", c.Events)
	}
	for _, e := range c.Events {
		if !wantEvents[e] {
			t.Fatalf("unexpected event %s in %v", e, c.Events)
		}
	}
}

func TestTopicsForEvents(t *testing.T) {
	topics := TopicsForEvents([]string{"USER_CREATED", "COMPANY_CREATED"})
	if len(topics) != 2 {
		t.Fatalf("got %v", topics)
	}
	crud := TopicsForEvents([]string{"TASK_CREATED", "TASK_DELETED"})
	if len(crud) != 2 || crud[0] == "" || crud[1] == "" {
		t.Fatalf("TASK_CREATED/DELETED topics: %v", crud)
	}
	invite := TopicsForEvents([]string{
		"REGISTRATION_INVITE_POLICY_UPDATED",
		"REGISTRATION_INVITE_CODE_ISSUED",
		"REGISTRATION_INVITE_CODE_REDEEMED",
	})
	if len(invite) != 3 {
		t.Fatalf("registration invite topics: %v", invite)
	}
}

func TestEventTopicCloudSnapshotAndBindingAdvanced(t *testing.T) {
	// taskCloudService publishes these events; the topics must resolve so
	// consumers/infra can align without an Unknown Topic Or Partition.
	cases := map[string]string{
		"LayerGraphSnapshotPersisted":        "layer-graph-snapshot-persisted",
		"JobStepFullArchived":                "job-step-full-archived",
		"CommentStartupLogArchived":          "comment-startup-log-archived",
		"StepFullCOSConfigUpdated":           "step-full-cos-config-updated",
		"CommentContainerBindingAdvanced":    "comment-container-binding-advanced",
		"CONTAINER_INSTRUCTION_IDLE_MARKED":  "container-instruction-idle-marked",
		"CONTAINER_INSTRUCTION_IDLE_CLEARED": "container-instruction-idle-cleared",
		"UserImpersonationStarted":           "user-impersonation-started",
		"UserImpersonationStopped":           "user-impersonation-stopped",
		"UserInboxMessageCreated":            "user-inbox-message-created",
	}
	for ev, wantTopic := range cases {
		topic, ok := EventTopic(ev)
		if !ok {
			t.Fatalf("EventTopic(%q) not registered", ev)
		}
		if topic != wantTopic {
			t.Fatalf("EventTopic(%q)=%q want %q", ev, topic, wantTopic)
		}
	}
	got := TopicsForEvents([]string{"LayerGraphSnapshotPersisted", "CommentContainerBindingAdvanced"})
	if len(got) != 2 {
		t.Fatalf("topics=%v want 2", got)
	}
}

func TestLoadIntentRegistrationInviteObservability(t *testing.T) {
	if _, err := FindMonorepoRoot(); err != nil {
		t.Skip("monorepo root not found:", err)
	}
	_, c, _, err := LoadIntent("registration_invite", "1_observability")
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != 18049 {
		t.Fatalf("port: %d", c.Port)
	}
	want := map[string]bool{
		"REGISTRATION_INVITE_POLICY_UPDATED": true,
		"REGISTRATION_INVITE_CODE_ISSUED":    true,
		"REGISTRATION_INVITE_CODE_REDEEMED":  true,
	}
	if len(c.Events) != len(want) {
		t.Fatalf("events: %v", c.Events)
	}
	for _, e := range c.Events {
		if !want[e] {
			t.Fatalf("unexpected event %s in %v", e, c.Events)
		}
	}
}

func TestLoadFromTempDomainEventsYaml(t *testing.T) {
	t.Setenv("CONF_ROOT", "")
	t.Setenv("DEPLOY_ROOT", "")
	dir := t.TempDir()
	// OPT-20260806-057: root 锚点已切到 conf/base.yaml（django 目录退役）
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "conf", "base.yaml"),
		[]byte("host: localhost\nport: 8001\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	deDir := filepath.Join(dir, "conf", "events", "domain-events")
	if err := os.MkdirAll(deDir, 0o755); err != nil {
		t.Fatal(err)
	}
	global := `transport: redis
internalSecret: sec
kafka:
  bootstrapServers: kafka:9093
redis:
  host: 127.0.0.1
  port: 6379
  streamKeyPrefix: "domain-events:"
`
	if err := os.WriteFile(filepath.Join(deDir, "config.yaml"), []byte(global), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Transport != domain.TransportRedis {
		t.Fatalf("transport %q", cfg.Transport)
	}
}
