#!/usr/bin/env bash
# 从 INFRA 机以 SSOT 驱动部署/更新 Host sh 上的 tencent-sh-1 GitLab CE（OPT-20260823-061）。
#
# 职责（替代手工 scp + compose up）：
#   1. 渲染 SSOT conf/infra/git-service-tencent-sh-1/ → 区域 .env（含 taskAuth OIDC bootstrap）
#   2. scp docker-compose.yml + .env + initializers/（zzz_*.rb）到 Host sh
#   3. ssh "docker compose up -d" 幂等应用（无配置变更时容器不重建）
#
# SSOT 原则：区域可调参数只改 conf/infra/git-service-tencent-sh-1/config.yaml；
# compose 配方在 gitService/docker-compose.tencent-sh-1.yml。
# 本脚本保证远端 compose 目录与 SSOT 一致，杜绝手工复制漂移。
#
# 用法:
#   deploy_tencent_sh_1_from_infra.sh            # 实际部署（生产 GitLab 重启风险，选低峰）
#   deploy_tencent_sh_1_from_infra.sh dry-run    # 只打印渲染 .env 与将执行的命令，不写远端
#   deploy_tencent_sh_1_from_infra.sh diff       # 对比本地渲染与远端现状（只读，安全）
#
# 环境变量覆盖: GITSERVICE_SH_SSH_HOST (默认 sh)、GITSERVICE_SH_COMPOSE_DIR (默认 /opt/daydaymoney/gitservice-tencent-sh-1)
set -euo pipefail
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GITSERVICE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
WORKSPACE_ROOT="$(cd "$GITSERVICE_DIR/.." && pwd)"

APP="git-service-tencent-sh-1"
SSOT_DIR="$WORKSPACE_ROOT/conf/infra/$APP"
SSOT_COMPOSE="$GITSERVICE_DIR/docker-compose.tencent-sh-1.yml"
SSOT_CONFIG="$SSOT_DIR/config.yaml"
REMOTE_DIR="${GITSERVICE_SH_COMPOSE_DIR:-/opt/daydaymoney/gitservice-tencent-sh-1}"
SSH_HOST="${GITSERVICE_SH_SSH_HOST:-sh}"
SSH_BIN="${GITSERVICE_SH_SSH_BIN:-ssh}"
SCP_BIN="${GITSERVICE_SH_SCP_BIN:-scp}"
CONNECT_TIMEOUT="${GITSERVICE_SH_SSH_CONNECT_TIMEOUT:-15}"
COMPOSE_PROJECT="gitservice-tencent-sh-1"

MODE="${1:-deploy}"
case "$MODE" in
  deploy|dry-run|diff) ;;
  *) echo "usage: $0 [deploy|dry-run|diff]" >&2; exit 2 ;;
esac

# 远端路径与主机由运维控制；拒绝元字符，保证 scp/ssh 载荷只有 cd + compose。
case "$REMOTE_DIR" in
  *[!A-Za-z0-9._/-]* | "")
    echo "error: GITSERVICE_SH_COMPOSE_DIR has unsafe characters: $REMOTE_DIR" >&2
    exit 2
    ;;
esac
case "$SSH_HOST" in
  *[!A-Za-z0-9._-]* | "")
    echo "error: GITSERVICE_SH_SSH_HOST has unsafe characters: $SSH_HOST" >&2
    exit 2
    ;;
esac

[[ -f "$SSOT_COMPOSE" ]] || { echo "error: SSOT compose missing: $SSOT_COMPOSE" >&2; exit 2; }
[[ -f "$SSOT_CONFIG" ]] || { echo "error: SSOT config missing: $SSOT_CONFIG" >&2; exit 2; }

