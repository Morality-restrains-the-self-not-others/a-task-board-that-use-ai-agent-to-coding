# Value Stream: conf-read/sync 脚本合并进 runAll

> Derived from design: `docs/design/merge-conf-scripts-into-runall.md`
> Date: 2026-06-05

## Value Summary

开发者通过 runAll 统一管理配置的读取与同步——配置脚本成为编排器的内置能力，消除 scripts/ 对 task2app 的反向依赖，代码重复消除。

## Related Value Streams

- **[2026-06-02-port-config-split-monorepo-conf-value-stream](2026-06-02-port-config-split-monorepo-conf-value-stream.md)**: 前置 — 已完成 port_config.json → conf/<app>/ YAML 迁移。本次变更将该迁移产生的配置工具脚本收归 runAll。
- **[2026-06-05-conf-directory-restructure-value-stream](2026-06-05-conf-directory-restructure-value-stream.md)**: 同步进行 — conf/ 二级目录重组。本次变更需对齐 `sync.sh` 和 `conf-read.py` 的路径解析逻辑。
- **[2026-06-02-docker-infra-conf-sync-value-stream](2026-06-02-docker-infra-conf-sync-value-stream.md)**: 前置 — conf sync 碎片机制已成熟。本次变更不改变 sync 行为，仅改变脚本物理位置。
- **关系类型**: extension — 在前置 conf 迁移已完成的基础上，将配置工具脚本从 monorepo 级 scripts/ 收归到 runAll 服务目录内。

## End-to-End Flow

[开发者执行 sync] → [runAll conf_sync.go 调用 runAll/scripts/conf-sync-all.sh] → [conf-sync.py 读取 sync.manifest.yaml] → [pick 字段生成 GENERATED YAML] → [taskSSE 启动时调用 runAll/scripts/conf-read.py snapshot-json] → [读取全量配置] → [服务正常启动]

## Value Increments

### Increment 1: conf_loader 提取 + 脚本迁移 (Thin Slice)

**Value to user:** 配置脚本归属 runAll，runAll/src/conf_sync.go 可直调本地脚本。

**Scope:**
- 创建 `runAll/scripts/conf_loader.py`（自包含 YAML 加载器，从 task2app/conf_loader 提取，无外部依赖）
- 移动 5 个脚本到 `runAll/scripts/`：conf-read.py, conf-sync.py, conf_lib.py, conf-sync-all.sh, migrate_port_config_to_conf.py
- 更新 `conf-read.py` 的 import：从 `config.conf_loader` → 从本地 `conf_loader`
- 更新 `conf-sync-all.sh` 内部对 `conf-sync.py` 的路径引用
- 更新 `runAll/src/conf_sync.go` 中两处 `scripts/conf-sync-all.sh` → `runAll/scripts/conf-sync-all.sh`

**Depends on:** port-config-split-monorepo-conf

**验证方式:**
- `python3 runAll/scripts/conf-read.py snapshot-json` 输出与迁移前一致
- `bash runAll/scripts/conf-sync-all.sh` exit 0
- `cd runAll && go test ./... -run TestSyncMonorepoConf` 通过

### Increment 2: 外部调用方路径更新

**Value to user:** 所有 8 个 sync.sh、taskSSE、CI 脚本正确引用新路径。

**Scope:**
- 更新 8 个 `conf/*/*/sync.sh` 的脚本路径：`$ROOT/scripts/conf-sync.py` → `$ROOT/runAll/scripts/conf-sync.py`
- 更新 `taskSSE/src/config.mjs` 两处 `scripts/conf-read.py` → `runAll/scripts/conf-read.py`
- 更新 `scripts/ci/check_conf_sync.sh` 的 `./scripts/conf-sync-all.sh` → `./runAll/scripts/conf-sync-all.sh`
- 更新 `conf/README.md` 文档路径引用

**Depends on:** Increment 1

**验证方式:**
- 逐一执行 8 个 sync.sh，确认均 exit 0
- `bash scripts/ci/check_conf_sync.sh` 通过
- 手工验证 `node taskSSE/src/index.mjs` 启动正常

### Increment 3: 旧文件清理 + 全仓验证

**Value to user:** 无冗余文件，单一配置管理入口。

**Scope:**
- 删除 `scripts/conf-read.py`, `scripts/conf-sync.py`, `scripts/conf_lib.py`, `scripts/conf-sync-all.sh`, `scripts/migrate_port_config_to_conf.py`
- 删除 `scripts/__pycache__/`
- 全仓 grep 确认无残留 `scripts/conf-read` / `scripts/conf-sync` 引用（排除 docs/ 历史文档）

**Depends on:** Increment 2

**验证方式:**
- `grep -rn 'scripts/conf-read\|scripts/conf-sync\|scripts/conf_lib' --include='*.py' --include='*.sh' --include='*.mjs' --include='*.go' | grep -v 'docs/' | grep -v '.git/' | grep -v '__pycache__'` 无输出
- 运行 `cd valueStream && go test ./...` 确认无 regression

## value-stream.yaml 变更

本次变更不新增独立 value stream entry。其为已有 `docker-infra-conf-sync` 和 `port-config-split-monorepo-conf` 流的维护性延续——脚本物理位置变更，sync/read 行为与输出不变。
