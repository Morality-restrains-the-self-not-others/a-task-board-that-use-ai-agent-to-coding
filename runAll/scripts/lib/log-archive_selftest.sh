#!/usr/bin/env bash
# Self-check: lib/log-archive.sh 短窗归档 + 批清理（不依赖外部服务，纯临时目录）。
# shellcheck disable=SC2034  # 测试期全局变量由被 source 的 lib 函数消费
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
LIB="${ROOT}/runAll/scripts/lib/log-archive.sh"
# shellcheck source=log-archive.sh
source "$LIB"

fail() { echo "FAIL: $*" >&2; exit 1; }

TMP="$(mktemp -d)"
trap 'rm -rf -- "$TMP"' EXIT

# ---------- archive_short_window ----------
mkdir -p "$TMP/logs/a" "$TMP/logs/b"
printf 'hello-1\n' > "$TMP/logs/a/svc.log"
: > "$TMP/logs/a/empty.log"
printf 'nightly-test-sweep-noise\n' > "$TMP/logs/a/nightly-test-sweep-20260824.log"
printf 'hello-2\n' > "$TMP/logs/b/other.log"

TRUNCATE_ARCHIVE_ENABLED=1
TRUNCATE_ARCHIVE_ROOT="$TMP/archive"
ARCHIVE_BATCH_TS="20260824T073000"
TRUNCATE_ARCHIVE_TAIL_BYTES=1024

archive_short_window "$TMP/logs/a"
archive_short_window "$TMP/logs/b"

[[ -f "$TMP/archive/20260824T073000/svc.log.gz" ]] || fail "svc.log.gz missing"
[[ -f "$TMP/archive/20260824T073000/other.log.gz" ]] || fail "other.log.gz missing"
if [[ -f "$TMP/archive/20260824T073000/empty.log.gz" ]]; then fail "empty log must not be archived"; fi
if [[ -f "$TMP/archive/20260824T073000/nightly-test-sweep-20260824.log.gz" ]]; then fail "nightly-test-sweep must be skipped"; fi
gunzip -c "$TMP/archive/20260824T073000/svc.log.gz" | grep -q 'hello-1' || fail "archived content mismatch"

# disabled → 不产生归档目录
rm -rf -- "$TMP/archive"
TRUNCATE_ARCHIVE_ENABLED=0
archive_short_window "$TMP/logs/a"
if [[ -d "$TMP/archive" ]]; then fail "archive must not be created when disabled"; fi

# 不存在的目录 → 幂等返回 0
archive_short_window "$TMP/does-not-exist"

# ---------- prune_archive_batches ----------
TRUNCATE_ARCHIVE_ENABLED=1
TRUNCATE_ARCHIVE_ROOT="$TMP/archive"
TRUNCATE_ARCHIVE_KEEP=2
mkdir -p "$TMP/archive/20260824T060000" "$TMP/archive/20260824T070000" "$TMP/archive/20260824T080000"
: > "$TMP/archive/20260824T060000/f"
: > "$TMP/archive/20260824T070000/f"
: > "$TMP/archive/20260824T080000/f"
prune_archive_batches
[[ -d "$TMP/archive/20260824T080000" ]] || fail "newest batch pruned"
[[ -d "$TMP/archive/20260824T070000" ]] || fail "second-newest batch pruned"
if [[ -d "$TMP/archive/20260824T060000" ]]; then fail "oldest batch not pruned"; fi

# 关闭时清理不执行
TRUNCATE_ARCHIVE_ENABLED=0
prune_archive_batches
if [[ ! -d "$TMP/archive/20260824T070000" ]]; then fail "prune must not run when disabled"; fi

echo "OK: log-archive short-window archive + batch prune"
