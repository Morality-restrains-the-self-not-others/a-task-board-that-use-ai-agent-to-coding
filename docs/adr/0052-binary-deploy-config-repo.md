# ADR-0052: 二进制部署；运行时配置与源码仓隔离

- **Status:** accepted
- **Date:** 2026-08-30
- **Author:** cursor
- **Deciders:** 头脑风暴 `/1-brainstorming-design-docs` 用户批准（approve_full，2026-08-30）

---

## Context

运行时与源码同树：`conf/` 在 monorepo（元规则 42）、`confload.FindMonorepoRoot` 依赖 `conf/base.yaml` / `dataMigrate/` / `.gitmodules`、runAll `working_dir` 指向服务源码目录。部署机必须 clone 源码才能启动；运行时拓扑随源码仓扩散。ADR-0027 已要求重启只 exec last-good 二进制，但产物仍长在源码树里。

需要：部署机不出现源码树；运行时配置单独仓库；产物（Go ELF、taskFE dist、taskEvents、Docker 配方）与源码评审面隔离。

## Decision

We will split delivery into three planes:

1. **Source repo** (this monorepo): code, tests, `dataMigrate` SQL, `conf.example/` only.
2. **Config repo** `daydaymoney-deploy`: private on **platform GitHub** (e.g. `github.com/task2money/daydaymoney-deploy`), not tenant GitLab. Holds full runtime `conf/`, `runAll.yaml`, Docker/GitLab-CE **recipes** (not `gitlab-ce` source), `releases.yaml` pins. One repo, `envs/<env>/`.
3. **Artifacts**: GitHub Packages (coordinates in `releases.yaml`). Deploy host runs `deploy-sync` then exec. No binaries in Git history.

Deploy host layout is `$DEPLOY_ROOT` with `CONF_ROOT` (default `$DEPLOY_ROOT/conf`). `confload` uses `FindConfigRoot`: env first, then `conf/base.yaml` walk-up; `.gitmodules` is not required. Production secrets are **placed manually on the host** (`conf-local/` mirroring `conf/` relative paths, PEM/OIDC keys); they are never committed to daydaymoney-deploy.

Application processes still must not migrate on startup. Schema SQL ships as a versioned tarball matching the binary gitsha. `db/registry.yaml` is deploy payload on `$DEPLOY_ROOT/db/` (copy or symlink). If it contains DSN passwords it stays host-only and is **not** committed to daydaymoney-deploy.

ADR-0027 remains: restart never compiles. On a deploy host, upgrade is **download-then-swap**; precise compile-restart remains a **source worktree** developer tool.

## Alternatives Considered

### Alternative 1: Overlay-only config repo; source keeps conf skeleton

- **Pros:** Smaller change to meta-rule 42
- **Cons:** Dual YAML merge; weak isolation
- **Why rejected:** User chose full runtime conf leaving the source repo

### Alternative 2: Deploy host still clones source, only forbids compile

- **Pros:** Path changes are small
- **Cons:** Source and config not isolated
- **Why rejected:** User chose no source tree on the deploy host

### Alternative 3: Host config repo on tenant GitLab (gitService)

- **Pros:** Same forge as product Git
- **Cons:** Mixes platform delivery ACL with tenant Git hosting
- **Why rejected:** User required platform GitHub (`daydaymoney-deploy`)

## Consequences

### Positive

- Deploy machines need no Go toolchain or source checkout
- Runtime parameters change without source PRs; code changes without leaking prod topology
- Config ACL can be ops-only, separate from developers

### Negative / Trade-offs

- Two Git repositories and a package feed to keep in sync
- Meta-rule 42 SSOT moves; local dev needs `CONF_ROOT`
- Transitional dual-write of `conf/` until P3

### Mitigations

- Phased P0–P4 in the design spec
- `conf.example/` bootstrap for new environments
- Pins default to one gitsha per release batch

### P4 host layout (accepted 2026-08-30 on this machine; clone-run 2026-08-31)

The config repo **is** the runnable checkout: `git clone github.com/task2money/daydaymoney-deploy` then `./scripts/up.sh` (secrets + artifact dir or GitHub Release pins). `$DEPLOY_ROOT` may be the clone itself (`conf` → `envs/current/conf` symlinks) or a separate assembled tree. **This host (2026-09-02):** live `$DEPLOY_ROOT` is `$HOME/bin/daydaymoney-deploy`. `/tmp/ram-deploy` is retired and must not be used as a script default.

`$DEPLOY_ROOT` contains `conf/`, recipe trees (`dockerInfra`, `gitService` without `gitlab-ce/`, `dataMigrate`, `db`), `bin/` + `artifacts/` (deploy-sync), no service `*.go`, no `.gitmodules`. **Do not** symlink MySQL `./data` at the source ram-work tree (empty-shell restore wiped the live datadir on 2026-08-30). Host datadir stays on the deploy volume.

`DEPLOY_MODE=1` remaps `./bin/<elf>` working_dir to the deploy root.

GitHub Release **tag `deploy-20260831-conf-local`** holds 17 Go ELFs plus `taskEvents-bin.tar.gz` (~748MB) and `taskFE-dist.tar.gz` (~3MB). Archive pins use `dest` + `unpack: tar.gz`. Host `releases.local.yaml` may still use `file://`.

Clone-run default when GitHub download fails: collect payloads on the source machine (`scripts/collect-deploy-binaries.sh` → `deploy-binaries/`, gitignored) and rsync that tree to the new node's `./artifacts/`. `./scripts/up.sh` installs locally and skips GitHub unless `FORCE_DEPLOY_SYNC=1`. Binaries still must not enter Git history.

**Precise compile on the source machine (2026-09-02):** `scripts/precise-compile.sh` runs each registered (or named) service `build_command` in the source `working_dir`, then collects into `$META_ROOT/deploy-binaries` with `COLLECT_SKIP_SHA=1`. It does not restart processes (ADR-0027). Deploy hosts still must not `go build`.

**Deploy 9999 orchestration (2026-09-02, ADR-0056):** On a split host the operator UI is deploy-side `:9999`. Those two buttons orchestrate source compile + conf-local rsync + incremental install; they do not compile on the deploy tree. See ADR-0056.

## References

- 设计: [2026-08-30-binary-deploy-config-repo-design.md](../superpowers/specs/2026-08-30-binary-deploy-config-repo-design.md)
- 意图: [binary-deploy-config-repo.intent.md](../intents/platform/binary-deploy-config-repo.intent.md)
- [ADR-0027](0027-restart-compile-separation.md) 重启与编译分离（本决策在部署机上对应 download-then-swap）
- [ADR-0022](0022-taskfe-nginx-static-resident.md) taskFE nginx 静态 + symlink
- [ADR-0035](0035-runall-orchestrator-independence.md) runAll 退出不拆业务进程
- [ADR-0046](0046-no-hardcoded-secrets.md) 密钥不进源码；本决策补充密钥不进配置仓 Git
- [ADR-0054](0054-conf-local-secrets-only.md) 机密只放 conf-local（取代 ADR-0053）
- 元规则 42 / 29 / 40 / 47