# ── 1. 渲染区域 .env（SSOT config.yaml 模板解析 + taskAuth OIDC bootstrap）──
render_env() {
  python3 - "$SSOT_CONFIG" "$WORKSPACE_ROOT" <<'PY'
import json
import os
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("error: python3-yaml required", file=sys.stderr)
    raise SystemExit(3)

main_yaml, workspace_root = sys.argv[1], sys.argv[2]
root = Path(workspace_root)
scripts = str(root / "runAll" / "scripts")
if scripts not in sys.path:
    sys.path.insert(0, scripts)
from conf_local import overlay_conf_file

# 与 sync_omniauth_oidc.sh 同源的模板展开（base.yaml → scheme/baseDomain/subdomains）。
base = overlay_conf_file(root / "conf" / "base.yaml") or {}
scheme = "https"
base_domain = str(base.get("baseDomain") or "daydaymoney.com")
m = {"scheme": scheme, "baseDomain": base_domain}
for k, tmpl in (base.get("subdomains") or {}).items():
    m[f"subdomains.{k}"] = str(tmpl).replace("${scheme}", scheme).replace("${baseDomain}", base_domain)


def expand(value: str) -> str:
    out = str(value)
    for k, v in m.items():
        out = out.replace("${" + k + "}", v)
    out = re.sub(
        r"\$\{(\w+):-([^}]*)\}",
        lambda mo: os.environ.get(mo.group(1), mo.group(2) or ""),
        out,
    )
    return out


# (a) SSOT config.yaml → 解析后 JSON → load_gitservice_config JSON 分支（避免模板被剥离）。
config = overlay_conf_file(Path(main_yaml))
resolved = {k: expand(v) if isinstance(v, str) else v for k, v in config.items()}

# load_gitservice_config.resolve(config_json, main_yaml)
sys.path.insert(0, str(root / "gitService/scripts"))
import load_gitservice_config as lgc  # noqa: E402

rendered = lgc.resolve(json.dumps(resolved), Path(main_yaml))
env = dict(rendered)

# (b) 从 task-auth OIDC bootstrap 取 issuer/client_secret/redirect_uri（gitlab-git-service-tencent-sh-1）。
oidc_issuer = ""
oidc_secret = ""
oidc_redirect = ""
ta_path = root / "conf" / "auth" / "task-auth" / "config.yaml"
if ta_path.is_file():
    ta = overlay_conf_file(ta_path) or {}
    oidc = ta.get("oidc") or {}
    target = str(resolved.get("oidcClientId") or "gitlab-git-service-tencent-sh-1")
    for client in oidc.get("bootstrapClients") or []:
        if str(client.get("clientId") or "") == target:
            oidc_secret = str(client.get("clientSecret") or "")
            oidc_redirect = expand(str(client.get("redirectUri") or "").strip())
            break
    oidc_issuer = expand(str(oidc.get("issuer") or "").strip())
if not oidc_issuer:
    oidc_issuer = env.get("GITLAB_OIDC_ISSUER", "") or expand(str(m.get("subdomains.gateway", "https://api.daydaymoney.com")))
if not oidc_secret:
    oidc_secret = ""
if not oidc_redirect:
    oidc_redirect = f"https://{env['GITLAB_HOSTNAME']}/users/auth/openid_connect/callback"

# (c) 组装最终 .env（compose 插值所需全集，排序保证确定性）。
env["GITLAB_HOME"] = env.get("GITLAB_HOME_FROM_CONF", "/var/lib/daydaymoney/gitService-tencent-sh-1")
env["GITLAB_OIDC_ISSUER"] = oidc_issuer
env["GITLAB_OIDC_CLIENT_SECRET"] = oidc_secret
env["GITLAB_OIDC_REDIRECT_URI"] = oidc_redirect
env["TRAE_TASKBILL_INTERNAL_SECRET"] = env.get("TRAE_TASKBILL_INTERNAL_SECRET", "")
env["TRAE_GITLAB_INTRANET_HOSTS"] = env.get("TRAE_GITLAB_INTRANET_HOSTS", "")
env["COMPOSE_PROJECT_NAME"] = env.get("COMPOSE_PROJECT_NAME", "gitservice-tencent-sh-1")

for key in ("GITLAB_HOME_FROM_CONF", "GITLAB_MIN_DOCKER_MEMORY_MIB", "GITLAB_ALLOWED_HOST_URL", "GITLAB_DISPLAY_HOST"):
    env.pop(key, None)

for key in sorted(env):
    if not key or key.startswith("#"):
        continue
    print(f"{key}={env[key]}")
PY
}

