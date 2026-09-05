#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
mkdir -p logs

# v4: canonical event/intent paths (matches config.AllIntents order)
INTENT_PATHS=(
  billing_transaction_created/1_process_billing_transaction
  billing_transaction_created/2_mark_user_tenant
  sse_message/1_send_sse_message
  sse_message/2_persist_job_execution_event
  email_sent/1_send_email
  invitation_created/1_send_invitation_email
  user_activated/1_send_welcome_notification
  user_created/0_create_company
  user_created/1_send_welcome_email
  user_created/2_sync_user_profile
  company_created/1_set_default_deliverable_system
  company_created/2_set_default_progress_system
  company_created/3_create_default_workspace
  company_created/5_init_tenant_feature_params
  company_created/6_grant_initial_resources
  workspace_created/1_process_workspace_creation
  project_updated/1_process_project_update
  task_completed/1_process_task_completion
  ai_assistant_reply_completed/1_persist_assistant_reply
  cloud_server_started/1_process_server_start
  cloud_server_stopped/1_process_server_stop
  cloud_server_start_auto/1_process_server_start_auto
  cloud_platform_authorization_created/1_process_cloud_platform_authorization
  relay_lifecycle/1_append_token_audit
  relay_lifecycle/2_open_runtime_session
  relay_lifecycle/3_clear_reachability
  relay_lifecycle/4_relay_workflow_update
  task_status_changed/1_release_servers_on_terminal
  task_status_changed/2_fanout_work_panel_sse
  workspace_machine_idle/1_recycle_idle_nodes
  task_comment_image_mentioned/1_start_vm_for_at_mention
  container_migrate_await_ready/1_await_running_heartbeat
  task_graceful_shutdown_await/1_hard_release_on_timeout
  registration_invite/1_observability
  task_created/1_fanout_work_panel_sse
  task_created/2_create_task_post
  task_deleted/1_fanout_work_panel_sse
  wechat_identity_conflict/1_audit_alert
  task_post_renewed/1_notify
  task_post_expired/1_notify
  task_post_expiry_scan/1_expire_posts
  member_joined/1_create_default_git_identity
  billing_order_comment_created/1_notify_other_side
  billing_referral_settle_scan/1_settle_due
  referral_code_expiry_scan/1_expire
  billing_profit_sharing_scan/1_process_pending
  cloud_csc_reconcile/1_sweep
  queued_auto_run_scan/1_dispatch
  user_account_deletion_execute_scan/1_execute_due
  oidc_sso_idempotency_cleanup/1_cleanup
  wechat_mp_cleanup/1_cleanup
  email_invite_expiry_scan/1_expire
  task_revision_recorded/1_observe
  project_revision_recorded/1_observe
  project_deleted/1_detach_task_projects
  gitlab_manual_node_fulfillment_queued/1_ops_alert
)

# Ports are read from conf/events/domain-events/{event}/config.yaml (single source of truth).

# Legacy v3 event slugs → primary intent path
EVENTS=(
  billing_transaction_created sse_message email_sent invitation_created
  user_activated user_created company_created workspace_created
  project_updated task_completed ai_assistant_reply_completed
  cloud_server_started cloud_server_stopped cloud_server_start_auto
  cloud_platform_authorization_created relay_lifecycle
  task_status_changed workspace_machine_idle task_comment_image_mentioned
  container_migrate_await_ready task_graceful_shutdown_await
  registration_invite task_created task_deleted
  wechat_identity_conflict
  task_post_renewed task_post_expired task_post_expiry_scan
  member_joined billing_referral_settle_scan referral_code_expiry_scan billing_profit_sharing_scan
  cloud_csc_reconcile queued_auto_run_scan user_account_deletion_execute_scan
)

intent_binary_name() {
  local event="$1" intent="$2"
  python3 -c "
import sys
sys.path.insert(0, '.')
# use go registry via subprocess-free echo from known naming rule
event, intent = sys.argv[1], sys.argv[2]
parts = ['task-events'] + event.split('_') + intent.split('_')
print('-'.join(parts))
" "$event" "$intent" 2>/dev/null || {
    local e="${event//_/-}" i="${intent//_/-}"
    echo "task-events-${e}-${i}"
  }
}

# Prefer exact GroupID from built binary dir listing
resolve_binary_name() {
  local event="$1" intent="$2"
  local dir="bin/${event}/${intent}"
  if [[ -d "$dir" ]]; then
    local exe
    exe=$(find "$dir" -maxdepth 1 -type f -name 'task-events-*' 2>/dev/null | head -1)
    if [[ -n "$exe" ]]; then
      basename "$exe"
      return
    fi
  fi
  intent_binary_name "$event" "$intent"
}

find_monorepo_root() {
  local dir="$ROOT"
  for _ in $(seq 1 10); do
    # OPT-20260806-057: django 目录退役，锚点切到 conf/base.yaml
    if [[ -f "$dir/conf/base.yaml" ]]; then
      echo "$dir"
      return 0
    fi
    local parent
    parent="$(dirname "$dir")"
    if [[ "$parent" == "$dir" ]]; then
      break
    fi
    dir="$parent"
  done
  return 1
}

