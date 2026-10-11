# taskAIEndPoint 使用说明

> **版本**: v1 | **状态**: ✅ current | **更新日期**: 2026-07-01

---

## 目录

1. [概述](#1-概述)
2. [架构定位](#2-架构定位)
3. [快速开始](#3-快速开始)
4. [API 参考](#4-api-参考)
5. [认证机制](#5-认证机制)
6. [预算管控](#6-预算管控)
7. [Provider 配置](#7-provider-配置)
8. [请求处理流程](#8-请求处理流程)
9. [错误处理](#9-错误处理)
10. [配置项](#10-配置项)
11. [可观测性](#11-可观测性)
12. [使用示例](#12-使用示例)
13. [故障排查](#13-故障排查)

---

## 1. 概述

**taskAIEndPoint** 是平台的 LLM API 网关服务（Go 实现，端口 `8013`），作为所有 LLM 请求的**唯一出入口**，负责：

| 能力 | 说明 |
|------|------|
| 🔐 **代理认证** | 验证容器/Agent 持有的 sub-token（代理令牌），确保容器不持有上游 API 主密钥 |
| 🚦 **预算门控** | 每次 LLM 调用前检查任务级模型预算，余额不足返回 `402 Payment Required` |
| 📡 **请求转发** | 将 OpenAI 兼容的 Chat Completions 请求转发至上游 LLM 提供商（DeepSeek、OpenAI 等） |
| 📊 **用量记录** | 每次调用后自动记录 token 消耗，支持幂等去重 |
| 🌊 **流式支持** | 完整支持 SSE (Server-Sent Events) 流式响应 |
| 🔍 **链路追踪** | 自动生成/传播 `X-Trace-Id`，输出结构化 JSON 日志供 Loki/Tempo 采集 |

### 适用场景

- Agent 容器内的代码需要调用 LLM，但不应持有上游 API Key
- 需要对每个任务的 LLM 调用进行预算控制和用量统计
- 需要统一的 LLM 流量入口做审计和监控

---

## 2. 架构定位

```
┌──────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Agent 容器   │────▶│  taskAIEndPoint   │────▶│  DeepSeek/OpenAI │
│ (sub-token)  │     │   (Go :8013)     │     │  (upstream LLM)  │
└──────────────┘     └────────┬─────────┘     └─────────────────┘
                              │ 内部 API
                              ▼
                     ┌──────────────────┐
                     │  saas-backend     │
                     │  (Django :8001)  │
                     │  - token验证      │
                     │  - 路由解析       │
                     │  - 凭证获取       │
                     │  - 预算门控       │
                     │  - 用量记录       │
                     └──────────────────┘
```

- **taskAIEndPoint** 是**纯代理网关**，不存储任何业务数据
- 所有业务逻辑（token 校验、预算判断、凭证查询）通过内部 API 委托给 Django
- 上游 Master API Key **始终存储在 Django 端**，绝不暴露给容器

---

## 3. 快速开始

### 3.1 启动服务

```bash
# 方式 1: 通过 runAll 启动（推荐，会自动处理依赖）
cd /path/to/monorepo
./run.sh start task-ai-endpoint

# 方式 2: 独立启动
cd taskAIEndPoint
./run.sh
```

### 3.2 验证服务

```bash
curl http://127.0.0.1:8013/api/health/
# {"status":"ok","service":"taskAIEndPoint"}
```

### 3.3 前置依赖

- **saas-backend** (Django `:8001`) — 必须先启动，taskAIEndPoint 的所有内部 API 都依赖它
- 确保 `conf/ai/task-ai-endpoint/config.yaml` 中的 `internalSecret` 与 Django 的 `settings.TASK_AI_ENDPOINT_INTERNAL_SECRET` 一致

---

## 4. API 参考

### 4.1 健康检查

```
GET /api/health/
```

**响应** `200 OK`:
```json
{
  "status": "ok",
  "service": "taskAIEndPoint"
}
```

---

### 4.2 LLM 代理 (Chat Completions)

这是核心接口，接收 OpenAI 兼容的 Chat Completions 请求并转发至上游 LLM。

```
POST /api/tenant/{tenant_id}/workspace/{workspace_id}/task/{task_id}/llm/{provider}/v1/{subpath}
```

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `tenant_id` | string | 租户（公司）ID |
| `workspace_id` | string | 工作空间 ID |
| `task_id` | string | 任务 ID |
| `provider` | string | LLM 提供商标识，如 `deepseek`、`openai` |
| `subpath` | string | OpenAI 兼容路径，如 `chat/completions` |

#### 请求头

| Header | 必需 | 说明 |
|--------|------|------|
| `Authorization` | ✅ | `Bearer <proxy_token>` — 代理令牌 |
| `Content-Type` | ✅ | `application/json` |
| `X-Trace-Id` | 推荐 | 链路追踪 ID，不传则自动生成 |
| `X-Request-Id` | 推荐 | 请求幂等键，不传则使用 `X-Trace-Id` |

#### 请求体

标准 OpenAI Chat Completions 格式：

```json
{
  "model": "deepseek-chat",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello!"}
  ],
  "stream": false,
  "temperature": 0.7,
  "max_tokens": 1024
}
```

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 模型名称，用于匹配预算配置和上游路由 |
| `messages` | array | ✅ | 消息列表，标准 OpenAI 格式 |
| `stream` | boolean | ❌ | 是否使用 SSE 流式响应，默认 `false` |

其他 OpenAI 参数（`temperature`、`max_tokens`、`top_p` 等）均透传至上游。

#### 响应

**非流式** (`"stream": false`): 直接透传上游 LLM 的 JSON 响应，包含 `choices` 和 `usage` 字段：

```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1719792000,
  "model": "deepseek-chat",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 8,
    "total_tokens": 23
  }
}
```

**流式** (`"stream": true`): SSE 流式输出，格式为：

```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

...

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":15,"completion_tokens":8,"total_tokens":23}}

data: [DONE]
```

> **注意**: 网关会自动注入 `stream_options.include_usage = true`，确保上游在最后的 chunk 中返回 `usage` 信息，用于准确记录 token 消耗。

---

## 5. 认证机制

### 5.1 Proxy Token（代理令牌）

容器/Agent 使用 **代理令牌 (proxy token)** 而非上游 API Key 来认证。代理令牌通过以下流程获得：

1. 任务启动时，Django 为容器创建 `CloudServerConfig` 记录，生成 `container_access_token`
2. `rewriteSubTokenProvidersForProxy`（taskCloudService）将所有 sub-token provider 的 URL 改写为网关地址，`api_key` 替换为代理令牌
3. 容器收到环境变量 `TASK_LLM_PROXY_TOKEN`，在 `Authorization: Bearer <token>` 中使用

### 5.2 Token 验证流程

```
容器请求 ────▶ taskAIEndPoint ────▶ POST /api/internal/task-ai-endpoint/validate-proxy-token/
                     │                         │
                     │                         ├── tenant_id, workspace_id, task_id, proxy_token
                     │                         │
                     │                         ▼
                     │               Django ────▶ Go taskCredentialService (:8015)
                     │                         │     POST /v1/token/validate
                     │                         │          │
                     │                         │          ├── token 无效 → 返回 None
                     │                         │          └── token 有效 → 返回 {tenant_id, workspace_id, task_id}
                     │                         │
                     │                         ├── Go 验证失败 → {"valid": false, "detail": "invalid proxy token"}
                     │                         ├── scope 不匹配 → {"valid": false, "detail": "token scope mismatch"}
                     │                         └── 验证通过 → {"valid": true}
                     │
                     ◀─── 401 Unauthorized（验证失败）
                     ◀─── 继续处理（验证通过）
```

> **注意**: Token 验证已迁移至 Go `taskCredentialService` (端口 `:8015`) 作为 SSOT (单一事实来源)。Django 不再通过 `CloudServerConfig.container_access_token` 直查 token，而是委托 Go 服务校验后再比对 scope。

### 5.3 内部 API 认证

taskAIEndPoint → Django 的内部调用通过 `X-TaskAIEndPoint-Internal-Secret` 共享密钥认证：

```python
# Django 侧校验
def check_task_ai_endpoint_secret(request) -> bool:
    secret = getattr(settings, 'TASK_AI_ENDPOINT_INTERNAL_SECRET', '') or ''
    if not secret:
        return True  # 未配置时放行（仅开发环境）
    return request.headers.get('X-TaskAIEndPoint-Internal-Secret') == secret
```

> ⚠️ **安全提醒**: 生产环境务必设置强随机密钥，本地开发默认值 `task-ai-endpoint-local-dev-secret-do-not-use-in-prod` 仅用于本地调试。

---

## 6. 预算管控

### 6.1 预算门控流程

当 provider 启用了预算 (`budget_enabled: true`)，每次 LLM 调用前会执行预算检查：

```
请求到达 ───▶ resolve_route() ───▶ budget_enabled?
                                        │
                        ┌───────────────┼───────────────┐
                        │ false                         │ true
                        ▼                               ▼
                  跳过预算检查               POST taskCloudService
                                            /api/internal/budget/reserve-or-deny/
                                                      │
                                          ┌───────────┼───────────┐
                                          │ allowed=true           │ allowed=false
                                          ▼                        ▼
                                      继续转发                  402 Payment Required
                                      LLM 调用                  {"error": {"code":
                                      │                          "budget_exhausted",
                                      ▼                          "message": "模型 xxx
                              POST Cloud record-usage            预算已用尽", ...}}
```

### 6.2 预算判断逻辑 (taskCloudService)

```python
# 预算判断核心逻辑（实现已在 Go Cloud；下为语义示意）
effective_limit = resolve_effective_budget_limit(task_row, default_row)
spent = 当前任务该模型的累计消耗

if effective_limit > 0 and spent >= effective_limit:
    return {"allowed": False, "code": "budget_exhausted", ...}
else:
    return {"allowed": True, ...}
```

- 预算粒度：**任务 + 模型** 维度（同一任务的不同模型各有独立预算）
- **不再**经 Django `/api/internal/task-ai-endpoint/budget/*` thin- 预算来源优先级：任务级单独设定 > 工作空间默认值 > 租户默认值
- `effective_limit = 0` 表示不限制

### 6.3 用量记录

LLM 调用完成后，网关从上游响应中提取 `usage.prompt_tokens` 和 `usage.completion_tokens`（同时兼容 `input_tokens`/`output_tokens` 字段名），异步记录至 `TaskModelBudget` 表。使用 **幂等键** (`idempotency_key`) 防止网络重试导致重复记录。

---

## 7. Provider 配置

### 7.1 配置入口

Provider 配置存储在 `TenantFeatureParams.providers` JSON 字段中，格式如下：

```json
{
  "providers": [
    {
      "provider": "deepseek",
      "base_url": "https://api.deepseek.com",
      "api_key": "sk-xxxxxxxxxxxxxxxx",
      "use_sub_token": true,
      "budget_enabled": true
    },
    {
      "provider": "openai",
      "base_url": "https://api.openai.com",
      "api_key": "sk-xxxxxxxxxxxxxxxx",
      "use_sub_token": true,
      "budget_enabled": false
    }
  ],
  "llm_budget_enabled": true
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `provider` | string | 提供商标识，对应 URL 路径中的 `{provider}` |
| `base_url` | string | 上游 LLM API 的基础 URL |
| `api_key` | string | **主密钥** — 仅存储在 Django 端，绝不暴露给容器 |
| `use_sub_token` | bool | 是否通过网关代理（必须为 `true` 才能使用此网关） |
| `budget_enabled` | bool | 是否启用预算控制 |

### 7.2 容器侧 URL 改写

当 `TASK_AI_ENDPOINT_ENABLED=true` 时，容器启动前会自动改写 provider 配置：

```
改写前:
  base_url: https://api.deepseek.com
  api_key:  sk-xxxxxxxxxxxxxxxx

改写后:
  base_url: http://{gateway}:8013/api/tenant/{t}/workspace/{w}/task/{task}/llm/deepseek/v1
  api_key:  <proxy_token>
  proxy_mode: task_ai_endpoint
```

容器代码**无需任何修改** — 它使用标准的 OpenAI SDK，只需将 `base_url` 指向网关即可。

---

## 8. 请求处理流程

一次完整的 LLM 代理请求经过以下阶段：

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 路径解析    parseLlmProxyPath()                          │
│    提取 tenant_id, workspace_id, task_id, provider, subpath │
├─────────────────────────────────────────────────────────────┤
│ 2. Token 提取   extractBearerToken()                        │
│    从 Authorization header 提取 Bearer token                │
├─────────────────────────────────────────────────────────────┤
│ 3. Token 验证   djangoValidateProxyToken()                  │
│    调用 Django 验证 proxy token 有效性 + scope 匹配         │
│    ❌ 401 Unauthorized                                     │
├─────────────────────────────────────────────────────────────┤
│ 4. 路由解析    djangoResolveRoute()                         │
│    获取 upstream_base_url, budget_enabled, use_sub_token    │
│    ❌ 403 Forbidden（use_sub_token=false）                  │
│    ❌ 404 Not Found（provider 未找到）                       │
├─────────────────────────────────────────────────────────────┤
│ 5. Body 解析   parseChatRequest()                           │
│    提取 model 名称，验证 JSON 格式                          │
│    ❌ 400 Bad Request                                      │
├─────────────────────────────────────────────────────────────┤
│ 6. 预算门控    djangoBudgetGate()  [仅 budget_enabled]      │
│    检查任务-模型维度预算是否充足                              │
│    ❌ 402 Payment Required（预算耗尽）                       │
├─────────────────────────────────────────────────────────────┤
│ 7. 凭证获取    djangoUpstreamCredential()                   │
│    获取上游 API Key                                         │
│    ❌ 404 Not Found（凭证未配置）                            │
├─────────────────────────────────────────────────────────────┤
│ 8. 上游转发    forwardUpstream()                            │
│    使用 upstream API Key 转发请求至 LLM 提供商               │
│    - 非流式: 读取完整响应 → 解析 usage → 透传               │
│    - 流式: 逐行 pipe SSE → 实时解析 usage                   │
│    ❌ 502 Bad Gateway（上游不可达/超时）                      │
├─────────────────────────────────────────────────────────────┤
│ 9. 用量记录    djangoCommitUsage()  [仅 budget_enabled]     │
│    记录 input_tokens + output_tokens，幂等去重               │
└─────────────────────────────────────────────────────────────┘
```

---

## 9. 错误处理

### 9.1 HTTP 状态码

| 状态码 | 含义 | 触发条件 |
|--------|------|---------|
| `200 OK` | 成功 | 请求正常完成 |
| `400 Bad Request` | 请求格式错误 | JSON 无效、缺少 `model` 字段、Body 读取失败 |
| `401 Unauthorized` | 认证失败 | 缺少 token、token 无效、scope 不匹配 |
| `402 Payment Required` | 预算耗尽 | 任务-模型维度的预算已达上限 |
| `403 Forbidden` | 禁止访问 | provider 未启用 sub-token (`use_sub_token=false`) |
| `404 Not Found` | 路由未找到 | URL 格式不匹配、provider 未配置 |
| `405 Method Not Allowed` | 方法不允许 | 非 POST 请求 |
| `502 Bad Gateway` | 上游错误 | Django 不可达、上游 LLM 连接失败/超时 |

### 9.2 错误响应格式

```json
{
  "detail": "错误描述信息"
}
```

预算耗尽错误有特殊格式：
```json
{
  "error": {
    "code": "budget_exhausted",
    "message": "模型 deepseek-chat 预算已用尽",
    "spent_amount": "10.50",
    "budget_limit": "10.00",
    "currency": "CNY"
  }
}
```

---

## 10. 配置项

### 10.1 环境变量

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `TASK_AI_ENDPOINT_HOST` | `127.0.0.1` | 监听地址 |
| `TASK_AI_ENDPOINT_PORT` | `8013` | 监听端口 |
| `TASK_AI_ENDPOINT_DJANGO_INTERNAL_API` | `http://127.0.0.1:8001` | Django 内部 API 基地址 |
| `TASK_AI_ENDPOINT_INTERNAL_SECRET` | (从 port_config 读取) | 与 Django 通信的共享密钥 |
| `TASK_AI_ENDPOINT_UPSTREAM_TIMEOUT_SEC` | `300` | 上游 LLM 请求超时（秒） |

### 10.2 配置文件

文件路径: `conf/ai/task-ai-endpoint/config.yaml`

```yaml
host: 127.0.0.1          # 监听地址
port: 8013                # 监听端口
enabled: true             # 是否启用
publicBaseUrl: http://127.0.0.1:8013     # 容器可访问的公开地址
djangoInternalApiBase: http://127.0.0.1:8001  # Django 内部 API
internalSecret: <secret>  # 与 Django 共享的密钥
upstreamTimeoutSec: 300   # 上游超时（秒）
logging:
  log_file: logs/task-ai-endpoint/endpoint.log
```

> **优先级**: 环境变量 > 配置文件。配置文件内 `port`/`host`/`upstreamTimeoutSec` 字段值覆盖环境变量。

### 10.3 Django 侧配置

在 `saas_project/settings.py` 中：

```python
# 是否启用 AI Endpoint 代理（控制容器侧 URL 改写）
TASK_AI_ENDPOINT_ENABLED = True

# 网关公开地址（容器可访问的地址）
TASK_AI_ENDPOINT_PUBLIC_BASE = "http://127.0.0.1:8013"

# 内部 API 认证密钥（必须与 Go 网关的 internalSecret 一致）
TASK_AI_ENDPOINT_INTERNAL_SECRET = "your-secure-random-secret"
```

---

## 11. 可观测性

### 11.1 日志格式

所有日志以结构化 JSON 输出至 stdout，由 runAll/Promtail 采集：

```json
{
  "time": "2026-07-01T10:30:00Z",
  "level": "INFO",
  "msg": "http_request",
  "service": "task-ai-endpoint",
  "trace_id": "abc123def456",
  "otel_trace_id": "abc123def4567890abcdef1234567890",
  "method": "POST",
  "path": "/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1/chat/completions",
  "status": 200,
  "duration_ms": 1523
}
```

### 11.2 链路追踪

- **X-Trace-Id**: 网关自动生成/传播，在响应头和所有内部调用中携带
- **otel_trace_id**: 自动从 `X-Trace-Id` 派生 32 字符 hex 值，兼容 OpenTelemetry / Tempo
- 可通过 Grafana Loki 按 `trace_id` 跨服务搜索完整请求链路

### 11.3 关键日志字段

| 字段 | 说明 |
|------|------|
| `trace_id` | 外部 trace ID（32 字符 hex 或自定义格式） |
| `otel_trace_id` | OpenTelemetry 兼容的 32 字符 hex trace ID |
| `service` | 固定为 `task-ai-endpoint` |
| `forward_stage` | 阶段标记，用于追踪请求处理的各个步骤 |
| `duration_ms` | 请求处理耗时（毫秒） |

---

## 12. 使用示例

### 12.1 基本调用

```bash
# 非流式 Chat Completions
curl -X POST http://127.0.0.1:8013/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1/chat/completions \
  -H "Authorization: Bearer <proxy_token>" \
  -H "Content-Type: application/json" \
  -H "X-Trace-Id: my-trace-001" \
  -H "X-Request-Id: req-001" \
  -d '{
    "model": "deepseek-chat",
    "messages": [
      {"role": "user", "content": "用一句话解释什么是 Go 语言"}
    ],
    "stream": false,
    "max_tokens": 100
  }'
```

### 12.2 流式调用

```bash
# SSE 流式 Chat Completions
curl -N -X POST http://127.0.0.1:8013/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1/chat/completions \
  -H "Authorization: Bearer <proxy_token>" \
  -H "Content-Type: application/json" \
  -H "X-Trace-Id: my-trace-002" \
  -d '{
    "model": "deepseek-chat",
    "messages": [
      {"role": "user", "content": "写一首关于编程的五言绝句"}
    ],
    "stream": true
  }'
```

### 12.3 Python SDK 集成

```python
from openai import OpenAI

# 网关地址 + proxy token
client = OpenAI(
    base_url="http://127.0.0.1:8013/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1",
    api_key="<proxy_token>",  # 代理令牌，非上游 API Key
)

# 非流式调用
response = client.chat.completions.create(
    model="deepseek-chat",
    messages=[
        {"role": "user", "content": "Hello!"}
    ],
)

print(response.choices[0].message.content)
print(f"Tokens used: {response.usage.total_tokens}")

# 流式调用
stream = client.chat.completions.create(
    model="deepseek-chat",
    messages=[{"role": "user", "content": "讲个笑话"}],
    stream=True,
)

for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### 12.4 Node.js / TypeScript 集成

```typescript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://127.0.0.1:8013/api/tenant/t1/workspace/w1/task/k1/llm/deepseek/v1",
  apiKey: "<proxy_token>",
});

const completion = await client.chat.completions.create({
  model: "deepseek-chat",
  messages: [{ role: "user", content: "Hello!" }],
});

console.log(completion.choices[0].message.content);
```

### 12.5 容器内自动配置

容器启动时，环境变量自动注入：

```bash
# 容器环境变量
TASK_AI_ENDPOINT_BASE_URL=http://gateway:8013
TASK_LLM_PROXY_TOKEN=<proxy_token>

# provider 配置已被改写，容器代码无需任何修改即可通过网关调用 LLM
```

---

## 13. 故障排查

### 13.1 常见问题

| 问题 | 可能原因 | 解决方法 |
|------|---------|---------|
| `401 missing proxy token` | 请求未携带 Authorization header | 确保 `-H "Authorization: Bearer <token>"` |
| `401 invalid proxy token` | proxy token 过期或不存在 | 检查 token 是否有效，任务是否已停止 |
| `401 token scope mismatch` | 路径中的 tenant/workspace/task 与 token 不匹配 | 检查 URL 路径参数是否正确 |
| `403 provider is not sub-token enabled` | provider 的 `use_sub_token=false` | 将 provider 设置为 sub-token 模式 |
| `402 budget_exhausted` | 任务在对应模型上的预算已耗尽 | 增加预算或使用其他模型 |
| `404 Not Found` | URL 路径格式不正确或 provider 未找到 | 检查 URL 格式是否匹配路由规则 |
| `502 django unreachable` | Django 内部 API 不可达 | 确认 saas-backend 已启动，端口 8001 |
| `502 upstream error` | 上游 LLM 服务不可达 | 检查网络和 provider base_url |

### 13.2 日志排查

```bash
# 查看服务日志
tail -f logs/task-ai-endpoint/endpoint.log

# 按 trace_id 搜索完整请求链路
grep "abc123" logs/task-ai-endpoint/endpoint.log

# 通过 Loki 搜索（如果已配置）
# 在 Grafana Explore 中:
# {job="task-ai-endpoint"} | json | trace_id="abc123"
```

### 13.3 调试检查清单

1. ✅ 确认 `saas-backend` 已启动且健康
2. ✅ 确认 `internalSecret` 在网关配置和 Django settings 中一致
3. ✅ 确认 provider 的 `use_sub_token=true`
4. ✅ 确认 proxy token 与 URL 中的 tenant_id/workspace_id/task_id 匹配
5. ✅ 确认上游 LLM provider 的 `base_url` 和 `api_key` 正确配置

---

## 参考链接

- **设计文档**: `docs/superpowers/specs/2026-05-30-task-ai-endpoint-budget-gateway-design.md`
- **实施计划**: `docs/superpowers/plans/2026-05-30-task-ai-endpoint-plan.md`
- **源代码**: `taskAIEndPoint/src/`
- **Django 内部 API**: 已删除（validate/route/credentials/budget 均迁 Go）
- **Proxy rewrite**: `taskCloudService/src/feature_params_proxy_rewrite.go`
- **配置文件**: `conf/ai/task-ai-endpoint/config.yaml`

## License

本仓库以 MIT License 授权，见 [LICENSE](./LICENSE)。
