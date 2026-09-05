#!/usr/bin/env bash
# Bring up a deploy root from an daydaymoney-deploy checkout (ADR-0052).
# Default: layout + cutover.env + optional artifact sync. Does not start the stack
# Host secrets: clone-root conf-local/ or SECRETS_DIR/conf-local (YAML + PEMs).
# Do not copy *.local.yaml into conf/. Do not rsync secrets/taskGateway or secrets/db.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [[ -d "$SCRIPT_DIR/../envs" ]]; then
  CONFIG_REPO="${CONFIG_REPO:-$(cd "$SCRIPT_DIR/.." && pwd)}"
else
  echo "up-from-config-repo: run from daydaymoney-deploy/scripts or set CONFIG_REPO" >&2
  exit 1
fi
DEPLOY_ROOT="${DEPLOY_ROOT:-$CONFIG_REPO}"
LAYOUT="${SCRIPT_DIR}/layout.sh"
if [[ ! -x "$LAYOUT" ]]; then
  LAYOUT="${SCRIPT_DIR}/layout-from-config-repo.sh"
fi
if [[ ! -x "$LAYOUT" ]]; then
  echo "up-from-config-repo: missing layout.sh" >&2
  exit 1
fi

log() { echo "[up-from-config-repo] $*"; }

overlay_conf_local() {
  local src="$1"
  if [[ ! -d "$src" ]]; then
    return 0
  fi
  mkdir -p "$DEPLOY_ROOT/conf-local"
  rsync -a --exclude 'README.md' --exclude '.gitkeep' "$src/" "$DEPLOY_ROOT/conf-local/"
  log "conf-local overlay from $src"
}

overlay_manual_secrets() {
  local src="$1"
  if [[ ! -d "$src" ]]; then
    return 0
  fi
  overlay_conf_local "$src/conf-local"
}

log "CONFIG_REPO=$CONFIG_REPO DEPLOY_ROOT=$DEPLOY_ROOT"
export CONFIG_REPO DEPLOY_ROOT
bash "$LAYOUT"

DEPLOY_ROOT="$(cd "$DEPLOY_ROOT" && pwd)"
mkdir -p "$DEPLOY_ROOT/conf"

overlay_conf_local "$CONFIG_REPO/conf-local"

if [[ -n "${SECRETS_DIR:-}" ]]; then
  if [[ ! -d "$SECRETS_DIR" ]]; then
    echo "up-from-config-repo: SECRETS_DIR is not a directory: $SECRETS_DIR" >&2
    exit 1
  fi
  overlay_manual_secrets "$SECRETS_DIR"
elif [[ -d "$CONFIG_REPO/secrets" ]]; then
  overlay_manual_secrets "$CONFIG_REPO/secrets"
else
  log "no secrets/ dir (place keys manually; see secrets.example/README.md)"
fi

INSTALL="${SCRIPT_DIR}/install-local-artifacts.sh"
LOCAL_ARTIFACTS=""
SKIP_GITHUB_ARTIFACTS=0
if [[ -n "${ARTIFACTS_DIR:-}" ]]; then
  if [[ ! -d "$ARTIFACTS_DIR" ]]; then
    echo "up-from-config-repo: ARTIFACTS_DIR is not a directory: $ARTIFACTS_DIR" >&2
    exit 1
  fi
  LOCAL_ARTIFACTS="$ARTIFACTS_DIR"
elif [[ -f "$DEPLOY_ROOT/artifacts/runAll" ]]; then
  LOCAL_ARTIFACTS="$DEPLOY_ROOT/artifacts"
elif [[ -f "$DEPLOY_ROOT/deploy-binaries/runAll" ]]; then
  LOCAL_ARTIFACTS="$DEPLOY_ROOT/deploy-binaries"
fi
if [[ -n "$LOCAL_ARTIFACTS" ]]; then
  if [[ ! -f "$INSTALL" ]]; then
    echo "up-from-config-repo: missing $INSTALL" >&2
    exit 1
  fi
  bash "$INSTALL" "$LOCAL_ARTIFACTS" "$DEPLOY_ROOT"
  if [[ -f "$LOCAL_ARTIFACTS/runAll" ]]; then
    SKIP_GITHUB_ARTIFACTS=1
    log "local artifacts from $LOCAL_ARTIFACTS (skip GitHub unless FORCE_DEPLOY_SYNC=1)"
  fi
fi

