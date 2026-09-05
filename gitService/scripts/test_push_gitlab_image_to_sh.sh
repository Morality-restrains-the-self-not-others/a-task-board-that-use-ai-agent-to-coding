#!/usr/bin/env bash
# Unit tests for push_gitlab_image_to_sh.sh（OPT-20260818-041）。
# 不实际推送镜像：GITSERVICE_SH_SSH_DRY_RUN=1 只验证管道命令与输入校验。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$SCRIPT_DIR/push_gitlab_image_to_sh.sh"
PASS=0
FAIL=0

assert_contains() {
  local name="$1" hay="$2" needle="$3"
  if [[ "$hay" == *"$needle"* ]]; then
    echo "PASS $name"
    PASS=$((PASS + 1))
  else
    echo "FAIL $name: missing [$needle] in [$hay]" >&2
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

# 推送 dry-run：应输出 docker save|gzip|ssh 管道。printf %q 会把空格转义为 '\ '，需归一化。
normalize_dry() {
  printf '%s' "${1//\\ / }" | sed 's/\\|/|/g'
}

run_push_dry() {
  local out
  out="$(GITSERVICE_SH_SSH_DRY_RUN=1 GITSERVICE_SH_SSH_BIN=ssh GITSERVICE_SH_SSH_HOST=sh \
    "$SCRIPT" "$@" 2>/dev/null || true)"
  normalize_dry "$out"
}

# 校验：非法主机名被拒绝
if GITSERVICE_SH_SSH_DRY_RUN=1 GITSERVICE_SH_SSH_BIN=ssh GITSERVICE_SH_SSH_HOST='sh;rm -rf /' \
   "$SCRIPT" >/dev/null 2>&1; then
  echo "FAIL host_validation: 非法主机名未被拒绝" >&2
  FAIL=$((FAIL + 1))
else
  echo "PASS host_validation"
  PASS=$((PASS + 1))
fi

# 校验：非法镜像名被拒绝（错误信息走 stderr，这里只断言非零退出）
set +e
GITSERVICE_SH_SSH_DRY_RUN=1 GITSERVICE_SH_SSH_BIN=ssh GITSERVICE_SH_SSH_HOST=sh \
  "$SCRIPT" --image 'gitlab/gitlab-ce:19.2.4-ce.0;ls' >/dev/null 2>&1
rc=$?
set -e
assert_rc "image_validation" "$rc" 2

# 推送 dry-run：SSOT 镜像名出现在管道中
out="$(run_push_dry)"
assert_contains "push_ssot_image" "$out" "docker save gitlab/gitlab-ce:"
assert_contains "push_ssot_pipe" "$out" "gzip -1"
assert_contains "push_ssot_load" "$out" "gunzip | docker load"

# GITLAB_IMAGE_OVERRIDE 生效
out="$(GITSERVICE_SH_SSH_DRY_RUN=1 GITSERVICE_SH_SSH_BIN=ssh GITLAB_IMAGE_OVERRIDE=gitlab/gitlab-ce:19.0.8-ce.0 "$SCRIPT" 2>/dev/null || true)"
out="$(normalize_dry "$out")"
assert_contains "override_image" "$out" "gitlab/gitlab-ce:19.0.8-ce.0"

# --check dry-run：查询远端镜像存在性
out="$(GITSERVICE_SH_SSH_DRY_RUN=1 GITSERVICE_SH_SSH_BIN=ssh "$SCRIPT" --check 2>/dev/null || true)"
out="$(normalize_dry "$out")"
assert_contains "check_docker_images" "$out" "docker images"
assert_contains "check_grep" "$out" "grep -qx"

echo "----"
echo "Result: PASS=$PASS FAIL=$FAIL"
[[ "$FAIL" -eq 0 ]]
