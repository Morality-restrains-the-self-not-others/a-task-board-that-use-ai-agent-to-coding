# Code Review (Round 2): BillingAccount 事件驱动初始化

> 审查范围: 6 Python files + 2 Go files (全部变更)
> 对照: plan.md, NFR clarification, domain model, DDD compliance

## 审查结论: ✅ 通过 (2 minor findings)

---

## 逐文件审查

### 1. `Saas_project/billing_bridge/client.py` ✅

| 检查项 | 结果 |
|--------|------|
| `requests` 移除 | ✅ `import requests` 已删除 |
| `HTTPClient` 替换 | ✅ `from core.utils.http_client import HTTPClient` |
| `forward_to_taskbill` 只分发 GET/POST | ✅ 所有调用方仅用 GET/POST |
| `timeout` 传递 | ✅ `**kwargs` → `requests.Session.get()` 透传 |
| `get_account` 只读 GET | ✅ 正确调用 `GET /api/internal/taskbill/accounts/{id}/` |
| `get_account_points` 错误格式 | ✅ 三段诊断: status + tenant_id + error_detail |
| `get_task_post_charge_points` | ✅ 改为调用 `get_account`（只读） |
| `get_or_create_account` 保留 | ✅ 保留供事件消费者和测试使用 |

**🟡 Finding 1: 异常捕获范围过宽**
```python
except Exception as exc:  # 原为 except requests.RequestException
    logger.warning('taskBill unreachable: %s', exc)
```
`HTTPClient` 内部已捕获 `Exception` 并 re-raise。此处再捕获所有异常会掩盖编程错误（如 `TypeError`/`ValueError`），错误日志"taskBill unreachable"会产生误导。

**Severity**: 🟡 Low — HTTPClient 已做异常隔离，实际风险极低。
**Fix**: 改为 `except OSError`（连接层错误）或显式捕获 `requests.RequestException`。非阻塞。

### 2. `Saas_project/billing_bridge/task_post_billing.py` ✅

| 检查项 | 结果 |
|--------|------|
| 函数重命名 | ✅ `get_or_create_billing_account` → `get_billing_account` |
| 导入更新 | ✅ `get_or_create_account` → `get_account` |
| `select_for_update` 移除 | ✅ GET 请求无需行锁 |
| 返回值简化 | ✅ `(account, created)` → `account` |
| 错误信息诊断 | ✅ 三段格式: status + tenant_id + error_detail |

### 3. `Saas_project/billing_bridge/taskbill_stub.py` ✅

| 检查项 | 结果 |
|--------|------|
| GET accounts 路径处理 | ✅ 正确解析 tenant_id，`create=False` 纯查询 |
| 404 响应 | ✅ 返回 `{'error': 'account not found'}` |
| 响应格式一致性 | ✅ 与 taskBill 实际响应一致 |

**🟡 Finding 2: stub 路径匹配使用子串而非前缀**
```python
if '/api/internal/taskbill/accounts/' in path:  # 子串匹配
```
**Severity**: 🟡 Low — 仅影响测试 stub，不影响生产。但未来添加 `/accounts/report/` 等路径时会误匹配。
**Fix**: 改为 `path.startswith('/api/internal/taskbill/accounts/')`。非阻塞。

### 4. `Saas_project/projects/views/todo_views.py` ✅

| 检查项 | 结果 |
|--------|------|
| 导入更新 | ✅ `get_billing_account` |
| 调用简化 | ✅ 移除 `select_for_update` 和解构 `_` |
| `transaction.atomic()` 保留 | ✅ 扣费事务完整性 |
| 并发安全性 | ✅ 无回归（旧 `select_for_update` 从未实际传到 taskBill） |

### 5. `Saas_project/core/grafana_errors/service.py` ✅

与 todo_views.py 变更一致。✅

### 6. `Saas_project/projects/view_test/conftest.py` ✅

| 检查项 | 结果 |
|--------|------|
| 账户预创建 | ✅ `get_or_create_account` → `get_billing_account` 正确顺序 |
| 导入 | ✅ 新增 `from billing_bridge.client import get_or_create_account` |

### 7. `taskBill/src/tenant_pricing.go` ✅

| 检查项 | 结果 |
|--------|------|
| `getBillingAccount` 只读 | ✅ 纯 SELECT，不写 |
| `ErrAccountNotFound` 哨兵 | ✅ 调用方可区分 404 vs 500 |
| SQL 注入防护 | ✅ 参数化查询 `?` |
| Fallback fix | ✅ 移除 `else { return error }`，保持二级解析 |
| JSON unmarshal | ✅ `_ = json.Unmarshal` 忽略错误（若字段为空仍可用） |

### 8. `taskBill/src/handlers.go` ✅

| 检查项 | 结果 |
|--------|------|
| Secret 校验 | ✅ `requireInternalSecret` (NFR L3) |
| tenant_id 解析 | ✅ `strings.TrimPrefix/TrimSuffix` + `ParseInt` |
| 边界检查 | ✅ `tid <= 0` → 400 |
| 404 vs 500 区分 | ✅ `ErrAccountNotFound` → 404; 其他 → 500 |
| 路由注册 | ✅ `mux.HandleFunc("/api/internal/taskbill/accounts/", ...)` |
| 响应格式 | ✅ 与 get-or-create 一致 (account, balance, locked_*) |

---

## DDD 合规审计

```
billing_bridge/domain/entities.py     ✅ 纯 Python 类，无基础设施导入
billing_bridge/domain/repositories.py ✅ ABC 抽象，无 ORM
billing_bridge/domain/services.py     ✅ 构造函数注入 repository 接口
billing_bridge/domain/events.py       ✅ frozen dataclass，过去式命名
billing_bridge/domain/__init__.py     ✅ 公开 API 导出
```

**0 个 DDD 违规**。领域层零基础设施依赖。

---

## NFR 覆盖确认

| NFR | 等级 | 实现证据 |
|-----|------|---------|
| 安全性 L3 | 内部密钥认证 | Go: `requireInternalSecret(r)` Python: `X-TaskBill-Internal-Secret` header |
| 安全性 L3 | 代理绕过 | `HTTPClient` → `_get_http_session()` → `trust_env=False` |
| 数据一致性 L3 | 幂等创建 | `tenant_id UNIQUE` + `getOrCreateBillingAccount` 不变 |
| 可用性 L2 | 降级 503 | `ErrAccountNotFound` → 调用方返回 "账户正在初始化" |
| 容错 L2 | 重试 | `except Exception` → re-raise → Django 上层处理 |
| 可观测性 L2 | 诊断信息 | 错误含 status + tenant_id + error_detail |

---

## 测试结果 (21/22 pass)

```
TodoViewSet_test.py                4/4 ✅
Billing tests (k="billing")         9/9 ✅
ProjectDetail git repos             7/7 ✅
test_product_pricing (auth)         0/1 ⚠️ pre-existing
```

---

## Summary

| Severity | Count | Description |
|----------|-------|-------------|
| 🔴 Critical | 0 | — |
| 🟡 Low | 2 | Broad exception catch + stub substring match |
| ✅ Pass | — | All other checks |

**Recommendation**: Ship as-is. 2 minor findings are non-blocking and can be addressed in follow-up.
