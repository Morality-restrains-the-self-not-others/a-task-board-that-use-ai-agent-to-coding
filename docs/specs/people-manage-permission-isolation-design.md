# 人员管理权限隔离 — 设计文档

**状态**: 草稿 (待审批)  
**日期**: 2026-06-28  
**作者**: Claude  

---

## 1. 问题描述

### 复现场景

1. 用户 `contact@daydaymoney.com` 创建了公司（租户 ID: `850256677331562496`），是公司创建者
2. 用户 `bowek64234@divahd.com` 被邀请为**普通成员**（`role=member`, `is_admin=False`）加入该公司
3. `bowek64234@divahd.com` 登录后访问 `http://183.250.1.132:4000/tenant/850256677331562496/people/manage/`
4. **预期**: 普通成员不应看到公司人员管理页面，应被拒绝访问或重定向
5. **实际**: 可以看到公司的完整人员列表、角色信息、成员状态等管理数据

### 影响范围

| 操作 | 当前权限检查 | 应有限制 |
|------|-------------|---------|
| 查看人员列表 (`company_members`) | 仅检查是否为该公司成员 | 应仅 admin/创建者可查看 |
| 邀请成员 (`invite`) | 仅检查是否为任何公司成员 | 应仅 admin/创建者可邀请 |
| 修改角色 (`update_role`) | 仅禁止修改创建者 | 应仅 admin/创建者可修改 |
| 启用/禁用成员 (`toggle_status`) | 仅禁止操作创建者 | 应仅 admin/创建者可操作 |
| 移除成员 (`destroy`) | 仅禁止移除创建者 | 应仅 admin/创建者可移除 |

---

## 2. 根因分析

### 2.1 后端 — CompanyMemberViewSet (`accounts/views/member_views.py`)

**核心漏洞**: 所有 action 方法的权限模型是「只要是该公司成员即可执行管理操作」，但正确的模型应该是「仅 admin 或创建者可以管理」。

```python
# 当前 company_members action (line 293-312) 的权限检查：
company_member = CompanyMember.objects.filter(
    user=current_user,
    company_id=company.id,
    is_active=True,
).first()
if not company_member:
    return Response({'error': '您不是该公司的成员'}, status=status.HTTP_403_FORBIDDEN)
# ❌ 缺少: 检查 company_member.is_admin 或 company.creator_id == user.id
```

受影响的 action 方法（均在 `CompanyMemberViewSet` 中）：

| Action | 行号 | 当前检查 | 缺失检查 |
|--------|------|---------|---------|
| `company_members` | 293-412 | 仅成员身份 | admin/creator |
| `invite` | 93-177 | 仅成员身份 | admin/creator |
| `update_role` | 414-463 | 仅防修改 creator | admin/creator |
| `toggle_status` | 465-484 | 仅防禁用 creator | admin/creator |
| `destroy` | 40-52 | 仅防移除 creator | admin/creator |

### 2.2 前端 — Sidebar (`components/Sidebar.vue`)

- "人员管理" 菜单项（含"邀请人"、"管理人员"、"管理分组"）对所有已认证用户可见
- 无 `v-if` / `v-show` 条件根据用户角色隐藏管理入口
- 路由配置 (`router.js` line 240-248) 仅设置 `requiresAuth: true`，无 `requiresAdmin` meta

### 2.3 数据模型 (`accounts/models/company_member.py`)

当前字段：
- `is_admin` (BooleanField): 标记是否为管理员
- `is_active` (BooleanField): 标记是否活跃

公司级别的角色仅有 `creator`（通过 `Company.creator_id` 关联）和 `is_admin` 两种区分，`member` 角色的用户 `is_admin=False`。

---

## 3. 设计方案

### 3.1 修复策略

**分层防御**:

1. **后端 API 层**（主要防线）: 在所有管理类 action 中增加 admin/creator 权限检查
2. **前端 UI 层**（辅助防线）: 根据用户角色隐藏管理入口，但不可作为安全依赖

### 3.2 后端修复 — 统一权限检查方法

在 `CompanyMemberViewSet` 中新增统一的权限检查辅助方法：

