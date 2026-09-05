# DDD 领域建模: conf/ 目录二级职能分组

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-05-conf-directory-restructure-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-05-conf-directory-restructure-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-05-conf-directory-restructure-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

本次变更为 **配置目录结构调整**，属于 DDD 技能明确定义的跳过条件：

- ✅ **配置变更** — 仅重命名目录路径（`git mv`），不涉及新的业务概念
- ✅ **无新限界上下文** — 现有 8 个 value stream domain 不变
- ✅ **无新实体/值对象/聚合** — 无业务逻辑变更
- ✅ **无新领域事件** — 无跨上下文通信变更

## 领域概念（仅文档记录）

设计文档中已识别的概念（供参考，无需生成领域模型文件）：

| 概念 | 类型 | 说明 |
|------|------|------|
| ServiceConfig | 实体 | 每服务的 config.yaml，已有概念 |
| ConfigFunction | 值对象 | 职能分组标签 (auth/core/gateway/ai/events/infra/billing/frontend/mock) |
| SyncManifestEntry | 值对象 | sync.manifest.yaml 的条目，已有概念 |
| FunctionConfigBundle | 聚合 | 一个职能下所有服务配置的集合（目录级聚合） |

这些概念已在现有 conf-sync 工具链（`conf-sync.py`、`sync.manifest.yaml`）中体现，无需新增领域模型文件。

## 对实施的影响

`ConfigFunction`（职能标签）是本变更唯一引入的新概念。它在以下位置体现：

- `conf/<职能>/` 目录名（文件系统层级，非代码）
- `conf/runAll.yaml` 的 `conf_app` 路径值
- `scripts/conf-sync-all.sh` 的 glob 模式

无需创建独立的领域层代码——目录结构本身就是 ConfigFunction 的实现。
