# 人员管理 — 公司切换器 + 无权限优雅降级

**状态**: 草稿 (待审批)  
**日期**: 2026-06-28  
**依赖**: `people-manage-permission-isolation-design.md`（上一轮权限门禁）

---

## 1. 问题描述

### 当前行为（权限门禁已实现）

用户 `bowek64234@divahd.com`（普通成员）访问 `/tenant/850256677331562496/people/manage/`：

1. 后端 `company_members` API 返回 **403 Forbidden**
2. 前端 `MemberList.vue` 收到非 ok 响应 → `throw new Error('获取成员列表失败')` → `alert('获取成员列表失败')`
3. 用户体验差：看到弹窗报错，不知道是因为没权限还是系统故障

### 期望行为

1. 页面顶部有一个**公司下拉选择器**，列出用户所属的全部公司
2. 对于**有管理权限**（admin/creator）的公司 → 正常加载成员列表
3. 对于**无管理权限**的公司 → 返回空数组 + 标注「没有访问权限」
4. **不弹 alert**，不显示「获取成员列表失败」这类技术错误文案

---

## 2. 根因分析

| 层级 | 问题 |
|------|------|
| **后端** `company_members` API | 对无权限用户返回 403，前端无法区分"没权限"和"系统故障" |
| **后端** `company_nicknames` | 已返回用户所属公司列表，但缺少 `is_admin`/`is_creator` 字段 |
| **前端** `MemberList.vue` | `fetchMembers()` 对所有非 2xx 统一 `alert('获取成员列表失败')` |
| **前端** `PeopleManage.vue` | 无公司切换器，URL 中的 tenant 是唯一的公司入口 |

---

## 3. 设计方案

### 3.1 后端变更

#### A. `company_members` API — 从 403 改为 200 + 权限标记

**当前响应** (无权限): `403 {"error": "仅公司管理员可执行此操作"}`

**新响应** (无权限):
```json
200
{
  "members": [],
  "meta": {
    "has_permission": false,
    "permission_message": "您没有该公司的人员管理权限，仅公司管理员和创建者可查看成员列表",
    "can_manage_budget_permissions": false,
    "llm_budget_enabled": false,
    "budget_raise_by_member_id": {}
  }
}
```

**有权限时不变**，仅 `meta.has_permission` 为 `true`。

修改位置: `member_views.py` `_require_company_admin` → 改为 `_check_company_admin` 返回三元组 `(has_permission, error_response_or_none, message)`。

#### B. Profile `company_nicknames` — 增加 `is_admin` / `is_creator`

在 `_build_profile_payload` 的 `company_nickname_items` 中追加字段：

```json
{
  "company_id": "...",
  "company_name": "My Company",
  "member_name": "...",
  "member_avatar_url": "...",
  "is_admin": true,
  "is_creator": false
}
```

修改位置: `user_views.py` `_build_profile_payload`

### 3.2 前端变更

#### A. `MemberList.vue` — 处理 `has_permission: false`

```javascript
// fetchMembers() 中:
const data = await response.json()
const { members: rows, meta } = parseCompanyMembersResponse(data)

if (!meta.has_permission) {
  // 无权限 — 不弹 alert，设置状态让 UI 展示友好提示
  hasPermission.value = false
  permissionMessage.value = meta.permission_message || '没有访问权限'
  members.value = []
  return
}
hasPermission.value = true
members.value = rows
```

模板新增:
```html
<div v-if="!hasPermission" class="text-center py-8">
  <p class="text-text-light text-lg">🔒 {{ permissionMessage }}</p>
</div>
```

#### B. `PeopleManage.vue` — 增加公司切换下拉框

```
┌─────────────────────────────────────────────┐
│ 管理人员                                      │
│ ┌─────────────────────────────┐             │
│ │ 🏢 My Company (管理员)    ▼ │  ← 下拉框    │
│ └─────────────────────────────┘             │
│   ┌─ Other Company (成员)        ← 下拉选项  │
│   └─ Third Company (管理员)                  │
│                                              │
│ 🔒 您没有该公司的人员管理权限      ← 无权限提示 │
│                                              │
│ [成员列表或空]                                │
└─────────────────────────────────────────────┘
```

- 切换公司 → 更新 URL `/tenant/{new_id}/people/manage/` → `MemberList` 自动重新 fetch
- 下拉框数据来源: `/api/user/{userId}/accounts/users/me/` 的 `company_nicknames`

### 3.3 不做的事情

- **不改变 URL 结构** — 继续使用 `/tenant/{id}/people/manage/`，公司切换 = 路由跳转
- **不在下拉框中实时查询权限** — 权限信息在 profile API 中随 `company_nicknames` 一起返回
- **不移除 Sidebar 权限检查** — 侧边栏仍按 `isCompanyAdmin` 隐藏整个人员管理菜单

---

## 4. 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `company-management` | `member-crud` | `company_members` 响应格式变更（403→200+标记） |
| `company-management` | `member-permission-gate` | 上一轮新增的步骤，本次调整门禁策略（拒绝→标记） |
| `user-auth` | `email-register`/`login` | profile `company_nicknames` 新增 `is_admin`/`is_creator` |

## 5. 领域概念

| 概念 | 说明 |
|------|------|
| CompanyAccessRight | 用户对公司的访问权限级别：`admin` / `member` / `creator` |
| CompanySwitcher | UI 组件，列出用户所属公司及其权限标签 |

## 6. 变更文件汇总

| 文件 | 操作 | 说明 |
|------|------|------|
| `accounts/views/member_views.py` | 修改 | `_require_company_admin` → 返回标记而非 403 |
| `accounts/views/user_views.py` | 修改 | `company_nicknames` 增加 `is_admin`/`is_creator` |
| `components/MemberList.vue` | 修改 | 处理 `has_permission: false`，不弹 alert |
| `views/PeopleManage.vue` | 修改 | 增加公司切换下拉框 |
| `accounts/view_test/CompanyMemberViewSet_test.py` | 修改 | 权限测试改为断言 200 + has_permission=false |