ENV_CONTENT="$(render_env)"
ENV_FILE="$(mktemp)"
trap 'rm -f "$ENV_FILE"' EXIT
printf '%s\n' "$ENV_CONTENT" > "$ENV_FILE"

# 泄漏守卫：渲染结果不得残留未解析模板变量。
if grep -q '\${' "$ENV_FILE"; then
  echo "error: rendered .env still contains unresolved template var:" >&2
  grep '\${' "$ENV_FILE" >&2 || true
  exit 2
fi

INITIALIZER_FILES=(
  "$GITSERVICE_DIR/initializers/zzz_fix_oidc_http.rb"
  "$GITSERVICE_DIR/initializers/zzz_fix_oidc_traceid.rb"
  "$GITSERVICE_DIR/initializers/zzz_trae_gitlab_traffic_quota.rb"
)
for f in "${INITIALIZER_FILES[@]}"; do
  [[ -f "$f" ]] || { echo "error: initializer missing: $f" >&2; exit 2; }
done

echo "rendered env:" >&2
cat "$ENV_FILE" >&2

# ── 2. dry-run：只打印将要执行的动作 ──
if [[ "$MODE" == "dry-run" ]]; then
  echo "--- dry-run commands ---"
  echo "$SCP_BIN -o BatchMode=yes -o ConnectTimeout=$CONNECT_TIMEOUT $SSOT_COMPOSE $SSH_HOST:$REMOTE_DIR/docker-compose.yml"
  echo "$SCP_BIN -o BatchMode=yes -o ConnectTimeout=$CONNECT_TIMEOUT $ENV_FILE $SSH_HOST:$REMOTE_DIR/.env"
  for f in "${INITIALIZER_FILES[@]}"; do
    echo "$SCP_BIN -o BatchMode=yes -o ConnectTimeout=$CONNECT_TIMEOUT $f $SSH_HOST:$REMOTE_DIR/initializers/$(basename "$f")"
  done
  echo "$SSH_BIN -o BatchMode=yes -o ConnectTimeout=$CONNECT_TIMEOUT $SSH_HOST 'cd $REMOTE_DIR && docker compose up -d'"
  exit 0
fi

# ── 3. diff：只读对比本地渲染与远端现状（不写远端）──
if [[ "$MODE" == "diff" ]]; then
  remote_env="$("$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" "cat $REMOTE_DIR/.env" 2>/dev/null || true)"
  remote_compose="$("$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" "cat $REMOTE_DIR/docker-compose.yml" 2>/dev/null || true)"
  local_compose="$(cat "$SSOT_COMPOSE")"

  changed=0
  if [[ "$local_compose" != "$remote_compose" ]]; then
    echo "DRIFT docker-compose.yml (local SSOT != remote)" >&2
    changed=1
  else
    echo "match docker-compose.yml" >&2
  fi

  if diff -u <(printf '%s\n' "$remote_env") "$ENV_FILE" >&2; then
    echo "match .env" >&2
  else
    echo "DRIFT .env (remote != SSOT render) see diff above" >&2
    changed=1
  fi

  if [[ "$changed" -eq 0 ]]; then
    echo "diff: no drift" >&2
    exit 0
  fi
  echo "diff: drift found (read-only report)" >&2
  exit 1
fi

# ── 4. deploy：scp + compose up -d（幂等）──
REMOTE_INIT_DIR="$REMOTE_DIR/initializers"
"$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" "mkdir -p '$REMOTE_INIT_DIR'"

"$SCP_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSOT_COMPOSE" "$SSH_HOST:$REMOTE_DIR/docker-compose.yml"
"$SCP_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$ENV_FILE" "$SSH_HOST:$REMOTE_DIR/.env"
for f in "${INITIALIZER_FILES[@]}"; do
  "$SCP_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$f" "$SSH_HOST:$REMOTE_INIT_DIR/$(basename "$f")"
done

echo "deploy: applying compose up -d on $SSH_HOST:$REMOTE_DIR" >&2
exec "$SSH_BIN" -o BatchMode=yes -o ConnectTimeout="$CONNECT_TIMEOUT" "$SSH_HOST" \
  "cd $REMOTE_DIR && docker compose up -d"
