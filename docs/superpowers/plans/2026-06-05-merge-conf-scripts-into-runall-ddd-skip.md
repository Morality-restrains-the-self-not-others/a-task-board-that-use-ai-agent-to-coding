# DDD 领域建模: conf-read/sync 脚本合并进 runAll

> 输入:
> - 设计文档: `docs/design/merge-conf-scripts-into-runall.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-05-merge-conf-scripts-into-runall-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-05-merge-conf-scripts-into-runall-nfr-clarification.md`

## 跳过声明

**跳过 DDD 建模。** 理由：

1. 本次变更是脚本文件物理位置迁移（`scripts/` → `runAll/scripts/`），不引入新的业务概念或数据实体
2. 设计的核心动作是：(a) 创建 `conf_loader.py` (b) 移动 5 个文件 (c) 更新 14 处路径引用 (d) 删除旧文件
3. 无新聚合、实体、值对象、领域服务或领域事件需要建模
4. 设计文档中标识的领域概念（`AppConfig`、`ConfApp` 聚合、`ConfigSynced` 事件）是描述性的——描述现有 conf/ 系统的运作方式，不需要在本次变更中新增代码结构

**现有的领域模型不变。** runAll 的 `domain/` 目录（`service_lifecycle_*`, `managed_service_entity`, `config_fingerprint_value_object` 等）继续作为配置管理的领域层，本次变更不影响它们。

## 后续

直接进入 `/6-plans-实施计划`——将设计文档中的 5 个执行步骤转为可验证的 checkbox 任务清单。
