#!/usr/bin/env bash
# 遍历 conf/auth/git-oauth/providers/ 下所有 GitLab provider YAML 文件，
# 将各 OAuth 应用的 scope/redirect_uri 同步到「该 website 所属」GitLab 容器的
# Doorkeeper Application（ADR-0014 多区域：禁止全部写入默认 gitlab 容器）。
# GitHub provider 自动跳过（使用 GitHub OAuth 机制，非 GitLab Doorkeeper）。
#
# GITLAB_CONTAINER 若设置，只同步映射到该容器名的 YAML（其余跳过）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROVIDERS_DIR="$WORKSPACE_ROOT/conf/auth/git-oauth/providers"
MAP_SCRIPT="$SCRIPT_DIR/gitlab_oauth_target_container.py"
FILTER_CONTAINER="${GITLAB_CONTAINER:-}"
MAX_WAIT=300

wait_for_rails() {
  local container="$1"
  local waited=0
  echo "等待 GitLab 容器 $container 初始化完成..."
  while (( waited < MAX_WAIT )); do
    if docker exec "$container" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
      echo "GitLab 容器 $container 初始化完成。"
      return 0
    fi
    sleep 5
    waited=$((waited + 5))
  done
  echo "GitLab 容器 $container 初始化超时（${MAX_WAIT}秒），跳过该实例。" >&2
  return 1
}

declare -A RAILS_READY=()

SYNC_COUNT=0
SKIP_COUNT=0
FAIL_COUNT=0

for PROVIDER_YAML in "$PROVIDERS_DIR"/http-*.yaml; do
  [[ -f "$PROVIDER_YAML" ]] || continue

  # 读取 provider/scope 元信息 + target 字段
  {
    read -r PROVIDER
    read -r SERVICE_PROVIDER
    read -r CLIENT_ID
    read -r CLIENT_SECRET
    read -r REQUESTED_SCOPE
    read -r REDIRECT_URI
    read -r WEBSITE
  } < <(python3 - "$PROVIDER_YAML" <<'PY'
import os, re, sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("\n\n\n\n\n\n")
    raise SystemExit(0)

path = Path(sys.argv[1])
if not path.is_file():
    print("\n\n\n\n\n\n")
    raise SystemExit(0)

repo_root = path
for _ in range(10):
    repo_root = repo_root.parent
    if (repo_root / "conf" / "base.yaml").is_file():
        break
scripts = str(repo_root / "runAll" / "scripts")
if scripts not in sys.path:
    sys.path.insert(0, scripts)
from conf_local import overlay_conf_file

# ── 解析 conf/base.yaml 中的 domain 模板 ──
_PLACEHOLDER_RE = re.compile(r"\$\{(scheme|baseDomain|subdomains\.[A-Za-z][A-Za-z0-9]*)\}")

def _build_domain_map():
    """从 conf/base.yaml 构建 domain map（仅域名寻址）。"""
    candidate = repo_root / "conf" / "base.yaml"
    if not candidate.is_file():
        return {}
    raw = overlay_conf_file(candidate) or {}
    _env_default_re = re.compile(r"\$\{(\w+):-([^}]*)\}")

    def _resolve_env_default(v: str) -> str:
        def _repl(m):
            return os.environ.get(m.group(1), m.group(2) or "")
        return _env_default_re.sub(_repl, str(v))

    scheme = os.environ.get("PUBLIC_SCHEME") or _resolve_env_default(
        str(raw.get("scheme", "https"))
    )
    scheme = scheme.strip().lower().rstrip(":/") or "https"
    base_domain = os.environ.get("BASE_DOMAIN") or _resolve_env_default(
        str(raw.get("baseDomain", ""))
    )
    if not base_domain:
        return {}
    domain_map = {"scheme": scheme, "baseDomain": base_domain}
    subdomains = raw.get("subdomains") or {}
    if isinstance(subdomains, dict):
        for key, template in subdomains.items():
            val = (
                str(template)
                .replace("${scheme}", scheme)
                .replace("${baseDomain}", base_domain)
            )
            domain_map[f"subdomains.{key}"] = val
    return domain_map

_DOMAIN_MAP = _build_domain_map()

def _resolve(val):
    """解析字符串值中的 ${baseDomain} / ${subdomains.xxx} 占位符。"""
    if not isinstance(val, str):
        return val
    return _PLACEHOLDER_RE.sub(
        lambda m: _DOMAIN_MAP.get(m.group(1), m.group(0)),
        val,
    )

# ── 读取 provider 配置 ──
data = overlay_conf_file(path) or {}
provider = str(data.get("provider") or "").strip().lower()
sp = str(data.get("service_provider") or "").strip()
target = data.get("target") or {}
# 模板解析后再输出
print(provider)
print(sp)
print(_resolve(target.get("client_id", "")))
print(_resolve(target.get("client_secret", "")))
print(_resolve(target.get("scope", "read_repository write_repository api read_user")))
print(_resolve(target.get("redirect_uri", "")))
print(_resolve(target.get("website", "")))
PY
  )

  # 跳过 GitHub provider（GitHub OAuth 不由 Doorkeeper 管理）
  if [[ "$PROVIDER" == "github" ]]; then
    echo "跳过 GitHub provider: $(basename "$PROVIDER_YAML") (service_provider=$SERVICE_PROVIDER)"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  if [[ "$PROVIDER" != "gitlab" ]]; then
    echo "跳过未知 provider 类型 '$PROVIDER': $(basename "$PROVIDER_YAML")"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  if [[ -z "$CLIENT_ID" ]]; then
    echo "跳过（无 client_id）: $(basename "$PROVIDER_YAML")"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  # 跳过含未展开模板变量的 redirect_uri（如 ${subdomains.gitoauth}），避免 Doorkeeper 校验失败中断整批同步
  if [[ "$REDIRECT_URI" == *'${'* ]]; then
    echo "跳过（redirect_uri 含未展开模板）: $(basename "$PROVIDER_YAML") redirect_uri=$REDIRECT_URI"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  CONTAINER="$(python3 "$MAP_SCRIPT" --website "$WEBSITE" --root "$WORKSPACE_ROOT" 2>/dev/null || true)"
  if [[ -z "$CONTAINER" ]]; then
    echo "跳过（website 未映射到 git-service 容器）: $(basename "$PROVIDER_YAML") website=$WEBSITE"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  if [[ -n "$FILTER_CONTAINER" && "$CONTAINER" != "$FILTER_CONTAINER" ]]; then
    echo "跳过（GITLAB_CONTAINER=$FILTER_CONTAINER 过滤）: $(basename "$PROVIDER_YAML") → $CONTAINER"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    echo "跳过（容器 $CONTAINER 未在本机运行）: $(basename "$PROVIDER_YAML")"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  if [[ -z "${RAILS_READY[$CONTAINER]+x}" ]]; then
    if wait_for_rails "$CONTAINER"; then
      RAILS_READY[$CONTAINER]=1
    else
      RAILS_READY[$CONTAINER]=0
    fi
  fi
  if [[ "${RAILS_READY[$CONTAINER]}" != "1" ]]; then
    echo "跳过（容器 $CONTAINER rails 未就绪）: $(basename "$PROVIDER_YAML")"
    SKIP_COUNT=$((SKIP_COUNT + 1))
    continue
  fi

  SCOPE="${REQUESTED_SCOPE:-read_repository write_repository api read_user}"
  APP_NAME="gitOauth ${SERVICE_PROVIDER:-$(basename "$PROVIDER_YAML" .yaml)}"
  echo "同步 Doorkeeper Application uid=$CLIENT_ID name=$APP_NAME → container=$CONTAINER scopes: $SCOPE"

  if ! docker exec "$CONTAINER" gitlab-rails runner "$(cat <<RUBY
