#!/usr/bin/env bash
# 架构视图自动归档 — 只保留最近 N 个版本，其余移入 architecture/archive/。
#
# 背景：application-integration / enterprise-landscape 每个迭代版本都会生成
#       v<N>-<view>-*.puml/.diff.archimate/.full.archimate/.mermaid.md 等文件
#       （.archimate 分增量 .diff 与全量 .full 两份），长期累积使主目录
#       充满历史细节。本脚本按「版本号」分组（同版本各格式 + .bak 视为一组），
#       保留最新 N 个版本，旧版本移入 archive/ 子目录（只移动不删除，git 历史完整）。
#       通配符 v[0-9]*-<view>-* 自动覆盖 .diff.archimate / .full.archimate 后缀，无需另行匹配。
#
# 用法（docs 仓库根）：
#   bash architecture/scripts/archive-old-views.sh            # 保留最近 5 版
#   bash architecture/scripts/archive-old-views.sh --keep 3   # 自定义保留数量
#   bash architecture/scripts/archive-old-views.sh --dry-run  # 只预览不移动
# 由 docs/.githooks/pre-commit 在检测到新视图版本时自动调用。
set -euo pipefail

# ── 参数解析 ────────────────────────────────────────────────
KEEP=5
DRY_RUN=0
while [ $# -gt 0 ]; do
  case "$1" in
    --keep) KEEP="$2"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    *) echo "❌ unknown arg: $1 (usage: --keep N / --dry-run)" >&2; exit 2 ;;
  esac
done
if ! [[ "$KEEP" =~ ^[0-9]+$ ]] || [ "$KEEP" -lt 1 ]; then
  echo "❌ --keep 必须为正整数" >&2
  exit 2
fi

# ── 定位 docs 仓库根 ────────────────────────────────────────
ROOT="$(cd "$(git rev-parse --show-toplevel 2>/dev/null || echo "$(dirname "$0")/../..")" && pwd)"
ARCH_DIR="$ROOT/architecture"
ARCHIVE_DIR="$ARCH_DIR/archive"
VIEWS="application-integration enterprise-landscape"

if [ ! -d "$ARCH_DIR" ]; then
  echo "⚠ architecture/ not found at $ROOT — archive skipped"
  exit 0
fi

[ "$DRY_RUN" -eq 1 ] && echo "▸ DRY RUN — 仅预览，不移动任何文件"
echo "▸ 保留规则: 每视图最近 $KEEP 个版本，其余 → archive/"

total_moved=0
for view in $VIEWS; do
  # 主目录下该视图的所有版本文件（不含 archive/）
  mapfile -t files < <(find "$ARCH_DIR" -maxdepth 1 -type f \
    -name "v[0-9]*-${view}-*" ! -path "*/archive/*" 2>/dev/null | sort)
  [ ${#files[@]} -eq 0 ] && continue

  # 提取版本号并按数字排序（v9 < v10 < v11），去重
  versions=()
  for f in "${files[@]}"; do
    v="$(basename "$f" | sed -E 's/^v([0-9]+)-.*/\1/')"
    versions+=("$v")
  done
  mapfile -t sorted < <(printf '%s\n' "${versions[@]}" | sort -n -u)

  # 保留最新 KEEP 个版本号
  total="${#sorted[@]}"
  keep_from=$((total - KEEP))
  if [ "$keep_from" -lt 0 ]; then keep_from=0; fi

  declare -A keep_set=()
  for ((i = keep_from; i < total; i++)); do
    keep_set["${sorted[$i]}"]=1
  done

  moved=0
  for f in "${files[@]}"; do
    v="$(basename "$f" | sed -E 's/^v([0-9]+)-.*/\1/')"
    if [ -n "${keep_set[$v]:-}" ]; then
      continue
    fi
    mkdir -p "$ARCHIVE_DIR"
    if [ "$DRY_RUN" -eq 1 ]; then
      echo "  [move] $(basename "$f")"
    else
      mv "$f" "$ARCHIVE_DIR/"
    fi
    moved=$((moved + 1))
  done

  if [ "$moved" -gt 0 ]; then
    echo "▸ $view: ${#sorted[@]} 个版本，保留 ${#keep_set[@]} 个（v${sorted[*]: -KEEP}），归档 $moved 个文件"
  else
    echo "▸ $view: ${#sorted[@]} 个版本 ≤ $KEEP，无需归档"
  fi
  total_moved=$((total_moved + moved))
done

echo "▸ 归档完成: $total_moved 个文件"$([ "$DRY_RUN" -eq 1 ] && echo "（DRY RUN，未实际移动）")
