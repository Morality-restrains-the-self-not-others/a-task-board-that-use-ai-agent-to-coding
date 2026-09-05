#!/usr/bin/env bash
# Copy source-less recipes + bootstrap scripts into an daydaymoney-deploy checkout (ADR-0052).
# Does not copy Go source, gitlab-ce, mysql data, PEMs, or config.local.yaml.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
META_ROOT="${META_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
DEST="${1:-}"
ENV_NAME="${DEPLOY_ENV:-current}"

log() { echo "[export-deploy-payload] $*"; }

if [[ -z "$DEST" ]]; then
  echo "usage: export-deploy-payload.sh <daydaymoney-deploy-checkout>" >&2
  exit 1
fi
mkdir -p "$DEST"
DEST="$(cd "$DEST" && pwd)"
ENV_DIR="$DEST/envs/$ENV_NAME"
mkdir -p "$ENV_DIR"

rsync_tree() {
  local src="$1" dst="$2"
  shift 2
  if [[ ! -d "$src" ]]; then
    log "skip missing $src"
    return 0
  fi
  mkdir -p "$dst"
  rsync -a --delete \
    --exclude '.git/' --exclude '.githooks/' --exclude '.claude/' \
    --exclude '__pycache__/' --exclude '*.pyc' \
    "$@" "$src/" "$dst/"
  log "rsync $src -> $dst"
}

rsync_tree "$META_ROOT/dockerInfra" "$ENV_DIR/dockerInfra" \
  --exclude 'data/' --exclude 'data.corrupt.*' --exclude '*.go'

rsync_tree "$META_ROOT/gitService" "$ENV_DIR/gitService" \
  --exclude 'gitlab-ce/' --exclude 'gitlab-ce' \
  --exclude 'gitlab_home/' --exclude 'playwright/' --exclude 'domain/' \
  --exclude '*.go'

rsync_tree "$META_ROOT/AiMonitor" "$ENV_DIR/AiMonitor" \
  --exclude '*.go' --exclude 'tests/'

rsync_tree "$META_ROOT/taskGateway" "$ENV_DIR/taskGateway" \
  --exclude 'logs/' --exclude '*.go' --exclude '*.pem'

rsync_tree "$META_ROOT/taskSSE" "$ENV_DIR/taskSSE" \
  --exclude '*.go' --exclude 'test/' --exclude 'tests/'

mkdir -p "$ENV_DIR/taskEvents"
if [[ -f "$META_ROOT/taskEvents/run.sh" ]]; then
  cp -a "$META_ROOT/taskEvents/run.sh" "$ENV_DIR/taskEvents/run.sh"
  log "copy taskEvents/run.sh"
fi

rsync_tree "$META_ROOT/taskFE" "$ENV_DIR/taskFE" \
  --exclude 'node_modules/' --exclude 'app/src/' \
  --exclude 'app/tests/' --exclude 'app/node_modules/' --exclude 'tests/' \
  --exclude 'app/dist/' --exclude 'app/public/' --exclude '*.go' \
  --exclude 'full-suite.log' --exclude '*.png' --exclude 'preview.log'

rsync_tree "$META_ROOT/dataMigrate" "$ENV_DIR/dataMigrate" \
  --exclude '*.go' --exclude '*.pem'

rsync_tree "$META_ROOT/db" "$ENV_DIR/db" \
  --exclude '*.sqlite3' --exclude '*.db' --exclude '*.go' \
  --exclude 'hooks/' --exclude 'load/' --exclude '*.pem' --exclude 'scripts/ci/'

rsync_tree "$META_ROOT/runAll/scripts" "$ENV_DIR/runAll/scripts" \
  --exclude 'tests/'

