# 设计文档：修复跨公司数据隔离 — tenant_id 作用域缺失

## 1. 问题概述

**现象**: 用户 `bowek64234@divahd.com` 在页面 `/tenant/850256677331562496/work-panel`（已切换到 `example-user` 的公司），创建任务时触发请求：

```
GET /api/tenant/850256677331562496/projects/workspace-access/workspace-collaborators/?workspace_id=858972707541250048
```

响应返回了 **bowek 自己公司** (`858949970286374912`) 的 workspace `857903329669984256`（"默认工作空间"），而非 `example-user` 公司的数据。

**问题本质**: 后端多租户接口收到 `tenant_id` 但未用于作用域查询，`.first()` 非确定性选择了错误公司的数据；前端公司切换时保留了上一个公司的查询参数。

## 2. 根因分析（双层问题）

### 2.1 后端根因（PRIMARY — 数据安全）

**文件**: `Saas_project/projects/views/workspace_access_views.py:211`

```python
def workspace_collaborators(self, request, tenant_id=None, workspace_id=None):
    # tenant_id 从 URL 接收 ✅
    current_user = self.request.user
    company_member = CompanyMember.objects.filter(user_id=str(current_user.pk)).first()
    # ❌ .first() 无排序 → 非确定性选择公司
    # ❌ tenant_id 完全未用于作用域查询
```

**攻击路径**:
1. 用户属于 A、B 两个公司
2. 在 `/api/tenant/B/...` 下访问（URL 中是 B 的 tenant_id）
3. `.first()` 随机返回 A 的 company_member
4. `workspace.company != company_member.company` 检查 workspace 是否属于 A（而非 B）
5. 如果 workspace 也属于 A → 检查通过 → **B 的 URL 下泄露 A 的数据**

**根本缺陷**: `CompanyMember.objects.filter(user_id=...).first()` 对多公司用户非确定性；`tenant_id` 参数未被用于约束公司选择。

### 2.2 前端根因（CONTRIBUTING — 用户体验）

**文件**: `front_project/app/src/components/Navbar.logic.vue:196-200`

```javascript
const switchCompany = (companyId) => {
  const path = window.location.pathname.replace(/^\/tenant\/\d+/, '') || '/work-panel/'
  const search = window.location.search || ''  // ❌ 保留了包括 ?workspace_id= 在内的所有查询参数
  window.location.href = '/tenant/' + companyId + path + search
}
```

**文件**: `front_project/app/src/views/WorkPanel.vue:339-341`

```javascript
if (finalWorkspaceId && currentWorkspace.value.id !== finalWorkspaceId) {
  currentWorkspace.value.id = finalWorkspaceId  // ❌ URL workspace_id 覆盖 API 返回值，无租户验证
}
```

**因果链**:
1. 用户在 A 公司页面，URL 含 `?workspace_id=A_ws`
2. 切换到 B 公司 → `switchCompany` 保留查询参数
3. URL 变为 `/tenant/B/work-panel/?workspace_id=A_ws`
4. `WorkPanel.initData()` 用 URL 的 `A_ws` 覆盖了 API 返回的 B 的 workspace
5. `CreateTaskModal` 用 `A_ws` 调 `workspace-collaborators` API

## 3. 受影响范围

### 3.1 CRITICAL — 后端接收 tenant_id 但未作用域

| # | 文件 | 行号 | 方法 | 风险 |
|---|------|------|------|------|
| 1 | `projects/views/workspace_access_views.py` | 18 | `get_queryset` | 全局作用域缺失 |
| 2 | 同上 | 41 | `workspace_permissions` | 跨公司权限暴露 |
| 3 | 同上 | 76 | `set_permission` | 跨公司写权限 |
| 4 | 同上 | 141 | `remove_permission` | 跨公司删权限 |
| 5 | 同上 | 176 | `company_workspaces` | 事后验证不足 |
| 6 | 同上 | 211 | `workspace_collaborators` | **本次触发** |
| 7 | `accounts/views/group_views.py` | 27, 37 | `get_queryset`, `perform_create` | 跨公司分组操作 |
| 8 | `accounts/views/member_views.py` | 207 | `invite` | 跨公司邀请 |
| 9 | 同上 | 615 | `pending_invitations` | 跨公司受邀列表 |
| 10 | 同上 | 689 | `resend_invitation_link` | 跨公司重发邀请 |

**共 11 处 CRITICAL 实例**（3 个文件）。

### 3.2 HIGH — 登录/重定向使用 .first() 无排序

