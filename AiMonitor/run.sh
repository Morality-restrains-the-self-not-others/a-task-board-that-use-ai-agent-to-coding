#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

if ! command -v docker >/dev/null 2>&1; then
  echo "错误: 未找到 docker，请先安装 Docker Desktop 或 Docker Engine。" >&2
  exit 1
fi

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
  elif docker-compose version >/dev/null 2>&1; then
    docker-compose "$@"
  else
    echo "错误: 未找到 Docker Compose（需 \`docker compose\` 或 \`docker-compose\`，Compose V2）。" >&2
    exit 1
  fi
}

# 镜像下载收敛到编译（build）阶段执行：runAll「编译」按钮 → bash AiMonitor/run.sh pull。
# 启动阶段不再拉取镜像，避免健康检查超时/页面卡在下载。
pull_images() {
  echo "正在拉取 Docker 镜像（编译阶段，避免启动时阻塞）..."
  compose pull --ignore-buildable --ignore-pull-failures "$@" 2>&1 || echo "警告: 部分镜像拉取失败，将尝试使用本地缓存。" >&2
}

ensure_images() {
  # 启动前检查本地镜像是否齐备；缺失时提示先执行编译（镜像下载不在启动阶段进行）。
  local images
  images="$(compose config --images 2>/dev/null || true)"
  if [[ -z "$images" ]]; then
    echo "警告: 无法枚举 compose 镜像列表（compose config --images 失败），跳过镜像检查。" >&2
    return 0
  fi
  local image missing=0
  while IFS= read -r image; do
    if ! docker image inspect "$image" >/dev/null 2>&1; then
      echo "本地缺少镜像: $image"
      missing=1
    fi
  done <<< "$images"
  if [[ "$missing" -eq 1 ]]; then
    echo "错误: 缺少上述镜像。请先在 runAll 页面点击「编译」（或执行 bash AiMonitor/run.sh pull）下载镜像后再启动。" >&2
    exit 1
  fi
}

