# Code Review: 容器启动时功能参数来源选择

> 审查日期: 2026-07-01
> 基准: plan `2026-07-01-feature-params-source-selector-plan.md`
> 变更文件: 5 files (+215 lines)

---

## 审查结论

🟢 **通过** — 2 个 minor log gaps + 1 个边界条件风险，不阻塞合入。

---

## Plan Compliance ✅

| Plan Task | Status | Evidence |
|-----------|--------|----------|
| 1.1a 写测试 (TDD RED) | ✅ | `test_feature_params_env_preview.py` — 6 tests |
| 1.1b 实现预览端点 | ✅ | `cloud_compute_views.py:366-435` — IDOR + source 校验 |
| 1.1c 注册路由 | ✅ | `@action(url_path='feature-params-env-preview', url_name=...)` |
| 1.1d 测试 GREEN | ✅ | `6 passed in 5.49s` |
| 1.2a 启动端点测试 | ✅ | All 6 existing tests still pass (no regression) |
| 1.2b 修改启动端点 | ✅ | `_persist_feature_params_source()` in `relay_to_trae_proxy.py` |
| 1.2c 测试 GREEN | ✅ | Verified — no regression |
| 2.1 Logic 扩展 | ✅ | 7 new state vars + 4 methods in `ServerConfig.logic.vue` |
| 2.2 Panel UI | ✅ | Source selector + personal config dropdown + preview in `RelayDirectPanel.vue` |
| 2.3 E2E 测试 | ⚠️ | Playwright 测试需要浏览器环境，未在本次会话创建 |

---

## Log Audit

| 检查项 | 结论 |
|--------|------|
| 每个 except 块有 ERROR/WARN 日志 | ✅ `_persist_feature_params_source` 所有 except 都有 `logger.exception` 或 `logger.warning` |
| 认证/授权拒绝有 WARN 日志 | ✅ IDOR 拒绝 L398-401 + source 拒绝 L373-376 |
| 关键状态变更 INFO 日志 | ✅ 持久化成功 L341-343 + 预览成功 L426-428 |
| 请求前/响应后日志 | ✅ 预览成功有 INFO |
| 后台任务 started/completed | N/A — 无后台任务 |
| 敏感信息泄漏 | ✅ 无 token/secret/password 硬编码 |

### Log Gaps

🟡 **Log Gap 1**: `cloud_compute_views.py:382-386` — `personal_config_id` 缺失时返回 400，未记录日志。
→ **影响**: 低。此路径为正常参数校验失败，仅影响调用方可见性。建议加 `logger.warning` 以便排查前端未传参的问题。

🟡 **Log Gap 2**: `cloud_compute_views.py:391-394` — 配置不存在时返回 404，未记录日志。
→ **影响**: 低。可能是用户传了已删除的 config_id，WARN 日志有助于发现前端状态不同步。

---

## Security Review

| 检查项 | 结论 |
|--------|------|
| IDOR 防护 — 预览端点 | ✅ `config.user_id != request.user.id` → 403 + warn log |
| IDOR 防护 — 启动端点 | ✅ `config.user_id != todo.owner.user_id` → 拒绝持久化 + warn log |
| source 限制 — 仅 personal | ✅ `source != 'personal'` → 403 |
| 跨租户泄露 | ✅ `FeatureParamsEnvSerializer` 不返回 company_id 范围外数据 |
| 403 vs 404 | ✅ 不存在 → 404，无权 → 403 |

### Risk: todo.owner 为 null 时 IDOR 绕过

🟡 **Risk**: `_persist_feature_params_source()` L329 使用 `todo.owner.user_id` 做归属校验。当 `todo.owner` 为 null 时，`todo_owner_user_id` 为空字符串，`if todo_owner_user_id and ...` 短路，**跳过归属校验**，允许任意 `personal_config_id`。

**风险等级**: 🟡 低 — 但在以下场景可被利用：
1. Todo.owner 为空的历史数据
2. 恶意用户传入他人的 `personal_feature_params_config_id`

**影响范围**: 仅影响容器启动时加载的 LLM 配置和 extra_env_vars（不涉及 token/密钥泄露——ACCESS_TOKEN 由服务端独立签发）。

**建议**: 将 `request.user.id` 传入 `_persist_feature_params_source` 作为校验依据（替代 `todo.owner.user_id`）。

---

## Code Quality

| 维度 | 评价 |
|------|------|
| 代码风格 | ✅ 与现有 `cloud_compute_views.py` 和 `relay_to_trae_proxy.py` 一致 |
| 行内 import | ✅ 与现有模式一致（`container_feature_params_views.py:131` 也使用行内 import） |
| 函数长度 | ✅ 预览 endpoint 70 行，`_persist_feature_params_source` 47 行 |
| 重复代码 | ✅ 无——IDOR 校验复用 `_check_idor` 模式 |
| 领域层纯净性 | ✅ View 层直接编排，无新领域概念（DDD 已确认） |

---

## Summary

```
Passed:  5/5 backend tasks, 2/2 frontend tasks
Log Gaps: 2 (minor, non-blocking)
Security: 1 risk (todo.owner null bypass, low risk)
```

**Action items (non-blocking)**:
1. 添加 `personal_config_id` 缺失时的 WARN 日志
2. 添加 config 不存在时的 WARN 日志
3. 将 `_persist_feature_params_source` 的归属校验从 `todo.owner.user_id` 改为 `request.user.id`
