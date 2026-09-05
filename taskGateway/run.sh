#!/usr/bin/env bash
set -euo pipefail

# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# OPT-20260901-005: 同机双根（源码仓 + clone-run/DEPLOY_ROOT）时 docker compose 默认项目名
# 都是 taskgateway，`compose down` 会拆掉另一棵树的容器。DEPLOY_MODE=1 或 DEPLOY_ROOT 存在时
# 改用 taskgateway-deploy 隔离；显式 COMPOSE_PROJECT_NAME 始终优先。
# ⚠️ 同机双根时 :18081 仍只能有一方发布（两边端口映射相同），项目名隔离只避免误拆容器。
resolve_compose_project_name() {
  if [[ -n "${COMPOSE_PROJECT_NAME:-}" ]]; then
    printf '%s' "$COMPOSE_PROJECT_NAME"
    return 0
  fi
  if [[ "${DEPLOY_MODE:-}" == "1" || -n "${DEPLOY_ROOT:-}" ]]; then
    printf '%s' "taskgateway-deploy"
  else
    printf '%s' "taskgateway"
  fi
}

COMPOSE_PROJECT_NAME="$(resolve_compose_project_name)"
export COMPOSE_PROJECT_NAME
TASKGATEWAY_APISIX_CONTAINER="${TASKGATEWAY_APISIX_CONTAINER:-${COMPOSE_PROJECT_NAME}-apisix-1}"
export TASKGATEWAY_APISIX_CONTAINER

docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  else
    docker-compose "$@"
  fi
}

# APISIX (uid 636) creates logs/*.log as 0644; host cron cannot truncate them.
# After start/reload, widen mode so runAll/scripts/truncate-ram-work-logs.sh can
# in-place truncate without rm (container still appends fine to a+rw files).
ensure_logs_host_writable() {
  local container="$TASKGATEWAY_APISIX_CONTAINER"
  if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$container"; then
    return 0
  fi
  if docker exec "$container" sh -c 'chmod a+rw /usr/local/apisix/logs/*.log 2>/dev/null || true'; then
    echo "[taskGateway] logs/*.log mode set a+rw for host truncate (container=$container)"
  else
    echo "[taskGateway] warning: failed to chmod logs/*.log in $container" >&2
  fi
}

# Host-owned stale nginx.pid / worker_events.sock → EACCES for uid 636 (FE-20260717-APISIX-PID-EACCES).
# Missing bind-mount path: Docker creates logs/ as root:root 755; uid 636 cannot create error.log
# (nginx emerg, container Exited 1). Host chmod 777 often fails; fix the volume as root in Docker.
#
# apache/apisix standalone entrypoint (uid 636) rewrites the bind-mounted config:
#   echo "$(sed … config.yaml)" > config.yaml
# git/rsync 0644 + owner 1000 → Permission denied → Exited 1 → :18081 connection refused
# (clone-run / .daydaymoney-deploy-seed). Do not git-commit 0666; chmod only at start.
prepare_apisix_logs_for_start() {
  local log_dir="$ROOT/logs"
  mkdir -p "$log_dir"
  chmod 777 "$log_dir" 2>/dev/null || true
  docker run --rm --user 0 -v "$log_dir":/logs alpine sh -c \
    'chmod 777 /logs && (chmod a+rw /logs/*.log 2>/dev/null || true)' \
    || echo "[taskGateway] warning: docker chmod logs failed" >&2
  rm -f "$log_dir/nginx.pid" "$log_dir/worker_events.sock"
  echo "[taskGateway] prepared apisix logs dir (removed stale nginx.pid / worker_events.sock)"
  local conf_yaml="$ROOT/apisix/config.yaml"
  if [[ -f "$conf_yaml" ]]; then
    chmod a+rw "$conf_yaml" 2>/dev/null || true
    docker run --rm --user 0 -v "$conf_yaml":/cfg.yaml alpine chmod a+rw /cfg.yaml \
      || echo "[taskGateway] warning: docker chmod config.yaml failed" >&2
    echo "[taskGateway] prepared apisix/config.yaml for uid 636 standalone rewrite"
  fi
}

cmd="${1:-start}"

