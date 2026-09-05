#!/usr/bin/env bash
# 验证：taskFE 编译必须「先临时目录 → 再原子切 public/html symlink」；失败不得抹掉现网 release。
set -euo pipefail
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
app_dir="$(cd "$script_dir/.." && pwd)"
taskfe_dir="$(cd "$app_dir/.." && pwd)"
fail=0

bash -n "$script_dir/runall-lifecycle.sh" || fail=1
bash -n "$script_dir/atomic-vite-build.sh" || fail=1

atomic="$(cat "$script_dir/atomic-vite-build.sh")"
lifecycle="$(cat "$script_dir/runall-lifecycle.sh")"
pkg="$app_dir/package.json"

# ── 1. SSOT 脚本存在且实现 staging → releases + symlink ──
if ! printf '%s\n' "$atomic" | grep -qE 'public/\.next|STAGING='; then
  echo "FAIL: atomic-vite-build.sh 须写入 public/.next staging" >&2
  fail=1
fi
if ! printf '%s\n' "$atomic" | grep -qE 'ln -sfn'; then
  echo "FAIL: atomic-vite-build.sh 须 ln -sfn 切换 html symlink" >&2
  fail=1
fi
if printf '%s\n' "$atomic" | grep -qE '^\s*rm -rf (dist|public/html)\s*$'; then
  echo "FAIL: atomic-vite-build.sh 禁止开局 rm -rf dist 或 public/html" >&2
  fail=1
fi
if printf '%s\n' "$atomic" | grep -qE 'docker (stop|rm)[[:space:]].*taskfe-nginx|compose .*(up --force|up -d --force|recreate)'; then
  echo "FAIL: atomic-vite-build.sh 禁止 docker stop / recreate taskfe-nginx" >&2
  fail=1
fi
if ! printf '%s\n' "$atomic" | grep -qE '保留现有'; then
  echo "FAIL: 构建失败时应明示保留现有 html/release" >&2
  fail=1
fi

# ── 2. lifecycle build 与 npm build 均委托 SSOT ──
if ! printf '%s\n' "$lifecycle" | grep -qE 'atomic-vite-build\.sh'; then
  echo "FAIL: runall-lifecycle.sh build 须调用 atomic-vite-build.sh" >&2
  fail=1
fi
if ! grep -q 'atomic-vite-build.sh' "$pkg"; then
  echo "FAIL: package.json build:vite 须调用 atomic-vite-build.sh" >&2
  fail=1
fi
if grep -qE '"build:vite": "rm -rf dist &&' "$pkg"; then
  echo "FAIL: package.json build:vite 仍先 rm -rf dist" >&2
  fail=1
fi

# ── 3. start 拒绝无 public/html/index.html；不走 vite preview ──
lifecycle_start="$(sed -n '/^  start)/,/^  ;;$/p' "$script_dir/runall-lifecycle.sh")"
if ! printf '%s\n' "$lifecycle_start" | grep -qE 'public/html/index.html|html/index.html'; then
  echo "FAIL: start 分支必须检查 public/html/index.html" >&2
  fail=1
fi
if printf '%s\n' "$lifecycle" | grep -qE 'npm run preview|exec npm run preview'; then
  echo "FAIL: runAll start 不得再 exec npm run preview" >&2
  fail=1
fi
if ! printf '%s\n' "$lifecycle" | grep -qE 'docker compose'; then
  echo "FAIL: start/stop 须走 docker compose" >&2
  fail=1
fi
if printf '%s\n' "$lifecycle" | grep -qE 'pkill|kill -9'; then
  echo "FAIL: stop 禁止 pkill / kill -9 :4000" >&2
  fail=1
fi

seed_old_release() {
  local root="$1"
  mkdir -p "$root/public/releases/old" "$root/node_modules/.bin" "$root/scripts"
  echo '<html>KEEP</html>' > "$root/public/releases/old/index.html"
  ln -sfn "releases/old" "$root/public/html"
}

# ── 4. 行为：失败路径不得删除已有 html（仿真假 vite）──
probe_dir="$(mktemp -d)"
seed_old_release "$probe_dir"
cat > "$probe_dir/node_modules/.bin/vite" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
outdir=""
prev=""
for a in "$@"; do
  if [[ "$prev" == "--outDir" ]]; then outdir="$a"; fi
  prev="$a"
done
mkdir -p "${outdir#../}" 2>/dev/null || true
exit 0
EOF
chmod +x "$probe_dir/node_modules/.bin/vite"
cp "$script_dir/atomic-vite-build.sh" "$probe_dir/scripts/atomic-vite-build.sh"
(
  cd "$probe_dir"
  export PATH="$probe_dir/node_modules/.bin:$PATH"
  sed -i 's/npx vite build/vite build/' scripts/atomic-vite-build.sh
  set +e
  bash scripts/atomic-vite-build.sh >/tmp/atomic_fail_probe.out 2>&1
  rc=$?
  set -e
  if [[ "$rc" -eq 0 ]]; then
    echo "FAIL: 残缺 staging 时 atomic build 应失败" >&2
    fail=1
  fi
  if [[ ! -f public/html/index.html ]]; then
    echo "FAIL: 失败后现网 public/html/index.html 被抹掉" >&2
    fail=1
  fi
  if ! grep -q 'KEEP' public/html/index.html; then
    echo "FAIL: 失败后 html 内容被改写" >&2
    fail=1
  fi
  if [[ -d public/.next ]]; then
    echo "FAIL: 失败后应清理残缺 public/.next" >&2
    fail=1
  fi
)
rm -rf "$probe_dir"

