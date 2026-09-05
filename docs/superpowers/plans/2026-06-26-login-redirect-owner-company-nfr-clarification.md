# NFR 澄清: Login Redirect to Owner Company

> 输入:
> - 设计文档: `/debug` session analysis
> - 价值流文档: `docs/superpowers/plans/2026-06-26-login-redirect-owner-company-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L0 | 不适用 — 无新数据路径 |
| 安全性 | L0 | 不适用 — 使用现有认证/授权 |
| 数据一致性 | L0 | 不适用 — 仅改变查询条件 |

## 跳过声明

**所有 NFR 类别均跳过。**

理由:
- 此修复仅修改 `_login_redirect_url` 函数内部查询逻辑：从 `CompanyMember.objects.filter(user_id=).first()` 改为 `Company.objects.filter(creator_id=).first()`
- 无新增数据库字段 — `creator_id` 是 `accounts_company` 表的已有字段
- 无新增外部依赖或 HTTP 调用
- 无新增数据流 — 查询路径仍是 Django ORM → SQLite
- 响应时间、吞吐量等性能指标不变
- 认证/授权逻辑不变

## 权衡与边界

**明确不做什么:**
- 不引入新的 CompanyOwner 模型或字段
- 不改变前端路由架构
- 不增加额外的权限检查

**升级触发条件:**
- 如果未来需要支持多公司 owner（co-owner），届时需要 NFR 重新评估
