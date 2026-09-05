#!/usr/bin/env bash
# 只有 conf 树时仍能解析 infra YAML（OPT-20260830-015）。
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
# shellcheck source=resolve_infra_conf.sh
source "$HERE/resolve_infra_conf.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

pass=0
fail=0
assert_eq() {
  local name="$1" got="$2" want="$3"
  if [[ "$got" == "$want" ]]; then
    echo "PASS  $name"
    pass=$((pass + 1))
  else
    echo "FAIL  $name — got=$got want=$want" >&2
    fail=$((fail + 1))
  fi
}

CONF="$TMP/conf-only"
mkdir -p "$CONF/infra/redis" "$CONF/infra/kafka" "$CONF/infra/portainer"
printf 'host: 10.9.9.9\nport: 6379\n' >"$CONF/infra/redis/config.yaml"
printf 'host: ${INFRA_HOST:-10.8.8.8}\nkafkaUiPort: 18080\n' >"$CONF/infra/kafka/config.yaml"
printf 'host: 10.7.7.7\nport: 9000\n' >"$CONF/infra/portainer/config.yaml"

got="$(CONF_ROOT="$CONF" resolve_infra_conf redis)"
assert_eq "CONF_ROOT redis path" "$got" "$CONF/infra/redis/config.yaml"

DEPLOY="$TMP/deploy"
mkdir -p "$DEPLOY/conf/infra/kafka"
printf 'host: 10.6.6.6\n' >"$DEPLOY/conf/infra/kafka/config.yaml"
got="$(
  unset CONF_ROOT
  DEPLOY_ROOT="$DEPLOY" resolve_infra_conf kafka
)"
assert_eq "DEPLOY_ROOT kafka path" "$got" "$DEPLOY/conf/infra/kafka/config.yaml"

host="$(python3 "$HERE/infra_conf_host.py" "$CONF/infra/redis/config.yaml")"
assert_eq "redis host literal" "$host" "10.9.9.9"

host="$(env -u INFRA_HOST python3 "$HERE/infra_conf_host.py" "$CONF/infra/kafka/config.yaml")"
assert_eq "kafka host default from YAML" "$host" "10.8.8.8"

host="$(INFRA_HOST=10.1.1.1 python3 "$HERE/infra_conf_host.py" "$CONF/infra/kafka/config.yaml")"
assert_eq "INFRA_HOST env wins YAML default" "$host" "10.1.1.1"

echo "----"
echo "resolve_infra_conf tests: $pass passed, $fail failed"
[[ "$fail" -eq 0 ]] || exit 1
