#!/usr/bin/env bash
# 解析 conf/infra/<area>/config.yaml（ADR-0052：CONF_ROOT / DEPLOY_ROOT 优先）。
# 用法：source 本文件后 resolve_infra_conf <area>
# shellcheck disable=SC2034
resolve_infra_conf() {
  local area="${1:?resolve_infra_conf: area required}"
  local rel="infra/${area}/config.yaml"
  local here workspace_root
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  workspace_root="$(cd "$here/../.." && pwd)"
  if [[ -n "${CONF_ROOT:-}" ]]; then
    if [[ -f "${CONF_ROOT}/${rel}" ]]; then
      printf '%s\n' "${CONF_ROOT}/${rel}"
      return 0
    fi
    if [[ -f "${CONF_ROOT}/conf/${rel}" ]]; then
      printf '%s\n' "${CONF_ROOT}/conf/${rel}"
      return 0
    fi
  fi
  if [[ -n "${DEPLOY_ROOT:-}" && -f "${DEPLOY_ROOT}/conf/${rel}" ]]; then
    printf '%s\n' "${DEPLOY_ROOT}/conf/${rel}"
    return 0
  fi
  printf '%s\n' "${workspace_root}/conf/${rel}"
}
