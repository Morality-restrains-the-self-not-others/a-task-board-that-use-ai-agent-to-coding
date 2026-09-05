#!/usr/bin/env bash
# ============================================================================
# trae-agent-docker-push-scan.sh — 约束 46：镜像相关变更 → 登记 pending 推送
# ============================================================================
# 由 auto-commit.sh（Stop / SessionEnd）调用。检测 trae-agent 是否含会影响
# onlineServiceJS 镜像的变更（脏工作树，或 HEAD 相对上次成功推送 SHA 超前），
# 若有则写入 .runall/trae_agent_docker_push_pending。
#
# fail-open：任一步失败仅 stderr 告警，exit 0，不阻断 auto-commit。
# ============================================================================
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel 2>/dev/null)" || exit 0
cd "$REPO_ROOT" || exit 0

SUB="trae-agent"
SUB_PATH="$REPO_ROOT/$SUB"
RUNALL_DIR="${TRAE_AGENT_DOCKER_PUSH_STATE_DIR:-$REPO_ROOT/.runall}"
PENDING_FILE="$RUNALL_DIR/trae_agent_docker_push_pending"
SHA_FILE="$RUNALL_DIR/trae_agent_docker_push_sha"

if [ ! -d "$SUB_PATH" ]; then
  exit 0
fi

# 路径是否「影响镜像」：排除文档与测试，命中 onlineServiceJS / trae_agent / 构建上下文。
is_image_affecting() {
  local p="$1"
  case "$p" in
    *.md|*.mdc|*.rst) return 1 ;;
  esac
  case "$p" in
    docs/*|*/docs/*) return 1 ;;
    */test/*|*/tests/*|*/__tests__/*) return 1 ;;
    *.test.*|*_test.*|*.spec.*) return 1 ;;
  esac
  case "$p" in
    onlineServiceJS/*|trae_agent/*) return 0 ;;
    Dockerfile|*/Dockerfile) return 0 ;;
    docker/*|*/docker/*) return 0 ;;
    pyproject.toml|poetry.lock|uv.lock|requirements*.txt|package.json|package-lock.json|yarn.lock|pnpm-lock.yaml)
      return 0 ;;
    *.toml|*.lock)
      # 根目录构建相关
      case "$p" in
        */*) return 1 ;;
        *) return 0 ;;
      esac
      ;;
  esac
  return 1
}

mkdir -p "$RUNALL_DIR" || true

NEED=0
REASON=""

# ── 1. 脏工作树（子模块或普通目录）────────────────────────────────
if [ -d "$SUB_PATH/.git" ] || [ -f "$SUB_PATH/.git" ]; then
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    # porcelain: XY PATH 或 XY ORIG -> PATH
    path="${line:3}"
    case "$path" in
      *" -> "*) path="${path##* -> }" ;;
    esac
    path="${path#\"}"
    path="${path%\"}"
    if is_image_affecting "$path"; then
      NEED=1
      REASON="dirty:$path"
      break
    fi
  done < <(git -C "$SUB_PATH" status --porcelain 2>/dev/null || true)
fi

# ── 2. HEAD 相对水位线超前（已提交未推镜像）──────────────────────
if [ "$NEED" -eq 0 ] && { [ -d "$SUB_PATH/.git" ] || [ -f "$SUB_PATH/.git" ]; }; then
  HEAD_SHA="$(git -C "$SUB_PATH" rev-parse HEAD 2>/dev/null || true)"
  LAST_SHA=""
  if [ -f "$SHA_FILE" ]; then
    LAST_SHA="$(tr -d '[:space:]' < "$SHA_FILE" 2>/dev/null || true)"
  fi
  if [ -n "$HEAD_SHA" ] && [ "$HEAD_SHA" != "$LAST_SHA" ]; then
    # 水位线缺失或不同：检查自 LAST_SHA..HEAD 是否有镜像相关文件变更
    if [ -z "$LAST_SHA" ] || ! git -C "$SUB_PATH" cat-file -e "$LAST_SHA^{commit}" 2>/dev/null; then
      # 无有效水位线：用最近一次提交的文件列表粗判（避免无变更空仓误登）
      while IFS= read -r path; do
        [ -z "$path" ] && continue
        if is_image_affecting "$path"; then
          NEED=1
          REASON="head-no-watermark:$path"
          break
        fi
      done < <(git -C "$SUB_PATH" diff-tree --no-commit-id --name-only -r HEAD 2>/dev/null || true)
    else
      while IFS= read -r path; do
        [ -z "$path" ] && continue
        if is_image_affecting "$path"; then
          NEED=1
          REASON="ahead:$path"
          break
        fi
      done < <(git -C "$SUB_PATH" diff --name-only "$LAST_SHA"..HEAD 2>/dev/null || true)
    fi
  fi
fi

if [ "$NEED" -eq 1 ]; then
  {
    echo "pending=1"
    echo "reason=$REASON"
    echo "scanned_at=$(date -Iseconds 2>/dev/null || date)"
    if [ -d "$SUB_PATH/.git" ] || [ -f "$SUB_PATH/.git" ]; then
      echo "head_sha=$(git -C "$SUB_PATH" rev-parse HEAD 2>/dev/null || true)"
    fi
  } > "$PENDING_FILE" || true
  echo "trae-agent-docker-push-scan: registered pending ($REASON)" >&2
else
  # 无镜像相关变更时不清除已有 pending（可能后台推送仍在进行）；仅无新需求时保持原状
  :
fi

exit 0
