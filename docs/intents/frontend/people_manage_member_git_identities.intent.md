# 前端：PeopleManage 成员 Git 身份管理

> 状态: **implemented** | 日期: 2026-08-12

## 意图

在 `/tenant/:id/people/manage/` 成员列表为每名成员提供 Git 身份管理入口（多身份）。

## 验收

1. 操作列有「Git 身份」按钮
2. 弹窗列出该公司该成员身份；可新增 / 设默认 / 删除
3. 错误展示带 `data-traceId`
4. API：`/api/git-identities/tenant/{tenantId}/member/{memberId}/`
