package config

import "strings"

// Intent consumer health ports use block 18020–18049 (see port-conflict-resolution design).
const (
	IntentPortBase = 18020
)

// IntentDefinition describes one per-intent consumer binary (v4).
type IntentDefinition struct {
	EventSlug      string
	IntentSlug     string
	EventType      string
	Port           int
	GroupID        string
	DefaultEnabled bool
}

// AllIntents is the canonical v4 registry (port_config / bin overlay may override).
var AllIntents = []IntentDefinition{
	{EventSlug: "billing_transaction_created", IntentSlug: "1_process_billing_transaction", EventType: "BILLING_TRANSACTION_CREATED", Port: 18020, GroupID: "task-events-billing-transaction-created-1-process-billing-transaction", DefaultEnabled: true},
	{EventSlug: "billing_transaction_created", IntentSlug: "2_mark_user_tenant", EventType: "BILLING_TRANSACTION_CREATED", Port: 18053, GroupID: "task-events-billing-transaction-created-2-mark-user-tenant", DefaultEnabled: true},
	{EventSlug: "sse_message", IntentSlug: "1_send_sse_message", EventType: "SSE_MESSAGE", Port: 18021, GroupID: "task-events-sse-message-1-send-sse-message", DefaultEnabled: true},
	{EventSlug: "sse_message", IntentSlug: "2_persist_job_execution_event", EventType: "SSE_MESSAGE", Port: 18062, GroupID: "task-events-sse-message-2-persist-job-execution-event", DefaultEnabled: true},
	{EventSlug: "email_sent", IntentSlug: "1_send_email", EventType: "EMAIL_SENT", Port: 18022, GroupID: "task-events-email-sent-1-send-email", DefaultEnabled: false},
	{EventSlug: "invitation_created", IntentSlug: "1_send_invitation_email", EventType: "INVITATION_CREATED", Port: 18023, GroupID: "task-events-invitation-created-1-send-invitation-email", DefaultEnabled: false},
	{EventSlug: "user_activated", IntentSlug: "1_send_welcome_notification", EventType: "USER_ACTIVATED", Port: 18024, GroupID: "task-events-user-activated-1-send-welcome-notification", DefaultEnabled: false},
	{EventSlug: "user_created", IntentSlug: "0_create_company", EventType: "USER_CREATED", Port: 18025, GroupID: "task-events-user-created-0-create-company", DefaultEnabled: true},
	{EventSlug: "user_created", IntentSlug: "1_send_welcome_email", EventType: "USER_CREATED", Port: 18026, GroupID: "task-events-user-created-1-send-welcome-email", DefaultEnabled: false},
	{EventSlug: "user_created", IntentSlug: "2_sync_user_profile", EventType: "USER_CREATED", Port: 18038, GroupID: "task-events-user-created-2-sync-user-profile", DefaultEnabled: true},
	{EventSlug: "company_created", IntentSlug: "1_set_default_deliverable_system", EventType: "COMPANY_CREATED", Port: 18027, GroupID: "task-events-company-created-1-set-default-deliverable-system", DefaultEnabled: true},
	{EventSlug: "company_created", IntentSlug: "2_set_default_progress_system", EventType: "COMPANY_CREATED", Port: 18028, GroupID: "task-events-company-created-2-set-default-progress-system", DefaultEnabled: true},
	{EventSlug: "company_created", IntentSlug: "3_create_default_workspace", EventType: "COMPANY_CREATED", Port: 18029, GroupID: "task-events-company-created-3-create-default-workspace", DefaultEnabled: true},
	{EventSlug: "company_created", IntentSlug: "5_init_tenant_feature_params", EventType: "COMPANY_CREATED", Port: 18054, GroupID: "task-events-company-created-5-init-tenant-feature-params", DefaultEnabled: true},
	{EventSlug: "company_created", IntentSlug: "6_grant_initial_resources", EventType: "COMPANY_CREATED", Port: 18055, GroupID: "task-events-company-created-6-grant-initial-resources", DefaultEnabled: true},
	{EventSlug: "workspace_created", IntentSlug: "1_process_workspace_creation", EventType: "WORKSPACE_CREATED", Port: 18030, GroupID: "task-events-workspace-created-1-process-workspace-creation", DefaultEnabled: true},
	{EventSlug: "project_updated", IntentSlug: "1_process_project_update", EventType: "PROJECT_UPDATED", Port: 18031, GroupID: "task-events-project-updated-1-process-project-update", DefaultEnabled: true},
	{EventSlug: "task_completed", IntentSlug: "1_process_task_completion", EventType: "TASK_COMPLETED", Port: 18032, GroupID: "task-events-task-completed-1-process-task-completion", DefaultEnabled: true},
	{EventSlug: "ai_assistant_reply_completed", IntentSlug: "1_persist_assistant_reply", EventType: "AI_ASSISTANT_REPLY_COMPLETED", Port: 18033, GroupID: "task-events-ai-assistant-reply-completed-1-persist-assistant-reply", DefaultEnabled: true},
	{EventSlug: "cloud_server_started", IntentSlug: "1_process_server_start", EventType: "CLOUD_SERVER_STARTED", Port: 18034, GroupID: "task-events-cloud-server-started-1-process-server-start", DefaultEnabled: true},
	{EventSlug: "cloud_server_stopped", IntentSlug: "1_process_server_stop", EventType: "CLOUD_SERVER_STOPPED", Port: 18035, GroupID: "task-events-cloud-server-stopped-1-process-server-stop", DefaultEnabled: true},
	{EventSlug: "cloud_server_start_auto", IntentSlug: "1_process_server_start_auto", EventType: "CLOUD_SERVER_START_AUTO", Port: 18036, GroupID: "task-events-cloud-server-start-auto-1-process-server-start-auto", DefaultEnabled: true},
	{EventSlug: "cloud_platform_authorization_created", IntentSlug: "1_process_cloud_platform_authorization", EventType: "CLOUD_PLATFORM_AUTHORIZATION_CREATED", Port: 18037, GroupID: "task-events-cloud-platform-authorization-created-1-process-cloud-platform-authorization", DefaultEnabled: true},
	{EventSlug: "relay_lifecycle", IntentSlug: "1_append_token_audit", EventType: "RELAY_LIFECYCLE", Port: 18039, GroupID: "task-events-relay-lifecycle-1-append-token-audit", DefaultEnabled: true},
	{EventSlug: "relay_lifecycle", IntentSlug: "2_open_runtime_session", EventType: "RELAY_LIFECYCLE", Port: 18040, GroupID: "task-events-relay-lifecycle-2-open-runtime-session", DefaultEnabled: true},
	{EventSlug: "relay_lifecycle", IntentSlug: "3_clear_reachability", EventType: "RELAY_LIFECYCLE", Port: 18041, GroupID: "task-events-relay-lifecycle-3-clear-reachability", DefaultEnabled: true},
	{EventSlug: "relay_lifecycle", IntentSlug: "4_relay_workflow_update", EventType: "RELAY_LIFECYCLE", Port: 18042, GroupID: "task-events-relay-lifecycle-4-relay-workflow-update", DefaultEnabled: true},
	{EventSlug: "task_status_changed", IntentSlug: "1_release_servers_on_terminal", EventType: "TASK_STATUS_CHANGED", Port: 18043, GroupID: "task-events-task-status-changed-1-release-servers-on-terminal", DefaultEnabled: true},
	{EventSlug: "task_status_changed", IntentSlug: "2_fanout_work_panel_sse", EventType: "TASK_STATUS_CHANGED", Port: 18048, GroupID: "task-events-task-status-changed-2-fanout-work-panel-sse", DefaultEnabled: true},
	{EventSlug: "workspace_machine_idle", IntentSlug: "1_recycle_idle_nodes", EventType: "WORKSPACE_MACHINE_IDLE", Port: 18044, GroupID: "task-events-workspace-machine-idle-1-recycle-idle-nodes", DefaultEnabled: true},
	{EventSlug: "task_comment_image_mentioned", IntentSlug: "1_start_vm_for_at_mention", EventType: "TASK_COMMENT_IMAGE_MENTIONED", Port: 18045, GroupID: "task-events-task-comment-image-mentioned-1-start-vm-for-at-mention", DefaultEnabled: true},
	{EventSlug: "container_migrate_await_ready", IntentSlug: "1_await_running_heartbeat", EventType: "CONTAINER_MIGRATE_AWAIT_READY", Port: 18046, GroupID: "task-events-container-migrate-await-ready-1-await-running-heartbeat", DefaultEnabled: true},
	{EventSlug: "task_graceful_shutdown_await", IntentSlug: "1_hard_release_on_timeout", EventType: "TASK_GRACEFUL_SHUTDOWN_AWAIT", Port: 18047, GroupID: "task-events-task-graceful-shutdown-await-1-hard-release-on-timeout", DefaultEnabled: true},
	{EventSlug: "registration_invite", IntentSlug: "1_observability", EventType: "REGISTRATION_INVITE_CODE_ISSUED", Port: 18049, GroupID: "task-events-registration-invite-1-observability", DefaultEnabled: true},
	{EventSlug: "task_created", IntentSlug: "1_fanout_work_panel_sse", EventType: "TASK_CREATED", Port: 18050, GroupID: "task-events-task-created-1-fanout-work-panel-sse", DefaultEnabled: true},
	{EventSlug: "task_created", IntentSlug: "2_create_task_post", EventType: "TASK_CREATED", Port: 18052, GroupID: "task-events-task-created-2-create-task-post", DefaultEnabled: true},
	{EventSlug: "task_deleted", IntentSlug: "1_fanout_work_panel_sse", EventType: "TASK_DELETED", Port: 18051, GroupID: "task-events-task-deleted-1-fanout-work-panel-sse", DefaultEnabled: true},
	{EventSlug: "wechat_identity_conflict", IntentSlug: "1_audit_alert", EventType: "WECHAT_IDENTITY_CONFLICT", Port: 18056, GroupID: "task-events-wechat-identity-conflict-1-audit-alert", DefaultEnabled: true},
	// v15 任务帖存续期：续存成功 / 到期下架通知 + 每日到期扫描（timer runner，无 Kafka 消费者）
	{EventSlug: "task_post_renewed", IntentSlug: "1_notify", EventType: "TASK_POST_RENEWED", Port: 18057, GroupID: "task-events-task-post-renewed-1-notify", DefaultEnabled: true},
	{EventSlug: "task_post_expired", IntentSlug: "1_notify", EventType: "TASK_POST_EXPIRED", Port: 18058, GroupID: "task-events-task-post-expired-1-notify", DefaultEnabled: true},
	{EventSlug: "task_post_expiry_scan", IntentSlug: "1_expire_posts", EventType: "TASK_POST_EXPIRY_SCAN", Port: 18059, GroupID: "task-events-task-post-expiry-scan-1-expire-posts", DefaultEnabled: true},
	{EventSlug: "member_joined", IntentSlug: "1_create_default_git_identity", EventType: "MEMBER_JOINED", Port: 18060, GroupID: "task-events-member-joined-1-create-default-git-identity", DefaultEnabled: true},
	// OPT-20260811-084：订单评论产生后通知对侧（一期 SSE 订单级通道）
	{EventSlug: "billing_order_comment_created", IntentSlug: "1_notify_other_side", EventType: "BILLING_ORDER_COMMENT_CREATED", Port: 18061, GroupID: "task-events-billing-order-comment-created-1-notify-other-side", DefaultEnabled: true},
	// OPT-20260816-026：推荐结算迁出 taskBill 进程内 ticker → timer worker（无 Kafka，POST taskBill settle-due）
	{EventSlug: "billing_referral_settle_scan", IntentSlug: "1_settle_due", EventType: "BILLING_REFERRAL_SETTLE_SCAN", Port: 18067, GroupID: "task-events-billing-referral-settle-scan-1-settle-due", DefaultEnabled: true},
	// OPT-20260816-031：推荐码过期回写迁出 taskReferral 进程内 ticker → timer worker（POST taskReferral expire-codes）
	{EventSlug: "referral_code_expiry_scan", IntentSlug: "1_expire", EventType: "REFERRAL_CODE_EXPIRY_SCAN", Port: 18066, GroupID: "task-events-referral-code-expiry-scan-1-expire", DefaultEnabled: true},
	// OPT-20260816-029：微信分账迁出 taskBill 进程内 daemon → timer worker（POST taskBill process-pending）
	{EventSlug: "billing_profit_sharing_scan", IntentSlug: "1_process_pending", EventType: "BILLING_PROFIT_SHARING_SCAN", Port: 18064, GroupID: "task-events-billing-profit-sharing-scan-1-process-pending", DefaultEnabled: true},
	// OPT-20260816-028：孤儿+泄漏 CSC 对账收敛到单个 sweep timer（先 orphan 打标，再 leaked 清资源；无 Kafka）
	{EventSlug: "cloud_csc_reconcile", IntentSlug: "1_sweep", EventType: "CLOUD_CSC_RECONCILE", Port: 18063, GroupID: "task-events-cloud-csc-reconcile-1-sweep", DefaultEnabled: true},
	// OPT-20260816-030：排队自动开跑迁出 taskTaskService 进程内 30s ticker → timer worker（POST dispatch-once）
	{EventSlug: "queued_auto_run_scan", IntentSlug: "1_dispatch", EventType: "QUEUED_AUTO_RUN_SCAN", Port: 18065, GroupID: "task-events-queued-auto-run-scan-1-dispatch", DefaultEnabled: true},
	{EventSlug: "user_account_deletion_execute_scan", IntentSlug: "1_execute_due", EventType: "USER_ACCOUNT_DELETION_EXECUTE_SCAN", Port: 18068, GroupID: "task-events-user-account-deletion-execute-scan-1-execute-due", DefaultEnabled: true},
	// OPT-20260825-030：auth_oidc_sso_idempotency 过期清理 timer（POST taskAuth cleanup；无 Kafka）
	{EventSlug: "oidc_sso_idempotency_cleanup", IntentSlug: "1_cleanup", EventType: "OIDC_SSO_IDEMPOTENCY_CLEANUP", Port: 18069, GroupID: "task-events-oidc-sso-idempotency-cleanup-1-cleanup", DefaultEnabled: true},
	// OPT-20260826-003：微信服务号挂起行/过期票据行清理 timer（POST taskAuth wechat-mp-cleanup；无 Kafka）
	{EventSlug: "wechat_mp_cleanup", IntentSlug: "1_cleanup", EventType: "WECHAT_MP_CLEANUP", Port: 18070, GroupID: "task-events-wechat-mp-cleanup-1-cleanup", DefaultEnabled: true},
	// OPT-20260829-003：超管邮箱邀请过期清理 timer（POST taskAuth email-invites/expire-due；无 Kafka）
	{EventSlug: "email_invite_expiry_scan", IntentSlug: "1_expire", EventType: "EMAIL_INVITE_EXPIRY_SCAN", Port: 18071, GroupID: "task-events-email-invite-expiry-scan-1-expire", DefaultEnabled: true},
	{EventSlug: "task_revision_recorded", IntentSlug: "1_observe", EventType: "TASK_REVISION_RECORDED", Port: 18072, GroupID: "task-events-task-revision-recorded-1-observe", DefaultEnabled: true},
	{EventSlug: "project_revision_recorded", IntentSlug: "1_observe", EventType: "PROJECT_REVISION_RECORDED", Port: 18073, GroupID: "task-events-project-revision-recorded-1-observe", DefaultEnabled: true},
	{EventSlug: "project_deleted", IntentSlug: "1_detach_task_projects", EventType: "PROJECT_DELETED", Port: 18074, GroupID: "task-events-project-deleted-1-detach-task-projects", DefaultEnabled: true},
	// OPT-20260902-014：GitLab 手动建节点待处理 → ops 邮件告警（只读，幂等键=order_id）
	{EventSlug: "gitlab_manual_node_fulfillment_queued", IntentSlug: "1_ops_alert", EventType: "GITLAB_MANUAL_NODE_FULFILLMENT_QUEUED", Port: 18075, GroupID: "task-events-gitlab-manual-node-fulfillment-queued-1-ops-alert", DefaultEnabled: true},
}

