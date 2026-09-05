#!/usr/bin/env bash
# Unit tests for deploy_tencent_sh_1_from_infra.sh (SSOT 驱动部署 tencent-sh-1，OPT-20260823-061).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PASS=0
FAIL=0

assert_contains() {
  local name="$1" hay="$2" needle="$3"
  if [[ "$hay" == *"$needle"* ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: missing [$needle]" >&2
    FAIL=$((FAIL + 1))
  fi
}

assert_not_contains() {
  local name="$1" hay="$2" needle="$3"
  if [[ "$hay" != *"$needle"* ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: unexpected [$needle]" >&2
    FAIL=$((FAIL + 1))
  fi
}

assert_rc() {
  local name="$1" rc="$2" want="$3"
  if [[ "$rc" -eq "$want" ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: rc=$rc want=$want" >&2
    FAIL=$((FAIL + 1))
  fi
}

# dry-run 不触碰远端，只渲染 .env 并打印命令。
dry_out="$(bash "$SCRIPT_DIR/deploy_tencent_sh_1_from_infra.sh" dry-run 2>&1)"

# 渲染 env 正确性
assert_contains "env-external-url" "$dry_out" "GITLAB_EXTERNAL_URL=https://gitlab-tencent-sh-1.daydaymoney.com"
assert_contains "env-external-host" "$dry_out" "GITLAB_EXTERNAL_HOST=gitlab-tencent-sh-1.daydaymoney.com"
assert_contains "env-region" "$dry_out" "TRAE_GITLAB_REGION=tencent-sh-1"
assert_contains "env-oidc-issuer" "$dry_out" "GITLAB_OIDC_ISSUER=https://api.daydaymoney.com"
assert_contains "env-oidc-redirect" "$dry_out" "GITLAB_OIDC_REDIRECT_URI=https://gitlab-tencent-sh-1.daydaymoney.com/users/auth/openid_connect/callback"
assert_contains "env-oidc-client" "$dry_out" "GITLAB_OIDC_CLIENT_ID=gitlab-git-service-tencent-sh-1"
assert_contains "env-gitlab-home" "$dry_out" "GITLAB_HOME=/var/lib/daydaymoney/gitService-tencent-sh-1"
assert_contains "env-ssh-port" "$dry_out" "GITLAB_SSH_PORT=2223"
assert_not_contains "env-no-template" "$dry_out" '\${'
assert_not_contains "env-no-host-placeholder" "$dry_out" "0.0.0.0"

# dry-run 打印将执行的命令
assert_contains "cmd-compose-src" "$dry_out" "docker-compose.tencent-sh-1.yml"
assert_contains "cmd-compose-scp" "$dry_out" "docker-compose.yml"
assert_contains "cmd-env-scp" "$dry_out" "gitservice-tencent-sh-1/.env"
assert_contains "cmd-ssh-up" "$dry_out" "docker compose up -d"
assert_contains "cmd-ssh-host" "$dry_out" " sh "
assert_contains "cmd-initializer" "$dry_out" "zzz_trae_gitlab_traffic_quota.rb"

# 非法 mode 拒绝
set +e
bash "$SCRIPT_DIR/deploy_tencent_sh_1_from_infra.sh" bogus >/dev/null 2>&1
rc=$?
set -e
assert_rc "invalid-mode-fails" "$rc" 2

# 不安全的远端目录拒绝
set +e
GITSERVICE_SH_COMPOSE_DIR='/tmp/evil; rm -rf /' bash "$SCRIPT_DIR/deploy_tencent_sh_1_from_infra.sh" dry-run >/dev/null 2>&1
rc=$?
set -e
assert_rc "unsafe-remote-dir-fails" "$rc" 2

# 不安全的 SSH 主机拒绝
set +e
GITSERVICE_SH_SSH_HOST='sh; rm -rf /' bash "$SCRIPT_DIR/deploy_tencent_sh_1_from_infra.sh" dry-run >/dev/null 2>&1
rc=$?
set -e
assert_rc "unsafe-ssh-host-fails" "$rc" 2

echo "result pass=$PASS fail=$FAIL"
if [[ "$FAIL" -ne 0 ]]; then
  exit 1
fi
