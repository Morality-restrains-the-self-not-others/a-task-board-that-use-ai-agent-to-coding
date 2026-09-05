---
name: 2-role-permission
description: "Step 2 - 角色权限分析：对 /1-brainstorming 产出的设计文档进行角色和权限维度的审核、补充与增强，确保每个改动点都有明确的权限边界。"
---

# /2-role-permission — 角色权限分析

对 `/1-brainstorming` 产出的设计文档进行角色权限维度的审核与增强。确保每个新增/修改的端点、服务、数据访问路径都有明确的权限边界。

---

## 输入

上一步产出的设计文档（`design-*.md`），通常位于 `.claude/skills/1-brainstorming*/` 目录下。

---

## 执行步骤

### Step 1: 读取设计文档

找到最新的设计文档，通读全部章节。重点关注：

- **改动文件清单** — 每个涉及的文件都是权限分析的目标
- **设计方案** — 新增的端点、服务函数、数据查询是权限检查的候选点
- **领域概念清单** — 新实体/聚合需要定义访问边界

### Step 2: 权限影响分析

对设计文档中的每个改动点，逐项分析：

#### 检查维度

| 维度 | 分析问题 |
|------|----------|
| **主体 (Who)** | 谁能执行这个操作？superuser / tenant_admin / workspace_admin / editor / viewer / owner？ |
| **资源 (What)** | 操作的对象属于哪个层级？System / Tenant / Workspace / Resource？ |
| **操作 (Action)** | 读/写/删/管理？是否需要区分？ |
| **范围 (Scope)** | 是否跨租户？是否限定工作空间？是否仅自己的资源？ |
| **条件 (Condition)** | 是否有状态约束？如：仅草稿可删、仅激活用户可操作 |
| **缺失风险** | 当前设计中是否遗漏了权限检查？是否存在 IDOR 风险？ |

#### 输出表格

对设计文档中每个 API 端点/服务函数/数据访问，在分析报告中输出：

```
| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST /api/xxx | tenant_admin | Workspace | write | IsAuthenticated | ⚠️ 缺失 | 添加 IsTenantAdmin + workspace 归属校验 |
| GET /api/yyy | workspace_member | Workspace | read | has_workspace_access | ✅ 充分 | — |
```

### Step 3: 角色和权限建模（仅当设计引入新角色/权限时）

若设计涉及以下情况，需进行权限建模：

- 新增角色类型（如 `reviewer`、`auditor`、`billing_manager`）
- 新增权限粒度（如从 workspace 级细化到 task 级）
- 新增跨资源访问模式（如 user A 可查看 user B 的任务）

#### 建模输出

1. **角色定义**：
   ```
   role: <角色名>
   display_name: <展示名>
   description: <用途>
   permissions: [<资源>:<操作>, ...]
   scope: tenant | workspace | resource
   ```

2. **角色层级**：新角色在现有层级中的位置
   ```
   superuser
     └─ tenant_admin
          └─ <新角色>
               └─ workspace_editor
   ```

3. **权限检查路径**：新增权限在各层的执行方式
   - L1 Middleware: 是否需要全局拦截？
   - L2 Permission Class: 是否需要新增 DRF Permission？
   - L3 Decorator: 是否需要新增 `@require_*` 装饰器？
   - L4 Service Guard: 是否需要新增 `_can_*()` 检查函数？

### Step 4: 安全审计检查

对设计文档中的改动做快速安全审计：

- [ ] **IDOR 风险**: 所有含 `/<resource>_id/` 的 URL 是否校验了资源归属？
- [ ] **权限提升**: 所有 `PATCH`/`PUT` 端点是否校验了操作者权限？
- [ ] **跨租户泄露**: 所有查询是否限定了 `tenant_id` / `company_id`？
- [ ] **403 vs 404**: 无权限用户请求不存在资源时是否返回 403（而非 404）防止资源探测？
- [ ] **user_id 注入**: 是否存在允许调用方指定他人 `user_id` 的接口？
- [ ] **敏感操作**: 删除、转账、权限变更等操作是否有额外的确认或审计日志？

### Step 5: 测试用例补充

输出需要新增的权限测试用例清单：

