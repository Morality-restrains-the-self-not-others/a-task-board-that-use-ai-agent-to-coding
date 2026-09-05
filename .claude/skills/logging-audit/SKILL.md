---
name: logging-audit
description: >
  [REFERENCE SKILL — auto-invoked by Step 8 (build) and Step 9 (review).]
  Log audit reference: log levels, must-log paths, forbidden content, structured format spec,
  and audit checklist. Not user-invoked — serves as the canonical logging standard consumed
  by the build and review pipeline steps.
---

# logging-audit — 可观测性驱动的日志审计与补全

> **定位**: 横切关注点工具技能，不占据独立流水线编号。
> 在 Step 8（构建）中伴随实现写入日志，在 Step 9（审查）中对日志做专项审计。

---

## 执行顺序（在流水线中的位置）

```
Step 5  NFR     → 澄清日志需求（结构化？级别？PII 脱敏？）
Step 6  DDD     → 领域事件即天然日志点
Step 8  Build   → [本技能] 伴随实现写入必要日志 ← 主调用点
Step 9  Review  → [本技能] 审计日志覆盖度与质量 ← 次调用点
```

### 为什么放在 Step 8 和 Step 9 之间

| 阶段 | 做什么 | 不做什么 |
|------|--------|----------|
| **Step 5 NFR** | 决定日志格式（JSON/plain）、聚合方式、保留策略 | 不写具体日志语句 |
| **Step 6 DDD** | 识别领域事件（`UserCreated`, `OrderPaid`...）；**每个业务意图对应至少一个事件契约并规划 MQ 投递** | 不决定日志级别 |
| **Step 8 Build** ✅ | **伴随每段实现代码写入必要日志**；意图成功路径必须 `publish` 对应事件 | 不等到最后补日志/补事件 |
| **Step 9 Review** ✅ | **逐文件审计日志覆盖度：错误路径、外部调用、状态变更**；审计 Intent→Event 投递 | 不跳过无日志/无事件的代码块 |

**核心原则**: 日志不是事后补的——它和功能代码一起写、一起审。

### 排障侧：有 data-traceId 时日志优先

本技能规范**写入**侧（`trace_id` 必填、结构化 JSON）。Agent **排障**时若错误带 **`data-traceId`**，须**优先**用该 ID 检索 Loki/Grafana 并重建全链路路径，再改代码——见：

- `.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`
- `.claude/skills/1-brainstorming-design-docs/SKILL.md`「TraceId 驱动的 Grafana 日志分析」
- 元规则 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md`

实现审计时额外确认：失败路径日志含可检索的 `trace_id`，且与前端 `data-traceId` / 响应头 `X-Trace-Id` 对齐。

---

## 日志级别速查表

| 级别 | 语义 | 触发条件 | 示例 |
|------|------|----------|------|
| **ERROR** | 需要人工介入 | 异常被吞、外部服务不可达、数据不一致 | `log.error("Payment failed", exc_info=True, extra={"order_id": id})` |
| **INFO** | 关键业务节点 | 状态变更、外部调用、认证决策、资源创建/删除 | `log.info("Project created", extra={"project_id": id, "user_id": uid})` |
| **WARN** | 降级/重试/预期外但可恢复 | 重试耗尽、回退到默认值、权限拒绝、**幂等命中跳过** | `log.warning("idempotency skip", extra={"event_type": t, "idempotency_key": fingerprint})` |
| **INFO** | 关键业务节点 | 状态变更、外部调用、认证决策、资源创建/删除 | `log.info("Project created", extra={"project_id": id, "user_id": uid})` |
| **DEBUG** | 诊断细节 | 函数入参/出参、中间计算值、缓存命中 | `log.debug("Looking up user", extra={"user_id": uid})` |

### 反模式

```
❌ log.info("Starting process...")           # 无上下文
❌ log.error("Error: " + str(e))             # 丢失 traceback
❌ log.debug(f"Password: {password}")        # 泄露敏感信息
❌ try: ... except: pass                     # 异常被吞，无日志
```

---

## 必须记录日志的 6 类关键路径

### 1. 错误路径（最高优先级）

```python
# ✅ 正确：记录完整上下文 + traceback
try:
    result = external_service.call(payload)
