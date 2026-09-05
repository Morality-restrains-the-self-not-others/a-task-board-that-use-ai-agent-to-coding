# NFR 澄清: 跨公司数据隔离 — tenant_id 作用域修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-28-cross-company-data-isolation-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-cross-company-data-isolation-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L4 | 多租户数据隔离：100% API 请求作用于 URL tenant_id 对应的公司 |
| 数据一致性 | L2 | 租户作用域查询确定性：`.first()` 替换为 `company_id=tenant_id` 过滤 |
| 可维护性 | L2 | 统一工具函数 `resolve_company_member_for_tenant` 覆盖 11 处调用点 |
| 性能 | L1 | 新增 company_id 过滤条件对查询性能无负面影响（已有索引） |
| 可用性 | L1 | 无新增可用性要求（修复不改变 API 契约） |
| 合规与隐私 | L3 | 防止跨租户数据泄露，满足多租户 SaaS 数据隔离基线要求 |

## 逐增量 NFR 分析

### Increment 1: 统一工具函数 + 后端 CRITICAL 视图修复

#### NFR 类别: 安全性
- **等级**: L4 - 极致（安全命脉）
- **量化目标**: 11 处 CRITICAL `.first()` 全部替换为 `resolve_company_member_for_tenant()`，0 处遗漏；所有 workspace/group/member 视图的 CompanyMember 查询 100% 经 tenant_id 过滤
- **质量场景**: QS-01, QS-02

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**: 同一用户的 API 请求在相同 tenant_id 下始终返回同一公司的数据（确定性查询替代非确定性 `.first()`）
- **质量场景**: QS-03

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `resolve_company_member_for_tenant` 单一函数覆盖所有调用点；新视图如需要 tenant 作用域直接复用该函数
- **质量场景**: QS-04

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 新查询路径（增加 `company_id=` 过滤）P95 延迟不超过原查询的 120%
- **质量场景**: QS-05

### Increment 2: 后端 HIGH 风险实例修复

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: 登录/重定向不再随机跳转到错误公司；序列化器 company queryset 确定性排序
- **质量场景**: QS-06

### Increment 3: 前端查询参数清理

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: 公司切换后 0 个跨租户查询参数保留（workspace_id、project_id 等）；WorkPanel 不盲信 URL workspace_id
- **质量场景**: QS-07

### Increment 4: 测试覆盖

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 每个修复的视图有对应的多公司隔离测试；`resolve_company_member_for_tenant` 有独立单元测试
- **质量场景**: QS-08

## 质量场景

### QS-01: 多公司用户 workspace collaborator 隔离
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L4 |
| 场景描述 | 属于 A、B 两公司的用户访问 `/api/tenant/B/.../workspace-collaborators/?workspace_id=A_ws` 时被拒绝 |
| 刺激源 | 已认证用户（同时属于公司 A 和 B） |
| 刺激 | GET 请求带 A 公司的 workspace_id，URL 中 tenant_id 为 B |
| 制品 | `workspace_collaborators` API |
| 环境 | 正常 |
| 响应 | 403 Forbidden `{'error': '无权访问该工作空间'}` |
| 响应度量 | 返回 403（非 200）；不返回公司 A 的任何数据 |

### QS-02: 多公司用户 invite 隔离
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L4 |
| 场景描述 | 属于 A、B 两公司的用户调用 `/api/tenant/A/.../invite/` 时，邀请创建在 A 公司而非 B 公司（即使 `.first()` 刚好返回 B 的 CompanyMember） |
| 刺激源 | 已认证用户 |
| 刺激 | POST 请求发送邀请，URL tenant_id = A |
| 制品 | `CompanyMemberViewSet.invite` |
| 环境 | 正常 |
| 响应 | 201 Created，邀请的 company_id = A |
| 响应度量 | 数据库 `accounts_invitation.company_id` = A 的 tenant_id |

### QS-03: 确定性 company_member 查询
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 场景描述 | 用户属于公司 A（created_at 更早）和 B，对 `/api/tenant/B/...` 发请求，始终使用 B 的 CompanyMember |
| 刺激源 | 已认证用户 |
| 刺激 | 连续 10 次 API 请求 |
| 制品 | `resolve_company_member_for_tenant` |
| 环境 | 正常 |
| 响应 | 10 次请求全部使用 company_id=B 的 CompanyMember |
| 响应度量 | 0 次使用错误公司的 CompanyMember；确定性 100% |