bootstrap_runall_from_release() {
  if [[ -x "$DEPLOY_ROOT/bin/runAll" ]]; then
    return 0
  fi
  local pins="$DEPLOY_ROOT/releases.yaml"
  if [[ ! -f "$pins" ]]; then
    log "no bin/runAll and no releases.yaml (set ARTIFACTS_DIR)"
    return 0
  fi
  local gh_bin="${GH:-gh}"
  if [[ ! -x "$gh_bin" ]] && ! command -v "$gh_bin" >/dev/null 2>&1; then
    log "no bin/runAll; install gh or set ARTIFACTS_DIR to bootstrap"
    return 0
  fi
  local parsed
  parsed="$(python3 - "$pins" <<'PY'
import shlex
import sys

import yaml

rel = yaml.safe_load(open(sys.argv[1], encoding="utf-8")) or {}
arts = rel.get("artifacts") or {}
pin = arts.get("runAll") or {}
pkg = str(pin.get("package") or "")
if not pkg.startswith("github://") or "@" not in pkg:
    sys.exit(1)
rest, tag = pkg[len("github://") :].rsplit("@", 1)
parts = rest.split("/")
if len(parts) < 3:
    sys.exit(1)
print("BOOT_REPO=" + shlex.quote(parts[0] + "/" + parts[1]))
print("BOOT_TAG=" + shlex.quote(tag))
PY
)" || {
    log "could not parse runAll github:// pin from releases.yaml"
    return 0
  }
  # shellcheck disable=SC2086
  eval "$parsed"
  mkdir -p "$DEPLOY_ROOT/bin"
  if download_github_runall "$gh_bin" "$BOOT_TAG" "$BOOT_REPO" "$DEPLOY_ROOT/bin"; then
    chmod +x "$DEPLOY_ROOT/bin/runAll" || true
    log "bootstrapped bin/runAll from GitHub Release $BOOT_TAG"
  else
    log "GitHub download of runAll failed (HTTP/2 PROTOCOL_ERROR: retried HTTP/1.1; set ARTIFACTS_DIR or GITHUB_TOKEN)"
  fi
}

github_http1_godebug() {
  local existing="${GODEBUG:-}"
  if [[ "$existing" == *http2client=0* ]]; then
    printf '%s' "$existing"
    return 0
  fi
  if [[ -n "$existing" ]]; then
    printf '%s,http2client=0' "$existing"
  else
    printf 'http2client=0'
  fi
}

# gh's Go HTTP/2 client hits GitHub PROTOCOL_ERROR; force HTTP/1.1, retry, then curl.
download_github_runall() {
  local gh_bin="$1" tag="$2" repo="$3" dest="$4"
  local retries="${GH_DOWNLOAD_RETRIES:-3}"
  local sleep_s="${GH_DOWNLOAD_SLEEP:-1}"
  local n godebug
  godebug="$(github_http1_godebug)"
  n=1
  while [[ "$n" -le "$retries" ]]; do
    if GODEBUG="$godebug" "$gh_bin" release download "$tag" --repo "$repo" --pattern runAll --dir "$dest"; then
      if [[ -f "$dest/runAll" ]]; then
        return 0
      fi
    fi
    log "gh release download attempt $n/$retries failed (tag=$tag)"
    if [[ "$n" -lt "$retries" && "$sleep_s" != "0" ]]; then
      sleep "$sleep_s"
    fi
    n=$((n + 1))
  done
  download_github_runall_curl "$gh_bin" "$tag" "$repo" "$dest"
}

download_github_runall_curl() {
  local gh_bin="$1" tag="$2" repo="$3" dest="$4"
  local curl_bin="${CURL:-curl}"
  if [[ ! -x "$curl_bin" ]] && ! command -v "$curl_bin" >/dev/null 2>&1; then
    log "curl not available for HTTP/1.1 GitHub fallback"
    return 1
  fi
  local token="${GITHUB_TOKEN:-${GH_TOKEN:-}}"
  if [[ -z "$token" ]]; then
    token="$(GODEBUG="$(github_http1_godebug)" "$gh_bin" auth token 2>/dev/null || true)"
  fi
  local api="${GITHUB_API_URL:-https://api.github.com}"
  local auth_h=()
  if [[ -n "$token" ]]; then
    auth_h=(-H "Authorization: Bearer ${token}")
  fi
  local meta asset_url
  meta="$("$curl_bin" --http1.1 -fsSL \
    "${auth_h[@]}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "${api}/repos/${repo}/releases/tags/${tag}")" || {
    log "curl GitHub release metadata failed tag=$tag"
    return 1
  }
  asset_url="$(RUNALL_ASSET_JSON="$meta" python3 - <<'PY'
import json
import os

rel = json.loads(os.environ["RUNALL_ASSET_JSON"])
for a in rel.get("assets") or []:
    if a.get("name") == "runAll":
        print(a.get("url") or "")
        break
PY
)"
  if [[ -z "$asset_url" ]]; then
    log "release $tag has no runAll asset"
    return 1
  fi
  "$curl_bin" --http1.1 -fsSL \
    "${auth_h[@]}" \
    -H "Accept: application/octet-stream" \
    -o "$dest/runAll" \
    "$asset_url" || {
    log "curl GitHub asset download failed tag=$tag"
    return 1
  }
  [[ -f "$dest/runAll" ]]
}