except ExternalServiceError as e:
    logger.error(
        "External service call failed",
        exc_info=True,
        extra={
            "service": "payment-gateway",
            "endpoint": "/v1/charge",
            "request_id": request_id,
            "user_id": user_id,
            "duration_ms": elapsed_ms,
        },
    )
    raise
```

**检查规则**: 每个 `except` 块（不含 `pass` 的）必须有 ≥1 条 ERROR 或 WARN 日志。

### 2. 外部 API 调用

```python
# ✅ 正确：记录请求摘要 + 响应状态 + 耗时
start = time.monotonic()
response = http_client.post(url, json=payload)
elapsed_ms = (time.monotonic() - start) * 1000
logger.info(
    "Outbound API call",
    extra={
        "method": "POST",
        "url": url,
        "status_code": response.status_code,
        "duration_ms": elapsed_ms,
        "request_id": request_id,
    },
)
```

**检查规则**: 每个跨进程/跨网络调用必须有请求前（DEBUG）和响应后（INFO）日志。

### 3. 状态变更（领域事件）

```python
# ✅ 正确：记录变更前后的状态
logger.info(
    "Order status changed",
    extra={
        "order_id": order.id,
        "from_status": "pending",
        "to_status": "paid",
        "user_id": user_id,
        "payment_method": "wechat",
    },
)
```

**检查规则**: 每个数据库 UPDATE/DELETE 操作（非查询）建议有 INFO 日志；关键业务实体必须有。

### 4. 认证与授权决策

```python
# ✅ 正确
logger.info(
    "Access denied",
    extra={
        "user_id": user_id,
        "resource": "Project",
        "resource_id": project_id,
        "required_permission": "project.admin",
        "user_permissions": user_perms,
    },
)
```

**检查规则**: 每次 auth 拒绝必须有 WARN 日志，包含 who/what/why。

### 5. 后台任务 / 异步作业

```python
# ✅ 正确
logger.info("Background job started", extra={"job_id": job_id, "task": "send_email"})
try:
    result = do_work()
    logger.info("Background job completed", extra={"job_id": job_id, "result": result})
except Exception:
    logger.exception("Background job failed", extra={"job_id": job_id})
```

**检查规则**: 每个后台任务必须有 started/completed/failed 三条日志中的至少两条。

### 6. 资源生命周期

```python
# ✅ 正确
logger.info("Project created", extra={"project_id": p.id, "user_id": uid, "company_id": cid})
logger.info("Project archived", extra={"project_id": p.id, "user_id": uid, "reason": reason})
```

**检查规则**: 每个 CRUD 的 C/U/D 操作建议有 INFO 日志。

---

## 禁止记录的内容

| 类别 | 示例 | 处理方式 |
|------|------|----------|
| 密码/密钥 | `password`, `secret_key`, `api_key` | 永远不记录 |
| Token | JWT, OAuth access_token, session_id | 只记录 hash 前 8 位 |
| 完整 PII | 身份证号、银行卡号、完整手机号 | 脱敏：`138****1234` |
| 完整 Email | `user@example.com` | 脱敏：`u***@example.com` |
| 大体积 Body | 上传文件内容、Base64 图片 | 只记录 size |

```python
# ✅ 正确：脱敏
logger.info("User login", extra={"email": mask_email(email), "ip": request.ip})