# ── 5. 行为：成功路径 staging → release + 切 symlink（假 vite 写出完整 index）──
ok_dir="$(mktemp -d)"
seed_old_release "$ok_dir"
echo '<html>OLD</html>' > "$ok_dir/public/releases/old/index.html"
cat > "$ok_dir/node_modules/.bin/vite" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
outdir=""
prev=""
for a in "$@"; do
  if [[ "$prev" == "--outDir" ]]; then outdir="$a"; fi
  prev="$a"
done
target="${outdir#../}"
mkdir -p "$target"
echo '<html>NEW</html>' > "$target/index.html"
EOF
chmod +x "$ok_dir/node_modules/.bin/vite"
cp "$script_dir/atomic-vite-build.sh" "$ok_dir/scripts/atomic-vite-build.sh"
(
  cd "$ok_dir"
  export PATH="$ok_dir/node_modules/.bin:$PATH"
  sed -i 's/npx vite build/vite build/' scripts/atomic-vite-build.sh
  bash scripts/atomic-vite-build.sh >/tmp/atomic_ok_probe.out 2>&1
  if [[ ! -L public/html ]]; then
    echo "FAIL: 成功后 public/html 必须是 symlink" >&2
    fail=1
  fi
  if ! grep -q 'NEW' public/html/index.html; then
    echo "FAIL: 成功后 html 未换成新产物" >&2
    fail=1
  fi
  if [[ -d public/.next ]]; then
    echo "FAIL: 成功后应清理 public/.next" >&2
    fail=1
  fi
  if [[ ! -f public/releases/old/index.html ]]; then
    echo "FAIL: 成功后应保留上一 release（releaseKeep）" >&2
    fail=1
  fi
)
rm -rf "$ok_dir"

# ── 6. 行为：vite 缺失时自动 npm ci 再构建 ──
auto_dir="$(mktemp -d)"
seed_old_release "$auto_dir"
mkdir -p "$auto_dir/bin"
cat > "$auto_dir/bin/npm" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$PWD/node_modules/.bin"
cat > "$PWD/node_modules/.bin/vite" <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
outdir=""
prev=""
for a in "$@"; do
  if [[ "$prev" == "--outDir" ]]; then outdir="$a"; fi
  prev="$a"
done
target="${outdir#../}"
mkdir -p "$target"
echo '<html>NEW</html>' > "$target/index.html"
EOS
chmod +x "$PWD/node_modules/.bin/vite"
EOF
chmod +x "$auto_dir/bin/npm"
cp "$script_dir/atomic-vite-build.sh" "$auto_dir/scripts/atomic-vite-build.sh"
(
  cd "$auto_dir"
  export PATH="$auto_dir/bin:$auto_dir/node_modules/.bin:$PATH"
  sed -i 's/npx vite build/vite build/' scripts/atomic-vite-build.sh
  bash scripts/atomic-vite-build.sh >/tmp/atomic_auto_ci.out 2>&1
  if ! grep -q 'npm ci 完成' /tmp/atomic_auto_ci.out; then
    echo "FAIL: vite 缺失时应自动 npm ci 后再构建" >&2
    fail=1
  fi
  if ! grep -q 'NEW' public/html/index.html; then
    echo "FAIL: 自动 npm ci 后 html 未换成新产物" >&2
    fail=1
  fi
)
rm -rf "$auto_dir"

# ── 7. 行为：npm ci 失败时 exit 127 且保留现网 html ──
ci_fail_dir="$(mktemp -d)"
seed_old_release "$ci_fail_dir"
mkdir -p "$ci_fail_dir/bin" "$ci_fail_dir/scripts"
cat > "$ci_fail_dir/bin/npm" <<'EOF'
#!/usr/bin/env bash
echo "simulated npm ci failure" >&2
exit 1
EOF
chmod +x "$ci_fail_dir/bin/npm"
cp "$script_dir/atomic-vite-build.sh" "$ci_fail_dir/scripts/atomic-vite-build.sh"
(
  cd "$ci_fail_dir"
  export PATH="$ci_fail_dir/bin:$PATH"
  set +e
  bash scripts/atomic-vite-build.sh >/tmp/atomic_ci_fail.out 2>&1
  rc=$?
  set -e
  if [[ "$rc" -ne 127 ]]; then
    echo "FAIL: npm ci 失败时应 exit 127（实际 $rc）" >&2
    fail=1
  fi
  if ! grep -q 'KEEP' public/html/index.html; then
    echo "FAIL: npm ci 失败后现网 html 被抹掉" >&2
    fail=1
  fi
)
rm -rf "$ci_fail_dir"

