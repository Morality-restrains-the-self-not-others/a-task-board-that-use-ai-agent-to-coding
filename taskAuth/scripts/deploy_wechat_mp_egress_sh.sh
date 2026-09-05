#!/usr/bin/env bash
# Deploy wechat-mp-egress binary to Host sh (ADR-0059).
# Usage:
#   bash taskAuth/scripts/deploy_wechat_mp_egress_sh.sh            # build + scp + restart
#   bash taskAuth/scripts/deploy_wechat_mp_egress_sh.sh dry-run
set -euo pipefail
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY all_proxy || true

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SSH_HOST="${WECHAT_MP_EGRESS_SSH_HOST:-sh}"
REMOTE_DIR="${WECHAT_MP_EGRESS_REMOTE_DIR:-/opt/daydaymoney/wechat-mp-egress}"
MODE="${1:-deploy}"

case "$MODE" in
  deploy|dry-run) ;;
  *) echo "usage: $0 [deploy|dry-run]" >&2; exit 2 ;;
esac

mkdir -p "$ROOT/taskAuth/bin"
echo "==> build wechat-mp-egress"
(
  cd "$ROOT/taskAuth"
  CGO_ENABLED=0 go build -o bin/wechat-mp-egress ./cmd/wechat-mp-egress
)

SECRET=""
if [[ -f "$ROOT/conf-local/infra/wechat-mp-egress/config.yaml" ]]; then
  SECRET="$(python3 -c "import yaml; d=yaml.safe_load(open('$ROOT/conf-local/infra/wechat-mp-egress/config.yaml')) or {}; print((d.get('internalSecret') or '').strip())")"
fi
if [[ -z "$SECRET" ]]; then
  echo "error: set internalSecret in conf-local/infra/wechat-mp-egress/config.yaml" >&2
  exit 2
fi

if [[ "$MODE" == "dry-run" ]]; then
  echo "dry-run: would scp bin → ${SSH_HOST}:${REMOTE_DIR}/wechat-mp-egress and restart"
  exit 0
fi

echo "==> ensure remote dir ${REMOTE_DIR}"
ssh -o BatchMode=yes "$SSH_HOST" "mkdir -p '$REMOTE_DIR'"

echo "==> scp binary"
scp -o BatchMode=yes "$ROOT/taskAuth/bin/wechat-mp-egress" "${SSH_HOST}:${REMOTE_DIR}/wechat-mp-egress"

echo "==> write unit + restart"
ssh -o BatchMode=yes "$SSH_HOST" bash -s <<EOF
set -euo pipefail
cat >/etc/systemd/system/wechat-mp-egress.service <<'UNIT'
[Unit]
Description=WeChat MP API egress (Host sh)
After=network.target

[Service]
Type=simple
WorkingDirectory=${REMOTE_DIR}
Environment=WECHAT_MP_EGRESS_INTERNAL_SECRET=${SECRET}
ExecStart=${REMOTE_DIR}/wechat-mp-egress -listen 0.0.0.0:8030
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT
chmod +x ${REMOTE_DIR}/wechat-mp-egress
systemctl daemon-reload
systemctl enable wechat-mp-egress
systemctl restart wechat-mp-egress
sleep 1
curl -sf http://127.0.0.1:8030/healthz
echo
systemctl --no-pager -l status wechat-mp-egress | head -20
EOF

echo "==> probe from INFRA"
curl -sf --max-time 5 "http://1.117.67.121:8030/healthz"
echo
echo "OK: wechat-mp-egress on ${SSH_HOST}:8030"
