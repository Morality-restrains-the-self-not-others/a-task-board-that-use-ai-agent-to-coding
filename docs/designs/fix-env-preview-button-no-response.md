# Fix: 预览环境变量按钮点击无响应

**日期**: 2026-07-01
**状态**: 待审批
**类型**: Bug Fix

---

## 问题描述

任务详情页「直接启动」面板中，选择「个人配置」来源并选定配置后，点击「预览环境变量」按钮没有任何界面反馈——既不显示环境变量预览，也不显示错误提示。

**复现步骤**:
1. 进入任务详情页 → 直接启动 → 功能参数来源 → 选择「个人配置」
2. 从二级下拉中选择一个个人配置
3. 点击「预览环境变量」按钮
4. 观察：按钮短暂变为「加载中...」后恢复，无预览内容，无错误提示

---

## 根因分析

### 根因: URL 路径构造错误 — `relayToTraeApiUrl()` 注入了多余的 `relay-to-trae/` 段

`ServerConfig.logic.vue` 中 `fetchEnvPreview()` 使用 `relayToTraeApiUrl()` 构造 API URL:

```js
// ❌ 当前代码 (ServerConfig.logic.vue ~L495)
const resp = await apiFetch(
  relayToTraeApiUrl(`feature-params-env-preview/?${params.toString()}`),
  ...
)
```

`relayToTraeApiUrl()` (L337-343) 的模式为:
```
/api/tenant/{tid}/workspace/{wid}/task/{tid}/cloud/compute/relay-to-trae/{suffix}
```

实际请求 URL:
```
/api/tenant/.../cloud/compute/relay-to-trae/feature-params-env-preview/?source=personal&...
```

但后端 `get_feature_params_env_preview` 是 `CloudComputeViewSet` 的 DRF `@action`，挂载路径为:

```
saas_project/urls.py:333  →  cloud/task_cloud_urls.py  →  cloud/urls.py (DRF router)
```

实际后端路由:
```
/api/tenant/{tid}/workspace/{wid}/task/{tid}/cloud/compute/feature-params-env-preview/
```

**`relay-to-trae/` 段不在后端路由中** → 请求命中 404，前端未处理非 `ok` 响应 → 静默失败，用户看不到任何反馈。

### URL 对比

| 组件 | URL 路径 |
|------|---------|
| 前端构造 (错误) | `.../cloud/compute/relay-to-trae/feature-params-env-preview/` |
| 后端路由 (正确) | `.../cloud/compute/feature-params-env-preview/` |
| 差异 | 多了 `relay-to-trae/` — 该段仅用于 `instance_callback_urls.py` 中的容器回调 |

### 次要缺陷: 非 ok 响应无用户反馈

```js
if (resp.ok) {
  // ... 处理成功
}
// ❌ 没有 else 分支 — 404/500 等静默忽略
```

---

## 解决方案

### 修复 1: 直接构造 URL，绕过 relayToTraeApiUrl

`feature-params-env-preview` 是 `CloudComputeViewSet` 的端点，不是 relay-to-trae 代理端点。应使用 `resolveRelayContextIds()` 直接拼接路径。

**文件**: `front_project/app/src/components/ServerConfig.logic.vue`

```diff
-    const resp = await apiFetch(
-      relayToTraeApiUrl(`feature-params-env-preview/?${params.toString()}`),
-      {
-        credentials: 'include',
-        headers: { Accept: 'application/json' },
-      },
-    )
+    const { tenant_id, workspace_id, task_id } = ctx
+    const previewUrl = `/api/tenant/${tenant_id}/workspace/${workspace_id}/task/${task_id}/cloud/compute/feature-params-env-preview/?${params.toString()}`
+    const resp = await apiFetch(previewUrl, {
+      credentials: 'include',
+      headers: { Accept: 'application/json' },
+    })
```

### 修复 2: 增加失败反馈

```diff
     if (resp.ok) {
       const data = await resp.json()
       resolvedEnvPreview.value = data.env || {}
       envPreviewExpanded.value = true
+    } else {
+      const errData = await resp.json().catch(() => ({}))
+      console.error('预览环境变量失败:', resp.status, errData.message || resp.statusText)
     }
```

### 不需要改动

- **后端** — 端点已正确注册在 `cloud/compute/feature-params-env-preview/`，无需修改
- **`relayToTraeApiUrl()`** — 该函数设计用于 relay-to-trae 代理端点（如 `start/`, `status/`, `health/`），不应修改

---

## Domain Concept Inventory

无新增领域概念。修复仅涉及前端 URL 构造。

---

## Value Stream Impact

本次修复属于 `feature-params-source-selector` 价值流 Increment 2 的缺陷修正。不新增 value stream 条目。

---

## 总结清单

- 根因: `fetchEnvPreview()` 通过 `relayToTraeApiUrl()` 构造 URL，多注入了 `relay-to-trae/` 段导致 404
- 修复 1: 直接拼接 `.../cloud/compute/feature-params-env-preview/` URL
- 修复 2: 增加非 ok 响应的错误日志
