#!/usr/bin/env bash
# 静态验收：Docker nginx 常驻、hashed 资源真 404、health、conf SSOT。
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
app_dir="$(cd "$script_dir/.." && pwd)"
taskfe_dir="$(cd "$app_dir/.." && pwd)"
repo_root="$(cd "$taskfe_dir/.." && pwd)"
fail=0

nginx_conf="$taskfe_dir/nginx/nginx.conf"
compose="$taskfe_dir/docker-compose.yml"
lifecycle="$script_dir/runall-lifecycle.sh"
vue_conf="$repo_root/conf/frontend/vue/config.yaml"
runall="$repo_root/conf/runAll.yaml"

bash -n "$lifecycle" || fail=1
[[ -f "$nginx_conf" ]] || { echo "FAIL: 缺少 nginx.conf" >&2; fail=1; }
[[ -f "$compose" ]] || { echo "FAIL: 缺少 docker-compose.yml" >&2; fail=1; }

ngx="$(cat "$nginx_conf" 2>/dev/null || true)"
cmp="$(cat "$compose" 2>/dev/null || true)"
life="$(cat "$lifecycle" 2>/dev/null || true)"

# hashed 资源不得 SPA fallback
if ! printf '%s\n' "$ngx" | grep -qE 'location \^~ /static/assets/'; then
  echo "FAIL: nginx 须有 location ^~ /static/assets/" >&2
  fail=1
fi
assets_block="$(printf '%s\n' "$ngx" | awk '/location \^~ \/static\/assets\//,/^[[:space:]]*}/')"
if printf '%s\n' "$assets_block" | grep -qE 'index\.html'; then
  echo "FAIL: /static/assets/ 禁止 fallback 到 index.html" >&2
  fail=1
fi

# health
if ! printf '%s\n' "$ngx" | grep -qE 'location = /health'; then
  echo "FAIL: nginx 须有 location = /health" >&2
  fail=1
fi
if ! printf '%s\n' "$ngx" | grep -qE 'html/index.html'; then
  echo "FAIL: /health 须按 html/index.html 存在与否返回 200/503" >&2
  fail=1
fi

# access log 带 X-Trace-Id
if ! printf '%s\n' "$ngx" | grep -qE 'trace_id=.*http_x_trace_id|x_trace_id'; then
  echo "FAIL: access log 须记录 X-Trace-Id" >&2
  fail=1
fi

# compose：父目录卷、host network 听 0.0.0.0:4000、unless-stopped、SSOT 注释
if ! printf '%s\n' "$cmp" | grep -qE 'restart:.*unless-stopped'; then
  echo "FAIL: compose 须 restart: unless-stopped" >&2
  fail=1
fi
if ! printf '%s\n' "$cmp" | grep -qE 'network_mode: host'; then
  echo "FAIL: compose 须 network_mode: host（避免 Docker DNAT 把客户端 IP 写成 172.26.0.1）" >&2
  fail=1
fi
if ! printf '%s\n' "$cmp" | grep -qE '0\.0\.0\.0|TASKFE_NGINX_HOST'; then
  echo "FAIL: compose 须发布到 0.0.0.0（或 conf 导出的 HOST）" >&2
  fail=1
fi
if ! printf '%s\n' "$ngx" | grep -qE 'listen 4000'; then
  echo "FAIL: host network 下 nginx 须 listen 4000（SSOT conf port）" >&2
  fail=1
fi
if ! printf '%s\n' "$ngx" | grep -qE 'real_ip_header X-Forwarded-For'; then
  echo "FAIL: nginx 须 real_ip_header X-Forwarded-For 还原边缘客户端 IP" >&2
  fail=1
fi
if ! printf '%s\n' "$ngx" | grep -qE 'set_real_ip_from 172\.16\.0\.0/12'; then
  echo "FAIL: nginx 须信任 Docker RFC1918 网桥" >&2
  fail=1
fi
if ! printf '%s\n' "$ngx" | grep -qE 'proxy_pass http://127\.0\.0\.1:18081'; then
  echo "FAIL: host network 下 /api/ 须 proxy_pass 127.0.0.1:18081" >&2
  fail=1
fi
if ! printf '%s\n' "$cmp" | grep -qE 'SSOT: conf/frontend/vue/config.yaml'; then
  echo "FAIL: compose 默认值须注释 SSOT: conf/frontend/vue/config.yaml" >&2
  fail=1
fi
if ! printf '%s\n' "$cmp" | grep -qE './app/public:/srv/taskfe'; then
  echo "FAIL: 须 mount ./app/public:/srv/taskfe（父目录）" >&2
  fail=1
fi
if ! printf '%s\n' "$cmp" | grep -qE './app/static:/srv/taskfe-static'; then
  echo "FAIL: 须 mount ./app/static:/srv/taskfe-static（微信 MP_verify 等根路径静态文件）" >&2
  fail=1
fi

