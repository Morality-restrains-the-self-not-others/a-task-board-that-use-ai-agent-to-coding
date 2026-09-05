# Value Stream: runAll + conf/ 配置统一

> Derived from design: `docs/superpowers/specs/2026-06-03-runall-conf-migration-design.md`

## Value Summary

开发者只需维护 `conf/` 一套配置，runAll 自动从中派生健康检查 URL，消除 IP 变更时的配置漂移。

## Related Value Streams

- **[2026-06-02-port-config-split-monorepo-conf-value-stream](2026-06-02-port-config-split-monorepo-conf-value-stream.md)**: 前置 — 将端口配置从 port_config.json 拆分为 conf/<app>/config.yaml。本迁移在此基础上让 runAll 读取 conf/。
- **[2026-06-02-conf-hardening-phase45-value-stream](2026-06-02-conf-hardening-phase45-value-stream.md)**: 前置 — conf/ 配置加固。本迁移是 conf/ 统一化的延续。

## End-to-End Flow

[开发者编辑 conf/<app>/config.yaml] → [runAll 启动时读取 conf/runAll.yaml] → [自动解析 conf_app 映射] → [拼接 health_check URL] → [服务启动并探活成功]

## Value Increments

### Increment 1: 核心迁移 (Thin Slice)
**Value to user:** 开发者更改 conf/ 中的 IP/端口后，runAll 自动使用新值，无需手动同步 runAll.yaml。
**Scope:**
- 新建 `conf/runAll.yaml`（从 `runAll.yaml` 迁移编排逻辑）
- runAll Go 程序新增 `conf_app` / `health_path` 解析
- `remote_docker.host` 自动从 `conf/docker-infra/config.yaml` 读取
- 平台服务（12个）使用 `conf_app` 映射；基础设施服务（22个）保留显式 `url`/`tcp`
- 删除根目录 `runAll.yaml`
**Depends on:** port-config-split-monorepo-conf（`conf/<app>/` 已存在 host/port 字段）

### Increment 2: 验证与清理
**Value to user:** 确保迁移后运行一致性，更新相关文档和脚本。
**Scope:**
- 运行 `test.sh` 验证 Go 测试通过
- 更新 `runAll/run.sh`、`build.sh` 默认路径
- 更新 `runAll.yaml.ai.md` 规则文件
- 更新 `runAll/README.md` 中配置路径引用
**Depends on:** Increment 1
