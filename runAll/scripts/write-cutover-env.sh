#!/usr/bin/env bash
# Write $DEPLOY_ROOT/cutover.env — SSOT for deploy env (OPT-20260901-004).
# Consumers (run.sh / deploy-sync.sh / LoadConfig) source this file; the key set
# must not drift between generators. Callers: up-from-config-repo.sh,
# prepare-ram-deploy.sh.
#
# Usage: write-cutover-env.sh DEPLOY_ROOT [INFRA_HOST]
#   INFRA_HOST defaults to 10.2.150.68 (overridable via env).
#   SOURCE_ROOT from env, or /tmp/ram-work when that tree exists (ADR-0056).
set -euo pipefail

DEPLOY_ROOT="${1:-${DEPLOY_ROOT:?write-cutover-env: DEPLOY_ROOT required}}"
INFRA_HOST="${2:-${INFRA_HOST:-10.2.150.68}}"
SOURCE_ROOT="${SOURCE_ROOT:-}"
if [[ -z "$SOURCE_ROOT" && -d /tmp/ram-work/runAll ]]; then
  SOURCE_ROOT=/tmp/ram-work
fi
if [[ -z "$SOURCE_ROOT" ]]; then
  echo "[write-cutover-env] WARNING: SOURCE_ROOT empty; deploy 9999 精准编译重启 will not see source-tree registrations (ADR-0056)" >&2
fi

cat > "$DEPLOY_ROOT/cutover.env" <<EOF
export DEPLOY_ROOT=${DEPLOY_ROOT}
export CONF_ROOT=${DEPLOY_ROOT}/conf
export MONOREPO_ROOT=${DEPLOY_ROOT}
export DEPLOY_MODE=1
export SOURCE_ROOT=${SOURCE_ROOT}
export RUNALL_SKIP_BUILD=1
export RUNALL_BIN=${DEPLOY_ROOT}/bin/runAll
export RUNALL_CONFIG=${DEPLOY_ROOT}/conf/runAll.yaml
export RUNALL_CONSOLE_LOG=${DEPLOY_ROOT}/logs/runall-console.log
export RUNALL_LOG_ROOT=${DEPLOY_ROOT}/logs
export RUNALL_OWNERSHIP_STORE=${DEPLOY_ROOT}/.runall/ownership.json
export INFRA_HOST=\${INFRA_HOST:-${INFRA_HOST}}
export GODEBUG=http2client=0
EOF
echo "[write-cutover-env] wrote $DEPLOY_ROOT/cutover.env" >&2
