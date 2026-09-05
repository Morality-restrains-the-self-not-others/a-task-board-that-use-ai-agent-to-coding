#!/usr/bin/env bash
# 幂等：将分散的旧 SQLite 文件搬迁到 db/<service>/（见 registry.yaml）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

move_if_needed() {
  local src="$1"
  local dst="$2"
  if [[ -f "$dst" ]]; then
    echo "skip (exists): $dst"
    return 0
  fi
  if [[ ! -f "$src" ]]; then
    echo "skip (no source): $src"
    return 0
  fi
  mkdir -p "$(dirname "$dst")"
  mv "$src" "$dst"
  echo "moved: $src -> $dst"
  for ext in -wal -shm; do
    if [[ -f "${src}${ext}" ]]; then
      mv "${src}${ext}" "${dst}${ext}"
      echo "moved: ${src}${ext} -> ${dst}${ext}"
    fi
  done
}

move_if_needed "$ROOT/task2app/Saas_project/db.sqlite3" "$ROOT/db/saas/saas.sqlite3"
move_if_needed "$ROOT/taskAuth/data/auth.db" "$ROOT/db/task-auth/auth.sqlite3"
move_if_needed "$ROOT/taskBill/data/billing.db" "$ROOT/db/task-bill/billing.sqlite3"
move_if_needed "$ROOT/gitOauth/db.sqlite3" "$ROOT/db/git-oauth/git-oauth.sqlite3"
move_if_needed "$ROOT/task2app/Saas_Ai_Provider/db.sqlite3" "$ROOT/db/ai-provider/ai-provider.sqlite3"

echo "done."
