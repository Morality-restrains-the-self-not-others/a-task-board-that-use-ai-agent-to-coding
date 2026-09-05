#!/usr/bin/env bash
# register-precise-restart-scan.sh 自测：水位线抑制「已部署但仍脏」的重登记。
# 不依赖 git submodule file://（部分环境 protocol.file.allow=never），改用 gitlink + 嵌套仓。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCAN="$SCRIPT_DIR/register-precise-restart-scan.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

git init -q "$TMP/repo"
git -C "$TMP/repo" config user.email t@example.com
git -C "$TMP/repo" config user.name t
mkdir -p "$TMP/repo/conf" "$TMP/repo/.runall" "$TMP/repo/scripts/lib" "$TMP/repo/fakeSvc"

cat > "$TMP/repo/conf/runAll.yaml" <<'EOF'
version: "1"
groups:
  - name: test
    services:
      - name: fake-svc
        working_dir: fakeSvc
EOF

# 嵌套 git 仓模拟子模块工作树
git init -q "$TMP/repo/fakeSvc"
git -C "$TMP/repo/fakeSvc" config user.email t@example.com
git -C "$TMP/repo/fakeSvc" config user.name t
echo 'v1' > "$TMP/repo/fakeSvc/main.go"
git -C "$TMP/repo/fakeSvc" add main.go
git -C "$TMP/repo/fakeSvc" commit -qm 'init'
SHA="$(git -C "$TMP/repo/fakeSvc" rev-parse HEAD)"

cat > "$TMP/repo/.gitmodules" <<'EOF'
[submodule "fakeSvc"]
	path = fakeSvc
	url = ./fakeSvc
EOF

# 父仓以 gitlink(160000) 登记子模块，porcelain 脏时显示 ` M fakeSvc`
git -C "$TMP/repo" add conf .gitmodules
git -C "$TMP/repo" update-index --add --cacheinfo 160000,"$SHA",fakeSvc
git -C "$TMP/repo" commit -qm 'meta with fakeSvc gitlink'

cp "$SCAN" "$TMP/repo/scripts/lib/register-precise-restart-scan.sh"

REG="$TMP/repo/.runall/precise_restart_services.txt"
WM="$TMP/repo/.runall/precise_restart_consumed_at"
export RUNALL_PRECISE_RESTART_FILE="$REG"

run_scan() {
  bash "$TMP/repo/scripts/lib/register-precise-restart-scan.sh"
}

echo 'v2 dirty' > "$TMP/repo/fakeSvc/main.go"
DIRTY_MT="$(stat -c %Y "$TMP/repo/fakeSvc/main.go")"

# Case 1: 无水位线 → 应登记
: > "$REG"
run_scan
if ! grep -qE $'^fake-svc($|\t)' "$REG"; then
  echo "FAIL: expected auto-register without watermark" >&2
  echo "--- reg ---"; cat "$REG" >&2 || true
  echo "--- meta status ---"; git -C "$TMP/repo" status --porcelain >&2 || true
  echo "--- sub status ---"; git -C "$TMP/repo/fakeSvc" status --porcelain >&2 || true
  exit 1
fi

# Case 2: 水位线 >= 脏文件 mtime → 清空后不应重登
: > "$REG"
printf '%s\n' "$DIRTY_MT" > "$WM"
run_scan
if grep -qE $'^fake-svc($|\t)' "$REG"; then
  echo "FAIL: watermark should suppress re-register of already-consumed dirty tree" >&2
  cat "$REG" >&2 || true
  exit 1
fi

# Case 3: 水位线之后有新编辑 → 应登记
: > "$REG"
printf '%s\n' "$DIRTY_MT" > "$WM"
sleep 1
echo 'v3 newer' > "$TMP/repo/fakeSvc/main.go"
run_scan
if ! grep -qE $'^fake-svc($|\t)' "$REG"; then
  echo "FAIL: new edits after watermark should register" >&2
  cat "$REG" >&2 || true
  exit 1
fi

# Case 4 (OPT-20260810-055): 内容变更但 mtime 未变（工具改写保持 mtime）→ 应登记。
# 水位线设为远未来，mtime 判定必然抑制；内容摘要相对快照偏离 → OR 触发登记。
: > "$REG"
FUTURE_MT="$(( $(date +%s) + 3600 ))"
printf '%s\n' "$FUTURE_MT" > "$WM"
echo 'v4 content-only change' > "$TMP/repo/fakeSvc/main.go"
touch -d "@$DIRTY_MT" "$TMP/repo/fakeSvc/main.go"   # 恢复旧 mtime：内容变了但 mtime 不变
run_scan
if ! grep -qE $'^fake-svc($|\t)' "$REG"; then
  echo "FAIL: content change with preserved mtime should register (blob-digest OR)" >&2
  cat "$REG" >&2 || true
  exit 1
fi

# Case 5 (OPT-20260810-055): 内容未再变、mtime 仍不新于水位线 → 摘要一致，抑制。
: > "$REG"
run_scan
if grep -qE $'^fake-svc($|\t)' "$REG"; then
  echo "FAIL: unchanged content with old mtime should stay suppressed" >&2
  cat "$REG" >&2 || true
  exit 1
fi

echo "OK: register-precise-restart-scan watermark selftest passed"
