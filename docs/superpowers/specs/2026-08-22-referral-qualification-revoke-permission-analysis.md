# 角色权限分析：推荐分账资格取消与操作审计

> Derived from: `docs/superpowers/specs/2026-08-22-referral-qualification-revoke-design.md`

## 结论

不引入新角色。所有新增/收紧的管理员写接口沿用现有 **superuser** 门闩（`requireSuperuser`）。taskBill 内部接口仅允许带 `X-TaskBill-Internal-Secret` 的服务间调用。

## 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/referral/applications/` | superuser | System | read | `requireSuperuser` | ✅ 充分 | 列表新增字段不改变权限 |
| POST `.../applications/{id}/approve/` | superuser | System | write | `requireSuperuser` | ✅ 充分 | 强制 reason；禁止 IDOR（按 path id 取行，无租户越权面：资格是平台级） |
| POST `.../applications/{id}/reject/` | superuser | System | write | `requireSuperuser` | ✅ 充分 | 同上 |
| POST `.../applications/{id}/revoke/` | superuser | System | write | `requireSuperuser`（新建） | ✅ 设计已覆盖 | 仅活跃 approved 可撤 |
| GET `.../applications/{id}/audit/` | superuser | System | read | `requireSuperuser`（新建） | ✅ 设计已覆盖 | 不向被推荐用户暴露 |
| POST `/api/internal/taskbill/referral/disable-eligibility/` | taskReferral（internal secret） | System | write | `requireInternalSecret` | ✅ 设计已覆盖 | 禁止公网暴露 |
| 前端 `/system-admin/users/?tab=referral-apps` | superuser SPA | System | UI | 既有系统管理路由守卫 | ✅ 充分 | 按钮不替代服务端鉴权 |

## 数据访问

| 表 | Owner | 谁写 | 谁读 | 禁止 |
|----|-------|------|------|------|
| `referral_code` | taskReferral | 超管审批/取消、用户申请、过期 timer | 超管列表、用户 status | 其他服务直连 |
| `referral_qualification_audit` | taskReferral | 仅审批命令成功路径 | 超管 GET audit | 用户端、taskBill |
| `billing_referral_edge` / accrual | taskBill | disable-eligibility 内部 API | 计提/分账 | taskReferral 直连账单库 |

## IDOR / 越权

- 申请 ID 为自增 BIGINT，仅超管可枚举；非超管 403。
- 内部 disable-eligibility 不接受浏览器 Cookie，仅 secret。
- 审计内容含操作理由，可能含内部说明：仅超管可见，日志禁止 PII 堆砌（理由原文入库但不打完整 intro）。

## 不建模新角色

```
superuser
  └─ （本增量无新角色）
```
