# 设计文档: 统一日志层 — 跨服务日志可排查性增强

**日期**: 2026-06-29
**上下文**: 排查 `recharge_verify_sms` 503 错误时发现日志链路断裂，无法在 Grafana 中端到端追踪请求

---

## 1. 现状分析

### 1.1 当前日志架构

```
[APISIX] → [Django] → [taskAuth HTTP Bridge] → [taskAuth Go]
   │           │                 │                      │
   │     JsonTraceFormatter  raw requests          structured JSON
   │     → stdout (JSON)     (no logging)          → stdout (JSON)
   │           │                                      │
   └───────────┴──────────────┬───────────────────────┘
                              ↓
                    Docker stdout/stderr
                              ↓
                    Promtail (tail + JSON parse)
                              ↓
                    Loki → Grafana
```

### 1.2 现有日志收集配置

**Promtail** (`AiMonitor/promtail/promtail-local.yaml`):
- Tails `/var/log/runall/*.log`
- Extracts `trace_id`, `level`, `msg` from JSON payload
- Pushes to Loki

**Django** (`settings.py:1054-1128`):
- `JsonTraceFormatter`: 输出 JSON 行，包含 `ts`, `level`, `service`, `logger`, `msg`, `trace_id`, `otel_trace_id`, `exc`
- `HttpAccessLogMiddleware`: 每个请求记录 `method`, `path`, `status`, `duration_ms`
- 所有 logger 输出到 `console` (stdout)

**taskAuth** (Go):
- 输出结构化 JSON 到 stdout
- 包含 `trace_id`, `method`, `path`, `status`, `duration_ms`

### 1.3 发现的 6 个日志缺口

#### 缺口 1: trace_id 未传播到 taskAuth 🔴 严重

`accounts/taskauth_bridge/client.py:26-62` 的 `forward_to_taskauth` 不发送 `X-Trace-Id` header：

```python
req_headers = {'Content-Type': 'application/json'}
if headers:
    req_headers.update(headers)
if token:
    req_headers['Authorization'] = f'Token {token}'
# ← 缺少: X-Trace-Id 传播
```

**验证**: 搜索 taskAuth 日志中 Django trace_id `web-1782656298121-vpeh9uciuq` → **零结果**。Django 日志有该 trace，taskAuth 日志完全没有。

**影响**: Grafana 中无法将 Django 请求与下游 taskAuth 调用关联。503 故障时看不到 taskAuth 侧发生了什么。

#### 缺口 2: taskAuth 调用成功时无日志 🟡 中等

`forward_to_taskauth` 仅失败时记录 WARNING：
```python
except requests.RequestException as exc:
    logger.warning('taskAuth unreachable: %s', exc)
    raise
```

成功调用无任何日志。无法回答："Django 是否调用了 taskAuth？调用了哪个端点？响应是什么？耗时多少？"

#### 缺口 3: 充值流程缺少结构化业务字段 🟡 中等

`recharge_views.py` 的 `logger.warning()` 未使用 `extra=` 传递结构化字段：

```python
logger.warning(
    "RechargeVerifySmsView: taskAuth unavailable for user %s",
    request.user.id,
)
# ← user_id 在 msg 字符串中，不是结构化字段，Loki 无法按 user_id 过滤
```

`JsonTraceFormatter` 已支持 `method`, `path`, `status`, `duration_ms`, `internal_action`, `task_id` 作为结构化 extra keys，但充值视图未使用。

#### 缺口 4: 通知日志使用非 JSON 格式 🟡 中等

`accounts_notify_logging_config.py` 的 `notify_logger` 使用传统 `Formatter`：
```python
formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
```

输出非 JSON，Promtail 的 JSON pipeline 无法解析 → Grafana 中搜索困难。

#### 缺口 5: 无 taskAuth bridge 专用日志文件 🟢 低

`core.utils.http_client` 有专用 `http_client_file` handler，但 `accounts.taskauth_bridge.client` 没有。taskAuth 调用日志混入通用 stdout，不利于独立监控。

#### 缺口 6: 无 billing 领域专用 logger 🟢 低

