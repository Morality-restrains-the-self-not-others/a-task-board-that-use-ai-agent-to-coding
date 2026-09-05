package config

// EventDefinition describes one per-event consumer binary.
type EventDefinition struct {
	Slug      string // directory name under bin/, e.g. billing_transaction_created
	EventType string // domain event_type constant
	Port      int
	GroupID   string
}

// AllEvents is legacy v3 per-event registry (primary intent ports only).
var AllEvents = []EventDefinition{
	{Slug: "billing_transaction_created", EventType: "BILLING_TRANSACTION_CREATED", Port: 18020, GroupID: "task-events-billing-transaction-created"},
	{Slug: "sse_message", EventType: "SSE_MESSAGE", Port: 18021, GroupID: "task-events-sse-message"},
	{Slug: "email_sent", EventType: "EMAIL_SENT", Port: 18022, GroupID: "task-events-email-sent"},
	{Slug: "invitation_created", EventType: "INVITATION_CREATED", Port: 18023, GroupID: "task-events-invitation-created"},
	{Slug: "user_activated", EventType: "USER_ACTIVATED", Port: 18024, GroupID: "task-events-user-activated"},
	{Slug: "user_created", EventType: "USER_CREATED", Port: 18025, GroupID: "task-events-user-created"},
	{Slug: "company_created", EventType: "COMPANY_CREATED", Port: 18027, GroupID: "task-events-company-created"},
	{Slug: "workspace_created", EventType: "WORKSPACE_CREATED", Port: 18030, GroupID: "task-events-workspace-created"},
	{Slug: "project_updated", EventType: "PROJECT_UPDATED", Port: 18031, GroupID: "task-events-project-updated"},
	{Slug: "task_completed", EventType: "TASK_COMPLETED", Port: 18032, GroupID: "task-events-task-completed"},
	{Slug: "ai_assistant_reply_completed", EventType: "AI_ASSISTANT_REPLY_COMPLETED", Port: 18033, GroupID: "task-events-ai-assistant-reply-completed"},
	{Slug: "cloud_server_started", EventType: "CLOUD_SERVER_STARTED", Port: 18034, GroupID: "task-events-cloud-server-started"},
	{Slug: "cloud_server_stopped", EventType: "CLOUD_SERVER_STOPPED", Port: 18035, GroupID: "task-events-cloud-server-stopped"},
	{Slug: "cloud_server_start_auto", EventType: "CLOUD_SERVER_START_AUTO", Port: 18036, GroupID: "task-events-cloud-server-start-auto"},
	{Slug: "cloud_platform_authorization_created", EventType: "CLOUD_PLATFORM_AUTHORIZATION_CREATED", Port: 18037, GroupID: "task-events-cloud-platform-authorization-created"},
	{Slug: "relay_lifecycle", EventType: "RELAY_LIFECYCLE", Port: 18039, GroupID: "task-events-relay-lifecycle"},
	{Slug: "task_status_changed", EventType: "TASK_STATUS_CHANGED", Port: 18043, GroupID: "task-events-task-status-changed"},
	{Slug: "workspace_machine_idle", EventType: "WORKSPACE_MACHINE_IDLE", Port: 18044, GroupID: "task-events-workspace-machine-idle-1-recycle-idle-nodes"},
	{Slug: "task_comment_image_mentioned", EventType: "TASK_COMMENT_IMAGE_MENTIONED", Port: 18045, GroupID: "task-events-task-comment-image-mentioned-1-start-vm-for-at-mention"},
	{Slug: "container_migrate_await_ready", EventType: "CONTAINER_MIGRATE_AWAIT_READY", Port: 18046, GroupID: "task-events-container-migrate-await-ready-1-await-running-heartbeat"},
	{Slug: "task_graceful_shutdown_await", EventType: "TASK_GRACEFUL_SHUTDOWN_AWAIT", Port: 18047, GroupID: "task-events-task-graceful-shutdown-await-1-hard-release-on-timeout"},
	{Slug: "wechat_identity_conflict", EventType: "WECHAT_IDENTITY_CONFLICT", Port: 18056, GroupID: "task-events-wechat-identity-conflict-1-audit-alert"},
	{Slug: "member_joined", EventType: "MEMBER_JOINED", Port: 18060, GroupID: "task-events-member-joined-1-create-default-git-identity"},
}

// EventBySlug returns registry entry or false.
func EventBySlug(slug string) (EventDefinition, bool) {
	for _, e := range AllEvents {
		if e.Slug == slug {
			return e, true
		}
	}
	return EventDefinition{}, false
}

// EventSlugsImplemented lists bin/ events with Go handlers ( grows per phase ).
func EventSlugsImplemented() []string {
	return []string{
		"billing_transaction_created",
		"sse_message",
		"email_sent",
		"invitation_created",
		"user_activated",
		"user_created",
		"company_created",
		"workspace_created",
		"project_updated",
		"task_completed",
		"ai_assistant_reply_completed",
		"cloud_server_started",
		"cloud_server_stopped",
		"cloud_server_start_auto",
		"cloud_platform_authorization_created",
		"relay_lifecycle",
		"task_status_changed",
		"workspace_machine_idle",
		"task_comment_image_mentioned",
		"container_migrate_await_ready",
		"task_graceful_shutdown_await",
		"wechat_identity_conflict",
		"member_joined",
	}
}
