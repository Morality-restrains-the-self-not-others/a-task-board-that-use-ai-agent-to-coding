#!/usr/bin/env bash
set -euo pipefail
# App processes must not inherit shell HTTP(S)_PROXY (dev-only network accel).
# See .ai/01_project_constraints/23_app_startup_no_env_proxy.md
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"
# shellcheck source=../scripts/resolve_infra_conf.sh
source "$ROOT/../scripts/resolve_infra_conf.sh"
KAFKA_CONF="$(resolve_infra_conf kafka)"
if [[ -f "$KAFKA_CONF" ]]; then
  _kafka_host="$(python3 "$ROOT/../scripts/infra_conf_host.py" "$KAFKA_CONF")"
  export INFRA_HOST="${INFRA_HOST:-$_kafka_host}"
  echo "[docker-kafka] conf=${KAFKA_CONF} INFRA_HOST=${INFRA_HOST} (PLAINTEXT_HOST advertised)"
else
  echo "[docker-kafka] missing $KAFKA_CONF" >&2
  exit 1
fi

COMPOSE_FILE="docker-compose.yml"

readonly -a REQUIRED_IMAGES=(
  "confluentinc/cp-zookeeper:7.4.0"
  "confluentinc/cp-kafka:7.4.0"
  "provectuslabs/kafka-ui:v0.7.2"
)

if ! command -v docker >/dev/null 2>&1; then
  echo "错误: 未找到 docker，请先安装 Docker Desktop 或 Docker Engine。" >&2
  exit 1
fi

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$COMPOSE_FILE" "$@"
  elif docker-compose version >/dev/null 2>&1; then
    docker-compose -f "$COMPOSE_FILE" "$@"
  else
    echo "错误: 未找到 Docker Compose（需 docker compose 或 docker-compose）。" >&2
    exit 1
  fi
}

image_present() {
  docker image inspect "$1" >/dev/null 2>&1
}

ensure_images() {
  local missing=0
  for image in "${REQUIRED_IMAGES[@]}"; do
    if ! image_present "$image"; then
      missing=1
      echo "本地缺少镜像: $image"
    fi
  done
  if [[ "$missing" -eq 1 ]]; then
    echo "正在拉取缺失的 Kafka 栈镜像（仅一次）..."
    compose pull
  else
    echo "本地镜像已齐，跳过 pull。"
  fi
}

up_stack() {
  echo "正在启动 Kafka 栈（Zookeeper/Kafka/Kafka UI，不重复 pull）..."
  compose up -d --pull never --remove-orphans

  # 等待 Kafka broker 真正就绪（而不仅是容器 running）
  echo "等待 Kafka broker 健康检查通过..."
  local max_wait=120
  local elapsed=0
  while [[ $elapsed -lt $max_wait ]]; do
    if docker exec kafka-kafka-1 kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; then
      echo "Kafka broker 就绪 (耗时 ${elapsed}s)"
      break
    fi
    sleep 5
    elapsed=$((elapsed + 5))
  done
  if [[ $elapsed -ge $max_wait ]]; then
    echo "警告: Kafka broker 在 ${max_wait}s 内未就绪，但容器已启动。请检查 docker logs kafka-kafka-1" >&2
  fi
}

down_stack() {
  echo "正在停止 Kafka 栈容器..."
  compose down --remove-orphans "$@"
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

case "$mode" in
  stop)
    down_stack "$@"
    ;;
  managed|start)
    ensure_images
    up_stack
    compose ps
    ;;
esac