```python
def _require_company_admin(self, company_id):
    """要求当前用户是该公司 admin 或 creator，否则返回 403"""
    current_user = self.request.user
    try:
        company = Company.objects.get(id=company_id)
    except Company.DoesNotExist:
        return Response({'error': '租户不存在'}, status=status.HTTP_404_NOT_FOUND)
    
    # creator 拥有最高权限
    if str(company.creator_id) == str(current_user.pk):
        return None  # 放行
    
    member = CompanyMember.objects.filter(
        user=current_user,
        company_id=company.id,
        is_active=True,
    ).first()
    
    if not member:
        return Response({'error': '您不是该公司的成员'}, status=status.HTTP_403_FORBIDDEN)
    
    if not member.is_admin:
        return Response({'error': '仅公司管理员可执行此操作'}, status=status.HTTP_403_FORBIDDEN)
    
    return None  # 放行
```

然后在以下 action 方法开头调用此检查：

- `company_members` (GET) — 查看人员列表
- `invite` (POST) — 邀请成员
- `update_role` (PATCH) — 修改角色
- `toggle_status` (PATCH) — 启用/禁用
- `destroy` (DELETE) — 移除成员

### 3.3 前端修复 — 角色感知 UI

**方式 A** (推荐): 通过 profile API 获取当前用户在公司的角色信息，Sidebar 根据角色条件渲染

**方式 B**: 后端 `company_members` 返回的 `meta` 中已有用户自身的管理权限信息，但该接口本身需要先做权限隔离。

推荐从 `CompanyViewSet.current` (GET `/api/tenant/{id}/accounts/companies/current/`) 获取当前用户的 CompanyMember 信息（含 `is_admin` 字段），Sidebar 据此条件渲染。

### 3.4 不做的事情

- **不引入新的角色模型** — 当前 `creator` / `admin` / `member` 三级已足够
- **不修改前端路由守卫**为主防线 — 前端权限检查是 UX 优化，安全依赖在后端
- **不修改数据模型** — `CompanyMember.is_admin` 字段已满足需求

---

## 4. 价值流影响分析

查看 `conf/value-stream.yaml`，本次变更影响以下现有价值流：

### 4.1 直接影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `company-management` | `member-crud` | 所有成员管理 API 需增加角色校验 |
| `company-management` | `invitation-email` | 邀请 API 需限制为 admin |
| `company-management` | `pending-invitation-role` | 待处理邀请的查看权限 |

### 4.2 新增/变更字段

| 字段 | 说明 |
|------|------|
| `saas-backend.accounts_companymember.is_admin` | 已有字段，本次变更加强其运行时检查 |

### 4.3 测试影响

| 现有测试文件 | 需要的变更 |
|-------------|-----------|
| `accounts/view_test/CompanyMemberViewSet_test.py` | 新增普通成员访问被拒绝的测试用例 |
| `accounts/view_test/CompanyViewSet_test.py` | 验证公司信息 API 不受影响（普通成员仍可查看公司基本信息） |

### 4.4 是否需要新价值流

**不需要**。这是对现有 `company-management` 流的权限加固，不引入新的业务流程。

---

## 5. 领域概念清单

| 概念 | 类型 | 所属上下文 | 说明 |
|------|------|-----------|------|
| Company | 聚合根 | 组织与成员 | 公司/租户，creator_id 标识创建者 |
| CompanyMember | 实体 | 组织与成员 | 公司成员，is_admin 区分管理权限 |
| CompanyAdminPolicy | 领域服务 | 组织与成员 | 判定用户是否有公司管理权限（creator 或 admin） |
| MemberManagementAccessGranted | 隐含概念 | 组织与成员 | creator 或 is_admin=True 视为管理权限 |

---

## 6. 总结清单

- 后端 API: 新增 `_require_company_admin` 统一权限检查，应用到 5 个 action
- 前端 UI: Sidebar 根据 profile API 返回的角色信息隐藏管理入口
- 测试: 新增普通成员被拒绝的测试用例（至少覆盖 `company_members` 和 `invite`）
- 不改: 数据模型、角色体系、路由守卫（不强依赖前端权限）
