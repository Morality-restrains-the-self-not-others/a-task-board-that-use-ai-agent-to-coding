# NFR 澄清: 人员管理 — 公司切换器 + 无权限优雅降级

> 输入: `docs/specs/people-manage-company-switcher-graceful-design.md`, `2026-06-28-people-manage-company-switcher-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | 权限判断仍在后端，has_permission 标记不可伪造 |
| 可用性 | L2 | 无权限时展示友好提示，无 alert 弹窗阻断 |
| 性能 | L1 | company_nicknames 增加 2 字段（内存布尔判断） |
| 可观测性 | L1 | 无新增日志需求（沿用上一轮权限拒绝日志） |
| 可维护性 | L2 | API 语义从"拒绝"改为"标记"，扩展性更好 |

## 质量场景

### QS-01: 无权限用户看到友好提示
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 普通成员 |
| 刺激 | 访问 people/manage |
| 制品 | MemberList.fetchMembers |
| 环境 | 正常 |
| 响应 | 页面显示 "🔒 您没有该公司的人员管理权限" |
| 响应度量 | 不出现 alert 弹窗，DOM 包含 permission message |

### QS-02: has_permission 不可伪造
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意客户端 |
| 刺激 | 伪造 has_permission=true 的请求 |
| 制品 | company_members API |
| 环境 | 正常 |
| 响应 | 后端始终根据 DB 中的 is_admin/creator_id 判定，忽略客户端传参 |
| 响应度量 | 单元测试: 传入 has_permission=true 参数不影响后端判定 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 无新增领域概念 | 无需新建实体/VO | 复用已有 CompanyAdminPolicy |

## 跳过声明
- **数据一致性/容错/合规/可伸缩性**: 不适用，无新增数据路径。
