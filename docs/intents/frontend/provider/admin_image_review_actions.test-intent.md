# 测试意图：平台审核镜像操作列提供查看详情与按状态写操作

## 测试目标

验证平台审核容器镜像操作列按状态提供完整选项，详情弹窗可查看镜像地址/技能/运行环境，历史驳回识别 `action=reject`。

## 测试分层

- 单元（前端）：`taskAiProvider/frontend/tests/adminImageReviewActions.unit.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | status=`pending_review` | `reviewRowActions` | `[detail, approve, reject]` |
| T2 | status=`approved` | `reviewRowActions` | `[detail, unpublish]` |
| T3 | status=`rejected`/`draft`/空 | `reviewRowActions` | `[detail]` |
| T4 | histories 含 `reject` 与 `rejected` | `rejectHistories` | 两条均保留 |
| T5 | `AdminImageReviewTable.vue` | 读源码 | 含「查看详情」、`data-testid="admin-review-detail"`、通过/驳回/撤销上架 |
| T6 | `AdminImageReviewDetailModal.vue` | 读源码 | 含 `image_url`、`ImageResolveInfoPanel`、`runtime_environments` |
| T7 | `AdminImageReviewTab.vue` | 读源码 | `createClickGuard`；详情按钮 `Anti-Replay-OK: ui-only` |

## 数据与环境

- 前端单测读纯函数与 Vue 源码，无需浏览器、无需 staff token。

## 通过标准

```
cd taskAiProvider/frontend && npm run test:unit -- tests/adminImageReviewActions.unit.test.js
```

全绿。
