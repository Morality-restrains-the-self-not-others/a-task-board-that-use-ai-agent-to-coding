# 实施计划: 充值 SMS 验证状态持久化修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-29-recharge-sms-verification-status-loss-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-29-recharge-sms-verification-status-loss-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-29-recharge-sms-verification-status-loss-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-29-recharge-sms-verification-status-loss-ddd-domain-model.md`

## 变更范围

2 个文件，约 10 行代码。无新增领域层文件。

## 任务清单

### Increment 1: Identity 缓存刷新 (治本)

- [ ] **T1** — `accounts/taskauth_bridge/identity_client.py` — 新增 `invalidate_user_cache(user_id)` 方法
  - 路径: `task2app/Saas_project/accounts/taskauth_bridge/identity_client.py`
  - 内容: 在 `TaskAuthIdentityClient` 类中新增方法，`try/except` 包裹 `cache.delete(f'taskauth:identity:user:{user_id}')`，失败写 `logger.warning`
  - 验证: 方法存在，签名正确，无异常抛出

- [ ] **T2** — `accounts/taskauth_bridge/login_methods_resolver.py` — `upsert_phone_login_method()` 成功后调用缓存刷新
  - 路径: `task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py`
  - 内容: 在 `upsert_phone_login_method()` 最后（`forward_to_taskauth` 成功返回后）调用 `get_identity_client().invalidate_user_cache(uid)`
  - 验证: Identity 缓存 key `taskauth:identity:user:{uid}` 在 phone upsert 后被删除

- [ ] **T3** — `accounts/taskauth_bridge/login_methods_resolver.py` — `upsert_username_login_method()` 成功后同样刷新（一致性）
  - 路径: 同 T2
  - 内容: 同样在 `upsert_username_login_method()` 成功后调用 `invalidate_user_cache(uid)`
  - 验证: 同 T2

### Increment 2: RechargePhoneStatusView 防御性兜底

- [ ] **T4** — `billing_bridge/recharge_views.py` — `RechargePhoneStatusView.get()` 始终检查 SMS 缓存
  - 路径: `task2app/Saas_project/billing_bridge/recharge_views.py`
  - 内容: 重构 `get()` 方法 — 无论 `has_phone` 取值，都调用 `is_recharge_sms_verified_for_user(request.user)`；统一构建 Response
  - 验证: 无绑定手机号的用户在 SMS 验证后 GET status 返回 `sms_verified: true`

### 测试

- [ ] **T5** — `tests/test_billing_recharge_validation.py` — 新增 `test_upsert_phone_login_method_invalidates_identity_cache`
  - 验证: mock taskAuth HTTP，调用 `upsert_phone_login_method` 后检查 `cache.get('taskauth:identity:user:{uid}')` 为 None
  - 命令: `cd task2app/Saas_project && python -m pytest tests/test_billing_recharge_validation.py::test_upsert_phone_login_method_invalidates_identity_cache -xvs`

- [ ] **T6** — `tests/test_billing_recharge_validation.py` — 新增 `test_recharge_phone_status_sms_verified_without_bound_phone`
  - 验证: 无手机绑定 + SMS 缓存已设置 → GET status 返回 `sms_verified: true`
  - 命令: `cd task2app/Saas_project && python -m pytest tests/test_billing_recharge_validation.py::test_recharge_phone_status_sms_verified_without_bound_phone -xvs`

### 验证

- [ ] **T7** — 运行全部 billing recharge 测试套件
  - 命令: `cd task2app/Saas_project && python -m pytest tests/test_billing_recharge_validation.py -xvs`
  - 验证: 所有已有 + 新增测试通过

## 执行顺序

```
T1 → T2 → T3 → T4 → T5 → T6 → T7
```

T5-T6 可与 T1-T4 并行编写（测试先行）。

## NFR 验证

| QS | 验证方式 |
|----|---------|
| QS-01 (read-your-writes) | T5 验证 identity 缓存删除；手动测试：verify_sms → phone_status 两请求间隔 < 1s |
| QS-02 (兜底) | T6 验证即使 has_phone=false 也返回 sms_verified=true |
| QS-03 (缓存删除非阻断) | T1 中 `try/except` 确保 `cache.delete` 失败不抛异常 |