充值/计费操作使用 `logging.getLogger(__name__)` 混入通用日志流，无法独立设置日志级别或过滤。

---

## 2. 设计方案

### 2.1 核心原则

1. **trace_id 全链路传播**: 每个出站 HTTP 调用自动携带当前请求的 trace_id
2. **结构化字段优先**: 所有业务日志使用 `extra=` 传递可过滤字段
3. **JSON 格式统一**: 所有日志输出使用 `JsonTraceFormatter`
4. **成功+失败双面记录**: 关键外部调用成功和失败均有日志
5. **向后兼容**: 不改变现有日志消费者（Promtail pipeline、Grafana dashboard）

### 2.2 改动清单

#### 改动 1: `forward_to_taskauth` 自动传播 trace_id 🔴

**文件**: `accounts/taskauth_bridge/client.py`

```python
from core.logging.trace_context import get_trace_id

def forward_to_taskauth(path, *, method='POST', body=None, headers=None, token=None):
    base = _taskauth_base_url()
    if not base:
        raise RuntimeError('taskAuth base URL not configured')

    url = f'{base}{path}'
    req_headers = {'Content-Type': 'application/json'}
    
    # 自动传播 trace_id 到 taskAuth
    trace_id = get_trace_id()
    if trace_id:
        req_headers['X-Trace-Id'] = trace_id
    
    if headers:
        req_headers.update(headers)
    if token:
        req_headers['Authorization'] = f'Token {token}'
    # ... rest unchanged
```

**效果**: taskAuth 日志中将出现与 Django 相同的 `trace_id`，Grafana 可按 trace_id 串联完整调用链。

#### 改动 2: `forward_to_taskauth` 添加成功日志 🟡

**文件**: `accounts/taskauth_bridge/client.py`

```python
import time

def forward_to_taskauth(...):
    start = time.monotonic()
    try:
        resp = _taskauth_http.request(...)
        duration_ms = int((time.monotonic() - start) * 1000)
        logger.info(
            "taskAuth HTTP %s %s → %s (%dms)",
            method, path, resp.status_code, duration_ms,
            extra={
                'internal_action': 'taskauth_http',
                'method': method,
                'path': path,
                'status': resp.status_code,
                'duration_ms': duration_ms,
            },
        )
    except requests.RequestException as exc:
        duration_ms = int((time.monotonic() - start) * 1000)
        logger.warning(
            "taskAuth unreachable: %s %s (%dms): %s",
            method, path, duration_ms, exc,
            extra={
                'internal_action': 'taskauth_http_error',
                'method': method,
                'path': path,
                'duration_ms': duration_ms,
            },
        )
        raise
```

**效果**: 每次 taskAuth 调用都有结构化日志（含耗时、端点、状态码），可在 Grafana 中按 `internal_action=taskauth_http` 过滤。

#### 改动 3: `JsonTraceFormatter` 扩展结构化字段 🟡

**文件**: `core/logging/json_trace_formatter.py`

在现有 `_STRUCTURED_EXTRA_KEYS` 中增加业务字段：

```python
_STRUCTURED_EXTRA_KEYS = frozenset({
    # 现有
    "method", "path", "status", "duration_ms", "internal_action", "task_id",
    # 新增: 业务上下文
    "user_id", "tenant_id", "phone_masked", "operation",
})
```

**效果**: 视图层可通过 `logger.info(..., extra={'user_id': ..., 'operation': 'recharge_verify_sms'})` 传递可过滤字段。

#### 改动 4: 充值视图使用结构化日志 🟡

**文件**: `billing_bridge/recharge_views.py`

将现有 `logger.warning("...")` 改为带 `extra=` 的结构化日志：

```python
logger.warning(
    "RechargeVerifySmsView: taskAuth unavailable for user %s",
    request.user.id,
    extra={
        'user_id': str(request.user.id),
        'tenant_id': str(tenant_id) if tenant_id else None,
        'operation': 'recharge_verify_sms',
    },
)
```

#### 改动 5: 通知日志迁移到 JSON 格式 🟡

**文件**: `core/logging/accounts_notify_logging_config.py`

