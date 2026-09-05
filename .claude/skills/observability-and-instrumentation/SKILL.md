---
name: observability-and-instrumentation
description: 可观测性埋点规范。适用于所有生产环境代码——结构化日志、RED/USE 指标、分布式追踪、告警规则。确保每条代码路径在生产中可诊断。
source: adapted from addyosmani/agent-skills
---

# 可观测性与埋点（Observability & Instrumentation）

## 概述

不可观测的代码就是不可运维的代码。可观测性是回答"系统在做什么以及为什么"的能力——使用代码发出的遥测数据从外部回答。埋点不是在发布后才加的——它和代码一起编写，就像测试一样。如果一个功能不带遥测就上线，第一个用户报 bug 就变成了考古学而不是查询。

## 流程

### 1. 埋点前先定义"正常"

没有问题的遥测只是噪音。在添加任何埋点前，写下值班工程师会问的 2-4 个问题：

```
FEATURE: 支付重试
值班会问:
1. 多少比例的支付在首次尝试就成功 vs 重试后成功？
2. 当支付永久失败时，为什么？（供应商错误？超时？验证？）
3. 支付供应商是否比平时慢？
→ 每个信号必须帮助回答其中一个问题。
```

### 2. 为每个问题选择正确信号

| 信号 | 回答 | 成本 | 示例 |
|------|------|------|------|
| **结构化日志** | "这个具体案例发生了什么？" | 按事件，随流量增长 | `payment_failed` 带上错误码 |
| **指标** | "多频繁 / 多快，总体？" | 按系列固定，查询廉价 | 供应商调用的 p99 延迟 |
| **追踪** | "跨服务时间花在哪？" | 按请求，通常采样 | 一次慢支付按 hop 分解 |

规则：指标告诉你**有**问题，追踪告诉你**在哪**，日志告诉你**为什么**。

### 3. 结构化日志

记录事件，不是散文。每条日志行是 JSON 对象，有稳定的事件名和机器可读字段：

```go
// BAD: 字符串插值 — 不可查询，不一致
log.Printf("Payment %s failed for user %s after %d retries", id, userID, n)

// GOOD: 稳定事件名 + 结构化字段（Go slog）
logger.WarnContext(ctx, "payment_failed",
    slog.String("paymentId", id),
    slog.String("provider", "stripe"),
    slog.String("errorCode", err.Code),
    slog.Int("attempt", n),
)
```

```python
# GOOD: Python 结构化日志
logger.warning("payment_failed", extra={
    "paymentId": payment_id,
    "provider": "stripe",
    "errorCode": err.code,
    "attempt": n,
})
```

**日志级别 — 一致使用：**

| 级别 | 含义 | 值班动作 |
|------|------|---------|
| `error` | 不变量被破坏；有人可能需要行动 | 调查 |
| `warn` | 降级但已处理（重试成功、使用回退） | 观察趋势 |
| `info` | 重大业务事件（订单已下、任务已完成） | 无 |
| `debug` | 诊断详情 | 生产环境默认关闭 |

**Correlation ID 是强制要求。** 在系统边界生成（或接受）请求 ID，并附加到每条日志行、span、出站调用。没有它，你无法从交织的日志中重建单个请求。

```go
// Go: middleware 注入 correlation ID
func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        w.Header().Set("X-Request-ID", requestID)
        ctx := context.WithValue(r.Context(), requestIDKey, requestID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**绝不记录密钥、token、密码、完整 PII。** 这是硬规则——遥测管道是经典的数据泄漏路径。白名单字段；不要记录整个请求体。

### 4. 指标

对于请求驱动的服务，为**每个端点**和**每个外部依赖**埋 RED 指标：

- **R**ate（请求率）
- **E**rrors（失败率）
- **D**uration（延迟直方图，非平均值）

```go
// Go: Prometheus 直方图
var httpDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request duration",
        Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
    },
    []string{"method", "endpoint", "status_class"}, // '2xx' 不是 '200'
)
```

**基数（Cardinality）是故障模式。** 每个唯一标签组合是一个独立时间序列。标签必须来自小的固定集合。绝不用 user_id、raw URL、错误消息文本等无界值做标签。

```
OK 标签:    endpoint="/api/v1/tasks/:id"  status_class="5xx"  provider="stripe"
绝不标签:  user_id, email, request_id, full_url, error_message_text
```

绝不追踪平均值，始终追踪分位数：平均值隐藏了那 1% 体验极差的用户。使用直方图并读取 p50/p95/p99。

### 5. 分布式追踪

遵循 OpenTelemetry 标准。对于 Go 服务：

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// 自动 instrumentation 覆盖 HTTP/gRPC/DB 客户端
handler := otelhttp.NewHandler(mux, "task-service")
```

手动 span 仅围绕有意义的内部工作单元（如 `applyDiscounts`、`chargeProvider`），并附加值班要过滤的属性。跨每个异步边界传播 context — HTTP header、队列消息 metadata — 否则追踪在间隙处死亡。

### 6. 告警

告警关注**用户感受到的症状**，不是原因：

```
SYMPTOM (值得 page):              CAUSE (dashboard 而非 page):
错误率 > 1% 持续 5 分钟             CPU 85%
p99 延迟 > 2s                      一个 pod 重启
队列堆积 > 10 分钟                  磁盘 70%
```

每条告警规则：
1. **必须可操作** — 如果响应是"忽略，它自愈"，删除告警
2. **链接到 runbook** — 哪怕三行：什么含义、首要查询、升级路径
3. **阈值和持续时间**由 SLO 或历史数据证明，不由猜测
4. 只用两个级别：**page**（用户面，立即行动）和 **ticket**（降级，本周行动）

### 7. 验证遥测本身

埋点是代码；它可能错误。在标记完成前触发路径并查看实际输出：

- 在 staging 强制一个错误 → 通过 `requestId` 在日志中定位，确认字段是结构化的
- 发送测试流量 → 确认指标系列以预期标签出现
- 在追踪 UI 中跟踪跨服务的一个请求 → 没有断裂 span
- 测试触发每条新告警（临时降低阈值）→ 确认到达正确通道

## 项目技术栈对照

| 技术栈 | 日志 | 指标 | 追踪 |
|--------|------|------|------|
| Go | `log/slog` (1.21+) | `prometheus/client_golang` | `go.opentelemetry.io/otel` |
| Django | `logging.LoggerAdapter` | `django-prometheus` | `opentelemetry-instrumentation-django` |
| Vue | `console` → Sentry | Web Vitals API | 前端 trace 到后端 span |

## 红旗

- 包含重试/队列/外部调用的功能 PR 没有任何新遥测
- 字符串插值构建日志行而非结构化字段
- 无 correlation/request ID — 每条日志行是孤儿
- 指标标签使用 user_id、raw URL 或错误消息文本（基数炸弹）
- 延迟以平均值追踪，无分位数
- 密钥、token 或完整请求体出现在日志中
- "在我机器上正常"作为生产功能健康的唯一证据
