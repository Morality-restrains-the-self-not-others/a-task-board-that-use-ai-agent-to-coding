#!/usr/bin/env bash
# 本机 git-service 启动闸门（ADR-0047）。
# 上海等区域实例 GITSERVICE_CONF_APP != git-service，不拦截。

local_gitlab_start_allowed() {
  local conf_app="${1:-}"
  local enabled="${2:-false}"
  case "$enabled" in
    true|TRUE|True|1|yes|on) enabled=true ;;
    *) enabled=false ;;
  esac
  if [[ "$conf_app" != "git-service" ]]; then
    return 0
  fi
  [[ "$enabled" == "true" ]]
}

print_local_gitlab_start_refused() {
  cat <<'EOF'
错误: 本机 GitLab（git-service）默认禁止启动，避免 Puma/Sidekiq 空转占满 CPU。
请先改 conf：在 conf-local/infra/git-service/config.yaml 增加

  runAllStartEnabled: true

保存后再从 runAll 面板（:9999）启动 git-service。
SSOT: conf/infra/git-service/config.yaml（仓库默认 false）。上海实例不受此键约束。
EOF
}
