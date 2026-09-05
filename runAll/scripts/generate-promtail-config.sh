#!/usr/bin/env bash
# 从 runAll.yaml 提取服务列表 → 生成 promtail-local.yaml 的 per-service scrape_configs
# 用法: bash runAll/scripts/generate-promtail-config.sh [--dry-run]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNALL_YAML="${ROOT}/conf/runAll.yaml"
PROMTAIL_CONF="${ROOT}/AiMonitor/promtail/promtail-local.yaml"
DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

if [[ ! -f "$RUNALL_YAML" ]]; then
  echo "错误: 缺少 $RUNALL_YAML" >&2
  exit 1
fi

# 输出: 每行 "service_name|path1,path2,..."（path 相对 /var/log/runall）
# 主路径始终为 <service>.log；额外路径为历史旁路文件名，映射到同一 job，减少漏采。
SERVICE_PATHS=$(python3 -c "
import yaml
with open('${RUNALL_YAML}') as f:
    d = yaml.safe_load(f)
svcs = set()
for g in d.get('groups', []):
    for s in g.get('services', []):
        name = (s.get('name') or '').strip()
        if name:
            svcs.add(name)

# Status UI structured logs: appendStructuredLog('runall') tees to logs/runall.log.
# runAll is not a group service in conf/runAll.yaml, so it would otherwise be omitted.
svcs.add('runall')

# 旁路/旧路径 → 规范 runAll 服务名（仅作 scrape 兜底；新启动须写 logs/<service>.log）
ALIASES = {
    'go-relay': [
        'go_relayToTrae.restart.log',
        'go-relay/relay.restart.log',
        'go-relay/relay.log',
    ],
    'task-container-gateway': [
        'taskContainerGateway.restart.log',
        'task-container-gateway/gateway.log',
    ],
    'task-cloud-service': [
        'task-cloud-service-run.log',
        'task-cloud-service/task-cloud-service.log',
    ],
    'task-credential-service': [
        'taskCredentialService.restart.log',
    ],
    'task-project-service': [
        'task-project-service-run.log',
    ],
    'task-auth': [
        'task-auth/task-auth.log',
    ],
}

for name in sorted(svcs):
    paths = [f'{name}.log']
    for extra in ALIASES.get(name, []):
        if extra not in paths:
            paths.append(extra)
    print(name + '|' + ','.join(paths))
")

pipeline_stages() {
  cat <<'STAGES'
    pipeline_stages:
      # Accept both runAll-prefixed lines and raw slog JSON (taskEvents / manual tee).
      - regex:
          expression: '^(?:\S+ \((?:stdout|stderr)\) )?(?P<payload>.*)$'
      - json:
          expressions:
            ts: ts
            level: level
            msg: msg
            service: service
            trace_id: trace_id
            traceId: traceId
          source: payload
      # OPT-20260831-021: 用 JSON 的 ts 作为 Loki 时间戳。Seeked Offset:0 重刮时
      # 存量行不再按摄入时刻写入，Grafana 30m/1h volume 图不会在恢复前假平。
      - timestamp:
          source: ts
          format: RFC3339Nano
      # Plaintext fallback when JSON has no level (e.g. log.Fatalf "Config error: ...").
      - regex:
          expression: '(?i)\b(?P<plain_level>error|fatal|panic|warn(?:ing)?|info|debug)\b'
          source: payload
      - template:
          source: level
          template: '{{ if .level }}{{ .level }}{{ else if eq .plain_level "fatal" }}error{{ else if eq .plain_level "panic" }}error{{ else if eq .plain_level "warning" }}warn{{ else }}{{ .plain_level }}{{ end }}'
      - regex:
          expression: '\[trace_id=(?P<trace_id>[A-Za-z0-9._:-]{8,256})\]'
          source: payload
      - template:
          source: trace_id
          template: '{{ if .trace_id }}{{ .trace_id }}{{ else }}{{ .traceId }}{{ end }}'
      - regex:
          expression: '^[A-Za-z0-9._:-]{8,256}$'
          source: trace_id
      - labels:
          level:
          service:
      - structured_metadata:
          trace_id:
          msg:
      - output:
          source: payload
STAGES
}

SCRAPE_CONFIGS=""
while IFS='|' read -r svc paths_csv; do
  [[ -z "$svc" ]] && continue
  SCRAPE_CONFIGS+="
  - job_name: ${svc}
    static_configs:"
  IFS=',' read -r -a paths <<< "$paths_csv"
  for p in "${paths[@]}"; do
    SCRAPE_CONFIGS+="
      - targets:
          - localhost
        labels:
          job: ${svc}
          __path__: /var/log/runall/${p}"
  done
  SCRAPE_CONFIGS+=$'\n'
  SCRAPE_CONFIGS+="$(pipeline_stages)"
  SCRAPE_CONFIGS+=$'\n'
done <<< "$SERVICE_PATHS"

# APISIX 网关本地日志（access.log / error.log）：微信回调 502 等由 error_page
# 本地生成、不进入任何业务进程，只能按 X-Trace-Id 从网关日志检索。
# promtail 容器已挂载 ${APISIX_LOG_ROOT:-../taskGateway/logs} → /var/log/apisix。
SCRAPE_CONFIGS+="$(cat <<'APISIX_STAGES'
  - job_name: apisix-gateway
    static_configs:
      - targets:
          - localhost
        labels:
          job: task-gateway
          __path__: /var/log/apisix/access.log
      - targets:
          - localhost
        labels:
          job: task-gateway
          __path__: /var/log/apisix/error.log
      - targets:
          - localhost
        labels:
          job: task-gateway
          __path__: /var/log/apisix/taskgateway-access.log
    pipeline_stages:
      - regex:
          expression: '(?i)(?:x-trace-id|trace_id)["'']?\s*[=:]\s*["'']?(?P<trace_id>[A-Za-z0-9._:-]{8,256})'
      - regex:
          expression: '\[(?P<level>error|warn|crit|alert|emerg|notice|info)\]'
      - labels:
          level:
      - structured_metadata:
          trace_id:
APISIX_STAGES
)"

CONFIG=$(cat <<PROMTAIL
# Auto-generated by runAll/scripts/generate-promtail-config.sh
# DO NOT EDIT MANUALLY — regenerate: bash runAll/scripts/generate-promtail-config.sh
# Canonical path: /var/log/runall/<runAll-service-name>.log
# Alias paths (*.restart.log / *-run.log) map to the same job to reduce bypass漏采.
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /var/lib/promtail/positions.yaml

clients:
  - url: \${LOKI_PUSH_URL}

scrape_configs:${SCRAPE_CONFIGS}
PROMTAIL
)

SVC_COUNT=$(printf '%s\n' "$SERVICE_PATHS" | grep -c . || true)

if $DRY_RUN; then
  echo "$CONFIG"
  echo ""
  echo "---"
  echo "dry-run: 将写入 $PROMTAIL_CONF (${SVC_COUNT} 个服务)"
else
  echo "$CONFIG" > "$PROMTAIL_CONF"
  echo "已生成 $PROMTAIL_CONF (${SVC_COUNT} 个服务，含旁路别名路径)"
fi