### QS-04: 新视图复用工具函数
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 场景描述 | 新开发者添加需要 tenant 作用域的视图时，使用 `resolve_company_member_for_tenant` 而非手写 `.first()` |
| 刺激源 | 开发者 |
| 刺激 | 新增一个接收 tenant_id 的视图 |
| 制品 | `accounts/workspace_context.py` |
| 环境 | 开发环境 |
| 响应 | 直接调用 `resolve_company_member_for_tenant(user_id, tenant_id)` 即可获得正确结果 |
| 响应度量 | CI 静态检查：禁止在视图层直接调用 `CompanyMember.objects.filter(user_id=...).first()` |

### QS-05: 查询性能无退化
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 场景描述 | 增加 `company_id=` 过滤条件后查询延迟无明显增加 |
| 刺激源 | 正常用户流量 |
| 刺激 | 标准 workspace-collaborators 查询 |
| 制品 | 修改后的视图 |
| 环境 | 正常负载 |
| 响应 | P95 延迟 ≤ 原查询的 120% |
| 响应度量 | 与修复前的基准对比 |

### QS-06: 登录重定向确定性
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 场景描述 | 多公司用户登录后被重定向到确定性公司（而非随机） |
| 刺激源 | 用户登录 |
| 刺激 | 成功认证后的重定向 |
| 制品 | `auth_views.login` / `_login_redirect_url` |
| 环境 | 正常 |
| 响应 | 重定向到有明确排序的第一家公司 |
| 响应度量 | 连续 10 次登录，100% 重定向到同一公司 |

### QS-07: 公司切换清除跨租户参数
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 场景描述 | 用户在 A 公司页面（URL 含 `?workspace_id=A_ws`）切换到 B 公司后，URL 不含 `workspace_id=A_ws` |
| 刺激源 | 用户操作公司下拉框 |
| 刺激 | 选择公司 B |
| 制品 | Navbar `switchCompany` |
| 环境 | 正常 |
| 响应 | 页面导航到 `/tenant/B/work-panel/`（无 `?workspace_id=` 参数） |
| 响应度量 | URL query string 不含 `workspace_id` 键 |

### QS-08: 测试覆盖完整性
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 场景描述 | 所有修复点有对应的测试用例 |
| 刺激源 | CI 流水线 |
| 刺激 | 运行完整测试套件 |
| 制品 | 测试套件 |
| 环境 | CI |
| 响应 | 所有新增测试通过；多公司隔离断言全部 PASS |
| 响应度量 | 新增测试文件 ≥ 1（`test_workspace_context.py`）；3 个现有测试文件增加多公司用例 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L4: 多租户隔离 | 所有聚合根查询必须作用域到 Company（tenant）边界 | 仓储接口增加 `company_id` 参数；Company 是聚合边界锚点 |
| 安全性 L4: URL tenant_id 为真源 | tenant_id 不是可选提示而是强制约束 | 视图层和领域层之间通过 `resolve_company_member_for_tenant` 统一桥梁 |
| 数据一致性 L2: 确定性查询 | `.first()` 无排序禁止在租户上下文中使用 | 工具函数强制 `company_id=` 过滤，禁止无作用域查询 |
| 可维护性 L2: 统一工具函数 | 跨视图的 tenant 作用域逻辑集中到 `workspace_context.py` | DDD 应用服务层调用该工具函数而非直连 ORM |

## 权衡与边界

### 取舍
- **安全优先于性能**: 增加 `company_id=` 过滤条件（~0.1ms 额外开销）换取 100% 数据隔离保证 — 可接受
- **工具函数集中化优先于灵活性**: 所有调用点必须使用 `resolve_company_member_for_tenant`，不接受裸 `.first()` — 统一控制面
- **前端清除参数优先于深度链接**: 公司切换时主动清除跨租户查询参数，接受丢失「深层链接到特定 workspace」的能力 — 安全优先

### 明确不做什么
- 不在本修复中解决所有 `.first()` 无排序实例（仅 CRITICAL + HIGH，MEDIUM 延后到 debt cleanup）
- 不更改 CompanyMember 模型或数据库schema（纯查询层修复）
- 不引入新的中间件或装饰器（保持函数式，不增加框架复杂度）
- 不在前端引入 Pinia/Vuex 状态管理（保持现有 ref/computed 模式）

### 升级触发条件
- 当出现新的 CRITICAL `.first()` 实例 → CI 静态检查捕获 → 自动阻止合入
- 当用户公司数超过 1000 → `company_id` 索引需重新评估（当前 < 100 个公司无问题）
- 当需要支持跨公司资源共享场景 → 需要显式的「共享」模型替代当前「严格隔离」

## 跳过声明
- **可伸缩性**: 跳过。修复不改变系统架构或数据量级，水平扩展策略不变。
- **可用性**: 跳过。修复不引入新的故障模式或外部依赖。
- **可观测性**: 跳过。现有日志/追踪覆盖足够，修复不新增观测需求。
- **容错机制**: 跳过。修复不涉及网络调用、重试或超时策略。
