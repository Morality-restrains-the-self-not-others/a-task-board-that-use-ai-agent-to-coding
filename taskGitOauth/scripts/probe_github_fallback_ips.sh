#!/usr/bin/env bash
# 探测 github.com 拨号 fallback IP 健康度（OPT-20260811-049）。
#
# 背景：2026-08-11 exchange_failed 根因是本机 DNS 返回的 github.com 边缘不可达，
# taskGitOauth 改用内置 fallback IP 串行拨号（infrastructure/outbound_github_dial.go）。
# CDN 边缘 IP 会漂移，列表可能过期；本脚本逐 IP 探测 /login/oauth/access_token 的
# TLS+HTTP 可达性，供人工或 cron / AiMonitor 定时巡检。
#
# 判定：curl --resolve github.com:443:<IP> 走指定 IP 建连。POST 到 OAuth 换票端点，
# 无凭据时返回 4xx（HTTP 层通）；TLS 挂起/超时则判不可达。注意部分 IP 可能 TCP/TLS
# 成功但 HTTP 挂起（见 outbound_github_dial.go 注释）——因此以「收到 HTTP 状态码」为准。
#
# 用法：
#   bash taskGitOauth/scripts/probe_github_fallback_ips.sh
#   GITOAUTH_GITHUB_DIAL_FALLBACK_IPS=1.2.3.4,5.6.7.8 bash .../probe_github_fallback_ips.sh
#
# 退出码：0 = 至少一个 IP 可达；1 = 全部不可达（适合 cron 告警）。
# 对接 AiMonitor 示例（可选）：全部不可达时输出 JSON 行含 status=critical，供采集告警。
set -euo pipefail
# 探测走直连，避免被 dev HTTP(S)_PROXY 劫持（对齐 23_app_startup_no_env_proxy）。
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCE_FILE="${ROOT}/taskGitOauth/infrastructure/outbound_github_dial.go"

CURL_TIMEOUT_SEC="${CURL_TIMEOUT_SEC:-8}"
GITHUB_RESOLVE_HOST="github.com"
PROBE_URL="https://github.com/login/oauth/access_token"

# 解析 IP 列表：优先 env 覆盖，其次从 Go 源码 defaultGithubDialFallbackIPs 提取。
resolve_ips() {
  if [[ -n "${GITOAUTH_GITHUB_DIAL_FALLBACK_IPS:-}" ]]; then
    echo "${GITOAUTH_GITHUB_DIAL_FALLBACK_IPS//,/ }"
    return 0
  fi
  if [[ ! -f "$SOURCE_FILE" ]]; then
    echo "错误: 未设置 GITOAUTH_GITHUB_DIAL_FALLBACK_IPS 且缺少 $SOURCE_FILE" >&2
    return 1
  fi
  python3 - "$SOURCE_FILE" <<'PY'
import re, sys
text = open(sys.argv[1], encoding="utf-8").read()
# []string{ "ip", ... } 字面量：匹配 []string{ 之后的元素区
m = re.search(r"defaultGithubDialFallbackIPs\s*=\s*\[\s*\]\s*string\s*\{\s*(.*?)\s*\}", text, re.S)
if not m:
    sys.exit(1)
for ip in re.findall(r'"([0-9a-fA-F:.]+)"', m.group(1)):
    print(ip)
PY
}

probe_one() {
  local ip="$1"
  # POST 到换票端点，无凭据预期 4xx；若连接/TLS 挂起则 curl 超时。
  local code
  code="$(curl --resolve "${GITHUB_RESOLVE_HOST}:443:${ip}" \
    -m "${CURL_TIMEOUT_SEC}" \
    -s -o /dev/null -w '%{http_code}' \
    -X POST -d 'client_id=probe&client_secret=probe&code=probe' \
    "${PROBE_URL}" 2>/dev/null || true)"
  if [[ -z "$code" || "$code" == "000" ]]; then
    echo "timeout"
    return 1
  fi
  echo "http=${code}"
  return 0
}

main() {
  local ips
  if ! ips="$(resolve_ips)"; then
    echo '{"status":"critical","ok":false,"reason":"no_fallback_ips_resolved"}' >&2
    return 1
  fi
  if [[ -z "$ips" ]]; then
    echo '{"status":"critical","ok":false,"reason":"empty_fallback_ips"}' >&2
    return 1
  fi

  local reachable=0 total=0
  local line
  while read -r ip; do
    [[ -z "$ip" ]] && continue
    total=$((total + 1))
    if line="$(probe_one "$ip")"; then
      reachable=$((reachable + 1))
    fi
    printf 'probe_github_fallback_ip ip=%s %s\n' "$ip" "$line"
  done <<< "$ips"

  if (( reachable == 0 )); then
    echo "{\"status\":\"critical\",\"ok\":false,\"reachable\":${reachable},\"total\":${total}}" >&2
    return 1
  fi
  echo "{\"status\":\"ok\",\"ok\":true,\"reachable\":${reachable},\"total\":${total}}"
  return 0
}

main "$@"
