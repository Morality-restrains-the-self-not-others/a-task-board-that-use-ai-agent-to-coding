# 设计文档：公司切换后项目列表为空

**状态**: 草稿 (待审批)  
**日期**: 2026-06-28  
**账号**: `bowek64234@divahd.com`  
**URL**: `http://183.250.1.132:4000/tenant/850256677331562496/work-panel`  
**目标公司**: `example-user`

---

## 1. 问题描述

用户在 WorkPanel 页面切换到公司 `example-user` 后，点击「创建任务」，**项目列表为空**。已知该公司有大量项目。

此问题与刚刚修复的「公司切换后 Navbar 回退」bug 属于**同一根因家族**——`/me` API 缺少 `tenant_id` 上下文，导致错误的工作空间解析，进而项目列表 API 使用了错误的工作空间 ID 查询。

## 2. 根因分析

### 2.1 数据流追踪

```
公司切换 → URL: /tenant/<new_company_id>/work-panel
  │
  ▼
WorkPanel.initData()
  ├─ tenantId = URL path → new_company_id ✅ (正确)
  └─ GET /api/user/{userId}/accounts/users/me/
        │
        │  ⚠️ 此 URL 只含 user_id，不含 tenant_id
        │
        ├─ get_current_company(tenant_id=None) → 返回 .first() 公司
        └─ get_current_workspace(tenant_id=None)
              │
              ├─ get_workspace_by_tenant(session, None) → None
              ├─ get_session_workspace_id(request, None)
              │     └─ session['current_workspace_id']  ← 遗留 key，旧公司的 workspace！
              │
              ├─ Workspace.objects.filter(id=old_workspace_id, company_id__in=user_companies)
              │     └─ 验证通过（用户仍是旧公司成员）
              │
              └─ 返回旧公司的 workspace ❌
                    │
                    ▼
              currentWorkspace = {id: old_workspace_id, name: "旧公司的工作空间"}
                    │
                    ▼
              fetchProjects():
                GET /api/tenant/<new_company_id>/projects/?workspace_id=<old_workspace_id>
                    │
                    └─ 项目属于 new_company 但不属于 old_workspace → 空列表 ❌
```

### 2.2 关键代码路径

**前端** (`WorkPanel.vue:302`):
```javascript
const userResponse = await apiFetch(`/api/user/${userId}/accounts/users/me/`, {...})
//                                   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//                                   URL 路径只有 user_id，无 tenant_id
```

**后端** (`user_serializer.py:92-105`):
```python
def get_current_workspace(self, obj):
    request = self.context.get('request')
    tenant_id = None
    if request and hasattr(request, 'resolver_match') and request.resolver_match:
        tenant_id = request.resolver_match.kwargs.get('tenant_id')
        # ⚠️ /api/user/{uid}/accounts/users/me/ 的 kwargs 只有 user_id，无 tenant_id
        #    因此 tenant_id = None

    workspace_id = get_session_workspace_id(request, None)
    # → session['current_workspace_id'] ← 遗留全局 key，旧公司的 workspace
```

**分歧点**: `tenant_id` 从 URL path kwargs 提取，但 `/me` API 的 URL 不含 `tenant_id`。`get_current_workspace` 只能依赖 session 遗留的 `current_workspace_id`（旧公司 workspace），而非当前 URL 对应的公司 workspace。

### 2.3 与上一个 bug 的关联

| Bug | 症状 | 根因 |
|-----|------|------|
| #1: Navbar 回退 | 公司下拉框显示旧公司 | `current_company` 用 `.first()`，无 tenant 感知 |
| #2: 项目列表为空 | 创建任务时无项目可选 | `current_workspace` 用 session 遗留 key，无 tenant 感知 |

**共同根因**: `/me` API (`/api/user/{uid}/accounts/users/me/`) 的 URL 不含 `tenant_id`，导致后端无法根据当前页面 URL 解析公司/工作空间上下文。

## 3. 设计方案

### 3.1 方案选择：前端传参 + 后端增强

