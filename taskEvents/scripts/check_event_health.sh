#!/usr/bin/env bash
# G4.5 v4: curl health for all intent binaries (ports 18020-18049).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

FAIL=0
check() {
  local port="$1" name="$2"
  local url="http://127.0.0.1:${port}/api/health/"
  local attempt
  for attempt in 1 2 3 4 5; do
    if curl -sf --max-time 2 "$url" >/dev/null; then
      echo "OK  $name ($url)"
      return 0
    fi
    sleep 1
  done
  echo "FAIL $name ($url)"
  FAIL=1
}

# port event/intent (matches config/intent_registry.go AllIntents)
INTENTS=(
  "18020 billing_transaction_created/1_process_billing_transaction"
  "18021 sse_message/1_send_sse_message"
  "18022 email_sent/1_send_email"
  "18023 invitation_created/1_send_invitation_email"
  "18024 user_activated/1_send_welcome_notification"
  "18025 user_created/0_create_company"
  "18026 user_created/1_send_welcome_email"
  "18038 user_created/2_sync_user_profile"
  "18027 company_created/1_set_default_deliverable_system"
  "18028 company_created/2_set_default_progress_system"
  "18029 company_created/3_create_default_workspace"
  "18030 workspace_created/1_process_workspace_creation"
  "18031 project_updated/1_process_project_update"
  "18032 task_completed/1_process_task_completion"
  "18033 ai_assistant_reply_completed/1_persist_assistant_reply"
  "18034 cloud_server_started/1_process_server_start"
  "18035 cloud_server_stopped/1_process_server_stop"
  "18036 cloud_server_start_auto/1_process_server_start_auto"
  "18037 cloud_platform_authorization_created/1_process_cloud_platform_authorization"
  "18039 relay_lifecycle/1_append_token_audit"
  "18040 relay_lifecycle/2_open_runtime_session"
  "18041 relay_lifecycle/3_clear_reachability"
  "18042 relay_lifecycle/4_relay_workflow_update"
  "18043 task_status_changed/1_release_servers_on_terminal"
  "18048 task_status_changed/2_fanout_work_panel_sse"
  "18044 workspace_machine_idle/1_recycle_idle_nodes"
  "18045 task_comment_image_mentioned/1_start_vm_for_at_mention"
  "18046 container_migrate_await_ready/1_await_running_heartbeat"
  "18047 task_graceful_shutdown_await/1_hard_release_on_timeout"
  "18049 registration_invite/1_observability"
)

for row in "${INTENTS[@]}"; do
  port="${row%% *}"
  path="${row#* }"
  check "$port" "$path"
done

exit "$FAIL"
