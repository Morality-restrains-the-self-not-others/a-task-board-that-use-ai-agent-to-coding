# 测试意图：frontend gitlab_disk_manual_admin_fulfillment

对应 `docs/intents/frontend/gitlab_disk_manual_admin_fulfillment.intent.md`

## 测试目标

管理端能看到并操作待开通队列；租户连接页在 pending_admin 时只显示等待，不假装已开通。

## 测试分层

- `SystemAdminGitlabTenantPanel.click-guard.test.js`（扩展）
- `WorkspaceSettingsGitlabConnection.test.js`（既有 pending 文案）

## 用例矩阵

| ID | 场景 | 期望 |
|---|------|------|
| T1 | 超管面板 mock pending 列表 | 渲染租户/区域/GB |
| T2 | 点队列行 | 填入查询框的 tenant 与 slug |
| T3 | 点开通实施 | 一次 POST + Idempotency-Key（既有） |
| T4 | 连接页 pending_admin | 「等待管理员开通实施」；无已开通徽章 |

## 通过标准

相关 Vitest 全绿。