| 层级 | 动作 | 说明 |
|------|------|------|
| **前端** | `/me` API 调用增加 `?tenant_id=` query 参数 | 将当前 URL 的 tenant 传递给后端 |
| **后端** | `get_current_company` 和 `get_current_workspace` 增加 query param fallback | 当 `resolver_match.kwargs` 无 `tenant_id` 时，从 `request.GET` 读取 |
| **后端** | `get_current_workspace` 增加 tenant-aware 解析优先级 | 当 `tenant_id` 可用时，优先查找该 tenant 的 workspace |

### 3.2 前端变更

#### A. Navbar.logic.vue — `/me` 调用增加 tenant_id query

```javascript
// fetchCurrentUser() 中 meResponse 调用
const tenantParam = route.params.tenant ? `?tenant_id=${encodeURIComponent(route.params.tenant)}` : ''
const meResponse = await apiFetch(`/api/user/${userId}/accounts/users/me/${tenantParam}`, {...})
```

#### B. Sidebar.vue — `/me` 调用增加 tenant_id query

```javascript
// initData() 中
const tenantParam = route.params.tenant ? `?tenant_id=${encodeURIComponent(route.params.tenant)}` : ''
const response = await apiFetch(`/api/user/${userId}/accounts/users/me/${tenantParam}`, {...})
```

#### C. WorkPanel.vue — `/me` 调用增加 tenant_id query

```javascript
// initData() 中
const tenantParam = tenantIdFromUrl ? `?tenant_id=${encodeURIComponent(tenantIdFromUrl)}` : ''
const userResponse = await apiFetch(`/api/user/${userId}/accounts/users/me/${tenantParam}`, {...})
```

### 3.3 后端变更

#### D. user_serializer.py — `get_current_company` 增加 query param fallback

在已有的 `resolver_match` 提取后增加：

```python
def get_current_company(self, obj):
    # ... existing code ...
    request = self.context.get('request')
    tenant_id = None
    if request and hasattr(request, 'resolver_match') and request.resolver_match:
        tenant_id = request.resolver_match.kwargs.get('tenant_id')
    # ↓ 新增: query param fallback
    if not tenant_id and request:
        tenant_id = request.GET.get('tenant_id')
    # ... existing filter logic ...
```

#### E. user_serializer.py — `get_current_workspace` 同样增加 query param fallback

```python
def get_current_workspace(self, obj):
    # ... existing code ...
    if request and hasattr(request, 'resolver_match') and request.resolver_match:
        tenant_id = request.resolver_match.kwargs.get('tenant_id')
    # ↓ 新增: query param fallback
    if not tenant_id and request:
        tenant_id = request.GET.get('tenant_id')
    # ... existing workspace resolution logic (already tenant-aware when tenant_id is set) ...
```

### 3.4 变更文件汇总

| 文件 | 操作 | 行数估计 |
|------|------|---------|
| `front_project/app/src/components/Navbar.logic.vue` | 修改 | +2 行 — `/me` 调用加 `?tenant_id=` |
| `front_project/app/src/components/Sidebar.vue` | 修改 | +2 行 — `/me` 调用加 `?tenant_id=` |
| `front_project/app/src/views/WorkPanel.vue` | 修改 | +2 行 — `/me` 调用加 `?tenant_id=` |
| `Saas_project/accounts/serializers/user_serializer.py` | 修改 | +6 行 — `get_current_company` 和 `get_current_workspace` 增加 query param fallback |

### 3.5 不做的事情

- 不新增 API 端点
- 不改变 URL 结构
- 不引入新的 session key
- 不修改项目查询 API（下游自动修复）

---

## 4. 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `company-management` / `user-company-assoc` | 已有 | `get_current_company` 增加 query param fallback |
| `project-workspace` / `workspace-crud` | 已有 | `get_current_workspace` 增加 query param fallback，workspace 解析正确化 |

---

## 5. 领域概念

无新领域概念。`tenant_id` 的传递增强是基础设施层的上下文传播改进。

---

## 6. 测试影响

| 测试文件 | 改动 |
|----------|------|
| `accounts/view_test/UserViewSet_test.py` | 新增：`?tenant_id=` query param 场景下的 `current_company` / `current_workspace` 断言 |
| 前端组件测试 | 新增：`/me` API 调用包含 `?tenant_id=` 参数验证 |
| Playwright E2E | 新增：公司切换 → 工作空间正确 → 项目列表非空的端到端验证 |
