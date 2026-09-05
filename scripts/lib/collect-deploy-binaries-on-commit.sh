#!/usr/bin/env bash
# Fail-open pre-commit helper: refresh gitignored $META/deploy-binaries/.
# Never blocks git commit unless COLLECT_DEPLOY_BINARIES_REQUIRED=1.
set -u

if [[ "${SKIP_COLLECT_DEPLOY_BINARIES:-}" == "1" ]]; then
  exit 0
fi
if [[ "${CI:-}" == "true" || "${CI:-}" == "1" ]]; then
  exit 0
fi

find_meta() {
  local d="${SESSION_META_ROOT:-}"
  if [[ -n "$d" && -f "$d/.gitmodules" ]]; then
    printf '%s' "$d"
    return 0
  fi
  d="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
  while [[ "$d" != "/" ]]; do
    if [[ -f "$d/.gitmodules" && -x "$d/runAll/scripts/collect-deploy-binaries.sh" ]]; then
      printf '%s' "$d"
      return 0
    fi
    d="$(dirname "$d")"
  done
  return 1
}

META="$(find_meta || true)"
if [[ -z "${META:-}" ]]; then
  exit 0
fi

COLLECT="$META/runAll/scripts/collect-deploy-binaries.sh"
if [[ ! -x "$COLLECT" ]]; then
  exit 0
fi

STAGING="${COLLECT_STAGING:-/tmp/daydaymoney-release-20260831-conf-local}"
RAM_DEPLOY="${RAM_DEPLOY:-${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}}"
has_src=0
for p in \
  "$STAGING/runAll" \
  "$RAM_DEPLOY/artifacts/runAll" \
  "$META/bin/runAll" \
  "$META/runAll/bin/runAll"
do
  if [[ -f "$p" ]]; then
    has_src=1
    break
  fi
done
if [[ "$has_src" != "1" ]]; then
  echo "[collect-deploy-binaries] skip: no runAll source on this machine" >&2
  exit 0
fi

echo "▸ collect deploy-binaries (incremental, gitignored)" >&2
# COLLECT_SKIP_SHA=1：源码机 ELF 与 conf.example/releases.yaml 的 Release pin 本就不同，
# 钩子内不得校验 sha（避免 fail-open 刷屏 SHA MISMATCH）。部署前的 sha 校验由
# runAll/scripts/collect-deploy-binaries.sh 显式调用 / 手工复制时负责（OPT-20260902-009）。
if ! META_ROOT="$META" COLLECT_SOFT=1 COLLECT_SKIP_SHA=1 RAM_DEPLOY="$RAM_DEPLOY" COLLECT_STAGING="$STAGING" \
  bash "$COLLECT" "$META/deploy-binaries"
then
  echo "⚠ collect-deploy-binaries failed (not blocking commit); SKIP_COLLECT_DEPLOY_BINARIES=1 to silence" >&2
  if [[ "${COLLECT_DEPLOY_BINARIES_REQUIRED:-}" == "1" ]]; then
    exit 1
  fi
fi
exit 0
