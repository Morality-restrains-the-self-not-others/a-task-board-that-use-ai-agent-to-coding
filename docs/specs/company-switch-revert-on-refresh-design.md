# 公司切换后页面刷新回退到初始公司

**状态**: 草稿 (待审批)  
**日期**: 2026-06-28  
**URL**: `http://183.250.1.132:4000/tenant/850256677331562496/work-panel`  
**账号**: `bowek64234@divahd.com`

---

## 1. 问题描述

用户在 WorkPanel 页面通过 Navbar 的公司下拉框切换到公司 B，页面跳转到 `/tenant/B/work-panel` 并完整刷新。刷新后 **Navbar 公司下拉框显示的公司回退到页面初始加载时的公司**（公司 A），而非用户刚切换到的公司 B。

注意：WorkPanel 的**页面内容**（任务列表等）使用的是 URL 中的 tenant B（正确），但 Navbar 和 Sidebar 的公司选择器显示的是公司 A（错误）。两者不一致，用户困惑。

### 复现步骤

1. 用户登录，所属公司有 A 和 B
2. 打开 `/tenant/A/work-panel` → Navbar 显示公司 A
3. 在下拉框中选择公司 B → 跳转到 `/tenant/B/work-panel`
4. 页面刷新 → **Navbar 下拉框又显示公司 A**，但 URL 和页面内容是公司 B

---

## 2. 根因分析

### 2.1 数据流追踪

```
switchCompany(B)
  │
  ▼
window.location.href = '/tenant/B/work-panel'    ← 完整页面跳转，无 API 调用通知后端
  │
  ▼
页面加载 → onMounted()
  │
  ├─ Navbar.logic.vue → fetchCurrentUser()
  │     └─ GET /api/user/{id}/accounts/users/me/
  │           └─ UserSerializer.get_current_company()
  │                 └─ CompanyMember.objects.filter(user_id=...).first()  ← 总是返回 DB 中第一条
  │                       └─ 返回公司 A
  │                             └─ currentTenant.value = "A"  ← ❌ 忽略了 URL 中的 tenant=B
  │
  ├─ Sidebar.vue → initData()
  │     └─ 同上 /me API → currentTenant.value = "A"  ← ❌ 同样问题
  │
  └─ WorkPanel.vue → initData()
        └─ tenantIdFromUrl = pathSegments[1] = "B"  ← ✅ 优先使用 URL 中的 tenant
```

### 2.2 三层根因

| 层级 | 文件 | 行号 | 问题 |
|------|------|------|------|
| **后端** | `accounts/serializers/user_serializer.py` | 65-77 | `get_current_company` 用 `.first()` 取第一条 CompanyMember，无 URL/会话感知 |
| **前端 Navbar** | `components/Navbar.logic.vue` | 97-115 | `applyMePayload` 仅用 API 返回的 `current_company` 设置 `currentTenant`，忽略路由 `route.params.tenant` |
| **前端 Sidebar** | `components/Sidebar.vue` | 265-288 | `initData` 同样仅用 API 的 `current_company`，忽略路由参数 |

### 2.3 后端 get_current_company 代码

```python
# user_serializer.py:65-77
def get_current_company(self, obj):
    company_member = CompanyMember.objects.filter(user_id=str(obj.pk)).first()
    # .first() → 始终返回 DB 顺序的第一条，无论当前 URL 是什么
    if company_member:
        return {
            'id': str(company_member.company.id),
            'name': company_member.company.name,
            'is_admin': company_member.is_admin,
        }
    return None
```

对比 `get_current_workspace`（同一文件 79-105 行）——它**会尝试从 request 中解析 tenant_id**：

```python
def get_current_workspace(self, obj):
    request = self.context.get('request')
    tenant_id = None
    if request and hasattr(request, 'resolver_match') and request.resolver_match:
        tenant_id = request.resolver_match.kwargs.get('tenant_id')  # ← 有 URL 感知
    # ...
```

`get_current_company` 缺少同样的 URL 感知逻辑。

---

## 3. 设计方案

### 3.1 方案选择：前端为主 + 后端增强

| 层级 | 策略 | 理由 |
|------|------|------|
| **前端 (primary)** | Navbar/Sidebar 的 `currentTenant` 优先使用 `route.params.tenant` | 最小改动，与 WorkPanel 已有的 URL-first 模式一致 |
| **后端 (improvement)** | `get_current_company` 接受 `tenant_id` 参数，优先匹配 URL 中的公司 | 让 `/me` API 返回上下文正确的数据，其他消费者也受益 |

### 3.2 前端变更

#### A. Navbar.logic.vue — `applyMePayload` 增加 URL 优先逻辑

**文件**: `task2app/front_project/app/src/components/Navbar.logic.vue`

```javascript
// 修改 applyMePayload（约 97-126 行）
const applyMePayload = (userInfo, userId = '') => {
  // ... 现有 userData 设置不变 ...

  // 公司选择优先级：URL tenant > API current_company > companies[0]
  const routeTenant = route.params.tenant           // ← 新增
  if (routeTenant) {                                 // ← 新增
    // 验证 URL tenant 是否在用户所属公司列表中
    const match = (userInfo.companies || []).find(   // ← 新增
      c => String(c.id) === String(routeTenant)      // ← 新增
    )
    if (match) {
      currentTenant.value = String(routeTenant)      // ← 新增：URL 优先
    }
  }
  if (!currentTenant.value) {                        // ← 修改：仅当 URL 无匹配时才用 API
    if (userInfo.current_company && userInfo.current_company.id) {
      currentTenant.value = String(userInfo.current_company.id)
    } else if (userInfo.companies && userInfo.companies.length > 0) {
      currentTenant.value = String(userInfo.companies[0].id)
    }
  }

  // userCompanies 列表填充不变...
}
```