```
| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| 管理员可以<操作> | tenant_admin | POST /api/xxx | 200 |
| 普通成员不可<操作> | workspace_editor | POST /api/xxx | 403 |
| 跨租户不可<操作> | other_tenant_user | POST /api/xxx | 403 |
| 已禁用用户不可<操作> | disabled_user | POST /api/xxx | 403 |
```

---

## 输出

### 产出物

在 `.claude/skills/2-role-permission/` 目录下创建 `permission-analysis-<date>.md`，包含：

1. **权限影响矩阵** — 每个改动点的权限分析表（Step 2 输出）
2. **新增角色/权限建模** — 角色定义 + 层级 + 检查路径（Step 3 输出，如适用）
3. **安全检查结论** — 审计清单的逐项结论（Step 4 输出）
4. **测试用例清单** — 需新增的权限测试（Step 5 输出）
5. **风险评级** — 高/中/低风险项及缓解建议

### 对设计文档的回写

根据分析结果，在原设计文档中**追加**以下章节（如设计文档中尚不存在）：

```markdown
## 7. 权限影响分析

(插入 Step 2 的权限影响矩阵)

## 8. 角色与权限建模

(插入 Step 3 的角色定义和检查路径，如适用)

## 9. 安全审查结论

(插入 Step 4 的审计清单结论)
```

若权限分析发现**缺失权限检查**，需在改动文件清单中追加对应的权限相关改动。

---

## 判定规则

### 红灯 🚨（设计文档需返工）
- 存在明确的 IDOR 漏洞且设计未覆盖
- 敏感操作（删除/转账/权限变更）无任何权限检查
- 跨租户数据泄露风险

### 黄灯 ⚠️（设计可继续，但需在实施前解决）
- 权限检查依赖隐式假设（如「只有管理员会调用这个接口」）
- 新增端点仅有 `IsAuthenticated` 而无资源级权限检查
- 未定义新角色的权限边界

### 绿灯 ✅（设计充分）
- 所有改动点有明确的权限检查说明
- 角色权限边界清晰
- 安全审计检查全部通过

---

## 与后续步骤衔接

```
/1-brainstorming (产出设计文档)
        ↓
/2-role-permission (本技能 — 审核增强设计文档)
        ↓
/4-value-stream → /5-nfr → /6-ddd → /7-plans (将增强后的设计转化为实施计划，包含权限检查实现任务)
        ↓
/8-build (BDD 测例先行，包含权限测试)
        ↓
/9-review (代码审查时复查权限实现)
        ↓
/10-ship
```

---

## 现有权衡参考

分析时可参考项目现有多层权限体系：

| 层级 | 模型 | 角色/标记 | 典型检查方式 |
|------|------|-----------|-------------|
| System | `User.is_superuser` / 平台角色 | Boolean / role | `RequirePlatformPerm` / `IsPlatformStaff` |
| Tenant（逻辑资源组 v72） | page ⊃ ui_region | `region:*` / `page:*` | `RequireRegion` / `HasRegion` / `HasPage` — **租户控制台默认**；见 `.ai/01_project_constraints/45_tenant_logical_rbac_resource_groups.md` |
| Tenant（粗码过渡） | `auth_permission` 码 | e.g. `member:manage` | `RequirePerm` — 存量可并存，新能力勿仅用此路径 |
| Tenant（业务数据范围） | `tenant_resource_group_assignment` | group-res | `HasGroupResourceAccess` — **≠** 页面 ACL |
| Workspace | `WorkspaceAccess.role` | view/edit/admin | `has_workspace_access()` / `_is_workspace_admin()` |
| Resource | `IsTodoMutationOwner` | 属主 | `BasePermission.has_object_permission()` |
| Budget | `TenantBudgetPermission` | Boolean | `user_can_modify_task_budget()` |

**租户相关设计强制**：触及租户页面/API 时，Step 2 矩阵必须回答「对应 page/region/api 成员是否登记、BE 是否 RequireRegion」。

---

## 输出格式要求

涉及多项 × 多方向可选方案时，在报告末尾附上总结清单：

```
总结清单：
- 项A: 方案1（简述）、方案2（简述）
- 项B: 方案1（简述）、方案2（简述）、方案3（简述）
```