# ── 8. 行为：内存低于 HARD 阈值时告警，但不阻止正常构建 ──
pre_dir="$(mktemp -d)"
seed_old_release "$pre_dir"
cat > "$pre_dir/node_modules/.bin/vite" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
outdir=""
prev=""
for a in "$@"; do
  if [[ "$prev" == "--outDir" ]]; then outdir="$a"; fi
  prev="$a"
done
target="${outdir#../}"
mkdir -p "$target"
echo '<html>NEW</html>' > "$target/index.html"
EOF
chmod +x "$pre_dir/node_modules/.bin/vite"
cp "$script_dir/atomic-vite-build.sh" "$pre_dir/scripts/atomic-vite-build.sh"
(
  cd "$pre_dir"
  export PATH="$pre_dir/node_modules/.bin:$PATH"
  export TASKFE_BUILD_MEM_AVAIL_MB=2048
  export TASKFE_BUILD_AUTO_FREE=0
  sed -i 's/npx vite build/vite build/' scripts/atomic-vite-build.sh
  bash scripts/atomic-vite-build.sh >/tmp/atomic_mem_low.out 2>&1
  if ! grep -q '可用内存 2048MiB' /tmp/atomic_mem_low.out; then
    echo "FAIL: 内存低于 HARD 阈值时应告警" >&2
    fail=1
  fi
  if ! grep -q 'NEW' public/html/index.html; then
    echo "FAIL: 内存预检不应阻止正常构建" >&2
    fail=1
  fi
)
rm -rf "$pre_dir"

# ── 9. 行为：首次 OOM（rc=137）→ 释放后重试成功 ──
retry_dir="$(mktemp -d)"
seed_old_release "$retry_dir"
cat > "$retry_dir/node_modules/.bin/vite" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ -f "$PWD/.oom_attempted" ]]; then
  outdir=""
  prev=""
  for a in "$@"; do
    if [[ "$prev" == "--outDir" ]]; then outdir="$a"; fi
    prev="$a"
  done
  target="${outdir#../}"
  mkdir -p "$target"
  echo '<html>NEW</html>' > "$target/index.html"
  exit 0
fi
touch "$PWD/.oom_attempted"
echo "Simulated OOM kill" >&2
exit 137
EOF
chmod +x "$retry_dir/node_modules/.bin/vite"
cp "$script_dir/atomic-vite-build.sh" "$retry_dir/scripts/atomic-vite-build.sh"
(
  cd "$retry_dir"
  export PATH="$retry_dir/node_modules/.bin:$PATH"
  export TASKFE_BUILD_MEM_AVAIL_MB=5000
  export TASKFE_BUILD_AUTO_FREE=0
  sed -i 's/npx vite build/vite build/' scripts/atomic-vite-build.sh
  bash scripts/atomic-vite-build.sh >/tmp/atomic_oom_retry.out 2>&1
  if ! grep -q 'OOM' /tmp/atomic_oom_retry.out; then
    echo "FAIL: 首次 OOM 时应提示重试" >&2
    fail=1
  fi
  if ! grep -q 'NEW' public/html/index.html; then
    echo "FAIL: OOM 重试后 html 应换成新产物" >&2
    fail=1
  fi
)
rm -rf "$retry_dir"

# ── 10. 静态：构建脚本包含内存预检与 OOM 重试逻辑 ──
if ! printf '%s\n' "$atomic" | grep -qE 'MemAvailable|TASKFE_BUILD_MEM_AVAIL_MB'; then
  echo "FAIL: atomic-vite-build.sh 须含内存预检（MemAvailable）" >&2
  fail=1
fi
if ! printf '%s\n' "$atomic" | grep -qE 'exit 137|heap out of memory'; then
  echo "FAIL: atomic-vite-build.sh 须识别 OOM（rc=137 / heap out of memory）" >&2
  fail=1
fi
if ! printf '%s\n' "$atomic" | grep -qE 'kafka-kafka-ui-1'; then
  echo "FAIL: atomic-vite-build.sh 须登记可自动停止的非关键容器 kafka-ui" >&2
  fail=1
fi

# ── 11. compose 挂父目录 public（T8：容器内 html 仍是 symlink）──
compose="$taskfe_dir/docker-compose.yml"
if [[ ! -f "$compose" ]]; then
  echo "FAIL: 缺少 $compose" >&2
  fail=1
elif ! grep -qE 'app/public:/srv/taskfe' "$compose"; then
  echo "FAIL: compose 须 bind-mount app/public 父目录（含 html symlink）" >&2
  fail=1
elif grep -qE 'public/html:' "$compose"; then
  echo "FAIL: compose 禁止只 mount 已解析的 html 目标" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  echo "test_build_clean_dist.sh FAILED" >&2
  exit 1
fi
echo "test_build_clean_dist.sh OK"
