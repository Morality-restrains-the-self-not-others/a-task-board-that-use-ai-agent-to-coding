# 角色权限分析：功能参数多层级配置

**分析日期**: 2026-06-30  
**来源设计**: `docs/designs/feature-params-hierarchy.md`  
**分析范围**: 全部 13 个 API 端点 + 1 个领域服务 + 4 张新表

---

## 1. 权限影响矩阵

### 1.1 公司级（现有，不变）

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|----------|------|
| E1 | `GET /api/tenant/{tenant}/feature-params/` | 公司成员 | Tenant | read | `IsAuthenticated` + `user_companies` 成员校验 | ✅ 充分 | — |
| E2 | `POST /api/tenant/{tenant}/feature-params/` | 租户管理员 | Tenant | write | `IsAuthenticated` + `CompanyMember.is_admin` 检查 | ✅ 充分 | 当前设计仅在响应中返回 `is_tenant_admin` 标记，建议增加服务端拒绝（见风险） |

> ⚠️ **现有问题发现**: 当前 `manage_feature_params` POST 中 `is_admin` 仅用于 GET 响应标记，POST 保存时未做 `is_admin` 强制校验。任何公司成员理论上可以 POST 修改公司配置。设计需修复此问题。

### 1.2 工作空间级（新增）

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|----------|------|
| E3 | `GET /api/tenant/{tenant}/workspace/{workspace}/feature-params/` | 工作空间成员 | Workspace | read | `IsAuthenticated` | ⚠️ 缺失 | 添加 `has_workspace_access(user, workspace_id)` + company 归属校验 |
| E4 | `POST /api/tenant/{tenant}/workspace/{workspace}/feature-params/` | 工作空间管理员 | Workspace | write | `IsAuthenticated` | ⚠️ 缺失 | 添加 `WorkspaceAccess.role='admin'` 校验（或 `CompanyMember.is_admin` 作为租户管理员回退） |
| E5 | 设置 `Workspace.allow_personal_feature_params`（工作空间设置 PATCH） | 工作空间管理员 / 租户管理员 | Workspace | write | 取决于实现路径 | ⚠️ 缺失 | 若合并入 E4 POST 则复用；若独立端点则同样需要 admin 校验 |

### 1.3 个人级（新增）

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|----------|------|
| E6 | `GET /api/personal/feature-params-configs/` | 登录用户 | User | read (list own) | `IsAuthenticated` | ⚠️ 缺失 | 按 `user_id=request.user.id` 过滤，确保只能看到自己的配置 |
| E7 | `POST /api/personal/feature-params-configs/` | 登录用户 | User | write | `IsAuthenticated` | ⚠️ 缺失 | **强制** `user_id=request.user.id`（拒绝传入的 user_id）+ `company_id` 必须属于当前用户成员的公司 |
| E8 | `GET /api/personal/feature-params-configs/{id}/` | 配置所有者 | User | read | `IsAuthenticated` | ⚠️ 缺失 | 校验 `config.user_id == request.user.id` — **IDOR 风险点** |
| E9 | `PUT /api/personal/feature-params-configs/{id}/` | 配置所有者 | User | write | `IsAuthenticated` | ⚠️ 缺失 | 校验 `config.user_id == request.user.id` — **IDOR 风险点** |
| E10 | `DELETE /api/personal/feature-params-configs/{id}/` | 配置所有者 | User | delete | `IsAuthenticated` | ⚠️ 缺失 | 校验 `config.user_id == request.user.id` — **IDOR 风险点** |

### 1.4 任务绑定（新增/修改）

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|----------|------|
| E11 | `PATCH /api/tasks/{task_id}/feature-params/` | 任务 Owner | Task | write | `IsAuthenticated` + `IsTodoMutationOwner` | ⚠️ 部分缺失 | 除 owner 检查外，选 personal 时须校验：① 工作空间允许个人配置 ② `personal_config_id` 属于当前用户 ③ 配置未删除 |
| E12 | `POST /api/tasks/` 扩展（feature_params_source + personal_config_id） | 工作空间成员 | Task | create | `has_workspace_access` 装饰器 | ⚠️ 部分缺失 | 选 personal 时须额外校验：① 工作空间允许 ② personal_config_id 属于当前用户 ③ personal_config_id 的 `company_id` 与 workspace 的 company 一致 |