mode="start"
if [[ $# -gt 0 ]]; then
  case "$1" in
    start|managed|stop|pull)
      mode="$1"
      shift
      ;;
  esac
fi

# ensure_port_free checks whether a TCP port is still in use after compose down
# and forcefully frees it by killing the process or container holding it.
# This prevents "address already in use" errors when a previous stack left
# behind orphaned processes or when a non-Compose process occupies the port.
ensure_port_free() {
  local port="$1"
  local label="${2:-port $port}"

  # Check if anything is listening on this port
  local pids
  pids=$(lsof -t -iTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
  if [[ -z "$pids" ]]; then
    return 0
  fi

  echo "端口 $port ($label) 仍被占用 (PID: $(echo "$pids" | tr '\n' ' '))，正在释放..."

  # Try to identify and stop any docker container using this port first
  local container_id
  container_id=$(docker ps -q --filter "publish=$port" 2>/dev/null || true)
  if [[ -n "$container_id" ]]; then
    echo "  停止占用端口 $port 的容器 $container_id..."
    docker stop "$container_id" >/dev/null 2>&1 || true
    docker rm -f "$container_id" >/dev/null 2>&1 || true
  fi

  # Kill any remaining processes holding the port
  for pid in $pids; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      echo "  终止进程 $pid (端口 $port)..."
      kill -TERM "$pid" 2>/dev/null || true
      sleep 0.3
      kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null || true
    fi
  done

  # Wait up to 3 seconds for the port to free
  local waited=0
  while [[ $waited -lt 30 ]]; do
    if ! lsof -t -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "端口 $port 已释放。"
      return 0
    fi
    sleep 0.1
    waited=$((waited + 1))
  done

  echo "警告: 端口 $port 在 3 秒后仍未释放，将继续尝试启动。" >&2
  return 1
}

cleanup_previous_stack() {
  # Ensure stale containers from previous detached starts are removed.
  echo "正在停止并清理上一次 AiMonitor 容器..."
  compose down --remove-orphans >/dev/null 2>&1 || true

  # Aggressively free critical ports that may be held by orphaned processes
  # or containers not tracked by this compose stack.  Port 9090 (Prometheus) is
  # especially prone to this after a SIGKILL or docker daemon restart.
  if ! command -v lsof >/dev/null 2>&1; then
    echo "警告: 未找到 lsof 命令，跳过端口级清理。" >&2
    echo "  若启动时出现 'address already in use' 错误，请手动释放端口或安装 lsof:" >&2
    echo "    Debian/Ubuntu: sudo apt install lsof" >&2
    echo "    RHEL/CentOS:   sudo yum install lsof" >&2
    echo "    Alpine:        sudo apk add lsof" >&2
    return 0
  fi

  local critical_ports="9090:Prometheus 3000:Grafana 3100:Loki 3200:Tempo 4317:OTel-gRPC 4318:OTel-HTTP 9115:Blackbox"
  for entry in $critical_ports; do
    local port="${entry%%:*}"
    local label="${entry##*:}"
    ensure_port_free "$port" "$label" || true
  done
}

generate_prometheus_targets() {
  if command -v python3 >/dev/null 2>&1; then
    echo "正在根据 runAll.yaml 生成 Prometheus 抓取目标..."
    python3 "$ROOT/scripts/generate_prometheus_from_runall.py"
  else
    echo "警告: 未找到 python3，跳过 runAll.yaml 目标自动生成。" >&2
  fi
}

generate_promtail_config() {
  local script="$ROOT/../runAll/scripts/generate-promtail-config.sh"
  if [[ -x "$script" ]] || [[ -f "$script" ]]; then
    echo "正在根据 runAll.yaml 生成 Promtail scrape 配置..."
    bash "$script"
  else
    echo "警告: 未找到 generate-promtail-config.sh，跳过 Promtail 配置生成。" >&2
  fi
}

build_daydaymoney_grafana_plugin() {
  local plugin_dir="$ROOT/../DaydaymoneyGrafana"
  local dist_js="$plugin_dir/dist/module.js"
  local src_dir="$plugin_dir/src"

  if [[ ! -d "$plugin_dir" ]]; then
    echo "警告: 未找到 DaydaymoneyGrafana 目录，跳过插件构建。" >&2
    return 0
  fi

  if [[ ! -f "$dist_js" ]] || [[ "$src_dir" -nt "$dist_js" ]] || [[ "$plugin_dir/package.json" -nt "$dist_js" ]]; then
    echo "正在构建 DaydaymoneyGrafana 插件（Grafana 11.5.x 兼容）..."
    if ! command -v npm >/dev/null 2>&1; then
      echo "警告: 未找到 npm，跳过 DaydaymoneyGrafana 插件构建（不影响监控栈核心功能）。" >&2
      return 0
    fi
    if [[ ! -d "$plugin_dir/node_modules" ]]; then
      (cd "$plugin_dir" && npm ci && npm run build) || {
        echo "警告: DaydaymoneyGrafana 插件构建失败（npm ci/build），跳过。监控栈核心功能不受影响。" >&2
        return 0
      }
    else
      (cd "$plugin_dir" && npm run build) || {
        echo "警告: DaydaymoneyGrafana 插件构建失败（npm build），跳过。监控栈核心功能不受影响。" >&2
        return 0
      }
    fi
  fi
}

sync_runall_log_root() {
  local runall_yaml="$ROOT/../conf/runAll.yaml"
  if [[ ! -f "$runall_yaml" ]]; then
    runall_yaml="$ROOT/../runAll.yaml"
  fi
  if [[ -z "${RUNALL_LOG_ROOT:-}" ]] && [[ -f "$runall_yaml" ]] && command -v python3 >/dev/null 2>&1; then
    local raw
    raw="$(python3 -c "
import sys
from pathlib import Path
sys.path.insert(0, str(Path('$ROOT/../runAll/scripts').resolve()))
from conf_local import overlay_conf_file
d = overlay_conf_file(Path('${runall_yaml}'))
print((d.get('logging') or {}).get('file_root') or '')
")"
    if [[ -n "$raw" ]]; then
      # 相对路径 → 以 runAll/ 为基准解析为项目根 logs/
      if [[ "$raw" = /* ]]; then
        RUNALL_LOG_ROOT="$raw"
      else
        RUNALL_LOG_ROOT="$ROOT/../logs"
      fi
    fi
  fi
  export RUNALL_LOG_ROOT="${RUNALL_LOG_ROOT:-$ROOT/../logs}"
  if [[ ! -d "$RUNALL_LOG_ROOT" ]]; then
    echo "警告: 日志目录 $RUNALL_LOG_ROOT 不存在。请先启动 runAll（tee 日志）再查 Loki。" >&2
  fi
}

print_ready_hint() {
  echo ""
  compose ps
  echo ""
  echo "已就绪:"
  echo "  Prometheus  http://localhost:9090"
  echo "  Blackbox    http://localhost:9115"
  echo "  Loki        http://localhost:3100"
  echo "  Tempo       http://localhost:3200  (query API)"
  echo "  OTel Coll.  grpc://localhost:4317  http://localhost:4318  → Tempo"
  echo "  Grafana     http://localhost:3000  （默认 admin / admin，可用 .env 覆盖）"
  echo "  日志采集    promtail-local (runAll 自动管理) → 远程 Loki :3100"
  echo "              （本 compose 栈不含 Promtail；tee 在 \${RUNALL_LOG_ROOT:-<project_root>/logs}）"
  echo ""
  echo "Grafana Dashboards:"
  echo "  Distributed Trace View — traceId 瀑布图 + 关联日志（Datadog 式）"
  echo "  Trace Log Journey / Trace Log Explore — 纯日志检索"
  echo ""
  echo "验收: python3 scripts/verify_trace_stack.py"
  echo ""
  if docker compose version >/dev/null 2>&1; then
    echo "查看日志: docker compose -f \"$ROOT/docker-compose.yaml\" logs -f"
  else
    echo "查看日志: docker-compose -f \"$ROOT/docker-compose.yaml\" logs -f"
  fi
}

ensure_dotenv() {
  if [[ ! -f "$ROOT/.env" ]]; then
    if [[ -f "$ROOT/.env.example" ]]; then
      echo "未找到 .env，从 .env.example 复制默认配置..."
      cp "$ROOT/.env.example" "$ROOT/.env"
    else
      echo "未找到 .env 或 .env.example，使用内建默认值（Grafana admin/admin）。" >&2
    fi
  fi
}

case "$mode" in
  stop)
    echo "正在停止 AiMonitor 容器..."
    compose down --remove-orphans "$@"
    ;;
  pull)
    ensure_dotenv
    pull_images "$@"
    ;;
  managed)
    ensure_dotenv
    sync_runall_log_root
    generate_prometheus_targets
    generate_promtail_config
    build_daydaymoney_grafana_plugin
    echo "正在清理上一次 AiMonitor 容器（若存在）..."
    cleanup_previous_stack
    ensure_images
    echo "正在以 detach 模式启动 AiMonitor（runAll 托管，镜像已在编译阶段下载，不再重复拉取）..."
    compose up -d --pull never --remove-orphans "$@"
    # 清空/重建栈后确保 Promtail 常驻（与 runAll promtail-local 对齐）
    if [[ -x "$ROOT/../runAll/scripts/runall-local-promtail.sh" ]]; then
      bash "$ROOT/../runAll/scripts/runall-local-promtail.sh" up || true
    fi
    print_ready_hint
    ;;
  *)
    ensure_dotenv
    sync_runall_log_root
    generate_prometheus_targets
    generate_promtail_config
    build_daydaymoney_grafana_plugin
    echo "正在清理上一次 AiMonitor 容器（若存在）..."
    cleanup_previous_stack
    ensure_images
    echo "正在启动 AiMonitor（Prometheus + Loki + Tempo + OTel Collector + Promtail + Grafana + Blackbox Exporter，镜像已在编译阶段下载，不再重复拉取）..."
    compose up -d --pull never "$@"
    if [[ -x "$ROOT/../runAll/scripts/runall-local-promtail.sh" ]]; then
      bash "$ROOT/../runAll/scripts/runall-local-promtail.sh" up || true
    fi
    print_ready_hint
    ;;
esac
