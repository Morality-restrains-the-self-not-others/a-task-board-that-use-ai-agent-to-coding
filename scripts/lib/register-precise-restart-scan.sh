#!/usr/bin/env bash
# ============================================================================
# register-precise-restart-scan.sh — 「服务源码改动 → 精准编译重启自动登记」强制机制
# ============================================================================
# 由 auto-commit.sh（Stop / SessionEnd hook）调用：扫描 meta 仓库全部子模块的
# 未提交变更，凡 runAll 托管服务的源代码/构建脚本/配置有改动，自动将服务登记到
# <仓库根>/.runall/precise_restart_services.txt（42 号约束文硬约束的落地执行）。
#
# 登记规则（与 .ai/01_project_constraints/42_precise_restart_service_registration.md 对齐）:
#   - 触发: 子模块工作树含非文档/非测试变更（源码、build.sh、go.mod、
#     package.json、conf/<app>/config.yaml 等），且该子模块是 runAll.yaml 中
#     某服务的 working_dir（或首段别名，如 taskFE/app → taskFE）。
#   - 不触发: 仅文档（*.md / *.rst / docs/）、仅测试（tests/ / __tests__ /
#     *.test.* / *.spec.* / *_test.go / test_*.py）、.gitignore 类元文件、
#     未被 runAll 托管的工作目录、working_dir 为 "." 的基础设施服务（docker-* 等）。
#   - 同一 working_dir 挂多个服务（如 taskEvents 32 个）→ 全部登记。
#   - 已登记服务去重追加；文件被「精准编译重启」执行清空后，仅当存在
#     「新于 consumed-at 水位线」的源码变更时才自动重登（避免立刻把刚部署的脏树重登）。
#
# 局限: 仅覆盖「工作树仍脏」的变更（智能体会话典型流程：编辑 → Stop 触发扫描）。
# 已完成提交且工作树干净的子模块（如跨会话已提交的代码）不会回溯登记 —— 需人工
# 用 scripts/register-precise-restart.sh 登记。
#
# 幂等/失败策略: 无变更 → no-op；任一步失败仅输出 stderr 告警并继续（fail-open，
# 不阻断 auto-commit 主流程）。登记文件在 .gitignore 中（.runall/），纯运行时状态。
# ============================================================================
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$REPO_ROOT" || exit 0

CONF="$REPO_ROOT/conf/runAll.yaml"
REG_FILE="${RUNALL_PRECISE_RESTART_FILE:-$REPO_ROOT/.runall/precise_restart_services.txt}"
CONSUMED_AT_FILE="$(dirname "$REG_FILE")/precise_restart_consumed_at"

[ -f "$CONF" ] || { echo "precise-restart-scan: $CONF 不存在，跳过登记扫描" >&2; exit 0; }

# 读取精准编译重启消费水位线（unix 秒）；缺失/非法视为 0（不抑制重登）。
CONSUMED_AT=0
if [ -f "$CONSUMED_AT_FILE" ]; then
  _raw="$(tr -d '[:space:]' < "$CONSUMED_AT_FILE" 2>/dev/null || true)"
  case "$_raw" in
    ''|*[!0-9]*) CONSUMED_AT=0 ;;
    *) CONSUMED_AT="$_raw" ;;
  esac
fi

# 登记行兼容纯服务名与 Go 写回的 name\tstate\tts。
is_already_registered() {
  local svc="$1"
  grep -qE $'^'"${svc}"$'($|\t)' "$REG_FILE" 2>/dev/null
}

# 脏文件最新 mtime（秒）；无可 stat 文件时返回 0。
newest_code_mtime() {
  local sub="$1"
  shift
  local newest=0 mt f
  for f in "$@"; do
    [ -e "$REPO_ROOT/$sub/$f" ] || continue
    mt="$(stat -c %Y "$REPO_ROOT/$sub/$f" 2>/dev/null || true)"
    case "$mt" in
      ''|*[!0-9]*) continue ;;
    esac
    if [ "$mt" -gt "$newest" ]; then
      newest="$mt"
    fi
  done
  printf '%s\n' "$newest"
}