### 1.5 容器运行时（内部，修改现有）

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|----------|------|
| E13 | `POST /api/container/{tenant}/{workspace}/{task}/feature-params-env/` | Container | Task | read (env) | `container_access_token` 认证 + URL 路径与 token 归属一致性校验 | ✅ 充分 | 内部逻辑改为 resolver，不改变权限边界；`resolved_env` 含 API key 不对外暴露 |

### 1.6 运行记录查询（新增）

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|----------|------|
| E14 | `GET /api/tasks/{task_id}/feature-params-snapshots/` | 任务可见范围内的成员 | Task | read (脱敏摘要) | `IsAuthenticated` | ⚠️ 缺失 | 校验用户对该 task 所在 workspace 有 view 权限；仅返回 `providers_summary`（脱敏），**禁止**返回 `resolved_env` 含完整 API key |
| E14b | （未来）`GET /api/tasks/{task_id}/feature-params-snapshots/{id}/detail/` | 任务 Owner + 管理员 | Task | read (完整) | — | — | 完整 env 详情接口，需要 `IsTodoMutationOwner` 或 `WorkspaceAccess.role='admin'` |

### 1.7 领域服务内部调用

| # | 改动点 | 调用方 | 被访问资源 | 风险 | 建议 |
|---|--------|--------|-----------|------|------|
| S1 | `FeatureParamsResolver.resolve()` | container views / budget service | `TenantFeatureParams`, `WorkspaceFeatureParams`, `PersonalFeatureParamsConfig`, `Workspace` | 内部调用无直接用户输入 | resolver 不暴露给前端 API；仅在受控的后端路径中调用 |

---

## 2. 安全审计检查

按 OWASP 常见漏洞维度逐项扫描：

- [x] **IDOR 风险** — ⚠️ 发现 3 处高风险点
  - `E8/E9/E10`：`/api/personal/feature-params-configs/{id}/` 若不做 `user_id` 校验，用户 A 可通过遍历 ID 读取/修改/删除用户 B 的配置
  - `E11`：`PATCH /tasks/{task_id}/feature-params/` 的 `personal_feature_params_config_id` 若不做所属校验，用户可绑定他人的配置
  - **缓解**: 以上所有端点必须在服务端强制校验资源归属

- [x] **权限提升** — ⚠️ 发现 1 处现有缺陷
  - `E2`：当前 `manage_feature_params` POST 未强制 `is_admin` 校验，任何公司成员可修改公司级配置
  - **缓解**: 此轮实施中修复：POST 时强制检查 `CompanyMember.is_admin`

- [x] **跨租户泄露** — ⚠️ 发现 2 处需验证
  - `E7`：个人配置创建时 `company_id` 必须限定为用户所属公司，拒绝跨公司注入
  - `E12`：任务创建时的 `personal_feature_params_config_id` 的 company 必须与 workspace 的 company 一致

- [x] **403 vs 404** — ✅ 无新增风险
  - 所有返回 403 的路径已明确（权限不足），资源不存在返回 404。不涉及敏感资源探测场景

- [x] **user_id 注入** — ⚠️ 发现 1 处高风险点
  - `E7`：创建个人配置时必须**忽略**请求体中的 `user_id` 字段，强制使用 `request.user.id`。攻击者可通过传入他人 `user_id` 创建属于他人的配置
  - **缓解**: API 层 `user_id = request.user.id`（硬编码，不从请求体读取）

- [x] **敏感操作** — ✅ 无新增资金/权限变更操作
  - 删除个人配置（E10）为低风险操作，但需确保仅所有者可执行
  - `resolved_env` 包含 API key，快照查询（E14）必须脱敏

---

## 3. 新增角色/权限建模

本次设计**不引入新角色类型**。所有权限基于现有角色体系：

```
superuser (User.is_superuser)
  └─ tenant_admin (CompanyMember.is_admin=True)
       ├─ workspace_admin (WorkspaceAccess.role='admin')
       │    └─ workspace_editor (WorkspaceAccess.role='edit')
       │         └─ workspace_viewer (WorkspaceAccess.role='view')
       └─ personal_config_owner (user_id == config.user_id) ← 新增概念，非新角色
```

