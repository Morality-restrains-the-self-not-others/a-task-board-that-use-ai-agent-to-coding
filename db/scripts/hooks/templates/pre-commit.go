#!/bin/bash
# Go 仓库 pre-commit（共享库版 random_test_runner.sh）：
# 1) 优先运行与暂存 .go 文件同包的测试（100% 必跑）
# 2) 对其余含测例的包按比例随机抽测（默认 30%，保证至少 1 个）
# 3) 任一测试失败则阻止提交
# 库: scripts/hooks/lib/random_test_runner.sh（SSOT，由 deploy_repo_random_precommit.sh 分发）

set -u

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
LIB="$HOOK_DIR/lib/random_test_runner.sh"
if [ ! -f "$LIB" ]; then
    echo "ERROR: shared lib missing: $LIB" >&2
    echo "       redeploy: bash <monorepo-root>/scripts/deploy_repo_random_precommit.sh" >&2
    exit 1
fi
source "$LIB"

cleanup() {
    echo "如果有测例失败，记得修复"
    echo "如果提交失败，记得先尝试修复，不行再忽略测例"
}
trap cleanup EXIT

# 仅变更 hooks/文档时跳过抽测，便于自举提交钩子模板（非 --no-verify）
rt_bootstrap_skip && exit 0

cd "$(git rev-parse --show-toplevel)" || exit 1

echo "Running pre-commit Go tests in $(pwd)..."
export PRE_COMMIT=1
TEST_RATIO="${PRECOMMIT_TEST_RATIO:-30}"
STAGED_FILES="$(git diff --cached --name-only 2>/dev/null || true)"

TMP_ALL="$(mktemp)"
TMP_STAGED="$(mktemp)"
TMP_TO_RUN="$(mktemp)"
trap 'rm -f "$TMP_ALL" "$TMP_STAGED" "$TMP_TO_RUN"; cleanup' EXIT

rt_go_collect_dirs > "$TMP_ALL"
if [ ! -s "$TMP_ALL" ]; then
    echo "No Go test packages found. Commit can proceed."
    exit 0
fi

# 暂存 100%
while IFS= read -r f; do
    [ -z "$f" ] && continue
    dir="$(rt_go_dir_for_file "$f" 2>/dev/null)" || continue
    if rt_go_dir_has_tests "$dir"; then
        echo "$dir" >> "$TMP_STAGED"
    fi
done <<< "$STAGED_FILES"

if [ -s "$TMP_STAGED" ]; then
    sort -u "$TMP_STAGED" -o "$TMP_STAGED"
    echo "Running staged Go test packages..."
    while IFS= read -r dir; do
        [ -z "$dir" ] && continue
        rt_go_run_dir "$dir" || exit 1
    done < "$TMP_STAGED"
else
    echo "No staged Go files mapped to test packages."
fi

# 随机抽测（排除暂存目录）
echo "Running random Go test packages (ratio=${TEST_RATIO}%)..."
if [ -s "$TMP_STAGED" ]; then
    grep -vxFf "$TMP_STAGED" "$TMP_ALL" | rt_pick_random "$TEST_RATIO" > "$TMP_TO_RUN"
else
    rt_pick_random "$TEST_RATIO" < "$TMP_ALL" > "$TMP_TO_RUN"
fi
while IFS= read -r dir; do
    [ -z "$dir" ] && continue
    echo "Random pick: ${dir}"
    rt_go_run_dir "$dir" || exit 1
done < "$TMP_TO_RUN"

echo "All Go unit tests passed. Commit can proceed."
