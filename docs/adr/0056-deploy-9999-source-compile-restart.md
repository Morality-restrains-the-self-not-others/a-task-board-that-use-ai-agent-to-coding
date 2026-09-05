# ADR-0056: 部署机 9999 编排源码编译、conf-local rsync 与增量安装

- **Status:** accepted
- **Date:** 2026-09-02
- **Author:** cursor
- **Deciders:** 头脑风暴 `/1-brainstorming-design-docs` 用户批准（approve_sync_conflocal，2026-09-02）
- **Supersedes (partial):** ADR-0052「Precise compile-restart remains a source worktree developer tool」——按钮改在部署 9999；**编译仍只发生在 SOURCE_ROOT**

---

## Context

ADR-0052 分离源码与部署后，本机对外 9999 是 `DEPLOY_MODE=1` 的部署根 runAll。`resolveBuildCommand` 被清空，「精准编译重启」「全部重新编译」不能在部署树 `go build`。日常用 `update.sh` 整树覆盖 `artifacts/` 与 `conf-local/` 并重启 runAll，效率低。

源码侧已有 `precise-compile.sh`（只编译进 `deploy-binaries/`）。需要把现有 9999 按钮接到这条链上，且编译成功后同步机密 overlay。

## Decision

We will keep compile **off the deploy tree**. Deploy-side runAll (`http://192.168.1.10:9999/`) orchestrates:

1. Read registry from `$SOURCE_ROOT/.runall/` (same file Agents already write).
2. Run `$SOURCE_ROOT/scripts/precise-compile.sh` with `DEPLOY_MODE` unset in the child.
3. On compile success: `rsync -a --delete` `$SOURCE_ROOT/conf-local/` → `$DEPLOY_ROOT/conf-local/`; incremental install of changed artifacts (copy-then-mv).
4. Precise restart: `restartService` without `build_command`. Build-all: install only, no process kill (ADR-0027).
5. Never `git pull` the config repo, never `up.sh`, never restart the runAll orchestrator as part of these two buttons.

`SOURCE_ROOT` comes from `cutover.env`. Missing `SOURCE_ROOT` fails the buttons (no silent skip). Deploy hosts still must not `go build`.

## Alternatives Considered

### Alternative 1: Restore `build_command` on the deploy tree

- **Rejected:** Reintroduces source on the deploy host (ADR-0052).

### Alternative 2: Keep `update.sh` as the only path; rsync inside the script

- **Rejected:** 9999 buttons stay dead; still restarts runAll.

### Alternative 3: Compile buttons do not touch conf-local

- **Rejected:** User chose `approve_sync_conflocal`.

## Consequences

### Positive

- Daily code changes: one 9999 click; only registered services rebuild and restart.
- conf-local stays aligned with the source tree without wiping via `rm -rf` + full `cp`.
- ADR-0027 last-good and compile-then-swap remain.

### Negative / Trade-offs

- Same-host `SOURCE_ROOT` only; remote compile is a later iteration.
- Build-all rsyncs conf-local but running processes keep old YAML until restart.
- `rsync --delete` removes dest-only files under conf-local (same as old `update.sh`).

### Mitigations

- Fail-fast if `SOURCE_ROOT` is unset.
- Compile failure skips rsync and install.
- `update.sh` remains for bootstrap / config-repo pull / runAll relaunch.
- Stale `cutover.env` written before this ADR may omit `SOURCE_ROOT`. Then GET `/api/precise-restart/registrations` falls through to a relative `.runall/` under the orchestrator cwd (`$DEPLOY_ROOT/runAll/`) and the UI shows「没有已登记的服务」even when `$SOURCE_ROOT/.runall/precise_restart_services.txt` has names. Regenerate with `write-cutover-env.sh`; `preciseRestartFile` also parses `SOURCE_ROOT` from the cutover file so a rewritten env does not depend on the process having been started with that key.