### 3.1 新增权限概念

| 概念 | 定义 | 检查方式 |
|------|------|---------|
| `personal_config_owner` | 个人配置的创建者 | `config.user_id == request.user.id` |
| `workspace_personal_config_allowed` | 工作空间是否允许个人配置 | `workspace.allow_personal_feature_params` |
| `snapshot_detail_viewer` | 可查看快照完整 env 的用户 | `IsTodoMutationOwner` 或 `WorkspaceAccess.role='admin'` |

### 3.2 权限检查路径

| 层级 | 新增检查 | 实现方式 |
|------|---------|---------|
| L2 Permission Class | `IsPersonalConfigOwner` | DRF `BasePermission`，对 `{id}` 端点校验 `config.user_id == request.user.id` |
| L4 Service Guard | `_can_use_personal_config(workspace)` | 服务层函数，检查 `workspace.allow_personal_feature_params` |
| L4 Service Guard | `_validate_personal_config_binding(config_id, user_id, company_id)` | 校验 config 存在、属于该用户、属于该公司 |

---

## 4. 测试用例清单

### 4.1 公司级权限测试

| # | 测试场景 | 角色 | 操作 | 预期 |
|---|----------|------|------|------|
| T1 | 租户管理员可修改公司配置 | tenant_admin | POST /api/tenant/{t}/feature-params/ | 200 |
| T2 | 普通成员不可修改公司配置 | workspace_editor | POST /api/tenant/{t}/feature-params/ | 403 |
| T3 | 非公司成员不可查看公司配置 | other_user | GET /api/tenant/{t}/feature-params/ | 403 |

### 4.2 工作空间级权限测试

| # | 测试场景 | 角色 | 操作 | 预期 |
|---|----------|------|------|------|
| T4 | 工作空间管理员可修改工作空间配置 | workspace_admin | POST .../workspace/{w}/feature-params/ | 200 |
| T5 | 工作空间普通成员可查看工作空间配置 | workspace_viewer | GET .../workspace/{w}/feature-params/ | 200 |
| T6 | 工作空间普通成员不可修改工作空间配置 | workspace_viewer | POST .../workspace/{w}/feature-params/ | 403 |
| T7 | 非工作空间成员不可查看工作空间配置 | other_user | GET .../workspace/{w}/feature-params/ | 403 |
| T8 | 租户管理员可修改任意工作空间配置 | tenant_admin | POST .../workspace/{w}/feature-params/ | 200 |
| T9 | 管理员可设置 allow_personal_feature_params | workspace_admin | PATCH workspace | 200 |
| T10 | 普通成员不可设置 allow_personal_feature_params | workspace_viewer | PATCH workspace | 403 |

### 4.3 个人配置权限测试

| # | 测试场景 | 角色 | 操作 | 预期 |
|---|----------|------|------|------|
| T11 | 用户可创建个人配置 | any_member | POST /api/personal/feature-params-configs/ | 201 |
| T12 | 用户可列出自己的个人配置 | any_member | GET /api/personal/feature-params-configs/ | 200 (仅自己的) |
| T13 | 用户可修改自己的配置 | owner | PUT .../configs/{id}/ | 200 |
| T14 | 用户不可修改他人的配置（IDOR） | other_user | PUT .../configs/{id}/ | 403 |
| T15 | 用户不可删除他人的配置（IDOR） | other_user | DELETE .../configs/{id}/ | 403 |
| T16 | 用户不可查看他人的配置（IDOR） | other_user | GET .../configs/{id}/ | 403 |
| T17 | 创建时传入他人 user_id 应被忽略 | attacker | POST .../ {user_id: "victim_id"} | 201 (user_id=自己) |

### 4.4 任务绑定权限测试

