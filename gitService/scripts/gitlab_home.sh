#!/usr/bin/env bash
# GitLab durable home helpers for gitService.
# Sourced by run.sh; also unit-tested via test_gitlab_home.sh.
# shellcheck shell=bash

gitlab_home_default_path() {
  local xdg="${XDG_DATA_HOME:-}"
  if [[ -n "$xdg" ]]; then
    printf '%s\n' "$xdg/daydaymoney/gitService"
  else
    printf '%s\n' "${HOME}/.local/share/daydaymoney/gitService"
  fi
}

# Resolve order: GITLAB_HOME env > conf gitlabHome > default.
# Args: optional conf gitlabHome value (may be empty).
resolve_gitlab_home() {
  local conf_home="${1:-}"
  if [[ -n "${GITLAB_HOME:-}" ]]; then
    printf '%s\n' "$GITLAB_HOME"
    return 0
  fi
  if [[ -n "$conf_home" ]]; then
    printf '%s\n' "$conf_home"
    return 0
  fi
  gitlab_home_default_path
}

gitlab_home_fstype() {
  local path="$1"
  if command -v findmnt >/dev/null 2>&1; then
    # -T: filesystem containing path (path need not exist as mountpoint)
    findmnt -T "$path" -o FSTYPE -n 2>/dev/null || true
    return 0
  fi
  df -T "$path" 2>/dev/null | awk 'NR==2 {print $2}' || true
}

# Returns 0 if durable (or allow override), 1 if volatile and blocked.
assert_gitlab_home_durable() {
  local path="$1"
  local fstype
  fstype="$(gitlab_home_fstype "$path")"
  case "$fstype" in
    tmpfs|ramfs)
      if [[ "${GITLAB_HOME_ALLOW_TMPFS:-}" == "1" ]]; then
        echo "警告: GITLAB_HOME=$path 位于 $fstype，已设置 GITLAB_HOME_ALLOW_TMPFS=1，继续。" >&2
        return 0
      fi
      echo "错误: GITLAB_HOME=$path 位于易失文件系统 ($fstype)。" >&2
      echo "容器/工作区重建或主机重启会导致 GitLab 数据丢失。" >&2
      echo "请将 GITLAB_HOME 设到磁盘路径（默认: $(gitlab_home_default_path)），" >&2
      echo "或仅在明确接受丢数风险时设置 GITLAB_HOME_ALLOW_TMPFS=1。" >&2
      return 1
      ;;
  esac
  return 0
}

# True (0) if directory looks like a populated GitLab data root.
gitlab_home_has_substance() {
  local root="$1"
  [[ -d "$root/data/postgresql/data" ]] || [[ -f "$root/data/bootstrapped" ]] || [[ -d "$root/config/gitlab-rails" ]]
}

# True (0) when legacy should be migrated into target.
legacy_needs_migrate() {
  local legacy="$1"
  local target="$2"
  if ! gitlab_home_has_substance "$legacy"; then
    return 1
  fi
  if gitlab_home_has_substance "$target"; then
    return 1
  fi
  # Same resolved path → nothing to do
  local leg_abs tgt_abs
  leg_abs="$(cd "$legacy" 2>/dev/null && pwd -P)" || return 1
  mkdir -p "$target"
  tgt_abs="$(cd "$target" 2>/dev/null && pwd -P)" || return 1
  [[ "$leg_abs" != "$tgt_abs" ]]
}

ensure_gitlab_home_dirs() {
  local root="$1"
  mkdir -p "$root/config" "$root/logs" "$root/data" "$root/bootstrap_marks"
}

migrate_legacy_gitlab_home() {
  local legacy="$1"
  local target="$2"
  ensure_gitlab_home_dirs "$target"
  echo "迁移 GitLab 数据: $legacy → $target …"

  local copied=0
  if command -v rsync >/dev/null 2>&1; then
    if rsync -aHAX "$legacy/" "$target/" 2>/dev/null || rsync -a "$legacy/" "$target/" 2>/dev/null; then
      copied=1
    fi
  fi

  # Root-owned GitLab files often cannot be read by the invoking user; copy via a
  # short-lived privileged container (no host sudo required).
  if [[ "$copied" -ne 1 ]] || ! gitlab_home_has_substance "$target"; then
    if ! command -v docker >/dev/null 2>&1; then
      echo "错误: rsync 权限不足且无 docker，无法迁移 $legacy → $target" >&2
      return 1
    fi
    echo "使用 docker 容器以 root 身份复制（绕过宿主机文件权限）…"
    docker run --rm \
      -v "$legacy:/from:ro" \
      -v "$target:/to" \
      alpine:3.20 \
      sh -c 'cp -a /from/. /to/' || return 1
  fi

  if ! gitlab_home_has_substance "$target"; then
    echo "错误: 迁移后目标目录仍无实质 GitLab 数据: $target" >&2
    return 1
  fi
  echo "GitLabHomeMigrated legacy=$legacy target=$target"
  echo ""
  echo "提示: 迁移成功后，旧数据目录不再使用，可酌情清理："
  echo "  sudo rm -rf $legacy"
  echo "  或确认保留作为备份：ls -lh $legacy"
  return 0
}

# Ensure mounts point at GITLAB_HOME. If running container still uses another
# Source path, recreate container (data stays on host dirs).
gitlab_home_mount_matches() {
  local container="${1:-gitlab}"
  local expected="$2"
  local src
  src="$(docker inspect "$container" --format '{{range .Mounts}}{{if eq .Destination "/var/opt/gitlab"}}{{.Source}}{{end}}{{end}}' 2>/dev/null || true)"
  [[ -n "$src" && "$src" == "$expected/data" ]]
}