MONOREPO_ROOT="${MONOREPO_ROOT:-$(find_monorepo_root || echo "$ROOT/..")}"

port_bindable() {
  local port="$1"
  python3 -c "import socket; s=socket.socket(); s.bind(('127.0.0.1', int('$port'))); s.close()" 2>/dev/null
}

port_preflight() {
  local event="$1" intent="$2" port="$3"
  if port_bindable "$port"; then
    return 0
  fi
  echo "port ${port} not available for ${event}/${intent} (address in use)" >&2
  echo "hint: override in conf/events/domain-events/${event}/config.yaml" >&2
  return 1
}

port_for_path() {
  local target="$1"
  local event="${target%%/*}" intent="${target#*/}"
  local conf_file="$MONOREPO_ROOT/conf/events/domain-events/${event}/config.yaml"
  if [[ ! -f "$conf_file" ]]; then
    echo "missing config: $conf_file" >&2
    return 1
  fi
  local port
  port="$(python3 -c "
import yaml, sys
with open(sys.argv[1]) as f:
    cfg = yaml.safe_load(f)
block = cfg.get('intents', {}).get(sys.argv[2], {})
port = block.get('port', '')
if not port:
    sys.exit(1)
print(port)
" "$conf_file" "$intent" 2>/dev/null)" || {
    echo "intent '${intent}' has no port (missing 'intents.${intent}.port') in ${conf_file}" >&2
    return 1
  }
  echo "$port"
}

