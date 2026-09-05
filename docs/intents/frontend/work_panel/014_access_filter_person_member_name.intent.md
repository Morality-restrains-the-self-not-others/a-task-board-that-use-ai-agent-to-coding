# 014 — 工作面板「人」过滤选项展示公司成员名称

- 日期：2026-07-19
- 相关 URL：`/tenant/{tenantId}/work-panel/`
- 关联：`docs/superpowers/specs/2026-07-15-work-panel-access-filter-design.md`

## 问题

访问过滤下拉 `[data-alias="access-filter-person-option"]` 可见文案为 `user_id` 前 8 位（如 `85025667`），而非公司成员名称。

## 根因

`WorkspaceAccessSerializer.get_user_info` 未返回 `member_name`，且 `username` 仅取 `UserProfile.username`，缺失时回落 `user_id[:8]`。前端 `parseAccessFilterSubjects` 亦优先 `username`。

## 目标

| # | 标准 | 验收 |
|---|------|------|
| S1 | 人员选项优先展示 `CompanyMember.member_name` | 按钮文案为成员名 |
| S2 | 无成员名时回落 profile username，再回落 user_id 前缀 | 与 `DisplayName` 一致 |
| S3 | `user_info` 含 `member_name` 字段 | AccessManagement 等同源 UI 可用 |
| S4 | 前端解析优先 member_name，并可借协作人池补全 | 单元测覆盖 |

## 落点

- `projects/serializers/workspace_access_serializer.py`
- `front_project/app/src/utils/workPanelAccessFilter.js`

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 工作面板人员过滤展示成员名 | — | — | — | — | 纯展示字段 enrichment，无状态变更 |