# OPT-20260901-006: clone-run 的网关在 envs/current/，conf-local 在部署根（再上两级）。
# 显式 DEPLOY_ROOT/CONF_ROOT 优先，否则向上找到含 conf/base.yaml 的根。与 Go confload
# 的 FindConfigRoot 语义一致，避免走错根导致机密漏叠（HTTP 200、密钥空、厂商 AUTH 失败）。
resolve_deploy_root() {
  if [[ -n "${DEPLOY_ROOT:-}" ]]; then
    printf '%s' "$DEPLOY_ROOT"
    return 0
  fi
  if [[ -n "${CONF_ROOT:-}" && -f "${CONF_ROOT}/base.yaml" ]]; then
    printf '%s' "$(cd "$(dirname "$CONF_ROOT")" && pwd)"
    return 0
  fi
  local walk="$ROOT"
  local i
  for i in 1 2 3 4 5 6 7 8; do
    if [[ -f "$walk/conf/base.yaml" ]]; then
      printf '%s' "$walk"
      return 0
    fi
    walk="$(dirname "$walk")"
  done
  printf '%s' ""
}

setup_tls() {
  local cert_dir="$ROOT/certs"
  mkdir -p "$cert_dir"
  local cert="$cert_dir/dev-gateway.pem"
  local key="$cert_dir/dev-gateway-key.pem"
  local deploy_root
  deploy_root="$(resolve_deploy_root)"
  local src_dir=""
  if [[ -n "$deploy_root" ]]; then
    src_dir="$deploy_root/conf-local/gateway/task-gateway"
  fi
  local src_cert="$src_dir/dev-gateway.pem"
  local src_key="$src_dir/dev-gateway-key.pem"
  if [[ -n "$src_dir" && -f "$src_cert" && -f "$src_key" ]]; then
    cp -a "$src_cert" "$cert"
    cp -a "$src_key" "$key"
    echo "[taskGateway] staged TLS from conf-local/gateway/task-gateway"
    return 0
  fi
  if [[ -f "$cert" && -f "$key" ]]; then
    return 0
  fi
  if [[ "${DEPLOY_MODE:-}" == "1" ]]; then
    echo "[taskGateway] missing conf-local/gateway/task-gateway/*.pem (DEPLOY_MODE=1: will not openssl)" >&2
    return 1
  fi
  local host="${INFRA_HOST:-localhost}"
  local san="DNS:localhost,IP:127.0.0.1"
  if [[ "$host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    san="${san},IP:${host}"
  else
    san="${san},DNS:${host}"
  fi
  echo "[taskGateway] generating dev TLS cert in $cert_dir CN=$host"
  openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$key" -out "$cert" -days 825 \
    -subj "/CN=${host}" \
    -addext "subjectAltName=${san}"
}

routes_apply() {
  export TASK_GATEWAY_APISIX_IN_DOCKER="${TASK_GATEWAY_APISIX_IN_DOCKER:-1}"

  # OPT-20260901-006: clone-run 时 REPO=envs/current，机密 overlay 在部署根 conf-local。
  # source 部署根 cutover.env 并显式导出 DEPLOY_ROOT/CONF_ROOT，让 routes-to-apisix.py
  # 及其它 ROOT.parent 生成器拿到部署根——只靠向上 walk 会在中间误放非空 conf-local 时提前停。
  local deploy_root
  deploy_root="$(resolve_deploy_root)"
  if [[ -n "$deploy_root" ]]; then
    export DEPLOY_ROOT="$deploy_root"
    export CONF_ROOT="${CONF_ROOT:-$deploy_root/conf}"
    if [[ -f "$deploy_root/cutover.env" ]]; then
      # shellcheck source=/dev/null
      source "$deploy_root/cutover.env"
      echo "[taskGateway] sourced $deploy_root/cutover.env"
    fi
  fi

  # Content-based freshness (not mtime): mtime skip left loopback upstreams when
  # docker.upstreamHost / env changed while routes.yaml was unchanged — causing
  # container→127.0.0.1 Connection refused and readiness HTTP 502.
  #
  # ⚠️ bind-mount inode 陷阱（OPT-20260806-023）：apisix.yaml 以 bind mount 挂载进容器；
  # 若此前在宿主机用 git checkout / 原子写（rename）替换了该文件，容器内挂载点仍指向旧 inode，
  # 本脚本的 --check 基于宿主机文件内容会判定「陈旧」并重生成 —— 因此变更 routes.yaml 后
  # 必须走本函数（内容式新鲜度 + 重生成 + reload），不要 git checkout 后手工 exec reload。
  if python3 "$ROOT/scripts/routes-to-apisix.py" --check >/dev/null 2>&1; then
    echo "[taskGateway] apisix.yaml is up-to-date — skipping regeneration"
  else
    echo "[taskGateway] regenerating apisix.yaml from routes.yaml..."
    python3 "$ROOT/scripts/routes-to-apisix.py"
  fi

  # If APISIX is running, hot-reload the config.
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$TASKGATEWAY_APISIX_CONTAINER"; then
    echo "[taskGateway] hot-reloading APISIX routes..."
    docker exec "$TASKGATEWAY_APISIX_CONTAINER" apisix reload 2>/dev/null || true
  fi
}

sync_apisix_file() {
  local src="$1" dst_name="$2" dst_uid="${3:-636}"
  # Suppress errors with || true so script continues even if remote sync fails
  scp -q "$src" "zcpu:/tmp/$dst_name" 2>/dev/null || echo "[taskGateway] warning: failed to scp $dst_name to zcpu (ignoring)" >&2
  ssh zcpu "docker run --rm -v /Users/task2app/gitClone/ramDisk/ram-mount/taskGateway/apisix:/target -v /tmp:/src alpine sh -c \"cp /src/$dst_name /target/$dst_name && chown $dst_uid:$dst_uid /target/$dst_name\"" 2>/dev/null || echo "[taskGateway] warning: failed to sync $dst_name via docker on zcpu (ignoring)" >&2
}

case "$cmd" in
  setup-tls)
    setup_tls
    ;;
  prepare-host-mounts)
    prepare_apisix_logs_for_start
    ;;
  routes-apply)
    routes_apply
    ;;
  runtime-identities)
    # 供测试/排障查询：compose 项目名与 APISIX 容器名（OPT-20260901-005）。
    printf 'COMPOSE_PROJECT_NAME=%s\nTASKGATEWAY_APISIX_CONTAINER=%s\n' \
      "$COMPOSE_PROJECT_NAME" "$TASKGATEWAY_APISIX_CONTAINER"
    ;;
  resolve-deploy-root)
    # 供测试/排障查询：部署根解析（OPT-20260901-006）。
    printf '%s\n' "$(resolve_deploy_root)"
    ;;
  start)
    setup_tls
    routes_apply
    prepare_apisix_logs_for_start
    # sync_apisix_file "$ROOT/apisix/config.yaml" "config.yaml"
    # sync_apisix_file "$ROOT/apisix/apisix.yaml" "apisix.yaml"
    docker_compose -f "$ROOT/docker-compose.yml" up -d
    echo "[taskGateway] Docker containers started"
    # compose down→up 后 APISIX init_worker 实测可到 ~140s；默认 180s。
    # 未就绪默认只告警（容器已 up），由 runAll health_check 做最终判定。
    # TASKGATEWAY_APISIX_READY_FAIL=1 才 exit 1（本地排障）。
    apisix_health_url="${TASKGATEWAY_APISIX_HEALTH_URL:-http://127.0.0.1:18081/api/health/}"
    apisix_ready_timeout="${TASKGATEWAY_APISIX_READY_TIMEOUT_SEC:-180}"
    apisix_container="$TASKGATEWAY_APISIX_CONTAINER"
    if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$apisix_container"; then
      echo "[taskGateway] ERROR: APISIX container not running after compose up (not a slow init)" >&2
      docker_compose -f "$ROOT/docker-compose.yml" logs --tail=80 apisix >&2 || true
      apisix_ready_timeout=1
    fi
    apisix_ready=false
    for i in $(seq 1 "$apisix_ready_timeout"); do
      if curl -sf --max-time 2 "$apisix_health_url" >/dev/null 2>&1; then
        apisix_ready=true
        echo "[taskGateway] APISIX ready ($apisix_health_url) after ${i}s"
        break
      fi
      sleep 1
    done
    if [[ "$apisix_ready" != true ]]; then
      echo "[taskGateway] WARNING: APISIX not ready at $apisix_health_url after ${apisix_ready_timeout}s (compose already up)" >&2
      if [[ -f "$ROOT/logs/error.log" ]]; then
        echo "=== taskGateway APISIX error.log (last 40 lines) ===" >&2
        tail -40 "$ROOT/logs/error.log" >&2 || true
        echo "=== end error.log ===" >&2
      fi
      if [[ "${TASKGATEWAY_APISIX_READY_FAIL:-0}" == "1" ]]; then
        exit 1
      fi
    fi
    # Output container logs to stdout so runAll can capture them
    echo "=== taskGateway container logs (last 50 lines) ==="
    docker_compose -f "$ROOT/docker-compose.yml" logs --tail=50 2>&1 || true
    echo "=== end container logs ==="
    # Also output APISIX error log if it exists (for debugging plugin loading issues)
    # Wait briefly for error.log to be created and have content in case APISIX started but hasn't written to it yet
    for i in {1..10}; do
      if [[ -f "$ROOT/logs/error.log" ]] && [[ -s "$ROOT/logs/error.log" ]]; then
        break
      fi
      sleep 1
    done
    if [[ -f "$ROOT/logs/error.log" ]] && [[ -s "$ROOT/logs/error.log" ]]; then
      echo "=== taskGateway APISIX error.log (last 30 lines) ==="
      tail -30 "$ROOT/logs/error.log" 2>/dev/null || true
      echo "=== end error.log ==="
    else
      echo "[taskGateway] warning: error.log not found or empty after waiting"
    fi
    ensure_logs_host_writable
    echo "[taskGateway] https://${INFRA_HOST:-localhost}:18444 (HTTP :18081)"
    ;;
  stop)
    docker_compose -f "$ROOT/docker-compose.yml" down
    ;;
  reload)
    routes_apply
    prepare_apisix_logs_for_start
    # sync_apisix_file "$ROOT/apisix/apisix.yaml" "apisix.yaml"
    docker_compose -f "$ROOT/docker-compose.yml" restart apisix
    echo "=== taskGateway apisix restart logs ==="
    docker_compose -f "$ROOT/docker-compose.yml" logs --tail=20 apisix 2>&1 || true
    echo "=== end restart logs ==="
    # Wait for APISIX to recreate log files, then widen permissions for host truncate.
    for i in {1..10}; do
      if [[ -f "$ROOT/logs/error.log" ]]; then
        break
      fi
      sleep 1
    done
    ensure_logs_host_writable
    # ssh zcpu "docker restart taskGateway-apisix-1" 2>/dev/null || true
    ;;
  routes-watch)
    # Watch routes.yaml for changes and auto-apply.
    echo "[taskGateway] watching $ROOT/routes/routes.yaml for changes..."
    if ! command -v inotifywait &>/dev/null; then
      echo "[taskGateway] inotify-tools not installed — polling every 5s instead"
      local last_mtime
      last_mtime=$(stat -c %Y "$ROOT/routes/routes.yaml" 2>/dev/null || echo 0)
      while true; do
        sleep 5
        local curr_mtime
        curr_mtime=$(stat -c %Y "$ROOT/routes/routes.yaml" 2>/dev/null || echo 0)
        if [[ "$curr_mtime" -gt "$last_mtime" ]]; then
          echo "[taskGateway] routes.yaml changed — applying..."
          routes_apply
          last_mtime="$curr_mtime"
        fi
      done
    else
      while inotifywait -q -e modify,create "$ROOT/routes/routes.yaml" 2>/dev/null; do
        sleep 0.3  # debounce
        echo "[taskGateway] routes.yaml changed — applying..."
        routes_apply
      done
    fi
    ;;
  *)
    echo "usage: $0 {start|stop|reload|setup-tls|prepare-host-mounts|routes-apply|routes-watch}" >&2
    exit 1
    ;;
esac