# runAll/run.sh detaches the orchestrator in its own session (setsid -f nohup).
# Ship it so a clone-run brings up :9999 with `./runAll/run.sh`
# instead of a foreground exec whose SIGTERM orphans the stack (OPT-20260901-002).
if [[ -f "$META_ROOT/runAll/run.sh" ]]; then
  mkdir -p "$ENV_DIR/runAll"
  cp -a "$META_ROOT/runAll/run.sh" "$ENV_DIR/runAll/run.sh"
  chmod +x "$ENV_DIR/runAll/run.sh"
  log "copy runAll/run.sh"
fi

# go-relay 以 run.sh 为 monorepo 标记，overlay 还要 src/；镜像构建要 Dockerfile 与
# trae-agent 根 pyproject.toml / trae_agent（OPT-20260830-021）。
rsync_tree "$META_ROOT/trae-agent/onlineServiceJS" "$ENV_DIR/trae-agent/onlineServiceJS" \
  --exclude 'node_modules/' --exclude 'test-results/' --exclude 'playwright-report/' \
  --exclude 'e2e/' --exclude 'test/' --exclude '*.png' \
  --exclude 'docker/code-server/*.tar.gz'
for extra in pyproject.toml README.md; do
  if [[ -f "$META_ROOT/trae-agent/$extra" ]]; then
    mkdir -p "$ENV_DIR/trae-agent"
    cp -a "$META_ROOT/trae-agent/$extra" "$ENV_DIR/trae-agent/$extra"
  fi
done
for extra in trae_agent trae_agent_online; do
  if [[ -d "$META_ROOT/trae-agent/$extra" ]]; then
    rsync_tree "$META_ROOT/trae-agent/$extra" "$ENV_DIR/trae-agent/$extra" \
      --exclude '__pycache__/' --exclude '*.pyc'
  fi
done

mkdir -p "$DEST/scripts"
cp -a "$SCRIPT_DIR/layout-from-config-repo.sh" "$DEST/scripts/layout.sh"
cp -a "$SCRIPT_DIR/up-from-config-repo.sh" "$DEST/scripts/up.sh"
# up.sh calls this via $SCRIPT_DIR; omitting it makes update.sh rewrite cutover.env
# without SOURCE_ROOT (ADR-0056).
cp -a "$SCRIPT_DIR/write-cutover-env.sh" "$DEST/scripts/write-cutover-env.sh"
cp -a "$SCRIPT_DIR/install-local-artifacts.sh" "$DEST/scripts/install-local-artifacts.sh"
cp -a "$SCRIPT_DIR/check_p4_deploy_root.sh" "$DEST/scripts/check_p4_deploy_root.sh"
chmod +x "$DEST/scripts/layout.sh" "$DEST/scripts/up.sh" \
  "$DEST/scripts/write-cutover-env.sh" \
  "$DEST/scripts/install-local-artifacts.sh" "$DEST/scripts/check_p4_deploy_root.sh"

mkdir -p "$DEST/secrets.example"
if [[ -f "$SCRIPT_DIR/daydaymoney-secrets.example.md" ]]; then
  cp -a "$SCRIPT_DIR/daydaymoney-secrets.example.md" "$DEST/secrets.example/README.md"
fi
if [[ -f "$SCRIPT_DIR/daydaymoney-deploy.README.md" ]]; then
  cp -a "$SCRIPT_DIR/daydaymoney-deploy.README.md" "$DEST/README.md"
fi
rm -f "$DEST/secrets.example/HOST_SECRETS.md" "$DEST/HOST_SECRETS.md"

cat > "$DEST/.gitignore" <<'EOF'
/bin/
/artifacts/
/deploy-binaries/
/logs/
/.runall/
/secrets/
/conf-local/
**/*.local.yaml
**/config.local.yaml
releases.local.yaml
**/dockerInfra/**/data/
**/dockerInfra/**/data.corrupt.*
**/*.pem
**/*.key
taskGateway/logs/
envs/**/taskGateway/logs/
**/docker/code-server/*.tar.gz
/taskAiProvider/frontend/dist/
/taskFE/app/public/
EOF

log "done dest=$DEST env=$ENV_NAME"
