# 测试意图：Chrome 插件工作空间列表约定路径

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_workspace_list_convention_path.intent.md`

## 测试目标

锁定插件默认端点不再打已下线的 `/api/tenant/{companyId}/workspaces/`，并验证 `getWorkspaces` / `getMembers` 实际请求 URL 与响应解析。

## 测试分层

- 单元：`taskChromePlugin/test/api-endpoints.test.js`

## 用例矩阵

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | `getDefaultEndpoints().workspaces` | 单元 | `/api/projects/workspaces/tenant_id/{companyId}` |
| T2 | 其余项目/任务/云端点 | 单元 | 不含 `/api/tenant/{companyId}/workspaces` 等已下线模板；对齐 work-panel |
| T3 | `getWorkspaces('co1')` mock fetch | 单元 | GET `.../api/projects/workspaces/tenant_id/co1`；不请求 `/api/tenant/co1/workspaces` |
| T4 | `getWorkspaces()` 无 companyId | 单元 | 先 GET `/api/user/{uid}/accounts/users/me/`，再按 companies 拉约定工作空间路径 |
| T5 | `getMembers('co1')` | 单元 | GET `.../accounts/members/company_members/`；解析 `{ members: [] }` 为数组 |

## 数据与环境

- 不连真实网关；`fetch` mock。
- 基址 `https://example.test`。

## 通过标准

```bash
cd taskChromePlugin && node --test test/api-endpoints.test.js
```

全部 T1–T5 通过。