# ❌ 错误：泄露
logger.info(f"User {email} logged in with token {jwt}")
```

---

## 结构化日志规范

统一使用 JSON 格式，确保机器可解析：

```json
{
  "ts": "2026-06-30T10:30:00.123Z",
  "level": "info",
  "logger": "app.services.payment",
  "msg": "Payment processed",
  "trace_id": "a1b2c3d4e5f6",
  "request_id": "req_abc123",
  "user_id": "user_42",
  "extra": {
    "order_id": "ord_789",
    "amount": 99.00,
    "currency": "CNY",
    "duration_ms": 234
  }
}
```

### 必填字段

| 字段 | 说明 | 示例 |
|------|------|------|
| `ts` | ISO 8601 UTC | `2026-06-30T10:30:00.123Z` |
| `level` | 小写 `debug\|info\|warn\|error`（禁止 `INFO`/`WARNING`） | `info` |
| `message` | 人类可读的一句话 | `"Payment processed"` |
| `trace_id` | 全链路追踪 ID | `"a1b2c3d4e5f6"` |

### 推荐字段

| 字段 | 说明 |
|------|------|
| `request_id` | HTTP 请求 ID |
| `user_id` | 操作人 ID |
| `duration_ms` | 操作耗时 |
| `extra.*` | 业务自定义字段 |

---

## 审计清单（Step 9 使用）

执行日志审计时，按以下清单逐项检查：

### A. 覆盖率检查

- [ ] 每个 `except` 块（不含 `pass`）是否有 ERROR/WARN 日志？
- [ ] 每个外部 HTTP/gRPC/DB 调用是否有请求前 DEBUG + 响应后 INFO？
- [ ] 每个关键状态变更（业务实体 C/U/D）是否有 INFO 日志？
- [ ] 每个认证拒绝点是否有 WARN 日志（含 who/what/why）？
- [ ] 每个后台任务是否有 started/completed/failed 日志？
- [ ] 每个资源创建/删除是否有 INFO 日志？

### B. 质量检查

- [ ] 日志消息是否包含足够的定位信息（ID、状态、操作人）？
- [ ] 错误日志是否使用 `exc_info=True` / `logger.exception()` 记录 traceback？
- [ ] 是否有硬编码的敏感信息（搜索 `password`, `secret`, `token`, `key`）？
- [ ] 日志级别是否使用合理（ERROR 不是 WARN，INFO 不是 DEBUG）？

### C. 噪音检查

- [ ] 循环内是否有 INFO 日志？（高频路径用 DEBUG）
- [ ] 是否有 `log.info("here")` / `log.info("test")` 等无意义日志？
- [ ] 是否有被注释掉的日志语句？（要么恢复，要么删除）

---

## 语言专项指南

### Python / Django

```python
import logging
import time

logger = logging.getLogger(__name__)

# 使用 extra 传递结构化字段
logger.info("User action", extra={"user_id": uid, "action": "create_project"})

# 使用 logger.exception 自动记录 traceback
try:
    risky_operation()
except Exception:
    logger.exception("risky_operation failed", extra={"user_id": uid})

# Django 中间件确保 request_id 传播
# settings.py: MIDDLEWARE += ['app.middleware.RequestIDMiddleware']
```

### Go

```go
import "log/slog"

slog.Info("order created",
    slog.String("order_id", orderID),
    slog.String("user_id", userID),
    slog.Int64("duration_ms", elapsed.Milliseconds()),
)

// 错误路径
if err != nil {
    slog.Error("payment failed",
        slog.String("order_id", orderID),
        slog.Any("error", err),
    )
}
```

---

## 与 NFR 的衔接

在 Step 5 NFR 阶段明确了以下问题后，本技能才可高效执行：

| NFR 决策 | 对日志的影响 |
|----------|-------------|
| 日志格式（JSON vs plain） | 决定 `formatter` 配置 |
| 日志聚合（ELK / Loki / CloudWatch） | 决定 `trace_id` 格式和必填字段 |
| 保留策略（30d / 90d / 1y） | 决定日志详细程度（保留久的少记 PII） |
| PII 脱敏策略 | 决定哪些字段需要 `mask_*()` 函数 |
| 采样率（100% / 10% / 1%） | 决定哪些路径用 INFO vs DEBUG |

如果 NFR 未做这些决策，本技能默认采用：
- **格式**: JSON（可切换为 plain）
- **级别**: 开发/测试环境 DEBUG，生产环境 INFO
- **脱敏**: 密码/Token 绝不出现在日志中；Email/手机号模糊处理

---

## 完成后 — 下一步选择

审计完成后，使用 `AskUserQuestion` 让用户选择：

```
header: "下一步"
question: "日志审计完成。下一步做什么？"
multiSelect: false
options:
  1. label: "继续代码审查 (推荐)"
     description: "日志审计是审查的子集，继续完整 code review"
  2. label: "修复日志缺失"
     description: "回到 Step 8 补齐缺失的日志语句"
  3. label: "交付上线"
     description: "日志已达到可观测性标准，进入 Step 10"
```

- 用户选 1 → 调用 `/9-review`
- 用户选 2 → 调用 `/8-build`
- 用户选 3 → 调用 `/10-ship`
