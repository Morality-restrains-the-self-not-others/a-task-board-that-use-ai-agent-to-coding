# DDD：容器→SaaS 接口版本

限界上下文：**AI Provider Marketplace**（taskAiProvider）。

## 模型

- **Entity** `ContainerImage`：新增 `SaasInboundSkillVersion string`
- **VO** `SaasInboundSkillVersion`：非空、属于 Catalog 的 published 且非 sunset
- **Catalog**（只读端口）：`SkillVersionCatalog.Load() (current string, entries []SkillVersionEntry, err)`
  - 适配器读 `docs/skills/saas-container/versions.yaml`（与现有 skill markdown 同源）
- **Event** `ContainerImageSaasInboundSkillVersionAssigned` payload: `image_id`, `vendor_id`, `saas_inbound_skill_version`

## 不变量

1. 可编辑状态下写入的 skill version ∈ published \ {sunset}
2. 镜像 `version`（OCI/产品版本）与 skill version 独立
3. 审核通过后字段随镜像冻结（无独立 PATCH 只改 skill version 的员工接口）

## 端口

复用 `domain.EventBus`。不新增 Kafka topic（现适配器 `LogEventBus`）。
