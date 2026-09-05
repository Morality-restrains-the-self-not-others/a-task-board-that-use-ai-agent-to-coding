# 测试意图：Chrome 插件工作空间下拉去重并按租户区分同名项

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_workspace_list_dedupe.intent.md`

## 测试目标

锁定插件聚合工作空间时按公司 id / 工作空间 id 去重，并在跨租户同名时用公司名区分 option 文案。

## 测试分层

- 单元：`taskChromePlugin/test/workspace-list.test.js`
- 单元：`taskChromePlugin/test/api-endpoints.test.js`（T6 重复 company_id 只拉一次）

## 用例矩阵

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | `uniqueCompanies` 两个相同 id | 单元 | 只保留第一条 |
| T2 | `uniqueCompanies` `is_active: false` | 单元 | 跳过该条 |
| T3 | `appendWorkspaceRows` 相同工作空间 id | 单元 | merged 只一条 |
| T4 | 两租户同名「用户的工作空间」 | 单元 | option 文案含各自 `company_name` |
| T5 | 单租户且名称不冲突 | 单元 | option 文案仅为工作空间名 |
| T6 | `getWorkspaces()` `/me` 重复 company_id | 单元 | 工作空间 URL 只请求一次 |
| T7 | `getWorkspaces()` 两租户同名工作空间 | 单元 | 返回两条不同 id，并带各自 company_name |

## 数据与环境

- 不连真实网关；`getWorkspaces` 测例 `fetch` mock。
- 基址 `https://example.test`。

## 通过标准

```bash
cd taskChromePlugin && node --test test/workspace-list.test.js test/api-endpoints.test.js
```

全部 T1–T7 通过。
