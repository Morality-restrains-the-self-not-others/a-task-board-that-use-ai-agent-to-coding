#!/usr/bin/env bash
# 确保 GitLab OmniAuth OIDC 配置与 taskAuth OIDC Provider 同步。
# 用法:
#   sync_omniauth_oidc.sh              检查并报告 OmniAuth 状态
#   sync_omniauth_oidc.sh --reconfigure 检测到 OIDC 未注册或 redirect_uri 漂移时自动 gitlab-ctl reconfigure
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CONTAINER="${GITLAB_CONTAINER:-gitlab}"
TASKAUTH_CONFIG="$WORKSPACE_ROOT/conf/auth/task-auth/config.yaml"
BASE_CONFIG="$WORKSPACE_ROOT/conf/base.yaml"
RECONFIGURE=false
if [[ "${1:-}" == "--reconfigure" ]]; then
  RECONFIGURE=true
  shift || true
fi

echo "=== gitService OmniAuth OIDC 同步 ==="

# ── 1. 读取 taskAuth OIDC bootstrap client 配置（与 run.sh 同源解析） ──
if command -v python3 &>/dev/null; then
  mapfile -t _OIDC_CFG < <(python3 - "$TASKAUTH_CONFIG" "$BASE_CONFIG" <<'PY'
import os, re, sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("")
    print("")
    print("")
    raise SystemExit(0)

task_auth = Path(sys.argv[1])
base_path = Path(sys.argv[2])

def expand_env_default(value: str) -> str:
    return re.sub(
        r"\$\{(\w+):-([^}]*)\}",
        lambda m: os.environ.get(m.group(1), m.group(2) or ""),
        value,
    )

def load_domain_map() -> dict[str, str]:
    if not base_path.is_file():
        return {}
    with open(base_path, encoding="utf-8") as f:
        base = yaml.safe_load(f) or {}
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
data = yaml.safe_load(task_auth.read_text(encoding="utf-8")) or {}
oidc = data.get("oidc") or {}
client_id = ""
client_secret = ""
redirect_uri = ""
for client in oidc.get("bootstrapClients") or []:
    if str(client.get("clientId") or "") == "gitlab-git-service":
        client_id = str(client.get("clientId") or "")
        client_secret = str(client.get("clientSecret") or "")
        redirect_uri = resolve_templates(str(client.get("redirectUri") or "").strip(), domain_map)
        break
if not client_id:
    client_id = str(oidc.get("bootstrapClientId") or "")
    client_secret = str(oidc.get("bootstrapClientSecret") or "")
    redirect_uri = resolve_templates(str(oidc.get("bootstrapRedirectUri") or "").strip(), domain_map)
if os.environ.get("GITLAB_OIDC_CLIENT_ID"):
    client_id = os.environ["GITLAB_OIDC_CLIENT_ID"]
if os.environ.get("GITLAB_OIDC_CLIENT_SECRET"):
    client_secret = os.environ["GITLAB_OIDC_CLIENT_SECRET"]
if os.environ.get("GITLAB_OIDC_REDIRECT_URI"):
    redirect_uri = os.environ["GITLAB_OIDC_REDIRECT_URI"]
print(client_id)
print(client_secret)
print(redirect_uri)
PY
  )
  OIDC_CLIENT_ID="${_OIDC_CFG[0]:-}"
  OIDC_CLIENT_SECRET="${_OIDC_CFG[1]:-}"
  OIDC_REDIRECT_URI="${_OIDC_CFG[2]:-}"
else
  echo "python3 不可用，从环境变量读取" >&2
  OIDC_CLIENT_ID="${GITLAB_OIDC_CLIENT_ID:-gitlab-git-service}"
  OIDC_CLIENT_SECRET="${GITLAB_OIDC_CLIENT_SECRET:-}"
  OIDC_REDIRECT_URI="${GITLAB_OIDC_REDIRECT_URI:-}"
fi

echo "  oidc_client_id:     ${OIDC_CLIENT_ID:-<未配置>}"
echo "  oidc_client_secret: ${OIDC_CLIENT_SECRET:+***}"
echo "  oidc_redirect_uri:  ${OIDC_REDIRECT_URI:-<未配置>}"

if [[ -z "$OIDC_CLIENT_ID" ]]; then
  echo "跳过：OIDC bootstrap client 未在 $TASKAUTH_CONFIG 中配置" >&2
  exit 0
fi

# ── 2. 等待 GitLab 容器运行 ─────────────────────────────────────────────
if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  echo "GitLab 容器 $CONTAINER 未运行，跳过。" >&2
  exit 0
fi

# ── 3. 等待 GitLab 就绪 ─────────────────────────────────────────────────
wait_for_gitlab() {
  local max_wait=300
  local waited=0
  echo "等待 GitLab 初始化完成..."
  while (( waited < max_wait )); do
    if docker exec "$CONTAINER" gitlab-rails runner "puts 'ready'" >/dev/null 2>&1; then
      echo "GitLab 就绪。"
      return 0
    fi
    sleep 5
    waited=$((waited + 5))
  done
  echo "GitLab 初始化超时 (${max_wait}s)。" >&2
  return 1
}
wait_for_gitlab || exit 0

