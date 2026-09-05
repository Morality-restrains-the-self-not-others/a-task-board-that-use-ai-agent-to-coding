#!/usr/bin/env bash
# Assemble $DEPLOY_ROOT from an daydaymoney-deploy checkout (ADR-0052).
# Clone-as-root: DEPLOY_ROOT=CONFIG_REPO uses relative symlinks into envs/<env>/.
# Separate deploy root: rsync recipes + conf (never mysql data, never *.local.yaml).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_NAME="${DEPLOY_ENV:-current}"

log() { echo "[layout-from-config-repo] $*"; }

if [[ -z "${CONFIG_REPO:-}" ]]; then
  if [[ -d "$SCRIPT_DIR/../envs" ]]; then
    CONFIG_REPO="$(cd "$SCRIPT_DIR/.." && pwd)"
  else
    echo "layout-from-config-repo: set CONFIG_REPO to the daydaymoney-deploy checkout" >&2
    exit 1
  fi
fi
CONFIG_REPO="$(cd "$CONFIG_REPO" && pwd)"
DEPLOY_ROOT="${DEPLOY_ROOT:-$CONFIG_REPO}"
mkdir -p "$DEPLOY_ROOT"
DEPLOY_ROOT="$(cd "$DEPLOY_ROOT" && pwd)"
SRC="$CONFIG_REPO/envs/$ENV_NAME"
if [[ ! -d "$SRC/conf" ]]; then
  echo "layout-from-config-repo: missing $SRC/conf" >&2
  exit 1
fi

TREES=(
  dockerInfra gitService AiMonitor taskGateway taskSSE taskEvents
  taskFE dataMigrate db runAll trae-agent
)

same_root=0
if [[ "$DEPLOY_ROOT" == "$CONFIG_REPO" ]]; then
  same_root=1
fi

link_or_rsync() {
  local name="$1"
  local from="$SRC/$name"
  local to="$DEPLOY_ROOT/$name"
  if [[ ! -e "$from" && ! -d "$from" ]]; then
    log "skip missing $name"
    return 0
  fi
  if [[ "$same_root" -eq 1 ]]; then
    if [[ -e "$to" && ! -L "$to" ]]; then
      log "keep existing real dir $name"
      return 0
    fi
    ln -sfn "envs/${ENV_NAME}/${name}" "$to"
    log "symlink $name -> envs/${ENV_NAME}/${name}"
    return 0
  fi
  mkdir -p "$to"
  rsync -a --delete \
    --exclude 'data/' --exclude 'data.corrupt.*' \
    --exclude '*.local.yaml' --exclude 'config.local.yaml' \
    --exclude '*.pem' --exclude 'logs/' \
    --exclude 'bin/' --exclude 'app/public/' --exclude 'app/dist/' \
    "$from/" "$to/"
  log "rsync $name"
}

log "CONFIG_REPO=$CONFIG_REPO DEPLOY_ROOT=$DEPLOY_ROOT env=$ENV_NAME"

if [[ "$same_root" -eq 1 ]]; then
  if [[ -d "$DEPLOY_ROOT/conf" && ! -L "$DEPLOY_ROOT/conf" ]]; then
    log "keep existing real conf dir"
  else
    ln -sfn "envs/${ENV_NAME}/conf" "$DEPLOY_ROOT/conf"
    log "symlink conf -> envs/${ENV_NAME}/conf"
  fi
  if [[ ! -e "$DEPLOY_ROOT/releases.yaml" ]]; then
    ln -sfn "envs/${ENV_NAME}/releases.yaml" "$DEPLOY_ROOT/releases.yaml"
    log "symlink releases.yaml"
  fi
else
  mkdir -p "$DEPLOY_ROOT/conf"
  rsync -a --delete \
    --exclude 'config.local.yaml' --exclude '*.local.yaml' \
    "$SRC/conf/" "$DEPLOY_ROOT/conf/"
  log "rsync conf"
  if [[ ! -e "$DEPLOY_ROOT/releases.yaml" && -f "$SRC/releases.yaml" ]]; then
    cp -a "$SRC/releases.yaml" "$DEPLOY_ROOT/releases.yaml"
    log "copy releases.yaml (dest was missing)"
  fi
