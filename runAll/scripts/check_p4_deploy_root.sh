#!/usr/bin/env bash
# Assert $DEPLOY_ROOT is a source-less deploy tree (ADR-0052 P4).
set -euo pipefail
ROOT="${1:-${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}}"

fail() { echo "check_p4_deploy_root: $*" >&2; exit 1; }

[[ -d "$ROOT" ]] || fail "missing $ROOT"
[[ ! -e "$ROOT/.gitmodules" ]] || fail "found .gitmodules (source clone)"
[[ -f "$ROOT/conf/base.yaml" ]] || fail "missing conf/base.yaml"
[[ -x "$ROOT/bin/runAll" ]] || fail "missing bin/runAll"
[[ -x "$ROOT/bin/taskAuth" ]] || fail "missing bin/taskAuth"
[[ -x "$ROOT/trae-agent/onlineServiceJS/run.sh" ]] || fail "missing trae-agent/onlineServiceJS/run.sh"
[[ -f "$ROOT/trae-agent/onlineServiceJS/Dockerfile" ]] || fail "missing trae-agent/onlineServiceJS/Dockerfile"
[[ -f "$ROOT/trae-agent/onlineServiceJS/buildDocker.sh" ]] || fail "missing trae-agent/onlineServiceJS/buildDocker.sh"
[[ -d "$ROOT/trae-agent/onlineServiceJS/src" ]] || fail "missing trae-agent/onlineServiceJS/src"

go_hits="$(find "$ROOT" \( \
  -path "$ROOT/.git" -o -path "$ROOT/.git/*" \
  -o -path "$ROOT/.config-repo" -o -path "$ROOT/logs" \
  -o -path "$ROOT/dockerInfra/*/data" -o -path "$ROOT/dockerInfra/*/data/*" \
  -o -path "$ROOT/shareLib" -o -path "$ROOT/shareLib/*" \
\) -prune -o -name '*.go' -print 2>/dev/null | head)"
if [[ -n "$go_hits" ]]; then
  fail "found Go source: $go_hits"
fi

while IFS= read -r -d '' link; do
  target="$(readlink -f "$link" 2>/dev/null || readlink "$link")"
  if [[ "$target" == /tmp/ram-work/* ]]; then
    fail "source symlink $link -> $target"
  fi
done < <(find "$ROOT" -type l -print0 2>/dev/null)

echo "check_p4_deploy_root: ok $ROOT"
