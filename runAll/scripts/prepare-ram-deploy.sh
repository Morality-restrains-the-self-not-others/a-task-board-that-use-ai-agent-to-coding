#!/usr/bin/env bash
# Layout $DEPLOY_ROOT from github.com/task2money/daydaymoney-deploy (ADR-0052).
# Recipes come from the config repo, not the source monorepo. Host secrets stay
# on the machine (SECRETS_DIR or SECRETS_FROM_META=1).
set -euo pipefail

META_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# Live clone-run root on this host (ADR-0052). Default $HOME/bin/daydaymoney-deploy.
DEPLOY_ROOT="${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}"
CONFIG_REPO_URL="${CONFIG_REPO_URL:-git@github.com:task2money/daydaymoney-deploy.git}"
SEED="${META_ROOT}/.daydaymoney-deploy-seed"

log() { echo "[prepare-ram-deploy] $*"; }

clone_config_repo() {
  local dest="$1"
  if [[ -d "$dest/.git" ]]; then
    git -C "$dest" fetch --depth 1 origin
    git -C "$dest" reset --hard origin/HEAD
    return
  fi
  if git clone --depth 1 "$CONFIG_REPO_URL" "$dest"; then
    return
  fi
  if [[ -d "$SEED/.git" ]]; then
    log "clone failed; copying seed $SEED"
    mkdir -p "$dest"
    rsync -a --delete --exclude '.git/' "$SEED/" "$dest/"
    return
  fi
  echo "prepare-ram-deploy: cannot clone $CONFIG_REPO_URL and seed missing" >&2
  return 1
}

log "DEPLOY_ROOT=$DEPLOY_ROOT"
mkdir -p "$DEPLOY_ROOT"
clone_config_repo "$DEPLOY_ROOT/.config-repo"

CONFIG_REPO="$DEPLOY_ROOT/.config-repo"
if [[ ! -d "$CONFIG_REPO/envs/current/conf" ]]; then
  echo "prepare-ram-deploy: missing $CONFIG_REPO/envs/current/conf" >&2
  exit 1
fi

LAYOUT="$CONFIG_REPO/scripts/layout.sh"
if [[ ! -x "$LAYOUT" ]]; then
  LAYOUT="${META_ROOT}/runAll/scripts/layout-from-config-repo.sh"
fi
if [[ ! -x "$LAYOUT" ]]; then
  echo "prepare-ram-deploy: missing layout.sh in config repo and source tree" >&2
  exit 1
fi

log "layout recipes from daydaymoney-deploy"
CONFIG_REPO="$CONFIG_REPO" DEPLOY_ROOT="$DEPLOY_ROOT" DEPLOY_ENV="${DEPLOY_ENV:-current}" \
  bash "$LAYOUT"

if [[ "${SECRETS_FROM_META:-0}" == "1" || -n "${SECRETS_DIR:-}" ]]; then
  secret_root="${SECRETS_DIR:-${META_ROOT}}"
  if [[ -d "${secret_root}/conf-local" ]]; then
    mkdir -p "$DEPLOY_ROOT/conf-local"
    rsync -a --exclude 'README.md' --exclude '.gitkeep' "${secret_root}/conf-local/" "$DEPLOY_ROOT/conf-local/"
    log "conf-local overlay from ${secret_root}/conf-local"
  fi
fi

mkdir -p "$DEPLOY_ROOT/logs" "$DEPLOY_ROOT/.runall"

# OPT-20260901-004: cutover.env key set is SSOT in write-cutover-env.sh; both
# generators must share it so consumers never source a missing key.
bash "$SCRIPT_DIR/write-cutover-env.sh" "$DEPLOY_ROOT"

log "done"