# ── 4. 检查 OmniAuth openid_connect 提供者与 redirect_uri ───────────────
check_omniauth() {
  docker exec "$CONTAINER" gitlab-rails runner "
    providers = Gitlab::Auth::OAuth::Provider.providers.map(&:name)
    puts providers.sort.join(',')
  " 2>&1 || echo ""
}

current_redirect_uri() {
  docker exec "$CONTAINER" gitlab-rails runner "
    p = Gitlab.config.omniauth.providers.find { |x| x.name.to_s == 'openid_connect' }
    if p.nil?
      puts ''
    else
      args = p.args.is_a?(Hash) ? p.args : p.args.to_hash
      args = args.transform_keys(&:to_s) rescue args
      opts = args['client_options'] || args[:client_options] || {}
      opts = opts.transform_keys(&:to_s) rescue opts
      puts (opts['redirect_uri'] || opts[:redirect_uri]).to_s
    end
  " 2>/dev/null | tail -n1 | tr -d '\r' || echo ""
}

PROVIDERS=$(check_omniauth)
CURRENT_REDIRECT="$(current_redirect_uri)"
NEED_RECONFIGURE=false

if ! echo "$PROVIDERS" | grep -q "openid_connect"; then
  echo "⚠️  OmniAuth openid_connect 未注册（当前: ${PROVIDERS:-<无>}）"
  NEED_RECONFIGURE=true
elif [[ -n "$OIDC_REDIRECT_URI" && "$CURRENT_REDIRECT" != "$OIDC_REDIRECT_URI" ]]; then
  echo "⚠️  OmniAuth redirect_uri 漂移:"
  echo "     当前: ${CURRENT_REDIRECT:-<空>}"
  echo "     期望: $OIDC_REDIRECT_URI"
  NEED_RECONFIGURE=true
else
  echo "✅ OmniAuth openid_connect 提供者已注册"
  echo "  live redirect_uri: ${CURRENT_REDIRECT:-<空>}"
fi

if $NEED_RECONFIGURE; then
  if $RECONFIGURE; then
    echo ">>> 执行 gitlab-ctl reconfigure 以加载 OmniAuth 配置..."
    if docker exec "$CONTAINER" gitlab-ctl reconfigure 2>&1 | while IFS= read -r line; do
      echo "  [gitlab-ctl] $line"
    done; then
      echo "reconfigure 完成，重新等待 GitLab 就绪..."
      wait_for_gitlab || { echo "reconfigure 后 GitLab 未能就绪" >&2; exit 1; }

      PROVIDERS=$(check_omniauth)
      CURRENT_REDIRECT="$(current_redirect_uri)"
      if ! echo "$PROVIDERS" | grep -q "openid_connect"; then
        echo "❌ reconfigure 后仍未找到 openid_connect（当前: ${PROVIDERS:-<无>}）" >&2
        echo "   请检查 docker-compose.yml 中 GITLAB_OMNIBUS_CONFIG 的 omniauth_providers 配置。"
        exit 1
      fi
      if [[ -n "$OIDC_REDIRECT_URI" && "$CURRENT_REDIRECT" != "$OIDC_REDIRECT_URI" ]]; then
        echo "❌ reconfigure 后 redirect_uri 仍不匹配:" >&2
        echo "     当前: ${CURRENT_REDIRECT:-<空>}" >&2
        echo "     期望: $OIDC_REDIRECT_URI" >&2
        echo "   请确认容器环境变量 GITLAB_OIDC_REDIRECT_URI 已注入后重建容器。" >&2
        exit 1
      fi
      echo "✅ reconfigure 后 OmniAuth openid_connect 已对齐"
      echo "  live redirect_uri: $CURRENT_REDIRECT"
    else
      echo "❌ gitlab-ctl reconfigure 失败" >&2
      exit 1
    fi
  else
    echo "   提示：使用 --reconfigure 自动重载配置，或手动执行："
    echo "     docker exec $CONTAINER gitlab-ctl reconfigure"
  fi
fi

# ── 5. 显示 OmniAuth 摘要 ───────────────────────────────────────────────
echo "--- OmniAuth 设置摘要 ---"
docker exec "$CONTAINER" gitlab-rails runner "
  cfg = Gitlab.config.omniauth
  puts \"omniauth_enabled: #{cfg.enabled}\"
  puts \"providers: #{cfg.providers.map(&:name).join(',')}\"
  p = cfg.providers.find { |x| x.name.to_s == 'openid_connect' }
  if p
    args = p.args.is_a?(Hash) ? p.args : p.args.to_hash
    args = args.transform_keys(&:to_s) rescue args
    opts = args['client_options'] || {}
    opts = opts.transform_keys(&:to_s) rescue opts
    puts \"redirect_uri: #{opts['redirect_uri']}\"
  end
" 2>&1 || true

echo "=== OIDC 同步完成 ==="
