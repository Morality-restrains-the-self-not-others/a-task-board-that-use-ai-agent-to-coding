#!/usr/bin/env bash
# Install hand-copied Release payloads into a deploy root (ADR-0052 artifact plane).
# Does not download from GitHub. Archive dest matches releases.yaml dest/unpack.
set -euo pipefail

SRC="${1:-}"
ROOT="${2:-}"
if [[ -z "$SRC" || -z "$ROOT" ]]; then
  echo "usage: install-local-artifacts.sh <artifacts-dir> <deploy-root>" >&2
  exit 1
fi
if [[ ! -d "$SRC" ]]; then
  echo "install-local-artifacts: not a directory: $SRC" >&2
  exit 1
fi
ROOT="$(cd "$ROOT" && pwd)"
SRC="$(cd "$SRC" && pwd)"

log() { echo "[install-local-artifacts] $*"; }

archive_escapes() {
  tar -tzf "$1" | grep -Eq '(^|/)\.\.(/|$)|^/'
}

dest_for_archive() {
  local base="$1"
  local pins="$ROOT/releases.yaml"
  local dest=""
  if [[ -f "$pins" ]]; then
    dest="$(python3 - "$pins" "$base" <<'PY'
import sys

try:
    import yaml
except ImportError:
    sys.exit(0)
rel = yaml.safe_load(open(sys.argv[1], encoding="utf-8")) or {}
arts = rel.get("artifacts") or {}
name = sys.argv[2]
stem = name[:-7] if name.endswith(".tar.gz") else name
pin = arts.get(stem) or {}
if str(pin.get("unpack") or "") != "tar.gz":
    sys.exit(0)
print(pin.get("dest") or "")
PY
)" || true
  fi
  if [[ -z "$dest" ]]; then
    case "$base" in
      taskEvents-bin.tar.gz) dest="taskEvents/bin" ;;
      taskFE-dist.tar.gz) dest="taskFE/app/public" ;;
      taskAiProvider-frontend-dist.tar.gz) dest="taskAiProvider/frontend/dist" ;;
    esac
  fi
  printf '%s' "$dest"
}

verify_taskevents_workers() {
  local runsh="$ROOT/taskEvents/run.sh"
  local bin="$ROOT/taskEvents/bin"
  [[ -f "$runsh" ]] || return 0
  python3 - "$runsh" "$bin" <<'PY'
import re
import sys
from pathlib import Path

runsh = Path(sys.argv[1])
binroot = Path(sys.argv[2])
text = runsh.read_text(encoding="utf-8")
m = re.search(r"INTENT_PATHS=\((.*?)\)", text, re.S)
if not m:
    sys.exit(0)
missing = []
for rel in re.findall(r"[a-z0-9_]+/[a-z0-9_]+", m.group(1)):
    d = binroot / rel
    if not d.is_dir() or not any(d.glob("task-events-*")):
        missing.append(rel)
if missing:
    print(
        "install-local-artifacts: taskEvents-bin.tar.gz missing worker(s): "
        + ", ".join(missing)
        + " (re-collect on source: bash runAll/scripts/collect-deploy-binaries.sh)",
        file=sys.stderr,
    )
    sys.exit(1)
PY
}

# cp onto a running ELF hits ETXTBSY; write a sibling then mv (new inode).
# Same size+mtime as dest: skip (incremental; log unchanged).
install_over() {
  local src="$1" dest="$2"
  local tmp ss sd ms md
  mkdir -p "$(dirname "$dest")"
  if [[ -e "$dest" && "$src" -ef "$dest" ]]; then
    return 0
  fi
  if [[ -f "$dest" && -f "$src" ]]; then
    ss="$(stat -c %s "$src")"
    sd="$(stat -c %s "$dest")"
    ms="$(stat -c %Y "$src")"
    md="$(stat -c %Y "$dest")"
    if [[ "$ss" == "$sd" && "$md" -ge "$ms" ]]; then
      log "unchanged $(basename "$dest")"
      return 0
    fi
  fi
  tmp="${dest}.install.$$"
  cp -a "$src" "$tmp"
  mv -f "$tmp" "$dest"
}

unpack_tar_over() {
  local archive="$1" dest="$2"
  local tmp rel f
  tmp="$(mktemp -d "$ROOT/.unpack-XXXXXX")"
  tar --no-same-owner -xzf "$archive" -C "$tmp"
  while IFS= read -r -d '' f; do
    rel="${f#"$tmp"/}"
    mkdir -p "$dest/$(dirname "$rel")"
    mv -f "$f" "$dest/$rel"
  done < <(find "$tmp" -type f -print0)
  rm -rf "$tmp"
}

mkdir -p "$ROOT/bin" "$ROOT/artifacts"
art_abs="$(cd "$ROOT/artifacts" && pwd)"
shopt -s nullglob
for f in "$SRC"/*; do
  [[ -f "$f" ]] || continue
  base="$(basename "$f")"
  case "$base" in
    *.sha|README*|MANIFEST*|.*|*.workers) continue ;;
  esac
  if [[ "$SRC" != "$art_abs" ]]; then
    install_over "$f" "$ROOT/artifacts/$base"
  fi
  if [[ "$base" == *.tar.gz ]]; then
    if archive_escapes "$f"; then
      echo "install-local-artifacts: refusing archive with path escape: $base" >&2
      exit 1
    fi
    dest="$(dest_for_archive "$base")"
    if [[ -z "$dest" ]]; then
      log "skip unknown archive $base"
      continue
    fi
    if [[ "$dest" == /* || "$dest" == *..* ]]; then
      echo "install-local-artifacts: unsafe dest $dest for $base" >&2
      exit 1
    fi
    mkdir -p "$ROOT/$dest"
    unpack_tar_over "$f" "$ROOT/$dest"
    log "unpacked $base -> $dest"
    continue
  fi
  install_over "$f" "$ROOT/bin/$base"
  chmod +x "$ROOT/bin/$base" || true
  log "elf $base"
done
verify_taskevents_workers
