# NFR 澄清: 人员管理权限隔离

> 输入:
> - 设计文档: `docs/specs/people-manage-permission-isolation-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-28-people-manage-permission-isolation-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | 仅 admin/creator 可执行管理操作，普通成员 403 |
| 性能 | L1 | 无额外性能要求（仅增加内存中 is_admin 布尔判断） |
| 数据一致性 | L0 | 不适用（无新增数据读写路径） |
| 可用性 | L0 | 不适用（不改变部署拓扑或容错机制） |
| 可观测性 | L2 | 越权拒绝记录日志，便于审计 |
| 可维护性 | L1 | 统一的 `_require_company_admin` 方法，复用不重复 |

## 逐增量 NFR 分析

### Increment 1: 后端 API 权限门禁 (Thin Slice)

#### NFR 类别: 安全性
- **等级**: L3 - 增强（auth 域自动升级）
- **量化目标**: 5 个管理 action 全部实施权限门禁；普通成员访问 100% 返回 403
- **质量场景**: QS-01, QS-02

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 权限检查增量延迟 < 5ms（仅一次 DB 查询 + 内存比较）
- **质量场景**: QS-03

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: 每次越权拒绝打印 WARNING 级别日志，含 user_id + company_id + action
- **质量场景**: QS-04

### Increment 2: 前端 Sidebar 角色感知 (Enhancement)
- NFR 同 Increment 1，前端仅 UX 辅助，安全依赖仍在后端

## 质量场景

### QS-01: 普通成员查看人员列表被拒
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 普通成员（is_admin=False, 非 creator） |
| 刺激 | GET /api/tenant/{id}/accounts/members/company_members/ |
| 制品 | CompanyMemberViewSet.company_members action |
| 环境 | 正常负载 |
| 响应 | 403 Forbidden + `{"error": "仅公司管理员可执行此操作"}` |
| 响应度量 | 单元测试断言 `resp.status_code == 403` |

### QS-02: 普通成员邀请成员被拒
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 普通成员 |
| 刺激 | POST /api/tenant/{id}/accounts/members/invite/ |
| 制品 | CompanyMemberViewSet.invite action |
| 环境 | 正常负载 |
| 响应 | 403 Forbidden |
| 响应度量 | 单元测试断言 403；admin 仍可正常邀请（201） |

### QS-03: 权限检查不引入显著延迟
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | 任意已认证用户 |
| 刺激 | 调用 company_members API |
| 制品 | `_require_company_admin` 方法 |
| 环境 | 正常负载 |
| 响应 | 权限检查在 5ms 内完成 |
| 响应度量 | 仅做 Company.objects.get + CompanyMember.objects.filter，均为索引查询 |

### QS-04: 越权操作可审计
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 普通成员 |
| 刺激 | 尝试调用任意管理 action |
| 制品 | `_require_company_admin` 方法 |
| 环境 | 正常 |
| 响应 | WARNING 日志: `user_id={uid} company_id={cid} action={action} → 403 (not admin)` |
| 响应度量 | `grep` 日志文件可检索到越权记录 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L3 + admin/creator 权限检查 | 领域服务需注入 actor 参数 | CompanyAdminPolicy 领域服务接收 actor user_id |
| 可观测性 L2 + 越权审计 | 越权拒绝是领域事件候选 | 按需可扩展为 `MemberPermissionDenied` 领域事件（V1 仅 logger.warning） |
| 性能 L1 + 内存布尔判断 | 无需 CQRS 或额外缓存 | 保持简单：DB 查询 + is_admin 布尔判断 |

## 权衡与边界

### 取舍
- 选择在 ViewSet 层做权限检查（非 ORM/DB 层），保持代码简单，一行 `_require_company_admin` 调用即可
- 不做细粒度 RBAC（如按操作分类授权），当前 creator/admin/member 三级已满足需求

### 明确不做什么
- 不引入新的角色表或权限模型（如 RBAC 的 permission 表）
- 不修改数据库 schema
- 不修改 API 响应结构（仅增加 403 错误路径）
- 不在前端路由做权限守卫（仅 Sidebar 显隐作为 UX 辅助）
- 不引入中间件级权限检查（如在 DRF permission_classes 中实现）

### 升级触发条件
- 当角色类型超过 5 种且权限矩阵复杂时 → 引入 RBAC 权限模型
- 当需要按操作类型（读/写/删除）精细化授权时 → 引入 permission 表
- 当需要跨服务权限检查时 → 升级为网关层 forward-auth 注入角色信息

## 跳过声明
- **可伸缩性**: 跳过。权限检查仅增加 1 次 DB 查询，不改变系统扩展特性。
- **可用性**: 跳过。不改变部署拓扑或容错机制。
- **数据一致性**: 跳过。无新增数据写入路径，仅读取已有 is_admin/creator_id 字段。
- **合规与隐私**: 跳过。不涉及用户数据处理方式的变化。
- **容错机制**: 跳过。权限检查失败即返回 403，无重试/降级场景。
