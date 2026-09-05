# 功能意图：daydaymoney.yaml 仓库元信息全链路

**日期**: 2026-07-18  
**迭代**: daydaymoney-yaml-metadata  
**设计**: `docs/superpowers/specs/2026-07-18-daydaymoney-yaml-metadata-design.md`

## 业务意图

为每个服务仓库提供稳定、可版本管理的身份文件 `daydaymoney.yaml`（`service_id` + `tags`），使：

1. SaaS 项目标签可从仓库元信息自动填充；
2. 前端页面 header 暴露同一身份；
3. Chrome 插件与 Grafana 能用该身份**反查**用户可见的全部工作空间/项目归属（支持一对多）；
4. 服务日志携带同一身份，便于观测侧精确匹配。

## 关键约束

- YAML **不得**写入 `workspace_id` / `project_id` / `company_id`（归属在 SaaS 侧通过 tags 索引）。
- 反查结果允许多行；客户端不得假设唯一项目。

## 业务意图 → 事件对照

**无对应事件**：本迭代以只读 resolve/parse-yaml、日志字段注入与复用既有项目 `tags` PATCH 为主，不新增领域事件或 MQ 契约。

| 业务意图 | 领域事件 | MQ 契约 | 发布点 | 消费者 | 例外理由 |
|----------|----------|---------|--------|--------|----------|
| 解析 YAML / resolve 归属 | — | — | — | — | 无对应事件：只读查询，无业务状态变更 |
| 同步项目标签（PATCH tags） | — | — | — | — | 无对应事件：复用既有项目 tags 更新路径，无独立消费者 |

## API 落点

- Owner：`taskProjectService`（Go-first）
- `GET /api/tenant/{tid}/daydaymoney/resolve`
- `POST /api/tenant/{tid}/daydaymoney/parse-yaml`
- `GET /api/tenant/{tid}/projects/?tag=`
