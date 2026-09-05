#!/usr/bin/env bash
# 由 gitService/run.sh source：容器复用/漂移重建/bootstrap。
# 依赖调用方已定义 compose_file、assert_docker_still_running，并 export GITLAB_*。

cleanup_previous_stack() {
  compose_file down --remove-orphans >/dev/null 2>&1 || true
}

# 参数: $1 = 最大等待秒数 (默认 300)；0 = 就绪, 1 = 超时
wait_for_gitlab_ready() {
  local max_wait="${1:-300}"
  local waited=0
  local CONTAINER_NAME="${GITLAB_CONTAINER:-gitlab}"

  while (( waited < max_wait )); do
    if docker exec "$CONTAINER_NAME" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
      echo "GitLab 已就绪。"
      return 0
    fi
    sleep 5
    waited=$((waited + 5))
  done
  echo "GitLab 初始化超时（${max_wait}秒）" >&2
  return 1
}

# 确保 GitLab 容器可用：复用已有 → 启动已停止 → 首次创建。
# 设置 NEED_BOOTSTRAP=true 当容器是首次创建或重建时。
ensure_container() {
  local CONTAINER_NAME="${GITLAB_CONTAINER:-gitlab}"
  NEED_BOOTSTRAP=false

  gitlab_env_drift() {
    local env_blob=""
    env_blob="$(docker inspect "$CONTAINER_NAME" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null || true)"
    local current_external_host current_redirect current_external_url
    current_external_host="$(printf '%s\n' "$env_blob" | grep '^GITLAB_EXTERNAL_HOST=' | cut -d= -f2-)"
    current_redirect="$(printf '%s\n' "$env_blob" | grep '^GITLAB_OIDC_REDIRECT_URI=' | cut -d= -f2-)"
    current_external_url="$(printf '%s\n' "$env_blob" | grep '^GITLAB_EXTERNAL_URL=' | cut -d= -f2-)"
    if [[ -n "$current_external_host" && "$current_external_host" != "$GITLAB_EXTERNAL_HOST" ]]; then
      echo "external_host 配置变更: $current_external_host → $GITLAB_EXTERNAL_HOST"
      return 0
    fi
    if [[ -n "${GITLAB_EXTERNAL_URL:-}" && "$current_external_url" != "$GITLAB_EXTERNAL_URL" ]]; then
      echo "external_url 配置变更: ${current_external_url:-<空>} → $GITLAB_EXTERNAL_URL"
      return 0
    fi
    if [[ -n "${GITLAB_OIDC_REDIRECT_URI:-}" && "$current_redirect" != "$GITLAB_OIDC_REDIRECT_URI" ]]; then
      echo "OIDC redirect_uri 配置变更: ${current_redirect:-<空>} → $GITLAB_OIDC_REDIRECT_URI"
      return 0
    fi
    local current_signup current_pw_web current_pw_git
    current_signup="$(printf '%s\n' "$env_blob" | grep '^GITLAB_SIGNUP_ENABLED=' | cut -d= -f2-)"
    current_pw_web="$(printf '%s\n' "$env_blob" | grep '^GITLAB_PASSWORD_AUTH_WEB=' | cut -d= -f2-)"
    current_pw_git="$(printf '%s\n' "$env_blob" | grep '^GITLAB_PASSWORD_AUTH_GIT=' | cut -d= -f2-)"
    if [[ "${current_signup:-<unset>}" != "${GITLAB_SIGNUP_ENABLED:-false}" ]]; then
      echo "signup 开关变更: ${current_signup:-<unset>} → ${GITLAB_SIGNUP_ENABLED:-false}"
      return 0
    fi
    if [[ "${current_pw_web:-<unset>}" != "${GITLAB_PASSWORD_AUTH_WEB:-false}" ]]; then
      echo "passwordAuthWeb 变更: ${current_pw_web:-<unset>} → ${GITLAB_PASSWORD_AUTH_WEB:-false}"
      return 0
    fi
    if [[ "${current_pw_git:-<unset>}" != "${GITLAB_PASSWORD_AUTH_GIT:-false}" ]]; then
      echo "passwordAuthGit 变更: ${current_pw_git:-<unset>} → ${GITLAB_PASSWORD_AUTH_GIT:-false}"
      return 0
    fi
    local current_image want_image
    current_image="$(docker inspect "$CONTAINER_NAME" --format '{{.Config.Image}}' 2>/dev/null || true)"
    want_image="${GITLAB_IMAGE:-gitlab/gitlab-ce:19.2.4-ce.0}"
    if [[ -n "$current_image" && "$current_image" != "$want_image" ]]; then
      echo "image 变更: $current_image → $want_image"
      return 0
    fi
    return 1
  }

  if docker ps --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    local drift_msg=""
    if drift_msg="$(gitlab_env_drift)"; then
      echo "检测到 $drift_msg"
      echo "正在重建容器以应用新配置（数据卷保持不变）..."
      docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
      docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
      compose_file up -d || {
        echo "错误: docker compose up 失败，无法应用 GitLab 新配置" >&2
        exit 1
      }
      sleep 3
      assert_docker_still_running || exit 1
      compose_file ps
      NEED_BOOTSTRAP=true
      rm -f "${GITLAB_HOME:-$SCRIPT_DIR/gitlab_home}/bootstrap_marks/omniauth_oidc_synced" \
            "./.bootstrap_marks/omniauth_oidc_synced"
      return 0
    fi
    echo "GitLab 容器 $CONTAINER_NAME 已在运行，复用现有容器。"
    return 0
  fi

  if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    local drift_msg=""
    if drift_msg="$(gitlab_env_drift)"; then
      echo "检测到已停止容器的 $drift_msg"
      echo "正在重建容器以应用新配置（数据卷保持不变）..."
      docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
      compose_file up -d || {
        echo "错误: docker compose up 失败，无法应用 GitLab 新配置" >&2
        exit 1
      }
      sleep 3
      assert_docker_still_running || exit 1
      compose_file ps
      NEED_BOOTSTRAP=true
      rm -f "${GITLAB_HOME:-$SCRIPT_DIR/gitlab_home}/bootstrap_marks/omniauth_oidc_synced" \
            "./.bootstrap_marks/omniauth_oidc_synced"
      return 0
    fi
    echo "GitLab 容器 $CONTAINER_NAME 已存在但已停止，尝试启动..."
    if compose_file start gitlab; then
      echo "等待 GitLab 就绪..."
      if wait_for_gitlab_ready 300; then
        echo "容器复用成功。"
        return 0
      fi
      echo "容器启动成功但 GitLab 就绪超时，尝试重建..."
    else
      echo "容器启动失败，尝试重建..."
    fi
    cleanup_previous_stack
    compose_file up -d || {
      echo "错误: docker compose up 失败，无法重建 GitLab" >&2
      exit 1
    }
    sleep 3
    assert_docker_still_running || exit 1
    compose_file ps
    NEED_BOOTSTRAP=true
    return 0
  fi

  echo "首次创建 GitLab 容器..."
  compose_file up -d || {
    echo "错误: docker compose up 失败，无法创建 GitLab" >&2
    exit 1
  }
  sleep 3
  assert_docker_still_running || exit 1
  compose_file ps
  NEED_BOOTSTRAP=true
}