if [[ "$SKIP_GITHUB_ARTIFACTS" != "1" ]]; then
  bootstrap_runall_from_release
fi

if [[ "${SYNC_ARTIFACTS:-1}" == "1" && ! -x "$DEPLOY_ROOT/bin/runAll" ]]; then
  echo "up-from-config-repo: missing bin/runAll (rsync deploy-binaries/ to ./artifacts/ or set ARTIFACTS_DIR)" >&2
  exit 1
fi

PINS="$DEPLOY_ROOT/releases.local.yaml"
if [[ ! -f "$PINS" ]]; then
  PINS="$DEPLOY_ROOT/releases.yaml"
fi
if [[ -x "$DEPLOY_ROOT/bin/runAll" && -f "$PINS" && "${SYNC_ARTIFACTS:-1}" == "1" ]]; then
  if [[ "${FORCE_DEPLOY_SYNC:-0}" == "1" || "$SKIP_GITHUB_ARTIFACTS" != "1" ]]; then
    log "deploy-sync pins=$PINS"
    CONF_ROOT="$DEPLOY_ROOT/conf" DEPLOY_MODE=1 GODEBUG="$(github_http1_godebug)" \
      "$DEPLOY_ROOT/bin/runAll" -command deploy-sync -config "$DEPLOY_ROOT/conf/runAll.yaml" \
      || log "deploy-sync skipped or failed (last-good kept)"
  else
    log "skip GitHub deploy-sync (local artifacts present; FORCE_DEPLOY_SYNC=1 to override)"
  fi
fi

if [[ -n "${TASK_EVENTS_BIN:-}" && -d "$TASK_EVENTS_BIN" ]]; then
  mkdir -p "$DEPLOY_ROOT/taskEvents/bin"
  rsync -a "$TASK_EVENTS_BIN/" "$DEPLOY_ROOT/taskEvents/bin/"
  log "taskEvents/bin from TASK_EVENTS_BIN (overrides Release unpack)"
fi
if [[ -n "${TASKFE_PUBLIC:-}" && -d "$TASKFE_PUBLIC" ]]; then
  mkdir -p "$DEPLOY_ROOT/taskFE/app/public"
  rsync -a "$TASKFE_PUBLIC/" "$DEPLOY_ROOT/taskFE/app/public/"
  log "taskFE public from TASKFE_PUBLIC (overrides Release unpack)"
fi
if [[ -n "${TASKAIPROVIDER_FRONTEND:-}" && -d "$TASKAIPROVIDER_FRONTEND" ]]; then
  mkdir -p "$DEPLOY_ROOT/taskAiProvider/frontend/dist"
  rsync -a "$TASKAIPROVIDER_FRONTEND/" "$DEPLOY_ROOT/taskAiProvider/frontend/dist/"
  log "taskAiProvider frontend from TASKAIPROVIDER_FRONTEND (overrides Release unpack)"
fi

INFRA_HOST_DEFAULT="10.2.150.68"
if [[ -z "${INFRA_HOST:-}" && -f "$DEPLOY_ROOT/conf-local/infra-host.env" ]]; then
  # shellcheck disable=SC1090
  INFRA_HOST="$(sed -n 's/^[[:space:]]*INFRA_HOST=//p' "$DEPLOY_ROOT/conf-local/infra-host.env" | tail -n 1 | tr -d '\"' | tr -d "'")"
fi
INFRA_HOST="${INFRA_HOST:-$INFRA_HOST_DEFAULT}"

# OPT-20260901-004: cutover.env key set is SSOT in write-cutover-env.sh; no
# second heredoc here so the key set cannot drift from prepare-ram-deploy.sh.
bash "$SCRIPT_DIR/write-cutover-env.sh" "$DEPLOY_ROOT" "$INFRA_HOST"

if [[ "${START:-0}" == "1" ]]; then
  # shellcheck disable=SC1091
  source "$DEPLOY_ROOT/cutover.env"
  if [[ ! -x "${RUNALL_BIN}" ]]; then
    echo "up-from-config-repo: START=1 requires $DEPLOY_ROOT/bin/runAll" >&2
    exit 1
  fi
  if [[ ! -f "$DEPLOY_ROOT/runAll/run.sh" ]]; then
    echo "up-from-config-repo: START=1 requires envs/current/runAll/run.sh (re-export deploy payload)" >&2
    exit 1
  fi
  # OPT-20260901-002: launch via runAll/run.sh (setsid -f nohup) so the
  # orchestrator lives in its own session; a foreground exec would be SIGTERMed
  # when up.sh exits, taking :9999 down while managed processes keep running.
  log "starting runAll (START=1, detached via run.sh)"
  cd "$DEPLOY_ROOT"
  ./runAll/run.sh
  exit 0
fi

log "layout complete. Next: start infra, then ./runAll/run.sh (sources $DEPLOY_ROOT/cutover.env)."