uid = '${CLIENT_ID}'
secret = '${CLIENT_SECRET}'
scopes = '${SCOPE}'
redirect = '${REDIRECT_URI}'
name = '${APP_NAME}'
app = Doorkeeper::Application.find_or_initialize_by(uid: uid)
if app.new_record?
  app.secret = secret unless secret.empty?
  app.name = name
  app.redirect_uri = redirect
  app.scopes = scopes
  app.confidential = true
  app.organization_id = Organizations::Organization.default_organization.id
  app.save!
  puts "CREATED: #{app.name} scopes=#{app.scopes}"
else
  # 自愈：同步已变更字段
  updated = false
  if app.name.to_s.strip.empty? || app.name.to_s.strip != name
    app.name = name
    updated = true
  end
  if app.redirect_uri.to_s.strip != redirect.to_s.strip
    app.redirect_uri = redirect
    updated = true
  end
  if app.scopes.to_s.strip != scopes.to_s.strip
    app.scopes = scopes
    updated = true
  end
  if updated
    app.save!
    puts "UPDATED: #{app.name} scopes=#{app.scopes} redirect_uri=#{app.redirect_uri}"
  else
    puts "UNCHANGED: #{app.name} scopes=#{app.scopes}"
  end
end
RUBY
  )"; then
    echo "同步失败: $(basename "$PROVIDER_YAML") uid=$CLIENT_ID container=$CONTAINER（继续处理其余 provider）" >&2
    FAIL_COUNT=$((FAIL_COUNT + 1))
    continue
  fi

  SYNC_COUNT=$((SYNC_COUNT + 1))
done

echo "OAuth scope 同步完成: 已同步 $SYNC_COUNT 个 GitLab provider, 跳过 $SKIP_COUNT 个, 失败 $FAIL_COUNT 个"
if (( FAIL_COUNT > 0 )); then
  exit 1
fi
