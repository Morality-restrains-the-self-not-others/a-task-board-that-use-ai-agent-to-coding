# Feature-Params 公网接口迁 Go

## 意图

将公司/工作空间/个人 feature-params 公网 CRUD 从 Django 迁到 taskCloudService，经 taskGateway 切流；表暂仍由 saas-backend internal store 持有。

## 验收

1. Gateway `task-cloud-feature-params` 路由至 taskCloudService
2. Django 公网同路径直连 → 410
3. 保留 access-context / view=summary / 审计
4. Internal upsert/list/audit 可供 Cloud 调用
5. Go + Django 相关单测通过

## 变更日期

2026-07-19

## 业务意图 → 事件对照

**无对应事件**：公网 API 路由迁 Go（HTTP），无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 公网 feature-params CRUD 迁 Go | — | — | — | — | API 路由迁 Go（HTTP），无新增业务事件 |
