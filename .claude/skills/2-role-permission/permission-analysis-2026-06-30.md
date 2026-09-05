# 角色权限分析 — System Admin 真实数据填充 + 用户管理重构

日期: 2026-06-30
参考设计: `docs/superpowers/specs/2026-06-30-system-admin-real-data-design.md`
风险评级: 🟢 **低风险** — 所有新增端点受现有 `SystemAdminPermissionMiddleware` 保护

---

## 1. 权限影响矩阵

### Django 层端点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否充分 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `GET /api/system-admin/dashboard/` | superuser | System | read | `SystemAdminPermissionMiddleware` (is_superuser) | ✅ | 无额外措施 |
| `GET /api/system-admin/users/` | superuser | System | read | `SystemAdminPermissionMiddleware` (is_superuser) | ✅ | 无额外措施 |
| `POST /api/system-admin/users/` | superuser | System | write | `SystemAdminPermissionMiddleware` (is_superuser) | ✅ | 建议记录审计日志（谁创建了谁） |
| `PUT /api/system-admin/users/<id>/` | superuser | System | write | `SystemAdminPermissionMiddleware` (is_superuser) | ⚠️ | 需防止 superuser 自我降级（自锁保护） |
| `DELETE /api/system-admin/users/<id>/` | superuser | System | write | `SystemAdminPermissionMiddleware` (is_superuser) | ⚠️ | 需防止 superuser 自我禁用（自锁保护）；软删除保证可逆 |

### taskAuth 内部端点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否充分 | 建议 |
|--------|------|----------|------|------|----------|------|
| `GET /api/internal/users/` | Django backend | System | read | `requireInternalSecret` (X-TaskAuth-Internal-Secret) | ✅ | 内部端点，secret 验证充分 |
| `GET /api/internal/users/count/` | Django backend | System | read | `requireInternalSecret` (X-TaskAuth-Internal-Secret) | ✅ | 内部端点，无敏感字段暴露 |
| `PATCH /api/internal/users/{id}/` | Django backend | System | write | `requireInternalSecret` (X-TaskAuth-Internal-Secret) | ⚠️ | 需在 Django 侧验证调用方是 superuser（中间件已覆盖），taskAuth 侧仅需验证 secret |

### 前端层（权限门控）

| 改动点 | 主体 | 资源层级 | 检查 | 是否充分 | 建议 |
|--------|------|----------|------|----------|------|
| `SystemAdmin.vue` — dashboard 数据加载 | superuser | System | `router.js` meta.requiresAdmin 路由守卫 | ✅ | 路由守卫已验证 is_superuser |
| `SystemAdminUsers.vue` — 用户 CRUD UI | superuser | System | `router.js` meta.requiresAdmin 路由守卫 | ✅ | 所有操作按钮（添加/编辑/删除）仅 superuser 可见 |
| `SystemAdminSidebar.vue` — 导航结构 | superuser | System | Navbar `v-if="currentUser.isSuperuser"` | ✅ | 非管理员看不到系统管理入口 |

---

## 2. 角色与权限建模

### 本次变更不引入新角色

所有操作限定在现有 `superuser` 角色范围内，无需新增角色类型。

### 现有权限层级回顾

```
superuser (System-level)
  ├─ tenant_admin (Company-level)
  │    └─ workspace_admin (Workspace-level)
  │         └─ workspace_member (Workspace-level)
  └─ staff (System-level, lower privilege)
```

### 系统管理员操作权限矩阵

| 操作 | 权限要求 | 检查层 |
|------|----------|--------|
| 查看仪表盘统计数据 | `is_superuser = True` | L1 Middleware: `SystemAdminPermissionMiddleware` |
| 查看所有用户列表 | `is_superuser = True` | L1 Middleware |
| 创建新用户 | `is_superuser = True` | L1 Middleware + L4 Service: 自锁保护 |
| 更新用户信息/角色 | `is_superuser = True` | L1 Middleware + L4 Service: 自锁保护 |
| 禁用/启用用户 | `is_superuser = True` | L1 Middleware + L4 Service: 自锁保护 |
| 调用 taskAuth 内部 API | `X-TaskAuth-Internal-Secret` | taskAuth handler: `requireInternalSecret()` |

### 自锁保护规则

在 `system_admin_users.py` 的 `update_user` 和 `delete_user` 函数中需添加：

```python
def _prevent_self_lockout(request, target_user_id):
    """防止 superuser 自我降级或禁用。"""
    if str(request.user.id) == str(target_user_id):
        # 禁止修改自己的 is_superuser 或 is_active
        ...
```

具体规则：
- **禁止将自己降级为非 superuser**：若请求体含 `is_superuser: false` 且目标 ID = 当前用户 ID，返回 403
- **禁止禁用自己**：若请求体含 `is_active: false` 且目标 ID = 当前用户 ID，返回 403

---

## 3. 安全审查结论

### 审计清单

| 检查项 | 结论 | 说明 |
|--------|------|------|
| **IDOR 风险** | ✅ 无风险 | `PUT/DELETE /api/system-admin/users/<id>/` 中 `<id>` 是系统级资源，superuser 有权访问任何用户 |
| **权限提升** | ✅ 已覆盖 | 所有端点由 `SystemAdminPermissionMiddleware` 统一拦截，返回 401/403 |
| **跨租户泄露** | ✅ 无风险 | 用户列表为系统级视图，不限定 tenant；admin 有权查看全部用户。用户手机号后端脱敏 |
| **403 vs 404** | ✅ 合理 | 对所有 `/api/system-admin/*` 路径，未认证返回 401，非 superuser 返回 403（由中间件统一处理） |
| **user_id 注入** | ⚠️ 低风险 | taskAuth `PATCH /api/internal/users/{id}/` 接受 Django 传入的 user_id。Django 侧中间件已确保仅 superuser 可达，taskAuth 侧 secret 验证确保仅 Django 可调用。双层防护充分 |
| **敏感操作审计** | ⚠️ 建议增强 | 用户创建/更新/禁用操作建议添加 `logger.info` 记录操作者 + 目标 + 变更内容。当前设计中未明确提及审计日志 |
| **taskAuth secret 泄露** | ✅ 安全 | `X-TaskAuth-Internal-Secret` 仅在 Django settings 与 taskAuth 配置中，不对外暴露 |
| **手机号泄露** | ✅ 已设计 | 后端脱敏（`138****1234`），前端仅展示脱敏后的手机号 |