# 微信公众号域名校验文件：精确 location，禁止 SPA fallback 成 index.html
if ! printf '%s\n' "$ngx" | grep -qE 'location ~ \^/MP_verify_'; then
  echo "FAIL: nginx 须有 location ~ ^/MP_verify_ 服务微信校验文件" >&2
  fail=1
fi
mp_block="$(printf '%s\n' "$ngx" | awk '/location ~ \^\/MP_verify_/,/^[[:space:]]*}/')"
if printf '%s\n' "$mp_block" | grep -qE 'index\.html'; then
  echo "FAIL: /MP_verify_*.txt 禁止 fallback 到 index.html" >&2
  fail=1
fi
if ! printf '%s\n' "$mp_block" | grep -qE 'root /srv/taskfe-static'; then
  echo "FAIL: MP_verify location 须 root /srv/taskfe-static" >&2
  fail=1
fi
verify_file="$app_dir/static/MP_verify_TmQKnSMuqxrR7atQ.txt"
if [[ ! -f "$verify_file" ]]; then
  echo "FAIL: 缺少微信校验文件 app/static/MP_verify_TmQKnSMuqxrR7atQ.txt" >&2
  fail=1
fi
verify_body="$(tr -d '\r\n' < "$verify_file")"
if [[ "$verify_body" != "TmQKnSMuqxrR7atQ" ]]; then
  echo "FAIL: 微信校验文件正文须为 TmQKnSMuqxrR7atQ，实际=${verify_body}" >&2
  fail=1
fi

# conf SSOT
if ! grep -qE '^serve: nginx' "$vue_conf"; then
  echo "FAIL: conf/frontend/vue/config.yaml 须 serve: nginx" >&2
  fail=1
fi
if ! grep -qE '^nginxImage:' "$vue_conf"; then
  echo "FAIL: conf 须 nginxImage" >&2
  fail=1
fi
if ! grep -qE '^memLimit:' "$vue_conf"; then
  echo "FAIL: conf 须 memLimit" >&2
  fail=1
fi

# runAll detach：compose up -d 后进程退出
if ! grep -A 20 'name: taskFE' "$runall" | grep -q 'launch_mode: detach'; then
  echo "FAIL: conf/runAll.yaml taskFE 须 launch_mode: detach" >&2
  fail=1
fi

# lifecycle 从 conf 导出
if ! printf '%s\n' "$life" | grep -qE 'conf-read.py'; then
  echo "FAIL: lifecycle 须用 conf-read.py 消费 vue conf" >&2
  fail=1
fi

# ── Live HTTP 探测（OPT-20260825-037）────────────────────────────────
# 静态 conf 检查发现不了运行时挂载丢失：nginx 重构/漏挂卷时公网可能静默回成
# SPA HTML（200 但微信校验失败）。默认探测本地 :4000；公网探测用
# MP_VERIFY_PUBLIC_BASE 显式打开（CI 无网时避免误失败）。
verify_path="/MP_verify_TmQKnSMuqxrR7atQ.txt"
missing_path="/MP_verify_MissingProbe12345.txt"
want_body="$(tr -d '\r\n' < "$verify_file")"

probe_mp_verify() {
  local base="$1" tmp code ctype body miss
  tmp="$(mktemp)"
  if ! code="$(curl -sS -o "$tmp" -w '%{http_code}' --max-time 8 "${base}${verify_path}")"; then
    echo "FAIL: ${base}${verify_path} 不可达" >&2
    rm -f "$tmp"; fail=1; return
  fi
  ctype="$(curl -sS -o /dev/null -w '%{content_type}' --max-time 8 "${base}${verify_path}")"
  body="$(tr -d '\r\n' < "$tmp")"
  rm -f "$tmp"
  [[ "$code" == "200" ]] || { echo "FAIL: ${base}${verify_path} HTTP=${code} 期望 200" >&2; fail=1; }
  case "$ctype" in
    text/plain*) ;;
    *) echo "FAIL: ${base}${verify_path} Content-Type=${ctype} 期望 text/plain" >&2; fail=1; ;;
  esac
  [[ "$body" == "$want_body" ]] || { echo "FAIL: ${base}${verify_path} 正文不匹配（期望 ${want_body}）" >&2; fail=1; }
  miss="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 8 "${base}${missing_path}")"
  [[ "$miss" == "404" ]] || { echo "FAIL: ${base}${missing_path} HTTP=${miss} 期望 404（禁止 SPA fallback 成 200 HTML）" >&2; fail=1; }
  echo "OK: ${base}${verify_path} -> ${code} ${ctype}"
}

if [[ "${MP_VERIFY_SKIP_LOCAL:-0}" != "1" ]]; then
  probe_mp_verify "http://127.0.0.1:4000"
fi
if [[ -n "${MP_VERIFY_PUBLIC_BASE:-}" ]]; then
  probe_mp_verify "${MP_VERIFY_PUBLIC_BASE}"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "test_nginx_static_resident.sh FAILED" >&2
  exit 1
fi
echo "test_nginx_static_resident.sh OK"