# ── OPT-20260810-055: 内容摘要快照补强 ──────────────────────────────────
# 工具改写内容但保持 mtime（罕见）或仅 git index 变化时，mtime 水位线会漏登。
# 这里用 git hash-object 摘要相对上次快照做 OR：当前摘要偏离快照即视为新变更。
# 快照格式: <submodule>\t<path>\t<digest>（每行）；文件缺失 → 空摘要（视为变化）。
BLOB_SNAPSHOT_FILE="$(dirname "$REG_FILE")/precise_restart_scan_blobs"

# 取某代码文件的当前 blob 摘要（文件缺失/不可读 → 空串）。
code_file_digest() {
  local sub="$1" p="$2"
  git -C "$REPO_ROOT/$sub" hash-object -- "$p" 2>/dev/null || true
}

# 该子模块全部代码文件内容是否与上次快照一致（快照无该文件记录 → 不一致）。
code_files_content_unchanged() {
  local sub="$1" f digest stored
  shift
  for f in "$@"; do
    digest="$(code_file_digest "$sub" "$f")"
    stored="$(awk -F '\t' -v s="$sub" -v f="$f" '$1==s && $2==f {print $3}' "$BLOB_SNAPSHOT_FILE" 2>/dev/null | head -1)"
    if [ -z "$stored" ] || [ "$stored" != "$digest" ]; then
      return 1
    fi
  done
  return 0
}

# 更新快照中该子模块代码文件条目为当前摘要（保留其它子模块条目）。
update_blob_snapshot() {
  local sub="$1" f digest
  shift
  mkdir -p "$(dirname "$BLOB_SNAPSHOT_FILE")"
  for f in "$@"; do
    digest="$(code_file_digest "$sub" "$f")"
    if [ -f "$BLOB_SNAPSHOT_FILE" ]; then
      grep -vF "$(printf '%s\t%s\t' "$sub" "$f")" "$BLOB_SNAPSHOT_FILE" > "$BLOB_SNAPSHOT_FILE.tmp"
      mv "$BLOB_SNAPSHOT_FILE.tmp" "$BLOB_SNAPSHOT_FILE"
    fi
    printf '%s\t%s\t%s\n' "$sub" "$f" "$digest" >> "$BLOB_SNAPSHOT_FILE"
  done
}

# ── 1. 构建 working_dir 首段 → 服务名映射（跳过 "." 基础设施服务）──────
# 结构: 6 空格 `      - name: <svc>` 后随 8 空格 `        working_dir: <dir>`。
declare -A SEG_TO_SVCS=()
_cur_svc=""
while IFS= read -r _line; do
  case "$_line" in
    '      - name: '*)
      _cur_svc="${_line#      - name: }"
      ;;
    '        working_dir: '*)
      if [ -n "$_cur_svc" ]; then
        _wd="${_line#        working_dir: }"
        if [ "$_wd" != "." ]; then
          _seg="${_wd%%/*}"   # taskFE/app → taskFE（与 register 脚本/后端解析一致）
          SEG_TO_SVCS["$_seg"]="${SEG_TO_SVCS["$_seg"]:-} $_cur_svc"
        fi
        _cur_svc=""
      fi
      ;;
  esac
done < "$CONF"

[ "${#SEG_TO_SVCS[@]}" -gt 0 ] || { echo "precise-restart-scan: runAll.yaml 未解析出可登记服务" >&2; exit 0; }

# ── 2. 收集子模块路径（仅扫描 .gitmodules 声明的子模块）────────────────
declare -A SUBMODULES=()
while IFS= read -r _p; do
  [ -n "$_p" ] && SUBMODULES["$_p"]=1
done < <(git config --file "$REPO_ROOT/.gitmodules" --get-regexp '^submodule\..*\.path$' 2>/dev/null | awk '{print $2}')
[ "${#SUBMODULES[@]}" -gt 0 ] || exit 0

