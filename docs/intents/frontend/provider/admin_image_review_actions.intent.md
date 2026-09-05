# 意图：平台审核镜像操作列提供查看详情与按状态写操作

## 背景与目标

`https://provider.daydaymoney.com/admin`「容器镜像审核」待审核行的操作列原先只渲染「通过」「驳回」。列表 API 已返回 `image_url`、技能、自动运行说明、运行环境，审核员无法在决策前查看这些字段。已驳回/草稿行操作列为空。

目标：任意状态至少提供「查看详情」；待审核另有通过/驳回；已上架另有撤销上架。详情弹窗展示镜像地址、架构、技能/自动运行与运行环境。历史驳回原因识别后端 `action=reject`（不只是 `rejected`）。

## 范围与边界

- 范围内：`taskAiProvider/frontend` 平台审核容器镜像 Tab 的操作列与详情/确认弹窗。
- 范围外：不新增审核 API、不改审批状态机、不改厂商门户版本操作。

## 约束与风险

- 写操作（通过/驳回/撤销上架）须 `createClickGuard` + `Idempotency-Key`；查看详情为 `Anti-Replay-OK: ui-only`。
- 禁止 `alert`/`confirm`/`prompt`；驳回与撤销上架原因用自定义模态。
- 请求失败展示须带 `data-traceId`。
- 无新的领域事件：审核仍走既有 POST approve/reject/unpublish。

## 验收标准

1. 待审核行操作列可见「查看详情」「通过」「驳回」。
2. 已上架行可见「查看详情」「撤销上架」。
3. 已驳回/草稿行可见「查看详情」，操作列不为空。
4. 详情弹窗含镜像地址、`ImageResolveInfoPanel`、运行环境表。
5. `review_histories` 中 `action=reject` 出现在「历史驳回原因」。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 查看镜像审核详情 | — | — | — | — | 纯前端展示列表已有字段 |
| 通过/驳回/撤销上架 | ContainerImageApproved / Rejected（既有日志） | — | `handleAdminContainerImages` | 镜像状态变更 | 无新 MQ；沿用既有 POST |

## 实施计划

1. `adminImageReviewActions.js` 作为操作矩阵与历史过滤 SSOT。
2. 抽出表格、详情弹窗、确认/原因弹窗。
3. 单测覆盖矩阵、`reject`/`rejected`、Vue 接线。

## 变更记录

- 2026-08-28：待审核操作列从仅通过/驳回补齐查看详情与按状态动作。
