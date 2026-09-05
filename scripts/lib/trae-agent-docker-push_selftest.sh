#!/usr/bin/env bash
# trae-agent-docker-push 扫描/执行自测（scratch 仓，DRY_RUN，不跑 docker）
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCAN="$SCRIPT_DIR/trae-agent-docker-push-scan.sh"
PUSH="$SCRIPT_DIR/trae-agent-docker-push.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

git init -q "$TMP/repo"
git -C "$TMP/repo" config user.email t@example.com
git -C "$TMP/repo" config user.name t

mkdir -p "$TMP/repo/trae-agent/onlineServiceJS/src" \
  "$TMP/repo/trae-agent/onlineServiceJS" \
  "$TMP/repo/scripts/lib" \
  "$TMP/repo/.runall"

# 假 buildDocker.sh（DRY_RUN 不会调用到真实内容，但脚本要求文件存在）
cat > "$TMP/repo/trae-agent/onlineServiceJS/buildDocker.sh" <<'EOF'
#!/usr/bin/env bash
echo "fake buildDocker"
exit 0
EOF
chmod +x "$TMP/repo/trae-agent/onlineServiceJS/buildDocker.sh"

# 嵌套 git 仓模拟 trae-agent 子模块
git init -q "$TMP/repo/trae-agent"
git -C "$TMP/repo/trae-agent" config user.email t@example.com
git -C "$TMP/repo/trae-agent" config user.name t
echo 'v1' > "$TMP/repo/trae-agent/onlineServiceJS/src/server.mjs"
git -C "$TMP/repo/trae-agent" add onlineServiceJS
git -C "$TMP/repo/trae-agent" commit -qm 'init online'

cp "$SCAN" "$TMP/repo/scripts/lib/trae-agent-docker-push-scan.sh"
cp "$PUSH" "$TMP/repo/scripts/lib/trae-agent-docker-push.sh"

export TRAE_AGENT_DOCKER_PUSH_STATE_DIR="$TMP/repo/.runall"
PENDING="$TMP/repo/.runall/trae_agent_docker_push_pending"
SHA_FILE="$TMP/repo/.runall/trae_agent_docker_push_sha"

run_scan() {
  bash "$TMP/repo/scripts/lib/trae-agent-docker-push-scan.sh"
}

# Case 1: 脏源码 → 应登记 pending
echo 'v2' > "$TMP/repo/trae-agent/onlineServiceJS/src/server.mjs"
rm -f "$PENDING"
run_scan
if [ ! -f "$PENDING" ]; then
  echo "FAIL: dirty onlineServiceJS src should register pending" >&2
  exit 1
fi

# Case 2: 仅文档脏 → 不应因文档新建 pending（先清 pending；水位线=HEAD 且无 ahead）
git -C "$TMP/repo/trae-agent" checkout -- onlineServiceJS/src/server.mjs
HEAD1="$(git -C "$TMP/repo/trae-agent" rev-parse HEAD)"
printf '%s\n' "$HEAD1" > "$SHA_FILE"
rm -f "$PENDING"
echo 'doc' > "$TMP/repo/trae-agent/onlineServiceJS/README.md"
run_scan
if [ -f "$PENDING" ]; then
  echo "FAIL: docs-only dirty should not register pending" >&2
  cat "$PENDING" >&2 || true
  exit 1
fi

# Case 3: 提交镜像相关变更且水位线落后 → 应登记
git -C "$TMP/repo/trae-agent" checkout -- onlineServiceJS/README.md 2>/dev/null || rm -f "$TMP/repo/trae-agent/onlineServiceJS/README.md"
echo 'v3' > "$TMP/repo/trae-agent/onlineServiceJS/src/server.mjs"
git -C "$TMP/repo/trae-agent" add onlineServiceJS/src/server.mjs
git -C "$TMP/repo/trae-agent" commit -qm 'feat: bump server'
rm -f "$PENDING"
# 水位线仍为旧 HEAD1
run_scan
if [ ! -f "$PENDING" ]; then
  echo "FAIL: committed ahead of watermark should register pending" >&2
  exit 1
fi

# Case 4: DRY_RUN 推送清除 pending 并更新水位线
export TRAE_AGENT_DOCKER_PUSH_DRY_RUN=1
bash "$TMP/repo/scripts/lib/trae-agent-docker-push.sh" --if-pending
if [ -f "$PENDING" ]; then
  echo "FAIL: dry-run push should clear pending" >&2
  exit 1
fi
HEAD2="$(git -C "$TMP/repo/trae-agent" rev-parse HEAD)"
GOT_SHA="$(tr -d '[:space:]' < "$SHA_FILE")"
if [ "$GOT_SHA" != "$HEAD2" ]; then
  echo "FAIL: watermark should be HEAD ($HEAD2) got='$GOT_SHA'" >&2
  exit 1
fi

# Case 5: SKIP 环境变量
echo 'v4' > "$TMP/repo/trae-agent/onlineServiceJS/src/server.mjs"
run_scan
export TRAE_AGENT_SKIP_DOCKER_PUSH=1
bash "$TMP/repo/scripts/lib/trae-agent-docker-push.sh" --if-pending
if [ ! -f "$PENDING" ]; then
  echo "FAIL: SKIP should leave pending intact" >&2
  exit 1
fi

echo "OK trae-agent-docker-push_selftest"