| # | 文件 | 行号 | 风险 |
|---|------|------|------|
| 1 | `accounts/taskauth_internal_views.py` | 120 | 登录后随机跳转到任一公司 |
| 2 | `accounts/taskauth_bridge/principal_loader.py` | 76 | `companies[0]` 非确定性 |
| 3 | `frontend_app/views/auth_views.py` | 60, 124 | 登录重定向随机公司 |
| 4 | `frontend_app/views/project_views.py` | 36 | 项目重定向随机公司 |
| 5 | `accounts/serializers/user_serializer.py` | 85, 110 | `.first()` fallback |
| 6 | `accounts/serializers/group_serializer.py` | 38 | 分组 queryset 作用域缺失 |

### 3.3 前端风险点

| # | 文件 | 风险 | 严重度 |
|---|------|------|--------|
| 1 | `Navbar.logic.vue:198` | 公司切换保留查询参数 | HIGH |
| 2 | `WorkPanel.vue:339-341` | URL workspace_id 覆盖 API 值 | HIGH |
| 3 | `WorkspaceSwitcher.vue:156` | workspace_id 固化到 URL | HIGH |
| 4 | `router.js:708-744` | 无租户变更时清理本地状态 | HIGH |
| 5 | `Projects.vue:291,300` | 租户回退时保留 workspace_id | MEDIUM |
| 6 | `CreateTaskModal.vue:1114` | 无 workspace→tenant 归属验证 | MEDIUM |

## 4. 修复方案

### 4.1 后端核心修复：统一工具函数

**新建文件**: `Saas_project/accounts/workspace_context.py`（已存在，增强）

```python
def resolve_company_member_for_tenant(user_id: str, tenant_id: str | None) -> CompanyMember | None:
    """
    根据 tenant_id 解析用户在该租户下的 CompanyMember。
    
    - tenant_id 必须提供：强制作用域到目标租户
    - 返回 None：用户不是该租户成员 → 调用方返回 403
    """
    if not tenant_id:
        raise ValueError("tenant_id is required for company scoping")
    
    try:
        tenant_id_int = int(tenant_id)
    except (ValueError, TypeError):
        return None
    
    return CompanyMember.objects.filter(
        user_id=str(user_id),
        company_id=tenant_id_int,
    ).first()
```

**修改**: 将所有 CRITICAL 实例中的:
```python
company_member = CompanyMember.objects.filter(user_id=str(current_user.pk)).first()
```
替换为:
```python
company_member = resolve_company_member_for_tenant(str(current_user.pk), tenant_id)
```

### 4.2 workspace_access_views.py 修复示例

```python
@action(detail=False, methods=['get'])
def workspace_collaborators(self, request, tenant_id=None, workspace_id=None):
    from accounts.workspace_context import resolve_company_member_for_tenant
    
    current_user = self.request.user
    company_member = resolve_company_member_for_tenant(str(current_user.pk), tenant_id)
    
    if not company_member:
        return Response({'error': '无权访问该租户'},
                       status=status.HTTP_403_FORBIDDEN)
    
    # 后续逻辑不变，此时 company_member 已确保属于 tenant_id 对应的公司
    ...
```

### 4.3 前端修复

#### 4.3.1 公司切换时清除跨租户查询参数

```javascript
// Navbar.logic.vue switchCompany
const switchCompany = (companyId) => {
  const path = window.location.pathname.replace(/^\/tenant\/\d+/, '') || '/work-panel/'
  // 清除跨租户查询参数
  const url = new URL(window.location.href)
  url.searchParams.delete('workspace_id')  // 清除旧公司的 workspace_id
  const cleanSearch = url.searchParams.toString()
  window.location.href = '/tenant/' + companyId + path + (cleanSearch ? '?' + cleanSearch : '')
}
```

#### 4.3.2 WorkPanel 不盲信 URL workspace_id

```javascript
// WorkPanel.vue initData — 信任 API > URL
if (userData?.current_workspace) {
  currentWorkspace.value = userData.current_workspace
  // 仅在 URL workspace_id 与 API 一致时保留，否则信任 API
  if (finalWorkspaceId && currentWorkspace.value.id !== finalWorkspaceId) {
    // 清理 URL 中的无效 workspace_id
    const url = new URL(window.location.href)
    url.searchParams.delete('workspace_id')
    window.history.replaceState({}, '', url.toString())
  }
}
```

### 4.4 已存在的相关 value stream

在 `conf/value-stream.yaml` 中已有 **planned** 步骤 `company-switch-url-context`（`company-management` 域），其描述与本修复部分重叠：

```yaml
- name: company-switch-url-context
  status: planned
  fields:
  - name: saas-backend.accounts_companymember.company_id
    description: get_current_company 新增 tenant_id aware filter
  - name: taskFE.runtime.currentTenant_resolution
    description: Navbar/Sidebar currentTenant 优先级从 API-only 改为 URL-first
  - name: saas-backend.projects_workspace.id
    description: get_current_workspace 在 tenant_id query param 可用时优先解析该 tenant 的 workspace
```

