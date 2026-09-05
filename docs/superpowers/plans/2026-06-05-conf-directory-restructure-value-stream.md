# Value Stream: conf/ 目录二级职能分组

> Derived from design: `docs/superpowers/specs/2026-06-05-conf-directory-restructure-design.md`

## Value Summary

开发者通过 `conf/<职能>/<服务>/` 二级目录结构快速定位服务配置，减少 flat 21 个目录的认知负担，职能边界与 runAll 编排组和价值流领域对齐。

## Related Value Streams

- **[2026-06-02-port-config-split-monorepo-conf-value-stream](2026-06-02-port-config-split-monorepo-conf-value-stream.md)**: 前置 — 已完成 Phase 1-3，将 port_config.json 拆分为 conf/<app>/config.yaml。本次变更是该结构的进一步组织优化。
- **[2026-06-02-conf-hardening-phase45-value-stream](2026-06-02-conf-hardening-phase45-value-stream.md)**: 前置 — conf/ 配置加固（CI 检查、直读 YAML）。本次变更需更新其 CI glob 路径。
- **[2026-06-03-runall-conf-migration-value-stream](2026-06-03-runall-conf-migration-value-stream.md)**: 前置 — runAll 已迁移到读取 conf/。本次变更需更新所有 `conf_app` 路径。
- **关系类型**: extension — 在前置 conf 迁移已完成的基础上，增加二级目录分组层。

## End-to-End Flow

[开发者查找配置] → [按职能定位目录] → [进入服务目录] → [编辑 config.yaml / 运行 sync] → [runAll 通过 conf_app 找到配置] → [服务正常启动]

## Value Increments

### Increment 1: Git 目录迁移 + Sync 路径更新 (Thin Slice)

**Value to user:** conf/ 目录变为二级结构，sync 机制在新路径下正常工作。

**Scope:**
- 在 conf/（独立 Git 仓库）内创建 9 个职能目录
- `git mv` 21 个服务目录到对应职能下
- 更新 7 个 `sync.manifest.yaml` 的 `from` 路径（`../<service>/` → `../../<职能>/<service>/`）
- 执行 `conf-sync-all.sh` 验证碎片生成一致
- 确保 `conf/logs/`、`conf/.runall/`、`conf/README.md`、`conf/runAll.yaml` 不变

**Depends on:** port-config-split-monorepo-conf (conf/ 目录已存在)

**验证方式:**
- `git status` 显示所有文件已 staged as renames（非 delete+add）
- `scripts/conf-sync-all.sh` exit 0，碎片内容与迁移前一致
- `git diff --stat` 仅显示路径变更，无内容变更（sync manifest 除外）

### Increment 2: 外部引用更新

**Value to user:** runAll 和所有工具脚本通过新路径正确加载配置。

**Scope:**
- `conf/runAll.yaml`: 更新所有 `conf_app` 值（~10 处），如 `task-auth` → `auth/task-auth`
- `scripts/conf-read.py`: 更新路径解析（如有硬编码）
- `scripts/conf-sync-all.sh`: glob `conf/*/sync.sh` → `conf/*/*/sync.sh`
- `scripts/ci/check_conf_sync.sh`: 同上
- `task2app/paths.conf` / FindMonorepoRoot: 标记文件 `conf/core/django/config.yaml` → `conf/core/django/config.yaml`

**Depends on:** Increment 1

**验证方式:**
- `runAll` 全栈启动，所有服务健康检查通过
- CI `check_conf_sync.sh` 通过

### Increment 3: 文档与价值流描述更新

**Value to user:** 所有文档引用与实际目录结构一致，新人可按文档导航。

**Scope:**
- `conf/README.md`: 更新目录结构图和路径引用
- `value-stream.yaml`: 更新涉及 conf 路径的 `description` 文本（4-5 处）
- 相关 `.ai.md` 伴读文件: 路径引用更新

**Depends on:** Increment 2

**验证方式:**
- `grep -r "conf/core/django/" docs/` 无遗留旧路径引用（或仅为历史文档）

## value-stream.yaml 变更

本次变更不新增独立 value stream entry，仅更新 `value-stream.yaml` 中已有步骤的 `description` 文本，将 `conf/<app>/` 路径更新为 `conf/<职能>/<app>/`。受影响条目约 4-5 处（与 conf 路径相关的字段描述）。