# ── 3. 判断「文档/测试专用」路径（42 号文：仅文档/测试变更不触发登记）──
is_doc_or_test() {
  local p="$1"
  case "$p" in
    *.md|*.rst|.gitignore|.gitattributes) return 0 ;;
  esac
  case "/$p" in
    */docs/*|*/tests/*|*/__tests__/*) return 0 ;;
  esac
  case "$p" in
    *.test.*|*.spec.*|*_test.go|test_*.py) return 0 ;;
  esac
  return 1
}

# ── 4. 扫描 meta 状态中的脏子模块，登记受影响服务 ──────────────────────
# meta porcelain 行: ` M taskFE` / `MM taskFE` / `? docs`（字段 2 为子模块路径）
REGISTERED=()
while IFS= read -r _meta_line; do
  [ -n "$_meta_line" ] || continue
  _sub="${_meta_line:3}"
  _sub="${_sub#\"}"; _sub="${_sub%\"}"
  [ -n "${SUBMODULES[$_sub]+x}" ] || continue        # 非子模块（meta 根文件）跳过
  [ -n "${SEG_TO_SVCS[$_sub]+x}" ] || continue       # 非服务工作目录跳过

  _svcs="${SEG_TO_SVCS[$_sub]}"
  _changes="$(git -C "$REPO_ROOT/$_sub" status --porcelain --untracked-files=all 2>/dev/null)" || continue
  [ -n "$_changes" ] || continue                     # 仅指针移动/工作树干净 → 不回溯登记

  # 收集非文档/测试脏路径；全部为文档/测试 → 不触发
  _code_files=()
  while IFS= read -r _cf; do
    _p="${_cf:3}"                                    # 去掉 `XY ` 状态前缀
    _p="${_p##*" -> "}"                              # rename 行取新路径
    _p="${_p#\"}"; _p="${_p%\"}"
    if ! is_doc_or_test "$_p"; then
      _code_files+=("$_p")
    fi
  done <<< "$_changes"
  [ "${#_code_files[@]}" -gt 0 ] || continue

  # 水位线抑制：脏源码 mtime 均不新于最近一次精准编译重启消费时刻 → 跳过
  # （变更已被编译重启消化，工作树仍脏不构成新登记理由）。
  # OPT-20260810-055 补强：mtime 未变但内容已变时，以 blob 摘要快照 OR ——
  # 当前摘要偏离快照即视为新变更，不因 mtime 陈旧而漏登。
  if [ "$CONSUMED_AT" -gt 0 ]; then
    _newest="$(newest_code_mtime "$_sub" "${_code_files[@]}")"
    if [ "$_newest" -gt 0 ] && [ "$_newest" -le "$CONSUMED_AT" ]; then
      if code_files_content_unchanged "$_sub" "${_code_files[@]}"; then
        update_blob_snapshot "$_sub" "${_code_files[@]}"
        continue
      fi
    fi
  fi
  update_blob_snapshot "$_sub" "${_code_files[@]}"

  for _svc in $_svcs; do
    REGISTERED+=("$_svc")
  done
done < <(git status --porcelain 2>/dev/null)
[ "${#REGISTERED[@]}" -gt 0 ] || exit 0

# ── 5. 去重追加登记 ────────────────────────────────────────────────────
mkdir -p "$(dirname "$REG_FILE")"
touch "$REG_FILE"
ADDED=()
for _svc in "${REGISTERED[@]}"; do
  if ! is_already_registered "$_svc"; then
    printf '%s\n' "$_svc" >> "$REG_FILE"
    ADDED+=("$_svc")
  fi
done

if [ "${#ADDED[@]}" -gt 0 ]; then
  echo "precise-restart-scan: 检测到服务源码变更，已自动登记: ${ADDED[*]} → $REG_FILE（点击 http://10.2.150.68:9999/ 「精准编译重启」执行）" >&2
else
  echo "precise-restart-scan: 变更服务均已登记（无新增）: ${REGISTERED[*]}" >&2
fi
exit 0
