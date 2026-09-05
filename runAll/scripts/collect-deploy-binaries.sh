#!/usr/bin/env bash
# Collect Release payloads into one directory for hand-copy. Do not commit the dest.
# Incremental: skip a file when dest exists with same size and mtime >= source.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
META_ROOT="${META_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
DEST="${1:-$META_ROOT/deploy-binaries}"
# Live clone-run tree (artifacts/ + installed ELFs). Default $HOME/bin/daydaymoney-deploy.
RAM_DEPLOY="${RAM_DEPLOY:-${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}}"
STAGING="${COLLECT_STAGING:-/tmp/daydaymoney-release-20260831-conf-local}"

log() { echo "[collect-deploy-binaries] $*"; }

ELFS=(
  runAll taskAuth taskBill taskCloudService taskProjectService
  taskTaskService taskTenantService taskReferral taskGitOauth
  taskCredentialService taskContainerGateway taskAiProvider
  taskAIComment taskAIEndPoint taskAgentSupport go_relayToTrae valueStream
)

find_src_for() {
  local name="$1" best="" best_m=0 f m
  local files=()
  if [[ -n "${COLLECT_SRC:-}" ]]; then
    files+=("$COLLECT_SRC/$name")
  else
    files+=(
      "$STAGING/$name"
      "$RAM_DEPLOY/artifacts/$name"
      "$META_ROOT/bin/$name"
      "$META_ROOT/$name/bin/$name"
    )
  fi
  for f in "${files[@]}"; do
    [[ -f "$f" ]] || continue
    m="$(stat -c %Y "$f")"
    if [[ -z "$best" || "$m" -gt "$best_m" ]]; then
      best="$f"
      best_m="$m"
    fi
  done
  printf '%s' "$best"
}

copy_if_changed() {
  local src="$1" dest="$2" base ss sd ms md
  base="$(basename "$src")"
  if [[ -f "$dest" ]]; then
    ss="$(stat -c %s "$src")"
    sd="$(stat -c %s "$dest")"
    ms="$(stat -c %Y "$src")"
    md="$(stat -c %Y "$dest")"
    if [[ "$ss" == "$sd" && "$md" -ge "$ms" ]]; then
      log "unchanged $base"
      return 0
    fi
  fi
  cp -a "$src" "$dest"
  log "copied $base from $src"
}

copy_named() {
  local name="$1"
  local src dest
  dest="$DEST/$name"
  src="$(find_src_for "$name")"
  if [[ -z "$src" ]]; then
    return 1
  fi
  copy_if_changed "$src" "$dest"
  if [[ "$name" != *.tar.gz ]]; then
    chmod +x "$dest" || true
  fi
  return 0
}

pack_if_missing() {
  local name="$1" src_dir="$2"
  if [[ -f "$DEST/$name" ]]; then
    return 0
  fi
  if [[ -d "$src_dir" ]]; then
    tar -czf "$DEST/$name" -C "$src_dir" .
    log "packed $name from $src_dir"
    return 0
  fi
  log "missing $name"
  return 1
}

# Live vendor SPA wins over a stale tarball (seed deploy otherwise 404s provider.*).
sync_provider_frontend_archive() {
  local name="taskAiProvider-frontend-dist.tar.gz"
  local dest="$DEST/$name"
  local live="" d dest_m newest
  for d in \
    "$META_ROOT/taskAiProvider/frontend/dist" \
    "$RAM_DEPLOY/taskAiProvider/frontend/dist"
  do
    if [[ -f "$d/index.html" ]]; then
      live="$d"
      break
    fi
  done
  if [[ -z "$live" ]]; then
    copy_named "$name" || pack_if_missing \
      "$name" "$RAM_DEPLOY/taskAiProvider/frontend/dist" || true
    return 0
  fi
  dest_m=0
  [[ -f "$dest" ]] && dest_m="$(stat -c %Y "$dest")"
  newest="$(stat -c %Y "$live/index.html")"
  newest="${newest:-0}"
  if [[ ! -f "$dest" || "$newest" -gt "$dest_m" ]]; then
    tar -czf "$dest" -C "$live" .
    log "packed $name from $live"
    return 0
  fi
  log "unchanged $name"
}

# Live taskEvents/bin wins over a stale tarball (new intents are otherwise never packed).
sync_taskevents_archive() {
  local live="" d dest="$DEST/taskEvents-bin.tar.gz" nsrc ndest dest_m newest found
  for d in "$META_ROOT/taskEvents/bin" "$RAM_DEPLOY/taskEvents/bin"; do
    [[ -d "$d" ]] || continue
    found="$(find "$d" -type f -name 'task-events-*' -print -quit || true)"
    if [[ -n "$found" ]]; then
      live="$d"
      break
    fi
  done
  if [[ -z "$live" ]]; then
    copy_named "taskEvents-bin.tar.gz" || pack_if_missing \
      "taskEvents-bin.tar.gz" "$RAM_DEPLOY/taskEvents/bin" || true
    return 0
  fi
  nsrc="$(find "$live" -type f -name 'task-events-*' | wc -l)"
  nsrc="${nsrc// /}"
  ndest=0
  if [[ -f "${dest}.workers" ]]; then
    ndest="$(tr -d '[:space:]' < "${dest}.workers")"
  elif [[ -f "$dest" ]]; then
    ndest="$(tar -tzf "$dest" | grep -c 'task-events-' || true)"
    ndest="${ndest// /}"
  fi
  dest_m=0
  [[ -f "$dest" ]] && dest_m="$(stat -c %Y "$dest")"
  newest="$(find "$live" -type f -name 'task-events-*' -printf '%T@\n' | sort -n | tail -1)"
  newest="${newest%.*}"
  newest="${newest:-0}"
  if [[ ! -f "$dest" || "$nsrc" -ne "$ndest" || "$newest" -gt "$dest_m" ]]; then
    tar -czf "$dest" -C "$live" .
    printf '%s\n' "$nsrc" > "${dest}.workers"
    log "packed taskEvents-bin.tar.gz from $live ($nsrc workers)"
    return 0
  fi
  printf '%s\n' "$nsrc" > "${dest}.workers"
  log "unchanged taskEvents-bin.tar.gz ($nsrc workers)"
}