parse_path() {
  local path="$1"
  if [[ "$path" != */* ]]; then
    # legacy event slug → primary intent
    local e="$path"
    for p in "${INTENT_PATHS[@]}"; do
      if [[ "$p" == "$e/"* ]]; then
        echo "$p"
        return 0
      fi
    done
    echo "unknown event slug: $e" >&2
    return 1
  fi
  echo "$path"
}

build_intent() {
  local path
  path="$(parse_path "$1")"
  local event="${path%%/*}" intent="${path#*/}"
  mkdir -p "bin/${event}/${intent}"
  local out="bin/${event}/${intent}/$(intent_binary_name "$event" "$intent")"
  local tmp="${out}.new.$$"
  if ! go build -tags otel_enabled -o "$tmp" "./cmd/${event}/${intent}"; then
    rm -f "$tmp"
    return 1
  fi
  mv -f "$tmp" "$out"
  echo "built $out"
}

build_event() {
  local slug="$1"
  local built=0
  for p in "${INTENT_PATHS[@]}"; do
    if [[ "$p" == "$slug/"* ]]; then
      build_intent "$p"
      built=1
    fi
  done
  if [[ "$built" -eq 0 ]]; then
    echo "unknown event: $slug" >&2
    exit 1
  fi
}

build_all() {
  for p in "${INTENT_PATHS[@]}"; do
    build_intent "$p"
  done
}

start_intent() {
  local path
  path="$(parse_path "$1")"
  local event="${path%%/*}" intent="${path#*/}"
  local port
  port="$(port_for_path "$path")" || {
    echo "cannot start ${path}: port_for_path failed (see above)" >&2
    exit 1
  }
  # ADR-0058: runAll canary overlap starts a SO_REUSEPORT peer beside the old
  # listener. Exclusive bind preflight and pidfile short-circuit would abort
  # the peer and leave runAll Retrying on the old health URL.
  if [[ "${RUNALL_CANARY_OVERLAP:-}" == "1" ]]; then
    echo "canary overlap: skip exclusive port preflight and pidfile for ${event}/${intent}" >&2
  else
    port_preflight "$event" "$intent" "$port" || return 1
  fi
  # ADR-0027: start execs last-good only. Compile via `run.sh build` or runAll 精准编译重启.
  local bin="bin/${event}/${intent}/$(resolve_binary_name "$event" "$intent")"
  if [[ ! -x "$bin" ]]; then
    echo "cannot start ${path}: missing binary ${bin} (run: bash run.sh build ${path})" >&2
    return 1
  fi
  local key="${event}__${intent}"
  local pidfile="logs/${key}.pid"
  local logfile="logs/${key}.log"
  # When runAll sets RUNALL_LOG_ROOT, write there so Promtail → Loki/Grafana can scrape
  # the same path as other platform services. Keep local logs/ as a symlink for operators.
  local runall_log=""
  if [[ -n "${RUNALL_LOG_ROOT:-}" ]]; then
    local bname
    bname="$(resolve_binary_name "$event" "$intent")"
    mkdir -p "$RUNALL_LOG_ROOT"
    runall_log="${RUNALL_LOG_ROOT}/${bname}.log"
  fi
  if [[ "${RUNALL_CANARY_OVERLAP:-}" != "1" ]] && [[ -f "$pidfile" ]] && ps -p "$(cat "$pidfile")" >/dev/null 2>&1; then
    echo "$(basename "$bin") already running pid=$(cat "$pidfile")"
    return
  fi
  if [[ -n "$runall_log" ]]; then
    touch "$runall_log"
    ln -sfn "$runall_log" "$logfile"
    "$bin" >> "$runall_log" 2>&1 &
  else
    "$bin" >> "$logfile" 2>&1 &
  fi
  echo $! > "$pidfile"
  echo "$(basename "$bin") started pid=$(cat "$pidfile")"
}

start_event() {
  local slug="$1"
  for p in "${INTENT_PATHS[@]}"; do
    if [[ "$p" == "$slug/"* ]]; then
      start_intent "$p"
    fi
  done
}

start_events() {
  for e in "${EVENTS[@]}"; do
    start_event "$e"
  done
}

start_all_intents() {
  for p in "${INTENT_PATHS[@]}"; do
    start_intent "$p"
  done
}

stop_intent() {
  local path
  path="$(parse_path "$1")"
  local event="${path%%/*}" intent="${path#*/}"
  local key="${event}__${intent}"
  local pidfile="logs/${key}.pid"
  local bname
  bname="$(resolve_binary_name "$event" "$intent")"
  if [[ -f "$pidfile" ]]; then
    kill "$(cat "$pidfile")" 2>/dev/null || true
    rm -f "$pidfile"
  fi
  # Prefer path-qualified match so `pkill -f $bname` cannot kill the parent shell
  # when OTEL_SERVICE_NAME / argv accidentally contains the same token.
  pgrep -f "bin/.*/${bname}( |$)" 2>/dev/null | xargs -r kill 2>/dev/null || true
}

stop_event() {
  local slug="$1"
  for p in "${INTENT_PATHS[@]}"; do
    if [[ "$p" == "$slug/"* ]]; then
      stop_intent "$p"
    fi
  done
}

stop_events() {
  for e in "${EVENTS[@]}"; do stop_event "$e"; done
}

stop_all_intents() {
  for p in "${INTENT_PATHS[@]}"; do stop_intent "$p"; done
}

status_intent() {
  local path
  path="$(parse_path "$1")"
  local event="${path%%/*}" intent="${path#*/}"
  local key="${event}__${intent}"
  local pidfile="logs/${key}.pid"
  local label
  label="$(resolve_binary_name "$event" "$intent")"
  if [[ -f "$pidfile" ]] && ps -p "$(cat "$pidfile")" >/dev/null 2>&1; then
    echo "${label} running pid=$(cat "$pidfile")"
  else
    echo "${label} stopped"
  fi
}

status_event() {
  local slug="$1"
  for p in "${INTENT_PATHS[@]}"; do
    if [[ "$p" == "$slug/"* ]]; then
      status_intent "$p"
    fi
  done
}

is_event() {
  local target="$1"
  for e in "${EVENTS[@]}"; do
    [[ "$e" == "$target" ]] && return 0
  done
  return 1
}

is_intent_path() {
  local target="$1"
  for p in "${INTENT_PATHS[@]}"; do
    [[ "$p" == "$target" ]] && return 0
  done
  return 1
}

cmd="${1:-}"
target="${2:-all}"

case "$cmd" in
  build)
    if [[ "$target" == "all" ]]; then
      build_all
    elif is_intent_path "$target"; then
      build_intent "$target"
    elif is_event "$target"; then
      build_event "$target"
    else
      echo "unknown target: $target" >&2; exit 1
    fi
    ;;
  start)
    if [[ "$target" == "all" ]]; then
      start_all_intents
    elif [[ "$target" == "events" ]]; then
      start_events
    elif is_intent_path "$target"; then
      start_intent "$target"
    elif is_event "$target"; then
      start_event "$target"
    else
      echo "unknown target: $target" >&2; exit 1
    fi
    ;;
  stop)
    if [[ "$target" == "all" ]]; then
      stop_all_intents
    elif [[ "$target" == "events" ]]; then
      stop_events
    elif is_intent_path "$target"; then
      stop_intent "$target"
    elif is_event "$target"; then
      stop_event "$target"
    else
      echo "unknown target: $target" >&2; exit 1
    fi
    ;;
  status)
    if [[ "$target" == "all" ]]; then
      for p in "${INTENT_PATHS[@]}"; do status_intent "$p"; done
    elif is_intent_path "$target"; then
      status_intent "$target"
    elif is_event "$target"; then
      status_event "$target"
    else
      echo "unknown target: $target" >&2; exit 1
    fi
    ;;
  integration-test)
    go test -tags=integration ./integration/...
    ;;
  prune-legacy-binaries)
    # v3 遗留：bin/{event}/task-events-* 与 bin/task-events-* 根目录副本（v4 在 bin/{event}/{intent}/）
    count=0
    while IFS= read -r f; do
      rm -f "$f"
      count=$((count + 1))
    done < <(find bin -maxdepth 2 -type f -name 'task-events-*' 2>/dev/null)
    echo "pruned $count legacy binary file(s) under bin/"
    ;;
  *)
    echo "usage: $0 {build|start|stop|status|integration-test|prune-legacy-binaries} [event/intent|event_slug|events|all]" >&2
    echo "intents (${#INTENT_PATHS[@]}):" >&2
    printf '  %s\n' "${INTENT_PATHS[@]}" >&2
    exit 1
    ;;
esac
