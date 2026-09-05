# 测试意图：工作面板任务卡创建人可读化与评论展开

## 对应功能意图

`work_panel_task_card_creator_comments.intent.md`

## 用例

| ID | 前置 | 操作 / 输入 | 期望 |
|----|------|-------------|------|
| T1 | 任务含 `created_by.username=alice` | 挂载看板任务卡 | 可见「创建人」与 `alice`；无「添加评论」 |
| T2 | 任务 `created_by` 无用户名 | 挂载看板任务卡 | 显示「未知用户」；头像不为「user」首字母 U |
| T3 | 默认折叠评论区 | 点击「评论」 | 出现评论面板与输入框；再点收起 |

## 落地

- Vitest：`task2app/front_project/app/src/components/TaskDetail.card-ux.test.js`

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版 |
