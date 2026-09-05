#!/usr/bin/env bash
# Ensure SSH reverse tunnels to the edge server (sh) are up.
# Edge nginx proxies api.*.daydaymoney.com → 127.0.0.1:<port> on the edge host;
# these tunnels map those ports back to the local dev machine.
# Idempotent: starts missing tunnels only. Safe to run from cron every few minutes.
# 8012 = GitLab CE (gitlab.daydaymoney.com)。2026-08-07 曾因清单缺失 + ExitOnForwardFailure
# 导致隧道中断后 5 分钟 cron 无法自愈，线上 502 持续数小时。勿从清单移除。
#
# 2026-09-01: pgrep 必须锚定 /usr/lib/autossh/autossh（禁止 autossh.*-R 误匹配
# Agent bash -c 诊断命令）。sshd LISTEN 仍在但 channel 半开时，对 SH
# 127.0.0.1:<port> 做 curl 探活，超时/拒绝则杀掉远端监听并重建（否则
# provider.daydaymoney.com 会 TLS 成功后 HTTP 挂死）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HEALTH_PY="${SCRIPT_DIR}/edge_tunnel_health.py"

# Loopback tunnels: edge nginx reaches 127.0.0.1:<port> on sh.
PORTS="${EDGE_TUNNEL_PORTS:-18081 8002 8003 8010 8011 8012 8013 8015}"
# Docker0-bind tunnels: reachable from containers via host-gateway (172.17.0.1)
# but NOT exposed to the public internet (needs GatewayPorts clientspecified).
# 8004 = taskBill (TRAE_TASKBILL_BASE for gitlab-tencent-sh-1 on sh, 2026-08-23).
DOCKER_BIND_PORTS="${EDGE_TUNNEL_DOCKER_PORTS:-8004}"
# Public tunnels: bind 0.0.0.0:<port> on sh (needs GatewayPorts clientspecified).
# 22 = GitLab gitlab-shell on this host (git@gitlab.daydaymoney.com → sh:22 → here).
PUBLIC_PORTS="${EDGE_TUNNEL_PUBLIC_PORTS:-22}"
LOG_DIR="${TUNNEL_LOG_DIR:-$HOME/.edge-tunnels}"
mkdir -p "$LOG_DIR"

autossh_running() {
  local bind="$1" p="$2"
  pgrep -f "/usr/lib/autossh/autossh .*-R ${bind}${p}:127.0.0.1:${p}" >/dev/null 2>&1
}

# Probe SH listener. Returns: 0=up, 1=down, 2=ssh unreachable (do not flap).
probe_remote_http() {
  local probe_host="$1" p="$2"
  local out curl_e http_code
  if ! out="$(ssh -o BatchMode=yes -o ConnectTimeout=5 sh \
    "curl -sS -m 2 -o /dev/null -w '%{http_code}' http://${probe_host}:${p}/; printf '\\nCURLE:%s' \"\$?\"" \
    2>/dev/null)"; then
    return 2
  fi
  http_code="$(printf '%s\n' "$out" | sed -n '1p')"
  curl_e="$(printf '%s\n' "$out" | sed -n 's/^CURLE://p' | tail -n1)"
  curl_e="${curl_e:-99}"
  python3 "$HEALTH_PY" probe-ok "$curl_e" "$http_code"
}

kill_stale() {
  local bind="$1" p="$2" probe_host="$3"
  pkill -f "/usr/lib/autossh/autossh .*-R ${bind}${p}:127.0.0.1:${p}" 2>/dev/null || true
  ssh -o BatchMode=yes -o ConnectTimeout=5 sh "fuser -k ${p}/tcp" >/dev/null 2>&1 || true
  echo "$(date '+%F %T') killed stale tunnel bind=${bind} port=${p} host=${probe_host}" >> "$LOG_DIR/tunnel-$p.log"
}

start_tunnel() {
  local bind="$1" p="$2"
  echo "$(date '+%F %T') starting tunnel for :$p (bind $bind)" >> "$LOG_DIR/tunnel-$p.log"
  nohup autossh -M 0 -N \
    -o "ServerAliveInterval 30" \
    -o "ServerAliveCountMax 3" \
    -o "ExitOnForwardFailure yes" \
    -o "ConnectTimeout 8" \
    -R "${bind}${p}:127.0.0.1:${p}" sh \
    >> "$LOG_DIR/tunnel-$p.log" 2>&1 &
}

ensure_http_tunnel() {
  local bind="$1" p="$2" probe_host="$3"
  local running=0
  if autossh_running "$bind" "$p"; then
    running=1
  fi
  if [[ "$running" -eq 1 ]]; then
    local probe_rc=0
    set +e
    probe_remote_http "$probe_host" "$p"
    probe_rc=$?
    set -e
    if [[ "$probe_rc" -eq 0 ]]; then
      return 0
    fi
    if [[ "$probe_rc" -eq 2 ]]; then
      echo "$(date '+%F %T') WARN tunnel_probe_ssh_failed bind=${bind} port=${p} action=skip" >> "$LOG_DIR/alerts.log"
      return 0
    fi
    echo "$(date '+%F %T') WARN tunnel_half_open bind=${bind} port=${p} action=restart" >> "$LOG_DIR/alerts.log"
    kill_stale "$bind" "$p" "$probe_host"
    sleep 1
  else
    echo "$(date '+%F %T') WARN tunnel_down bind=${bind} port=${p} action=restart" >> "$LOG_DIR/alerts.log"
  fi
  start_tunnel "$bind" "$p"
}

ensure_process_only() {
  local bind="$1" p="$2"
  if autossh_running "$bind" "$p"; then
    return 0
  fi
  echo "$(date '+%F %T') WARN tunnel_down bind=${bind} port=${p} action=restart" >> "$LOG_DIR/alerts.log"
  start_tunnel "$bind" "$p"
}

for p in $PORTS; do
  ensure_http_tunnel "" "$p" "127.0.0.1"
done
for p in $DOCKER_BIND_PORTS; do
  ensure_http_tunnel "172.17.0.1:" "$p" "172.17.0.1"
done
for p in $PUBLIC_PORTS; do
  ensure_process_only "0.0.0.0:" "$p"
done
