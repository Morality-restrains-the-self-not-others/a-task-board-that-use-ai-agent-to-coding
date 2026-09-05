# onlineServiceJS DEBUG_AGENT 调试日志设计

日期：2026-05-25  
状态：已批准  
范围：`trae-agent/onlineServiceJS` 出入站详细调试日志 + `relayToTrae=true` 直接启动默认注入 `DEBUG_AGENT=True`

## 1. 背景

当前 `onlineServiceJS` 的请求日志以摘要为主，排查跨服务调用问题时缺少完整上下文。  
本需求引入 `DEBUG_AGENT` 环境变量，在调试场景下记录完整请求/响应细节。

## 2. 目标与约束

### 2.1 目标

1. 当 `DEBUG_AGENT=True` 时，记录对外请求的完整入参和出参。
2. 当 `DEBUG_AGENT=True` 时，记录调用 `onlineServiceJS` 的请求入参与响应出参。
3. `relayToTrae=true` 页面直接启动时，默认携带 `DEBUG_AGENT=True` 并传入 `onlineServiceJS`。

### 2.2 关键约束

1. 调试日志字段 `method/url/headers/body` 按用户要求不做脱敏、不做截断。
2. 默认行为保持兼容：未开启 `DEBUG_AGENT` 时仍为现有摘要日志。
3. 不改动现有 token exchange、relay status push 主流程。

## 3. 设计方案

## 3.1 开关与布尔语义

- 新增 `DEBUG_AGENT` 开关，支持 `1/true/yes/on`（忽略大小写）为启用。
- 由 `outboundReqLog` 提供统一 `isDebugAgentEnabled()` 判定能力。

## 3.2 出站调用日志增强

在以下模块对 `fetch` 调用增加 debug 日志：

- `saasTaskCloud.postJson`
- `layerGitOauthPush.createGithubPullRequest`
- ~~`reachability.fetchPublicIpv4Domestic`~~（已移除：公网 IP 仅由 `TRAE_PUBLIC_IP` / `PUBLIC_IP` / 非 loopback `BUSINESS_API_ENDPOINT` 注入，镜像内不再外网探测）
- `stagedCommitSuggest.callOpenAiCompatibleChat`
- `server.callOpenAiCompatibleChat`

记录项：

- 请求：`method/url/headers/body`
- 响应：`status/headers/body`
- 异常：`method/url/error`

## 3.3 入站调用日志增强

在 `server.mjs` 增加 Express 中间件（仅在 `DEBUG_AGENT=True` 生效）：

- 记录请求 `method/url/headers/body`
- Hook `res.json` / `res.send` 记录响应 `status/headers/body`

## 3.4 relay 直启默认注入

- `relayToTraeUtils.RELAY_TO_TRAE_ENV_KEYS` 新增 `DEBUG_AGENT`
- `buildDefaultRelayToTraeEnvItems` 中默认值设为 `True`
- `ServerConfig.logic.vue` 继续透传 env，无需协议变更
- `relayToTrae/server.py` 已支持透传子进程环境，无需额外改造

## 4. 测试策略

1. 单测：`relayToTraeUtils` 增加 `DEBUG_AGENT` 默认值与键顺序断言。
2. E2E：`TaskDetail.relay-to-trae-direct-start.playwright` 断言 start payload 包含 `env.DEBUG_AGENT === 'True'`。
3. 回归：确认 `DEBUG_AGENT` 关闭时原摘要日志不受影响。

## 5. Domain Concept Inventory（供 DDD 步骤输入）

### Bounded Contexts

- 任务协作（TaskDetail 直启参数组装）
- Relay 编排（relayToTrae 启停与 env 透传）
- 可观测性（onlineServiceJS 请求生命周期日志）

### Key Entities

- `RelayStartEnv`
- `OnlineServiceDebugSwitch`
- `HttpDebugLogEntry`

### Candidate Aggregates

- `RelayRuntimeAggregate`
- `OnlineServiceRequestLogAggregate`

### Domain Events

- `RelayStartRequested`
- `OnlineServiceInboundRequestHandled`
- `OnlineServiceOutboundRequestCompleted`

## 6. Value Stream Impact Analysis

本需求影响现有价值流：

1. `task-detail-runtime-relay`（直接影响：直启默认注入 DEBUG_AGENT）
2. `task2app-outbound-governance`（直接影响：出站调试日志增强）

字段影响（日志/配置维度）：

- `runtime-env.DEBUG_AGENT`
- `reqLogs.outbound.log`（新增 debug 请求/响应明细）
- `logs/requests.log`（保留摘要；debug 明细同时写 reqLogs）

测试影响：

- `front_project/app/src/utils/relayToTraeUtils.test.js`
- `playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`

状态变化：

- 不新增价值流，基于现有 active 流做观测性增强。