| # | 测试场景 | 角色 | 操作 | 预期 |
|---|----------|------|------|------|
| T18 | 任务 Owner 可修改 feature_params_source | task_owner | PATCH /tasks/{id}/feature-params/ | 200 |
| T19 | 非 Owner 不可修改 feature_params_source | other_user | PATCH /tasks/{id}/feature-params/ | 403 |
| T20 | 创建任务时可指定 feature_params_source=company | workspace_editor | POST /tasks/ | 201 |
| T21 | 创建任务时可指定 feature_params_source=workspace | workspace_editor | POST /tasks/ | 201 |
| T22 | 工作空间允许个人配置时可选 personal | workspace_editor (ws allow=true) | POST /tasks/ {source=personal} | 201 |
| T23 | 工作空间禁止个人配置时不可选 personal | workspace_editor (ws allow=false) | POST /tasks/ {source=personal} | 400 |
| T24 | 绑定他人的 personal_config_id 应拒绝 | attacker | PATCH /tasks/{id}/feature-params/ {config_id=other's} | 400 |
| T25 | 绑定已删除的 personal_config_id 应拒绝 | task_owner | PATCH /tasks/{id}/feature-params/ {config_id=deleted} | 400 |

### 4.5 快照查询权限测试

| # | 测试场景 | 角色 | 操作 | 预期 |
|---|----------|------|------|------|
| T26 | 任务所在工作空间成员可查看快照列表（脱敏） | workspace_viewer | GET /tasks/{id}/feature-params-snapshots/ | 200 (providers_summary only) |
| T27 | 非工作空间成员不可查看快照 | other_user | GET /tasks/{id}/feature-params-snapshots/ | 403 |
| T28 | 快照列表不含 resolved_env（API key 脱敏） | any | GET /tasks/{id}/feature-params-snapshots/ | 响应中无 resolved_env 字段 |

---

## 5. 设计文档回写

需要在设计文档 `docs/designs/feature-params-hierarchy.md` 中追加以下内容：

### 5.1 API 设计补充权限列

每个 API 表格增加权限列：

| Method | Path | 权限 |
|--------|------|------|
| GET | /api/tenant/{t}/feature-params/ | IsAuthenticated + 公司成员 |
| POST | /api/tenant/{t}/feature-params/ | IsAuthenticated + **tenant_admin** |
| GET | .../workspace/{w}/feature-params/ | IsAuthenticated + workspace_access(view) |
| POST | .../workspace/{w}/feature-params/ | IsAuthenticated + workspace_access(admin) |
| GET | /api/personal/feature-params-configs/ | IsAuthenticated (filter: user_id=自己) |
| POST | /api/personal/feature-params-configs/ | IsAuthenticated (force: user_id=request.user.id) |
| GET/PUT/DELETE | .../configs/{id}/ | IsAuthenticated + **IsPersonalConfigOwner** |
| PATCH | /tasks/{id}/feature-params/ | IsAuthenticated + IsTodoMutationOwner + validate binding |
| GET | /tasks/{id}/feature-params-snapshots/ | IsAuthenticated + workspace_access(view) |

### 5.2 现有缺陷修复

在实施计划中增加：

> **Increment 0: 修复现有权限缺陷**
> 1. `manage_feature_params` POST 增加 `CompanyMember.is_admin` 强制校验

---

## 6. 风险评级

| 级别 | 数量 | 项目 |
|------|------|------|
| 🔴 高风险 | 4 | E8/E9/E10 IDOR（个人配置 CRUD 无 user_id 校验）、E7 user_id 注入、E2 现有公司配置无 admin 校验 |
| 🟡 中风险 | 3 | E11 任务绑定未校验 config 归属、E12 任务创建跨公司 config 注入、E14 快照 API key 泄露 |
| 🟢 低风险 | 0 | — |

**总体评估**: 🟡 黄灯 — 设计可继续，但高风险项必须在 Increment 3（管理 API）实施前全部解决。

---

## 总结清单

- **公司级现有缺陷**: POST 缺少 admin 校验 — 本次修复
- **工作空间级**: workspace admin 校验 + company 归属 — 新增
- **个人级 IDOR**: 所有 `{id}` 端点必须校验 `user_id` 归属 — 4 处高风险
- **user_id 注入**: 创建个人配置时硬编码 `user_id = request.user.id` — 1 处高风险
- **任务绑定**: personal source 须三重校验（workspace 允许 + config 属于用户 + config 属于同公司）
- **快照脱敏**: 列表接口不返回 `resolved_env`，完整内容需 IsTodoMutationOwner + admin 鉴权
- **测试覆盖**: 28 条权限测试用例，覆盖 IDOR / 跨租户 / 权限提升 / user_id 注入 / 403 vs 404
