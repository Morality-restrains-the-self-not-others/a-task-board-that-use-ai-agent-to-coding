# 价值流：容器→SaaS 接口版本

Mapping the approved design into a value stream.

## Related Value Streams

Greenfield 增量挂在既有「厂商镜像市场上架」流上，无独立 `*-value-stream.md` 与本主题同名。相关：`2026-07-16-ai-provider-python-to-go-migration-value-stream.md`（门户已在 Go）。本增量是该流的登记步骤增强，不撤销既有增量。

## 端到端流

厂商打开 skill 页了解契约 → 添加镜像版本时选择接口版本 → 草稿保存 → 提交审核 → 员工看到接口版本 → 上架后公开目录带该字段。

## 增量（按价值）

1. **P0 契约可识别**：versions.yaml + skill 页 badge + GET versions + GET md?version=
2. **P0 厂商必选**：列 + POST/PUT 校验 + 表单 select
3. **P1 审核/目录可见**：admin 表 + public catalog JSON

一期交付 1–3。运行时 env 注入为后续增量。

## 字段

| name | description |
|------|-------------|
| taskAiProvider.ai_provider_vendorcontainerimage.saas_inbound_skill_version | 已发布 inbound 契约版本 |