### 风险评估

| 风险 | 等级 | 缓解措施 |
|------|------|----------|
| superuser 自我锁出 | 🟡 中 | 在 `update_user`/`delete_user` 中实现自锁保护 |
| 缺乏审计日志 | 🟡 低 | `logger.info` 记录操作，后续接入 domain-event 审计 |
| taskAuth PATCH 端点被滥用 | 🟢 低 | 双层防护：Django middleware + taskAuth secret |
| 手机号原始值在 taskAuth 响应中传输 | 🟢 低 | taskAuth→Django 是内网通信；Django 侧脱敏后再返回前端 |

---

## 4. 测试用例清单

### 需新增的权限测试

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 未认证用户访问 dashboard | anonymous | `GET /api/system-admin/dashboard/` | 401 |
| 普通用户访问 dashboard | normal_user | `GET /api/system-admin/dashboard/` | 403 |
| 公司管理员访问 dashboard | tenant_admin | `GET /api/system-admin/dashboard/` | 403 |
| 超级管理员访问 dashboard | superuser | `GET /api/system-admin/dashboard/` | 200 |
| 超级管理员查看用户列表 | superuser | `GET /api/system-admin/users/` | 200 |
| 普通用户查看用户列表 | normal_user | `GET /api/system-admin/users/` | 403 |
| 超级管理员创建用户 | superuser | `POST /api/system-admin/users/` | 201 |
| 超级管理员更新用户 | superuser | `PUT /api/system-admin/users/<id>/` | 200 |
| 超级管理员自我降级 | superuser | `PUT /api/system-admin/users/<self_id>/` (is_superuser=false) | 403 |
| 超级管理员自我禁用 | superuser | `PUT /api/system-admin/users/<self_id>/` (is_active=false) | 403 |
| 超级管理员禁用其他用户 | superuser | `DELETE /api/system-admin/users/<other_id>/` | 200 (is_active=false) |
| taskAuth 内部端点无 secret | — | `GET /api/internal/users/` (no secret) | 403 |
| taskAuth 内部端点有效 secret | — | `GET /api/internal/users/` (with secret) | 200 |

### 需新增的集成测试

| 测试场景 | 文件 | 验证点 |
|----------|------|--------|
| 创建用户完整链路 | `cloudSystemAdmin/tests/test_system_admin_users.py` | taskAuth 注册 → Django 回调 → 用户出现在列表中 |
| 用户列表分页 | `cloudSystemAdmin/tests/test_system_admin_users.py` | limit/offset 分页正确 |
| 手机号脱敏 | `cloudSystemAdmin/tests/test_system_admin_users.py` | 返回数据中 phone 字段已脱敏 |
| 自锁保护 | `cloudSystemAdmin/tests/test_system_admin_users.py` | 修改自己 is_superuser 返回 403 |
| dashboard 数据聚合 | `cloudSystemAdmin/tests/test_system_admin_dashboard.py` | 各统计字段正确 |

---

## 5. 权限实现建议

### taskAuth `PATCH /api/internal/users/{id}/` 的安全设计

```go
// 仅允许修改白名单字段
var allowedFields = map[string]bool{
    "is_active":    true,
    "is_superuser": true,
    "is_staff":     true,
}

func handlePatchUser(w http.ResponseWriter, r *http.Request) {
    // 1. 验证 internal secret
    if !requireInternalSecret(r) { ... }
    
    // 2. 仅允许白名单字段
    for key := range body {
        if !allowedFields[key] {
            delete(body, key)
        }
    }
    
    // 3. 执行更新
    ...
}
```

### Django 自锁保护

```python
def _guard_self_lockout(request, target_user_id, data):
    """禁止 superuser 自我降级或禁用。"""
    if str(request.user.id) != str(target_user_id):
        return  # 不是自己，放行
    
    if 'is_superuser' in data and not data['is_superuser']:
        raise PermissionDenied('不能取消自己的超级管理员权限')
    if 'is_active' in data and not data['is_active']:
        raise PermissionDenied('不能禁用自己的账号')
```

---

## 判定结论

**绿灯 ✅** — 设计可继续。

现有 `SystemAdminPermissionMiddleware` 已为所有 `/api/system-admin/*` 端点提供了统一的 `is_superuser` 检查。taskAuth 内部端点通过 `X-TaskAuth-Internal-Secret` 保护。设计在权限维度充分，仅需在实现时添加自锁保护和审计日志记录。

总结清单：
- 权限门控: 方案1 — 复用现有 SystemAdminPermissionMiddleware（推荐）、方案2 — 每个视图独立添加 @permission_classes
- 自锁保护: 方案1 — 在 Django 视图函数中检查（推荐）、方案2 — 在 taskAuth PATCH handler 中检查
- 审计日志: 方案1 — logger.info 记录（当前阶段推荐）、方案2 — 发送 domain event 到 Kafka（后续增强）
- taskAuth PATCH 字段白名单: 方案1 — taskAuth handler 侧白名单过滤（推荐）、方案2 — Django 侧预过滤
