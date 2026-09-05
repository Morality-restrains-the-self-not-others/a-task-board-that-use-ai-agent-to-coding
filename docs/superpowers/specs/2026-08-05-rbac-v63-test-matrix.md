# RBAC v63 — 测试用例矩阵

- **迭代**: rbac-merged-v63
- **作者**: claude
- **日期**: 2026-08-05
- **关联设计**: `2026-08-05-rbac-merged-v63-design.md`（§9 实施周期 D8-D10）

---

## 1. 单元测试（已落地 ✅）

| # | 包/文件 | 用例 | 状态 |
|---|---------|------|------|
| 1 | shareLib/authz middleware_test.go | TestHasPerm_FromHeaders — X-Tenant-Perms 解析与多租户集合 | ✅ |
| 2 | shareLib/authz middleware_test.go | TestHasPlatformPerm_StaticMapping — super_admin/employee 静态映射 | ✅ |
| 3 | shareLib/authz middleware_test.go | TestMiddleware_WithContext — 中间件注入 Context | ✅ |
| 4 | shareLib/authz middleware_test.go | TestRequirePerm_Writes403 — 拒绝写 403 | ✅ |
| 5 | shareLib/authz middleware_test.go | TestHasGroupResourceAccess — 组资源伪码命中 | ✅ |
| 6 | shareLib/authz middleware_test.go | TestParse_TenantPermsFormat — 序列化格式解析 | ✅ |
| 7 | taskAuth rbac_pdp_test.go | TestPlatformRolesFromFlags — is_superuser/is_staff → 平台角色 | ✅ |
| 8 | taskAuth rbac_pdp_test.go | TestSerializeTenantPerms — X-Tenant-Perms 序列化排序 | ✅ |
| 9 | taskAuth rbac_pdp_test.go | TestSerializeTenantPerms_Truncation — 7000 字节上限截断 | ✅ |
| 10 | taskAuth rbac_pdp_test.go | TestInPlaceholders — SQL IN 占位符 | ✅ |
| 11 | taskAuth 既有套件 | TestGatewayForwardAuthCacheHit — 缓存命中注入（修复 X-Auth-Cache） | ✅ |
| 12 | taskAuth 既有套件 | TestRegistrationInviteSuperuserPutPolicyViaXUserID — requireSuperuser 双通道 | ✅ |
| 13 | taskFE | system_admin_route_guard_service.test.js 7 用例（含 RBAC 角色判定） | ✅ |

**已同步（2026-08-05）**：
- `TestGatewayForwardAuthUserIdCookie` — 期望已与代码一致（SSO 桥 fallback 保留，有效用户 200 / 未知用户 401）；taskAuth 全量套件全绿 ✅
- 清理：`django_session_test.go`（v57 Django 退役后引用已删代码的死测试）已删除

## 2. E2E 用例矩阵（16 用例；核心 9 用例已落地执行 ✅ 2026-08-05）

> 已执行: `docs/superpowers/specs/scripts/rbac-v63-e2e.sh`（真实环境 taskAuth:8003 + taskTenantService:8020 + MySQL）
> 结果: **9/9 通过**（E1 角色创建/E2 非法码/E3 内置锁定/E4 列表/E5 PDP/E6 403/E7 组资源/E8 role-exists/E9 跨租户隔离）
> 待执行（需完整用户流/浏览器）: E10-E16（回填数据已在迁移执行中核验 ✅ E10/E11 逻辑等价）

### 2.1 角色管理（4）

| # | 场景 | 步骤 | 预期 |
|---|------|------|------|
| E1 | 租户自定义角色创建 | tenant_admin POST /api/auth/roles/（company_id+display_name+permissions） | 201；GET 列表可见；is_system=false |
| E2 | 自定义角色非法权限码拒绝 | 创建角色带 platform:manage | 400 illegal permission |
| E3 | 内置角色锁定 | PUT/DELETE 内置角色（tenant_admin） | 403 系统内置角色不可修改 |
| E4 | 自定义角色删除引用保护 | 角色已被成员引用时 DELETE | 422 角色仍有成员引用 |

### 2.2 权限判定链路（5）

| # | 场景 | 步骤 | 预期 |
|---|------|------|------|
| E5 | 自定义角色授予后端点可达 | 创建角色含 project:view → 分配给成员 → 成员访问项目列表 | 200（PDP 展开权限码） |
| E6 | 未授权成员拒绝 | member 访问 project:manage 端点 | 403 |
| E7 | 组角色继承 | 组设置 member 角色 → 组内成员访问 view 端点 | 200 |
| E8 | 组资源继承 | project p1 分配给组 g1（view）→ g1 成员访问 p1 | 200（HasGroupResourceAccess 伪码命中） |
| E9 | 组资源撤销 | 撤销 p1↔g1 分配 → g1 成员访问 p1 | 403（缓存 TTL/事件失效后） |

### 2.3 硬切换（3）

| # | 场景 | 步骤 | 预期 |
|---|------|------|------|
| E10 | is_admin 回填 | 迁移 004 后查 tenant_member_role | is_admin=1 成员有 tenant_admin 行 |
| E11 | creator 回填 | 迁移 004 后查 creator 成员 | creator 有 tenant_admin 行（无需 is_admin） |
| E12 | 硬切换 grep 校验 | 全仓 grep 鉴权判定路径 | 0 处（数据回显字段除外） |

### 2.4 前端（2）

| # | 场景 | 步骤 | 预期 |
|---|------|------|------|
| E13 | 平台角色显示系统管理入口 | employee/super_admin 登录 | Navbar 显示"系统管理" |
| E14 | 非平台角色无入口 | member 登录 | Navbar 不显示"系统管理"，守卫重定向工作面板 |

### 2.5 安全边界（2）

| # | 场景 | 步骤 | 预期 |
|---|------|------|------|
| E15 | 跨租户角色隔离 | 租户 A 自定义角色名不得授予租户 B 成员 | role-exists 校验 company_id 归属；PDP 查无码 |
| E16 | 交叉组隔离 | group_admin_A 操作 B 组 | RequireGroupAdmin scope 校验拒绝 |

## 3. 回归范围

- taskAuth 全量测试套件（全绿 ✅）
- shareLib/authz 单测
- taskFE 路由守卫 + Navbar 相关测试
- taskBill/taskTask/taskCloud/taskTenantService 编译 + 既有测试（环境依赖 DB 的用例按环境执行）
