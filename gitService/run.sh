#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
# 第二实例：GITSERVICE_CONF_APP=git-service-tencent-sh-1 ./run.sh（上海精简 GitLab）
GITSERVICE_CONF_APP="${GITSERVICE_CONF_APP:-git-service}"

# ADR-0052: CONF_ROOT may be the conf dir or deploy root.
resolve_gitservice_conf_dir() {
  if [[ -n "${CONF_ROOT:-}" && -d "${CONF_ROOT}/infra" ]]; then
    echo "${CONF_ROOT}"
    return
  fi
  if [[ -n "${CONF_ROOT:-}" && -d "${CONF_ROOT}/conf/infra" ]]; then
    echo "${CONF_ROOT}/conf"
    return
  fi
  if [[ -n "${DEPLOY_ROOT:-}" && -d "${DEPLOY_ROOT}/conf/infra" ]]; then
    echo "${DEPLOY_ROOT}/conf"
    return
  fi
  echo "${WORKSPACE_ROOT}/conf"
}
GITSERVICE_CONF_DIR="$(resolve_gitservice_conf_dir)"
PORT_CONFIG_MAIN="${GITSERVICE_CONF_DIR}/infra/${GITSERVICE_CONF_APP}/config.yaml"
# ADR-0054：加载器只读 config.yaml + conf-local overlay；config.local.yaml 已废弃。
cd "$SCRIPT_DIR"

load_docker_helpers() {
  local helper=""
  for candidate in \
    "$SCRIPT_DIR/scripts/remote-compose-helper.sh" \
    "$WORKSPACE_ROOT/scripts/remote-compose-helper.sh" \
    "$SCRIPT_DIR/scripts/docker-desktop-helper.sh" \
    "$WORKSPACE_ROOT/scripts/docker-desktop-helper.sh"; do
    if [[ -f "$candidate" ]]; then
      helper="$candidate"
      break
    fi
  done
  if [[ -n "$helper" ]]; then
    # shellcheck source=/dev/null
    source "$helper"
    return 0
  fi

  # 远程 CPU 仅 clone gitService 时无 monorepo scripts/；使用 Linux 通用回退。
  docker_helper_compose_cmd() {
    if docker compose version &>/dev/null; then
      echo "docker compose"
    elif command -v docker-compose &>/dev/null; then
      echo "docker-compose"
    else
      return 1
    fi
  }
  docker_helper_daemon_ready() {
    docker info >/dev/null 2>&1
  }
  docker_helper_desktop_memory_mib() {
    return 1
  }
  docker_helper_try_start_desktop() {
    return 1
  }
  docker_helper_ensure_daemon() {
    local max_wait="${1:-60}" waited=0
    while (( waited < max_wait )); do
      if docker_helper_daemon_ready; then
        return 0
      fi
      sleep 1
      waited=$((waited + 1))
    done
    return 1
  }
  docker_helper_warn_gitlab_memory() {
    local min_mib="${1:-6144}"
  echo "提示: 远程 Docker 模式，建议宿主机可用内存 ≥ ${min_mib} MiB。" >&2
  }
}

load_docker_helpers

# shellcheck source=scripts/gitlab_home.sh
source "$SCRIPT_DIR/scripts/gitlab_home.sh"

compose_bin="$(docker_helper_compose_cmd)" || {
  echo "未找到 docker compose 或 docker-compose，请先安装 Docker Desktop / Docker Engine。" >&2
  exit 1
}
COMPOSE=($compose_bin)

wait_for_docker_daemon() {
  docker_helper_ensure_daemon "${GITLAB_DOCKER_WAIT_SECONDS:-60}"
}