本修复应当将此步骤从 `planned` 推进到 `active`，但需注意：这一步只覆盖了 `get_current_company`/`get_current_workspace` 的序列化器层，**未覆盖视图层的 11 处 `.first()` 非确定性**。本修复是此步骤的必要补充。

## 5. 执行计划

### Phase 1: 后端核心修复（数据安全）
1. 增强 `accounts/workspace_context.py` — 统一 `resolve_company_member_for_tenant()`
2. 修复 `projects/views/workspace_access_views.py` — 6 处
3. 修复 `accounts/views/group_views.py` — 2 处
4. 修复 `accounts/views/member_views.py` — 3 处

### Phase 2: 后端次要修复（用户体验）
5. 修复 `accounts/taskauth_internal_views.py` — 登录重定向排序
6. 修复 `accounts/taskauth_bridge/principal_loader.py` — companies 排序
7. 修复 `frontend_app/views/auth_views.py` — 重定向排序
8. 修复 `accounts/serializers/user_serializer.py` — fallback 排序

### Phase 3: 前端修复
9. 修复 `Navbar.logic.vue` — 切换时清除 workspace_id
10. 修复 `WorkPanel.vue` — 不盲信 URL workspace_id
11. router guard 增强

### Phase 4: 测试与验证
12. 为 `resolve_company_member_for_tenant` 编写单元测试
13. 为每个修复的视图编写多公司隔离测试
14. E2E 测试：跨公司 workspace 隔离

## 6. Domain Concepts

- **Bounded Contexts**: 租户/公司管理（accounts）、工作空间管理（projects）
- **Key Entities**: `CompanyMember`（租户作用域成员）、`Workspace`（公司下属工作空间）、`WorkspaceAccess`（工作空间访问权限）
- **核心约束**: `CompanyMember.company_id` 与 URL `tenant_id` 必须匹配
- **Aggregate Root**: `Company` — 所有查询必须作用域到公司边界内

## 7. 测试影响

| 测试文件 | 改动 |
|----------|------|
| `accounts/tests/test_workspace_context.py` | 新增：`resolve_company_member_for_tenant` 正确/错误/多公司用例 |
| `projects/view_test/WorkspaceAccessViewSet_test.py` | 增强：多公司用户交叉访问测试 |
| `projects/view_test/WorkspaceViewSet_switch_workspace_test.py` | 增强：跨公司 workspace 隔离 |
| `accounts/view_test/CompanyMemberViewSet_test.py` | 增强：invite/pending_invitations 多公司隔离 |
| `accounts/view_test/CompanyViewSet_test.py` | 增强：多公司分组隔离 |
| `accounts/tests/test_taskauth_bridge.py` | 增强：principal_loader companies 排序 |
| `playwright/front_project/tests/` | 新增：跨公司切换 workspace_id 清理 E2E 测试 |

## 8. 风险矩阵

| 风险 | 等级 | 缓解 |
|------|------|------|
| `.first()` → 确定性查询后用户突然无法访问数据 | 中 | URL 已有 `tenant_id`，转换是严格收紧而非放宽 |
| 前端清除 workspace_id 可能破坏深层链接 | 低 | Phase 3 仅清除跨租户的无效 workspace_id；同租户内保留 |
| 大量 CRITICAL 处修改可能引入回归 | 中 | 统一函数封装 + 每处独立测试 + CI 门禁 |
| 登录重定向排序变更影响全用户 | 高 | 仅在 `company-management` 域使用确定性排序（`created_at` 或其他明确规则）；Phase 2 需要用户确认排序策略 |

## 9. 未纳入范围

- 其他使用 `.first()` 但不用 tenant_id 的非 CRITICAL 处（MEDIUM/LOW 级别），将在后续 debt cleanup 中处理
- SaaS_Ai_Provider 服务 — 经确认无 CompanyMember 引用，不受影响
- 其他 Go 服务（gitOauth, gitService, taskEvents 等）— 不涉及用户公司作用域

## 10. 澄清待定

在进入实施前需要确认：

1. **排序策略**: `resolve_company_member_for_tenant` 中 `.first()` 现在只有一个公司（因为加了 `company_id=tenant_id_int` 过滤），无需排序。但登录/重定向场景无法从 URL 获取 tenant_id，需要确定公司选择策略（最早创建？最后活跃？）。

2. **前端范围**: 是否仅修复 `workspace_id` 查询参数，还是需要清理所有可能跨租户污染的参数（如 `project_id` 等）？