**为何在 applyMePayload 而非 fetchCurrentUser 中修改**：`applyMePayload` 是唯一设置 `currentTenant` 的地方，集中修改避免遗漏。

#### B. Sidebar.vue — `initData` 增加 URL 优先逻辑

**文件**: `task2app/front_project/app/src/components/Sidebar.vue`

```javascript
// 修改 initData（约 265-311 行）
const initData = async () => {
  try {
    const userId = getCookie('userId')
    if (!userId) { /* ... */ return }

    const response = await apiFetch(`/api/user/${userId}/accounts/users/me/`, { /* ... */ })
    if (response.ok) {
      const userData = await response.json()

      // 公司选择优先级：URL tenant > API current_company > companies[0]
      const routeTenant = route.params.tenant           // ← 新增
      let tenantId = ''                                 // ← 修改
      if (routeTenant) {                                // ← 新增
        const match = (userData.companies || []).find(  // ← 新增
          c => String(c.id) === String(routeTenant)     // ← 新增
        )
        if (match) tenantId = String(routeTenant)       // ← 新增
      }                                                 // ← 新增
      if (!tenantId) {                                  // ← 修改
        if (userData.current_company?.id) {
          tenantId = userData.current_company.id
        } else if (userData.companies?.length > 0) {
          tenantId = userData.companies[0].id
        }
      }

      if (tenantId) {
        currentTenant.value = tenantId
        // 查询管理员身份...（不变）
      }
    }
  } catch (error) { /* ... */ }
}
```

### 3.3 后端变更（可选增强）

#### C. UserSerializer.get_current_company — 增加请求上下文感知

**文件**: `task2app/Saas_project/accounts/serializers/user_serializer.py`

```python
def get_current_company(self, obj):
    from ..models import User
    if not isinstance(obj, User):
        return None

    # 尝试从请求 URL 中获取 tenant_id
    request = self.context.get('request')
    tenant_id = None
    if request and hasattr(request, 'resolver_match') and request.resolver_match:
        tenant_id = request.resolver_match.kwargs.get('tenant_id')

    company_members = CompanyMember.objects.filter(user_id=str(obj.pk))

    company_member = None
    if tenant_id and company_members:
        # 优先匹配 URL 中的 tenant
        company_member = company_members.filter(company_id=str(tenant_id)).first()
    if not company_member:
        company_member = company_members.first()  # fallback

    if company_member:
        return {
            'id': str(company_member.company.id),
            'name': company_member.company.name,
            'is_admin': company_member.is_admin,
        }
    return None
```

注意：部分 `/me` API 的调用场景中 URL 可能不含 `tenant_id`（如 `/profile/`、`/pricing/` 等），此时 `tenant_id` 为 None，回退到 `.first()` 行为，向后兼容。

### 3.4 不做的事情

- **不在前端新增 API 调用**通知后端"当前公司" — 已有 URL 作为真源，无需额外状态同步
- **不引入新的 localStorage/sessionStorage 字段** — URL 是唯一真源
- **不改变 `switchCompany` 的行为** — 继续使用完整页面跳转（与 workspace 切换的 `WorkspaceSwitcher` 模式一致）

---

## 4. 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `company-management` | `company-crud` | `get_current_company` 增加 tenant_id 感知（可选后端改动） |
| `company-management` | `user-company-assoc` | 前端公司选择器逻辑变更，URL tenant 优先 |

本次修复不引入新流、新步骤。Full value stream mapping 留给 step 3。

---

## 5. 领域概念

| 概念 | 说明 |
|------|------|
| **CurrentCompany** (current_company) | 用户当前操作的公司上下文 — 真源应为 URL `/tenant/:tenant`，API 字段仅作 fallback |
| **CompanySwitcher** | Navbar 中的 `<select>` 组件 — 通过 `switchCompany()` 跳转 URL 切换公司 |
| **CompanyMember** | 用户-公司关联 — `get_current_company` 的查询源，需按 `tenant_id` 过滤而非 `.first()` |
| **Tenant-Aware Context** | 设计原则 — 跨组件（Navbar/Sidebar/WorkPanel）统一使用 URL tenant 作为当前公司真源 |

---

## 6. 变更文件汇总

| 文件 | 操作 | 说明 |
|------|------|------|
| `front_project/app/src/components/Navbar.logic.vue` | 修改 | `applyMePayload` 中 `currentTenant` 优先使用 `route.params.tenant` |
| `front_project/app/src/components/Sidebar.vue` | 修改 | `initData` 中 `currentTenant` 优先使用 `route.params.tenant` |
| `Saas_project/accounts/serializers/user_serializer.py` | 修改 (可选) | `get_current_company` 增加 `tenant_id` URL 感知，优先匹配当前公司 |

---

## 7. 测试影响

| 测试文件 | 改动 |
|----------|------|
| `Navbar.logic.vue` 相关测试 | 覆盖：URL 有 tenant 时优先使用；URL 无 tenant 时 fallback 到 API |
| `Sidebar.vue` 相关测试 | 同上 |
| `UserSerializer` 测试 (`accounts/view_test/UserViewSet_test.py`) | 新增：`get_current_company` 在 `tenant_id` kwarg 存在时返回正确公司的断言 |
| Playwright E2E | 新增：WorkPanel 公司切换 → 刷新 → Navbar 显示正确公司的端到端验证 |
