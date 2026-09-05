#!/usr/bin/env bash
# 载入远程 Docker 快捷命令。加入 ~/.zshrc:
#   source /path/to/ram-mount/runAll/scripts/docker-shell.sh
#
# 可选：默认使用远程 context
#   export RAM_MOUNT_DOCKER_DEFAULT=remote

_ram_mount_docker_scripts="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

docker-remote() {
  "$_ram_mount_docker_scripts/docker-use-remote.sh"
}

docker-dctx() {
  "$_ram_mount_docker_scripts/docker-status.sh"
}

docker-tunnel() {
  "$_ram_mount_docker_scripts/docker-tunnel-remote.sh" "$@"
}

# 短 alias（仅交互 shell）
if [[ -n "${ZSH_VERSION:-}" ]] || [[ "${-#*i}" != "$-" ]]; then
  alias dr='docker-remote'
  alias ds='docker-dctx'
  alias dt='docker-tunnel'
fi

[[ "${RAM_MOUNT_DOCKER_DEFAULT:-remote}" == "remote" ]] && docker-remote >/dev/null 2>&1 || true