mode="start"
if [[ $# -gt 0 ]]; then
  case "$1" in
    start|managed|stop)
      mode="$1"
      shift
      ;;
  esac
fi

compose_file() {
  "${COMPOSE[@]}" -f "$SCRIPT_DIR/docker-compose.yml" "$@"
}

# shellcheck source=scripts/ensure_gitlab_container.sh
source "$SCRIPT_DIR/scripts/ensure_gitlab_container.sh"

load_gitservice_config() {
  if ! command -v python3 &>/dev/null; then
    echo "提示: 未检测到 python3，gitService 将使用环境变量或默认端口配置。" >&2
    export GITLAB_EXTERNAL_HOST="${GITLAB_EXTERNAL_HOST:-localhost}"
    export GITLAB_HTTP_PORT="${GITLAB_HTTP_PORT:-8012}"
    export GITLAB_SSH_PORT="${GITLAB_SSH_PORT:-2222}"
    export GITLAB_ALLOWED_HOST_URL="${GITLAB_ALLOWED_HOST_URL:-}"
    export GITLAB_HOSTNAME="${GITLAB_HOSTNAME:-localhost}"
    export GITLAB_DISPLAY_HOST="${GITLAB_DISPLAY_HOST:-127.0.0.1}"
    export GITLAB_EXTERNAL_URL="${GITLAB_EXTERNAL_URL:-http://${GITLAB_EXTERNAL_HOST}:${GITLAB_HTTP_PORT}}"
    # fallback only — SSOT: conf/infra/git-service/config.yaml signupEnabled/passwordAuth*
    export GITLAB_SIGNUP_ENABLED="${GITLAB_SIGNUP_ENABLED:-false}"
    export GITLAB_PASSWORD_AUTH_WEB="${GITLAB_PASSWORD_AUTH_WEB:-false}"
    export GITLAB_PASSWORD_AUTH_GIT="${GITLAB_PASSWORD_AUTH_GIT:-false}"
    export GITLAB_IMAGE="${GITLAB_IMAGE:-gitlab/gitlab-ce:19.2.4-ce.0}"
    export GITLAB_RUNALL_START_ENABLED="${GITLAB_RUNALL_START_ENABLED:-false}"
    export TRAE_TASKBILL_BASE="${TRAE_TASKBILL_BASE:-http://host.docker.internal:8004}"
    export TRAE_TASKBILL_INTERNAL_SECRET="${TRAE_TASKBILL_INTERNAL_SECRET:-}"
    export TRAE_GITLAB_REGION="${TRAE_GITLAB_REGION:-}"
    export TRAE_GITLAB_PUBLIC_HOST="${TRAE_GITLAB_PUBLIC_HOST:-${GITLAB_EXTERNAL_HOST:-}}"
    export TRAE_GITLAB_INTRANET_HOSTS="${TRAE_GITLAB_INTRANET_HOSTS:-}"
    return
  fi

  # 优先使用 conf-read.py（已内置模板解析 ${subdomains.xxx}），
  # 避免手动 yaml.safe_load 拿到的 allowedHost 含未解析模板变量被丢弃。
  local config_json=""
  local conf_read="$WORKSPACE_ROOT/runAll/scripts/conf-read.py"
  # 非默认实例禁止走 gitService conf-read（会读回 8012 现网配置）
  if [[ -f "$conf_read" && "$GITSERVICE_CONF_APP" == "git-service" ]]; then
    config_json=$(python3 "$conf_read" gitService --json 2>/dev/null || echo "")
  fi

  # 解析逻辑抽到 scripts/load_gitservice_config.py（独立可测模块，OPT-20260812-033）：
  # conf-read JSON 优先，YAML 兜底；单测直接 import 覆盖两分支。
  config_lines="$(python3 "$SCRIPT_DIR/scripts/load_gitservice_config.py" "$config_json" "$PORT_CONFIG_MAIN")"

  _oidc_secret_pre="${GITLAB_OIDC_CLIENT_SECRET:-}"
  _bill_secret_pre="${TRAE_TASKBILL_INTERNAL_SECRET:-${TASKBILL_INTERNAL_SECRET:-}}"
  while IFS='=' read -r key value; do
    [[ -z "${key:-}" ]] && continue
    export "$key=$value"
  done <<< "$config_lines"
  export GITLAB_OIDC_CLIENT_SECRET="${GITLAB_OIDC_CLIENT_SECRET:-$_oidc_secret_pre}"
  export TRAE_TASKBILL_INTERNAL_SECRET="${TRAE_TASKBILL_INTERNAL_SECRET:-$_bill_secret_pre}"
  export TRAE_TASKBILL_BASE="${TRAE_TASKBILL_BASE:-http://host.docker.internal:8004}"
  export TRAE_GITLAB_REGION="${TRAE_GITLAB_REGION:-}"
  export TRAE_GITLAB_PUBLIC_HOST="${TRAE_GITLAB_PUBLIC_HOST:-${GITLAB_EXTERNAL_HOST:-}}"
  export TRAE_GITLAB_INTRANET_HOSTS="${TRAE_GITLAB_INTRANET_HOSTS:-}"
}

load_gitservice_config
export GITLAB_CONTAINER="${GITLAB_CONTAINER_NAME:-gitlab}"
export GITLAB_IMAGE="${GITLAB_IMAGE:-gitlab/gitlab-ce:19.2.4-ce.0}"
# 分步升级可覆盖 conf imageTag：GITLAB_IMAGE_OVERRIDE=gitlab/gitlab-ce:19.0.8-ce.0
if [[ -n "${GITLAB_IMAGE_OVERRIDE:-}" ]]; then
  export GITLAB_IMAGE="$GITLAB_IMAGE_OVERRIDE"
  echo "GITLAB_IMAGE_OVERRIDE=$GITLAB_IMAGE" >&2
fi

# shellcheck source=scripts/local_gitlab_start_gate.sh
source "$SCRIPT_DIR/scripts/local_gitlab_start_gate.sh"
if [[ "$mode" != "stop" ]]; then
  if ! local_gitlab_start_allowed "$GITSERVICE_CONF_APP" "${GITLAB_RUNALL_START_ENABLED:-false}"; then
    print_local_gitlab_start_refused >&2
    echo "event=gitlab_start_refused conf_app=${GITSERVICE_CONF_APP} runAllStartEnabled=${GITLAB_RUNALL_START_ENABLED:-false}" >&2
    exit 1
  fi
  echo "event=gitlab_start_allowed conf_app=${GITSERVICE_CONF_APP} runAllStartEnabled=${GITLAB_RUNALL_START_ENABLED}" >&2
fi

prepare_gitlab_home() {
  local conf_home="${GITLAB_HOME_FROM_CONF:-}"
  # resolve_gitlab_home respects existing GITLAB_HOME env first.
  export GITLAB_HOME
  GITLAB_HOME="$(resolve_gitlab_home "$conf_home")"
  assert_gitlab_home_durable "$GITLAB_HOME" || exit 1
  ensure_gitlab_home_dirs "$GITLAB_HOME"
  local fstype
  fstype="$(gitlab_home_fstype "$GITLAB_HOME")"
  echo "GITLAB_HOME=$GITLAB_HOME (fstype=${fstype:-unknown})"
  export GITLAB_ADMIN_PAT_FILE="${GITLAB_ADMIN_PAT_FILE:-$GITLAB_HOME/.taskbill_admin_pat}"
}

# Requires Docker daemon. Migrates legacy ./gitlab_home and realigns container mounts.
migrate_and_realign_gitlab_home() {
  local legacy="$SCRIPT_DIR/gitlab_home"
  local did_migrate=0
  local CONTAINER_NAME="${GITLAB_CONTAINER:-gitlab}"

  if legacy_needs_migrate "$legacy" "$GITLAB_HOME"; then
    if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
      echo "迁移前停止 GitLab 容器以保持数据一致…"
      compose_file stop >/dev/null 2>&1 || docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    migrate_legacy_gitlab_home "$legacy" "$GITLAB_HOME" || exit 1
    did_migrate=1
    if [[ -d "$SCRIPT_DIR/.bootstrap_marks" ]]; then
      mkdir -p "$GITLAB_HOME/bootstrap_marks"
      cp -a "$SCRIPT_DIR/.bootstrap_marks/." "$GITLAB_HOME/bootstrap_marks/" 2>/dev/null || true
    fi
  fi

  # After migration, always recreate so mounts cannot stick to legacy paths.
  # Also recreate when an existing container's /var/opt/gitlab Source ≠ $GITLAB_HOME/data.
  if docker ps -a --format '{{.Names}}' | grep -qx "$CONTAINER_NAME"; then
    if [[ "$did_migrate" -eq 1 ]] || ! gitlab_home_mount_matches "$CONTAINER_NAME" "$GITLAB_HOME"; then
      echo "对齐容器挂载到 GITLAB_HOME=$GITLAB_HOME（宿主机数据保留）…"
      docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
      docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
  fi
}

prepare_gitlab_home

apply_remote_infra_host() {
  local infra_host="${RUNALL_INFRA_HOST:-}"
  infra_host="${infra_host#"${infra_host%%[![:space:]]*}"}"
  infra_host="${infra_host%"${infra_host##*[![:space:]]}"}"
  if [[ -z "$infra_host" ]]; then
    return 0
  fi
  # Strip URL scheme if present (e.g., http://host:port → host)
  if [[ "$infra_host" == http://* ]]; then
    infra_host="${infra_host#http://}"
  elif [[ "$infra_host" == https://* ]]; then
    infra_host="${infra_host#https://}"
  fi
  export GITLAB_EXTERNAL_HOST="$infra_host"
  export GITLAB_DISPLAY_HOST="$infra_host"
  export GITLAB_HOSTNAME="$infra_host"
  echo "远程 infra 覆盖: GITLAB_EXTERNAL_HOST=$GITLAB_EXTERNAL_HOST" >&2
}

apply_remote_infra_host

echo "GitService 配置来源: $PORT_CONFIG_MAIN (overlay: conf-local/infra/${GITSERVICE_CONF_APP}/config.yaml)"
echo "生效配置: host=$GITLAB_EXTERNAL_HOST, port=$GITLAB_HTTP_PORT, sshPort=$GITLAB_SSH_PORT"
if [[ -n "${GITLAB_EXTERNAL_URL:-}" ]]; then
  echo "external_url: $GITLAB_EXTERNAL_URL"
fi
if [[ -n "${GITLAB_ALLOWED_HOST_URL:-}" ]]; then
  echo "allowedHost: $GITLAB_ALLOWED_HOST_URL"
fi

# 同步 OIDC issuer / redirect_uri 到 GitLab 容器环境变量
# 优先已有环境变量；否则解析 task-auth oidc（含 ${subdomains.*}）；
# issuer 再回退到 base.yaml 的 ${subdomains.gateway}。
# redirect_uri 必须与 taskAuth bootstrapClients[].redirectUri 完全一致。
if [[ -z "${GITLAB_OIDC_ISSUER:-}" || -z "${GITLAB_OIDC_REDIRECT_URI:-}" ]]; then
  export WORKSPACE_ROOT
  OIDC_RESOLVED=$(python3 << 'PYEOF'
import os, re
from pathlib import Path

root = Path(os.environ.get("WORKSPACE_ROOT", "")).resolve()
base_path = root / "conf" / "base.yaml"
task_auth = root / "conf" / "auth" / "task-auth" / "config.yaml"
import sys
_scripts = str(root / "runAll" / "scripts")
if _scripts not in sys.path:
    sys.path.insert(0, _scripts)
from conf_local import overlay_conf_file

def expand_env_default(value: str) -> str:
    return re.sub(
        r"\$\{(\w+):-([^}]*)\}",
        lambda m: os.environ.get(m.group(1), m.group(2) or ""),
        value,
    )

def load_domain_map() -> dict[str, str]:
    if not base_path.is_file():
        return {}
    try:
        import yaml
    except ImportError:
        return {}
    base = overlay_conf_file(base_path) or {}
    scheme = expand_env_default(str(base.get("scheme") or "https"))
    if os.environ.get("PUBLIC_SCHEME"):
        scheme = os.environ["PUBLIC_SCHEME"]
    scheme = scheme.strip().lower().rstrip(":/") or "https"
    domain = expand_env_default(str(base.get("baseDomain") or ""))
    if os.environ.get("BASE_DOMAIN"):
        domain = os.environ["BASE_DOMAIN"]
    if not domain:
        return {}
    m = {"scheme": scheme, "baseDomain": domain}
    for k, tmpl in (base.get("subdomains") or {}).items():
        m[f"subdomains.{k}"] = (
            str(tmpl).replace("${scheme}", scheme).replace("${baseDomain}", domain)
        )
    return m

def resolve_templates(s: str, m: dict[str, str]) -> str:
    out = s
    for k, v in m.items():
        out = out.replace("${" + k + "}", v)
    return out

domain_map = load_domain_map()
issuer = ""
redirect_uri = ""
if task_auth.is_file():
    try:
        data = overlay_conf_file(task_auth) or {}
        oidc = data.get("oidc") or {}
        issuer = str(oidc.get("issuer") or "").strip()
        issuer = resolve_templates(issuer, domain_map)
        target_client = os.environ.get("GITLAB_OIDC_CLIENT_ID") or "gitlab-git-service"
        for client in oidc.get("bootstrapClients") or []:
            if str(client.get("clientId") or "") == target_client:
                redirect_uri = resolve_templates(str(client.get("redirectUri") or "").strip(), domain_map)
                break
        if not redirect_uri:
            legacy = str(oidc.get("bootstrapRedirectUri") or "").strip()
            if legacy:
                redirect_uri = resolve_templates(legacy, domain_map)
        if not redirect_uri and domain_map.get("scheme") and domain_map.get("subdomains.gitlab"):
            redirect_uri = f"{domain_map['scheme']}://{domain_map['subdomains.gitlab']}/users/auth/openid_connect/callback"
    except Exception:
        issuer = ""
        redirect_uri = ""
if (not issuer or "${" in issuer) and domain_map.get("subdomains.gateway"):
    issuer = domain_map["subdomains.gateway"]
print(issuer if issuer and "${" not in issuer else "")
print(redirect_uri if redirect_uri and "${" not in redirect_uri else "")
PYEOF
)
  ISSUER_RESOLVED="$(printf '%s\n' "$OIDC_RESOLVED" | sed -n '1p')"
  REDIRECT_RESOLVED="$(printf '%s\n' "$OIDC_RESOLVED" | sed -n '2p')"
  if [[ -z "${GITLAB_OIDC_ISSUER:-}" && -n "$ISSUER_RESOLVED" ]]; then
    export GITLAB_OIDC_ISSUER="$ISSUER_RESOLVED"
  fi
  if [[ -z "${GITLAB_OIDC_REDIRECT_URI:-}" && -n "$REDIRECT_RESOLVED" ]]; then
    export GITLAB_OIDC_REDIRECT_URI="$REDIRECT_RESOLVED"
  fi
fi
echo "OIDC issuer: ${GITLAB_OIDC_ISSUER:-<未设置>}"
echo "OIDC redirect_uri: ${GITLAB_OIDC_REDIRECT_URI:-<未设置>}"

# ── 启动前校验：external_host 不能是 loopback ──────────────────
validate_external_host() {
  case "$GITLAB_EXTERNAL_HOST" in
    localhost|127.*|::1|0.0.0.0)
      echo "" >&2
      echo "==============================================" >&2
      echo " 警告: GitLab external_url 主机名为 '$GITLAB_EXTERNAL_HOST'" >&2
      echo " 这会导致页面上显示的仓库 clone URL 无法从外部访问。" >&2
      echo "" >&2
      echo " 修复方式:" >&2
      echo "   1. 设置环境变量: export GITLAB_EXTERNAL_HOST=<你的域名>" >&2
      echo "   2. 或修改 conf/base.yaml 的 baseDomain / subdomains.gitlab" >&2
      echo "   3. 或设置: export BASE_DOMAIN=<根域名>" >&2
      echo "   4. 已运行的容器需要重建: docker compose down && bash run.sh" >&2
      echo "" >&2
      echo " 如这是本地开发环境且仅本机访问，可忽略此警告。" >&2
      echo "==============================================" >&2
      echo "" >&2
      ;;
  esac
}
validate_external_host

if ! wait_for_docker_daemon; then
  echo "Docker daemon 当前不可用。远程开发请确认 CPU 机 Docker 已运行；本机仅需 docker CLI + cpu-remote context。" >&2
  echo "排查: docker context show && docker info" >&2
  exit 1
fi

docker_helper_warn_gitlab_memory "${GITLAB_MIN_DOCKER_MEMORY_MIB:-6144}"

apply_gitlab_memory_tuning() {
  # 基准默认来自 conf（load_gitservice_config 已导出）；此处仅在 Docker 内存不足时降档。
  local min_mib="${GITLAB_MIN_DOCKER_MEMORY_MIB:-6144}"
  local current_mib=""
  current_mib="$(docker_helper_desktop_memory_mib 2>/dev/null || true)"
  if [[ -n "$current_mib" ]] && (( current_mib < min_mib )); then
    export GITLAB_MEM_LIMIT="${GITLAB_MEM_LIMIT_LOW:-2g}"
    export GITLAB_PUMA_WORKERS="${GITLAB_PUMA_WORKERS_LOW:-1}"
    export GITLAB_SIDEKIQ_CONCURRENCY="${GITLAB_SIDEKIQ_CONCURRENCY_LOW:-3}"
    echo "GitLab 精简模式: mem_limit=$GITLAB_MEM_LIMIT puma_workers=$GITLAB_PUMA_WORKERS sidekiq=$GITLAB_SIDEKIQ_CONCURRENCY（Docker 内存 ${current_mib} MiB < ${min_mib} MiB）" >&2
  fi
}

apply_gitlab_memory_tuning

assert_docker_still_running() {
  if docker_helper_daemon_ready; then
    return 0
  fi
  echo "" >&2
  echo "错误: Docker Engine 在 GitLab 启动过程中停止。" >&2
  echo "常见原因: Docker Desktop 分配内存不足（当前建议 ≥ ${GITLAB_MIN_DOCKER_MEMORY_MIB:-6144} MiB）。" >&2
  echo "请调大 Docker Desktop → Settings → Resources → Memory 后重试。" >&2
  return 1
}

case "$mode" in
  stop)
    if [[ "${1:-}" == "--clean" ]]; then
      echo "完全清理 GitLab 容器（不删除 GITLAB_HOME 数据）..."
      compose_file down --remove-orphans
      rm -rf "${GITLAB_HOME:-}/bootstrap_marks/"* ./.bootstrap_marks/* 2>/dev/null || true
      echo "容器已清理。持久数据仍在: ${GITLAB_HOME:-<未设置>}"
      echo "若需清空数据，请手动: rm -rf \"\$GITLAB_HOME\""
    else
      echo "正在停止 GitLab 容器（保留容器以加速下次启动）..."
      compose_file stop
      echo "提示: 使用 'bash gitService/run.sh stop --clean' 完全清理容器（仍保留 GITLAB_HOME 数据）。"
    fi
    exit 0
    ;;
  managed)
    echo "runAll 托管模式：确保 GitLab 容器可用..."
    migrate_and_realign_gitlab_home
    ensure_container
    ;;
  *)
    echo "确保 GitLab 容器可用..."
    migrate_and_realign_gitlab_home
    ensure_container
    ;;
esac

if [[ "$mode" != "managed" ]] && [[ -n "${HTTP_PROXY:-}${HTTPS_PROXY:-}${http_proxy:-}${https_proxy:-}" ]] && [[ "${NO_PROXY:-${no_proxy:-}}" != *"localhost"* ]]; then
  echo ""
  echo "提示: 检测到 HTTP(S) 代理环境变量，且 NO_PROXY 未包含 localhost。"
  echo "若浏览器或 curl 打不开页面，请将 localhost、127.0.0.1 加入 NO_PROXY/no_proxy，或对本地地址使用「绕过代理」。"
  echo ""
fi

WEB_URL="${GITLAB_EXTERNAL_URL:-http://${GITLAB_DISPLAY_HOST}:${GITLAB_HTTP_PORT}}"
SSH_URL="ssh://git@${GITLAB_DISPLAY_HOST}:${GITLAB_SSH_PORT}/group/project.git"

echo ""
echo "Web:   $WEB_URL"
echo "SSH:   $SSH_URL"
if [[ -n "${GITLAB_ALLOWED_HOST_URL:-}" && "$WEB_URL" != "$GITLAB_ALLOWED_HOST_URL" ]]; then
  echo "域名访问(可选): $GITLAB_ALLOWED_HOST_URL"
fi
echo "查看 root 初始密码:"
echo "  docker exec -it gitlab grep 'Password:' /etc/gitlab/initial_root_password"
echo ""
if [[ "$mode" != "stop" ]]; then
    run_bootstrap_if_needed
  fi