fi

for name in "${TREES[@]}"; do
  link_or_rsync "$name"
done

# OPT-20260830-025：禁止把 P4 mysql datadir 链到 ram-work 或已判定损坏的空壳。
unlink_bad_mysql_data() {
  local data="$DEPLOY_ROOT/dockerInfra/mysql/data"
  local detect="$DEPLOY_ROOT/dockerInfra/mysql/detect_corrupt_data.sh"
  if [[ ! -e "$data" && ! -L "$data" ]]; then
    return 0
  fi
  local target=""
  if [[ -L "$data" ]]; then
    target="$(readlink -f "$data" 2>/dev/null || readlink "$data" || true)"
    if [[ "$target" == /tmp/ram-work/* ]]; then
      log "remove mysql data symlink into ram-work: $data -> $target"
      rm -f "$data"
      return 0
    fi
  fi
  if [[ -x "$detect" ]] && "$detect" "$data"; then
    if [[ -L "$data" ]]; then
      log "remove corrupt mysql data symlink: $data"
      rm -f "$data"
    else
      log "corrupt mysql datadir at $data (not a symlink); start will refuse unless MYSQL_ALLOW_EMPTY_REINIT=1"
    fi
  fi
}
unlink_bad_mysql_data

mkdir -p "$DEPLOY_ROOT/taskGateway/logs" "$DEPLOY_ROOT/logs" "$DEPLOY_ROOT/bin" "$DEPLOY_ROOT/artifacts"
if ! chmod 777 "$DEPLOY_ROOT/taskGateway/logs" 2>/dev/null; then
  docker run --rm --user 0 -v "$DEPLOY_ROOT/taskGateway/logs":/logs alpine sh -c 'chmod 777 /logs'
fi
mkdir -p "$DEPLOY_ROOT/taskFE/app/public" "$DEPLOY_ROOT/taskEvents/bin"

# Separate deploy root: rewrite the copied (non-git) runAll.yaml.
# Clone-as-root: conf is a symlink into git-tracked YAML — do not sed it.
if [[ "$same_root" -eq 0 && -f "$DEPLOY_ROOT/conf/runAll.yaml" ]]; then
  sed -i "s|file_root: /tmp/ram-work/logs|file_root: ${DEPLOY_ROOT}/logs|" "$DEPLOY_ROOT/conf/runAll.yaml" || true
  sed -i "s|--config \\.\\./conf/value-stream.yaml|--config ${DEPLOY_ROOT}/conf/value-stream.yaml|" "$DEPLOY_ROOT/conf/runAll.yaml" || true
fi

# Always overlay logging.file_root via gitignored conf-local (runAll MergeConfLocal).
# Preserves other keys in an existing hand-copied conf-local/runAll.yaml.
rewrite_conf_local_file_root() {
  local overlay="$DEPLOY_ROOT/conf-local/runAll.yaml"
  local logs="${DEPLOY_ROOT}/logs"
  mkdir -p "$DEPLOY_ROOT/conf-local"
  python3 - "$overlay" "$logs" <<'PY'
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    sys.exit("layout-from-config-repo: PyYAML required to merge conf-local/runAll.yaml")

path = Path(sys.argv[1])
file_root = sys.argv[2]
data = {}
if path.exists():
    loaded = yaml.safe_load(path.read_text(encoding="utf-8"))
    if isinstance(loaded, dict):
        data = loaded
logging = data.get("logging")
if not isinstance(logging, dict):
    logging = {}
logging["file_root"] = file_root
data["logging"] = logging
path.write_text(yaml.safe_dump(data, sort_keys=False, allow_unicode=True), encoding="utf-8")
PY
  log "conf-local/runAll.yaml logging.file_root=$logs"
}
rewrite_conf_local_file_root

log "done"