将 `notify_logger` 的 handler formatter 切换为 `JsonTraceFormatter`：

```python
from core.logging.json_trace_formatter import JsonTraceFormatter

json_formatter = JsonTraceFormatter()
file_handler.setFormatter(json_formatter)
console_handler.setFormatter(json_formatter)
```

#### 改动 6: 新增 taskAuth bridge 专用日志文件 🟢

**文件**: `saas_project/settings.py` → `LOGGING` dict

```python
'taskauth_bridge_file': {
    'class': 'core.logging.flushing_handler.FlushingFileHandler',
    'filename': str(_KAFKA_LOG_DIR / 'taskauth_bridge.log'),
    'formatter': 'json_trace',
},
```

并在 loggers 中新增：
```python
'accounts.taskauth_bridge': {
    'handlers': ['console', 'taskauth_bridge_file'],
    'level': 'INFO',
    'propagate': False,
},
```

---

## 3. 改动文件清单

| # | 文件 | 改动类型 | 行数 |
|---|------|----------|------|
| 1 | `accounts/taskauth_bridge/client.py` | trace_id 传播 + 成功/失败日志 | ~15 行 |
| 2 | `core/logging/json_trace_formatter.py` | 扩展 _STRUCTURED_EXTRA_KEYS | ~3 行 |
| 3 | `billing_bridge/recharge_views.py` | 结构化 extra= 字段 | ~10 行 |
| 4 | `core/logging/accounts_notify_logging_config.py` | JSON formatter | ~5 行 |
| 5 | `saas_project/settings.py` | taskAuth bridge logger + file handler | ~15 行 |

---

## 4. 修复前后对比

### 修复前：排查 recharge_verify_sms 503

```
Grafana 搜索 trace_id=web-1782697460594-ea7gyu5dwof:
  ✅ Django: http_request POST /.../recharge_verify_sms/ → 503 (22ms)
  ❌ taskAuth: 无日志（trace_id 未传播）
  ❌ bridge: 无日志（成功时不记录）
  
结论: 知道 Django 返回了 503，但不知道 taskAuth 侧发生了什么
```

### 修复后：排查 recharge_verify_sms 503

```
Grafana 搜索 trace_id=web-1782697460594-ea7gyu5dwof:
  ✅ Django: http_request POST /.../recharge_verify_sms/ → 503 (22ms)
  ✅ bridge: taskauth_http_error POST /api/internal/login-methods/phone-taken/ (15ms): ConnectionError
  ✅ taskAuth: 无日志（确认 taskAuth 未收到请求）
  
结论: 一目了然 — taskAuth bridge 连接失败，taskAuth 未收到请求，问题在网络层而非 taskAuth
```

---

## 5. 领域概念清单

（本次为基础设施增强，不引入新领域概念）

| 概念 | 类型 | 上下文 |
|------|------|--------|
| TraceContext | Infrastructure | 跨服务 — contextvars 存储 request-scoped trace_id |
| JsonTraceFormatter | Infrastructure | 日志 — 统一 JSON 行格式 |
| HttpAccessLogMiddleware | Infrastructure | 日志 — 每请求自动记录 |

---

## 6. 价值流影响

无 `value-stream.yaml`。此增强影响所有依赖 taskAuth 的流程：
- **用户认证** — 登录/注册/token 解析可端到端追踪
- **充值流程** — SMS 验证可追踪至 taskAuth phone 绑定
- **GitHub OAuth** — OAuth 流程可追踪至 taskAuth identity

---

总结清单：
- 缺口1 (trace_id传播): 在 forward_to_taskauth 自动添加 X-Trace-Id header
- 缺口2 (成功日志): 每次 taskAuth 调用记录结构化 success/error 日志
- 缺口3 (业务字段): 扩展 JsonTraceFormatter + 视图层使用 extra=
- 缺口4 (JSON格式): 通知日志迁移到 JsonTraceFormatter
- 缺口5 (专用文件): 新增 taskauth_bridge logger + 文件 handler
- 缺口6 (billing logger): 可选，建议后续按需添加
