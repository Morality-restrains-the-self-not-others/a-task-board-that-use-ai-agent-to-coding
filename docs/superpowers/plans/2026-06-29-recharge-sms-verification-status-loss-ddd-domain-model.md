# 领域模型: 充值 SMS 验证状态持久化修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-29-recharge-sms-verification-status-loss-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-29-recharge-sms-verification-status-loss-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-29-recharge-sms-verification-status-loss-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 关键发现: 无新增领域概念

本修复为纯 Bug Fix，不引入新的实体、值对象、聚合或领域事件。变更仅限于基础设施层（缓存刷新）和应用层（视图逻辑）。

## 现有领域概念清单（被本次修复涉及）

| 概念 | 类型 | 限界上下文 | 本次变更 |
|------|------|-----------|---------|
| `User` | 聚合根 | Auth (认证) | 不变 |
| `PhoneLoginMethod` | 实体 (taskAuth 侧) | Auth (认证) | 不变 — `upsert_phone_login_method` 逻辑不变 |
| `VerificationCode` | 实体 | Accounts (账号) | 不变 |
| `BillingAccount` | 聚合根 | Billing (充值) | 不变 |
| `SmsVerifiedFlag` | 值对象 (缓存键) | Billing (充值) | 不变 — `billing_recharge_sms_ok:{uid}` 的读写逻辑不变 |

## NFR 驱动的领域模型影响

| NFR 决策 | 模型影响 | 具体动作 |
|----------|---------|---------|
| 数据一致性 L3 (read-your-writes) | `TaskAuthIdentityClient` 需要感知写操作副作用 | 在基础设施层增加 `invalidate_user_cache(user_id)` 方法，由 `upsert_phone_login_method` 成功后调用 |
| 容错 L1 (缓存删除非阻断) | 缓存刷新为 best-effort | `invalidate_user_cache()` 内部 `try/except`，失败仅 warning 日志 |

## 新增基础设施方法（非领域层）

```python
# accounts/taskauth_bridge/identity_client.py — TaskAuthIdentityClient
def invalidate_user_cache(self, user_id: str) -> None:
    """清除用户 identity 缓存（best-effort，失败不抛异常）。"""
    try:
        cache.delete(f'{CACHE_PREFIX}user:{user_id}')
    except Exception:
        logger.warning("identity_cache_invalidation_failed", extra={"user_id": user_id})
```

## 领域事件

**本次不引入新的领域事件。** SMS 验证是会话级短时状态（15 分钟 TTL），不适合用领域事件传递。若未来有跨上下文消费需求（如风控、审计），可引入 `PHONE_BOUND` 事件，但当前不在范围内。

## 自检

- [x] 无新增领域层文件（纯 Bug Fix，不需要新 `domain/` 文件）
- [x] 无 ORM 导入
- [x] 无外部服务导入
- [x] 变更仅在基础设施层和应用层