// IntentByPath returns registry entry for event/intent path.
func IntentByPath(eventSlug, intentSlug string) (IntentDefinition, bool) {
	for _, d := range AllIntents {
		if d.EventSlug == eventSlug && d.IntentSlug == intentSlug {
			return d, true
		}
	}
	return IntentDefinition{}, false
}

// IntentsForEvent lists all intents under an event.
func IntentsForEvent(eventSlug string) []IntentDefinition {
	var out []IntentDefinition
	for _, d := range AllIntents {
		if d.EventSlug == eventSlug {
			out = append(out, d)
		}
	}
	return out
}

// BinaryName returns executable file name for an intent.
func BinaryName(eventSlug, intentSlug string) string {
	def, ok := IntentByPath(eventSlug, intentSlug)
	if !ok {
		return ""
	}
	return def.GroupID
}

// IntentPathKey is event/intent for run.sh.
func IntentPathKey(eventSlug, intentSlug string) string {
	return eventSlug + "/" + intentSlug
}

// ParseIntentPath splits "event/intent".
func ParseIntentPath(path string) (eventSlug, intentSlug string, ok bool) {
	i := strings.Index(path, "/")
	if i <= 0 || i >= len(path)-1 {
		return "", "", false
	}
	return path[:i], path[i+1:], true
}

// AllIntentPaths returns every event/intent path in registry order.
func AllIntentPaths() []string {
	out := make([]string, 0, len(AllIntents))
	for _, d := range AllIntents {
		out = append(out, IntentPathKey(d.EventSlug, d.IntentSlug))
	}
	return out
}

// PrimaryIntentForEvent returns the first intent when v3 slug equals event slug.
func PrimaryIntentForEvent(eventSlug string) (IntentDefinition, bool) {
	intents := IntentsForEvent(eventSlug)
	if len(intents) == 0 {
		return IntentDefinition{}, false
	}
	if len(intents) == 1 {
		return intents[0], true
	}
	for _, d := range intents {
		if d.DefaultEnabled {
			return d, true
		}
	}
	return intents[0], true
}

// AllIntentPorts returns health check ports for enabled intents in registry order.
func AllIntentPorts() []int {
	out := make([]int, 0, len(AllIntents))
	for _, d := range AllIntents {
		out = append(out, d.Port)
	}
	return out
}