# Live taskFE/app/public wins over a stale tarball (frontend update otherwise never repacked).
sync_taskfe_dist_archive() {
  local name="taskFE-dist.tar.gz"
  local dest="$DEST/$name"
  local live="" d nsrc ndest dest_m newest
  for d in \
    "$META_ROOT/taskFE/app/public" \
    "$RAM_DEPLOY/taskFE/app/public"
  do
    # atomic-vite-build 切 public/html → releases/<id>，根目录没有 index.html
    if [[ -f "$d/index.html" || -f "$d/html/index.html" ]]; then
      live="$d"
      break
    fi
  done
  if [[ -z "$live" ]]; then
    copy_named "$name" || pack_if_missing \
      "$name" "$RAM_DEPLOY/taskFE/app/public" || true
    return 0
  fi
  nsrc="$(find "$live" -type f | wc -l)"
  nsrc="${nsrc// /}"
  ndest=0
  if [[ -f "$dest" ]]; then
    ndest="$(tar -tzf "$dest" | grep -v '/$' | wc -l || true)"
    ndest="${ndest// /}"
  fi
  dest_m=0
  [[ -f "$dest" ]] && dest_m="$(stat -c %Y "$dest")"
  newest="$(find "$live" -type f -printf '%T@\n' | sort -n | tail -1)"
  newest="${newest%.*}"
  newest="${newest:-0}"
  if [[ ! -f "$dest" || "$nsrc" -ne "$ndest" || "$newest" -gt "$dest_m" ]]; then
    tar -czf "$dest" -C "$live" .
    log "packed $name from $live ($nsrc files)"
    return 0
  fi
  log "unchanged $name ($nsrc files)"
}

# OPT-20260831-016: 手拷 deploy-binaries/ 后对照 conf.example/releases.yaml 的 sha 校验，
# 避免错 ELF / 旧 tag 静默装上新节点。
verify_release_shas() {
  if [[ "${COLLECT_SKIP_SHA:-0}" == "1" ]]; then
    log "skip sha-verify (COLLECT_SKIP_SHA=1)"
    return 0
  fi
  local rel="$META_ROOT/conf.example/releases.yaml"
  if [[ ! -f "$rel" ]]; then
    log "sha-verify: 无 $rel，跳过"
    return 0
  fi
  local rc=0
  python3 - "$rel" "$DEST" <<'PY' || rc=1
import hashlib, os, sys
try:
    import yaml
except ImportError:  # pragma: no cover
    sys.exit(0)
rel, dest = sys.argv[1], sys.argv[2]
with open(rel, encoding="utf-8") as f:
    data = yaml.safe_load(f) or {}
artifacts = data.get("artifacts") or {}
checked = 0
bad = []
for name, pin in artifacts.items():
    pkg = str(pin.get("package") or "")
    sha = str(pin.get("sha") or "").strip()
    if not pkg or not sha:
        continue
    fname = pkg.rsplit("/", 1)[-1].split("@")[0]
    path = os.path.join(dest, fname)
    if not os.path.isfile(path):
        continue
    digest = hashlib.sha256(open(path, "rb").read()).hexdigest()
    checked += 1
    if digest != sha:
        bad.append(fname)
if bad:
    print(f"SHA MISMATCH: {', '.join(bad)} vs conf.example/releases.yaml", file=sys.stderr)
    sys.exit(1)
print(f"sha-verify: {checked} 个文件与 releases.yaml 一致")
PY
  if [[ $rc -ne 0 ]]; then
    log "SHA-256 校验失败：$DEST 与 conf.example/releases.yaml 不一致"
    return 1
  fi
  return 0
}

mkdir -p "$DEST"
DEST="$(cd "$DEST" && pwd)"

for name in "${ELFS[@]}"; do
  if copy_named "$name"; then
    continue
  fi
  if [[ -f "$META_ROOT/bin/$name" ]]; then
    copy_if_changed "$META_ROOT/bin/$name" "$DEST/$name"
    chmod +x "$DEST/$name" || true
    continue
  fi
  if [[ -f "$META_ROOT/$name/bin/$name" ]]; then
    copy_if_changed "$META_ROOT/$name/bin/$name" "$DEST/$name"
    chmod +x "$DEST/$name" || true
    continue
  fi
  log "missing $name"
done

sync_taskevents_archive
sync_taskfe_dist_archive
sync_provider_frontend_archive

if [[ ! -f "$DEST/runAll" ]]; then
  echo "collect-deploy-binaries: missing runAll (set COLLECT_SRC or build first)" >&2
  if [[ "${COLLECT_SOFT:-0}" == "1" ]]; then
    exit 0
  fi
  exit 1
fi

verify_release_shas || exit 1

{
  echo "collected $(date -Iseconds)"
  echo "dest=$DEST"
  ls -lh "$DEST"
} > "$DEST/MANIFEST.txt"
log "wrote $DEST ($(du -sh "$DEST" | awk '{print $1}'))"
