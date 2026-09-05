#!/usr/bin/env bash
# Refresh ELF artifacts on a P4 tree. Recipes must already come from daydaymoney-deploy
# (prepare-ram-deploy.sh / layout-from-config-repo.sh). Never bind MySQL data to
# the source ram-work tree.
set -euo pipefail

META="${META_ROOT:-/tmp/ram-work}"
DEPLOY_ROOT="${DEPLOY_ROOT:-$HOME/bin/daydaymoney-deploy}"
ARTIFACTS="$DEPLOY_ROOT/artifacts"
CONFIG_REPO="${CONFIG_REPO:-$DEPLOY_ROOT/.config-repo}"

log() { echo "[materialize-p4] $*"; }

copy_elf() {
  local name="$1" src="$2"
  if [[ ! -x "$src" ]]; then
    log "skip missing elf $name ($src)"
    return 0
  fi
  cp -a "$src" "$ARTIFACTS/$name"
  chmod +x "$ARTIFACTS/$name"
  log "artifact $name"
}

log "DEPLOY_ROOT=$DEPLOY_ROOT"
mkdir -p "$ARTIFACTS" "$DEPLOY_ROOT/bin" "$DEPLOY_ROOT/logs"

LAYOUT="$CONFIG_REPO/scripts/layout.sh"
if [[ ! -x "$LAYOUT" ]]; then
  LAYOUT="${META}/runAll/scripts/layout-from-config-repo.sh"
fi
if [[ -x "$LAYOUT" && -d "$CONFIG_REPO/envs/current/conf" ]]; then
  log "layout recipes from config repo"
  CONFIG_REPO="$CONFIG_REPO" DEPLOY_ROOT="$DEPLOY_ROOT" bash "$LAYOUT"
fi

if [[ "${COPY_ELFS_FROM_META:-0}" != "1" ]]; then
  log "skip ELF copy (set COPY_ELFS_FROM_META=1 to bootstrap file:// pins from a source tree)"
  log "done"
  exit 0
fi

copy_elf runAll "$META/runAll/bin/runAll"
copy_elf taskAuth "$META/taskAuth/bin/taskAuth"
copy_elf taskBill "$META/taskBill/bin/taskBill"
copy_elf taskReferral "$META/taskReferral/bin/taskReferral"
copy_elf taskGitOauth "$META/taskGitOauth/bin/taskGitOauth"
copy_elf taskAiProvider "$META/taskAiProvider/bin/taskAiProvider"
copy_elf taskAgentSupport "$META/taskAgentSupport/bin/taskAgentSupport"
copy_elf taskAIEndPoint "$META/taskAIEndPoint/bin/taskAIEndPoint"
copy_elf taskContainerGateway "$META/taskContainerGateway/bin/taskContainerGateway"
copy_elf taskProjectService "$META/taskProjectService/bin/taskProjectService"
copy_elf taskTenantService "$META/taskTenantService/bin/taskTenantService"
copy_elf taskTaskService "$META/taskTaskService/bin/taskTaskService"
copy_elf taskCloudService "$META/taskCloudService/bin/taskCloudService"
copy_elf taskCredentialService "$META/taskCredentialService/bin/taskCredentialService"
copy_elf taskAIComment "$META/taskAIComment/bin/taskAIComment"
copy_elf go_relayToTrae "$META/go_relayToTrae/bin/go_relayToTrae"
copy_elf valueStream "$META/valueStream/bin/valueStream"

log "write releases.yaml (file:// pins)"
python3 - "$ARTIFACTS" "$DEPLOY_ROOT/releases.yaml" <<'PY'
import hashlib, os, sys, yaml
art, out = sys.argv[1], sys.argv[2]
artifacts = {}
for name in sorted(os.listdir(art)):
    path = os.path.join(art, name)
    if not os.path.isfile(path) or name.endswith(".sha"):
        continue
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    artifacts[name] = {
        "sha": h.hexdigest(),
        "package": "file://" + os.path.abspath(path),
    }
if not artifacts:
    raise SystemExit("no artifacts")
with open(out, "w", encoding="utf-8") as f:
    f.write("# P4 host pins. Prefer github:// after Release publish.\n")
    yaml.safe_dump({"artifacts": artifacts}, f, sort_keys=True)
print("pins", len(artifacts), "->", out)
PY

if [[ -x "$ARTIFACTS/runAll" ]]; then
  mkdir -p "$DEPLOY_ROOT/runAll/bin"
  tmp="$DEPLOY_ROOT/runAll/bin/runAll.install.$$"
  cp -a "$ARTIFACTS/runAll" "$tmp"
  chmod +x "$tmp"
  mv -f "$tmp" "$DEPLOY_ROOT/runAll/bin/runAll"
  tmp="$DEPLOY_ROOT/bin/runAll.install.$$"
  cp -a "$ARTIFACTS/runAll" "$tmp"
  chmod +x "$tmp"
  mv -f "$tmp" "$DEPLOY_ROOT/bin/runAll"
fi

log "done"