run_bootstrap_if_needed() {
  if [[ "${NEED_BOOTSTRAP:-false}" != "true" ]]; then
    echo "复用已有容器，跳过 bootstrap 脚本。"
    return 0
  fi

  echo "等待 GitLab 就绪后再跑 bootstrap（升级 migrations 可能需数分钟）…"
  wait_for_gitlab_ready "${GITLAB_UPGRADE_WAIT_SECONDS:-900}" || \
    echo "警告: GitLab 就绪超时，bootstrap 可能失败。" >&2

  local BOOTSTRAP_MARKS_DIR="${GITLAB_HOME:-$SCRIPT_DIR/gitlab_home}/bootstrap_marks"
  mkdir -p "$BOOTSTRAP_MARKS_DIR"
  mkdir -p "$SCRIPT_DIR/.bootstrap_marks"

  echo "同步本地 GitLab OAuth 应用 scope（与 port_config 对齐）…"
  "$SCRIPT_DIR/scripts/sync_local_oauth_app_scopes.sh" \
    && touch "$BOOTSTRAP_MARKS_DIR/oauth_scopes_synced" "$SCRIPT_DIR/.bootstrap_marks/oauth_scopes_synced" \
    || echo "提示: OAuth scope 同步未完成，可稍后手动执行 gitService/scripts/sync_local_oauth_app_scopes.sh" >&2

  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/omniauth_oidc_synced" ]]; then
    echo "同步 GitLab OmniAuth OIDC 配置…"
    "$SCRIPT_DIR/scripts/sync_omniauth_oidc.sh" --reconfigure \
      && touch "$BOOTSTRAP_MARKS_DIR/omniauth_oidc_synced" "$SCRIPT_DIR/.bootstrap_marks/omniauth_oidc_synced" \
      || echo "提示: OIDC 同步未完成，可稍后手动执行 gitService/scripts/sync_omniauth_oidc.sh --reconfigure" >&2
  else
    echo "OmniAuth OIDC 已同步，跳过。"
  fi

  echo "应用注册/账密策略（signup/passwordAuth，SSOT conf/infra/git-service/config.yaml）…"
  "$SCRIPT_DIR/scripts/apply_auth_policy.sh" \
    && touch "$BOOTSTRAP_MARKS_DIR/auth_policy_applied" "$SCRIPT_DIR/.bootstrap_marks/auth_policy_applied" \
    || echo "提示: 注册/账密策略未应用，可稍后手动执行 gitService/scripts/apply_auth_policy.sh" >&2

  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/oidc_ssl_fixed" ]]; then
    echo "应用 OIDC SSL 协议修复…"
    "$SCRIPT_DIR/scripts/fix_oidc_ssl.sh" \
      && touch "$BOOTSTRAP_MARKS_DIR/oidc_ssl_fixed" "$SCRIPT_DIR/.bootstrap_marks/oidc_ssl_fixed" \
      || echo "提示: OIDC SSL 修复未完成，可稍后手动执行 gitService/scripts/fix_oidc_ssl.sh" >&2
  else
    echo "OIDC SSL fix 已应用，跳过。"
  fi

  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/oidc_traceid_fixed" ]]; then
    echo "应用 OIDC TraceId 注入修复…"
    "$SCRIPT_DIR/scripts/fix_oidc_traceid.sh" \
      && touch "$BOOTSTRAP_MARKS_DIR/oidc_traceid_fixed" "$SCRIPT_DIR/.bootstrap_marks/oidc_traceid_fixed" \
      || echo "提示: OIDC TraceId 修复未完成，可稍后手动执行 gitService/scripts/fix_oidc_traceid.sh" >&2
  else
    echo "OIDC TraceId fix 已应用，跳过。"
  fi

  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/import_sources_enabled" ]]; then
    echo "启用 GitLab import sources（GitHub、GitLab 跨实例等）…"
    "$SCRIPT_DIR/scripts/enable_import_sources.sh" \
      && touch "$BOOTSTRAP_MARKS_DIR/import_sources_enabled" "$SCRIPT_DIR/.bootstrap_marks/import_sources_enabled" \
      || echo "提示: import_sources 配置未完成，可稍后手动执行 gitService/scripts/enable_import_sources.sh" >&2
  else
    echo "import_sources 已配置，跳过。"
  fi

  if [[ ! -f "$BOOTSTRAP_MARKS_DIR/public_projects_restricted" ]]; then
    echo "限制公开仓库创建（restricted_visibility_levels + default_project_visibility）…"
    "$SCRIPT_DIR/scripts/restrict_public_projects.sh" \
      && touch "$BOOTSTRAP_MARKS_DIR/public_projects_restricted" "$SCRIPT_DIR/.bootstrap_marks/public_projects_restricted" \
      || echo "提示: 公开仓库限制未完成，可稍后手动执行 gitService/scripts/restrict_public_projects.sh" >&2
  else
    echo "公开仓库限制已配置，跳过。"
  fi
}
