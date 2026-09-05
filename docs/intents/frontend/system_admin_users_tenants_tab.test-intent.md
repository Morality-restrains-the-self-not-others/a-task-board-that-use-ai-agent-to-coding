# 测试意图：系统管理员用户页「租户」Tab

## 测试目标

证明 Tab 可见、可切换、深链生效，面板拉取租户列表并正确渲染。

## 测试分层

| 层 | 位置 |
|----|------|
| composable | `taskFE/app/src/composables/useSystemAdminUsers.referral-tab.test.js`（扩 tenants） |
| 页面 | `taskFE/app/src/views/SystemAdminUsers.tenants-tab.test.js` |
| 面板 | `taskFE/app/src/components/SystemAdminTenantsPanel.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| Tab 文本 | 含「租户」 |
| 点击租户 Tab | 出现 `system-admin-tenants-panel` |
| `?tab=tenants` | activeTab=tenants，不 GET users |
| 有 items | 表格显示 name |
| items=[] | 暂无租户 |
| API 非 2xx | 错误节点 data-traceId |

## 通过标准

上述测例全绿。
