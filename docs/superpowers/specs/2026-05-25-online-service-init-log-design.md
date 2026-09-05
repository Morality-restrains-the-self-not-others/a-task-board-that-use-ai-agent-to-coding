# onlineServiceJS 启动环境快照日志（init.log）设计

日期：2026-05-25  
状态：待审批  
范围：`trae-agent/onlineServiceJS` 服务启动时写入 `logs/init.log`，记录本次启动使用的环境变量及其值

## 1. 背景与目标

当前 `onlineServiceJS` 启动后仅有运行日志与请求摘要日志，缺少“启动时实际使用了哪些环境变量”的证据。  
在排查 `DEBUG_AGENT`、`TASK_API_ENDPOINT_ORIGIN` 等配置生效问题时，需要先确认进程启动输入。

目标：

1. 在 `onlineServiceJS` 启动流程尽早落盘 `logs/init.log`。
2. 每次启动写入一条完整快照，便于回溯同一任务多次重启差异。
3. 保持现有启动流程与请求处理行为不变（仅新增可观测性）。

## 2. 设计方案（草案）

## 2.1 日志位置与格式

- 文件：`{ONLINE_PROJECT_STATE_ROOT}/logs/init.log`
- 每次启动追加写入一段结构化文本（单行 JSON）：
  - `ts`
  - `event`（固定 `onlineServiceJS.init`）
  - `pid`
  - `port`
  - `env`（本次启动环境键值对）

示例（说明格式，不代表最终字段全集）：

```json
{"ts":"2026-05-25T03:21:12.701Z","event":"onlineServiceJS.init","pid":12345,"port":"8765","env":{"DEBUG_AGENT":"True","TASK_API_ENDPOINT_ORIGIN":"http://api.daydaymoney.com","BUSINESS_API_ENDPOINT_ORIGIN":"http://127.0.0.1:8765","ACCESS_TOKEN":"..."}}
```

## 2.2 写入时机

- 在 `server.mjs` 的 `main()` 开始阶段执行（`runBootstrapTokenExchangeOnly()` 前）。
- 这样可以覆盖：
  - 正常启动
  - bootstrap 失败后退出
  - strict/non-strict 分支

## 2.3 环境变量采集策略

- 默认采集 `process.env` 全量键值（按你的需求“记录启动时用到的环境变量以及值”）。
- 为避免日志过大，提供一层可选治理（默认不开启）：
  - `INIT_LOG_ENV_KEYS`（若配置则仅记录白名单键）
- 不改变 `DEBUG_AGENT` 的“调试不脱敏不截断”原则；`init.log` 也按原值记录。

## 2.4 与现有日志关系

- `logs/requests.log`：继续保留请求摘要行。
- `reqLogs/outbound.log`：继续记录出站/调试 HTTP 细节。
- `logs/init.log`：新增“启动输入快照”维度，用于解释为何调试日志分支是否生效。

## 3. Domain Concept Inventory（供后续 DDD 使用）

### Bounded Contexts

- Relay 启动编排上下文（传递 env）
- onlineServiceJS 运行时上下文（消费 env 并启动服务）
- 可观测性上下文（启动快照与请求日志关联）

### Key Entities

- `OnlineServiceInitSnapshot`
- `OnlineServiceRuntimeEnv`

### Candidate Aggregates

- `OnlineServiceStartupAggregate`（聚合一次启动快照与启动结果）

### Domain Events

- `OnlineServiceInitLogged`
- `OnlineServiceBootstrappingStarted`

## 4. Value Stream Impact Analysis

受影响现有价值流（基于 `value-stream.yaml`）：

1. `task-detail-relay-debug-agent-observability`（直接影响：启动可观测性增强）
2. `task-detail-runtime-relay`（间接影响：排查 relay 直启参数透传）

字段影响：

- 新增文件维度：`onlineProject_state.logs.init.log`
- 不改动数据库字段。

测试影响（建议）：

- 新增 `onlineServiceJS` 单测：验证启动时写入 `logs/init.log`，且包含 `DEBUG_AGENT` 等关键键。
- 回归现有 `server.debugAgentInbound.test.mjs` / `saasTaskCloud.debugAgentOutbound.test.mjs`（确认无副作用）。

状态变化建议：

- 不新增独立价值流；在 `task-detail-relay-debug-agent-observability` 下新增一个 planned step（如 `relay-debug-agent-init-snapshot`）。

## 5. 风险与兼容性

1. **日志体积增长**：全量 env 可能较大；可通过 `INIT_LOG_ENV_KEYS` 白名单压缩。
2. **敏感值落盘**：按当前需求保留原值；后续若要治理可新增脱敏开关，不影响本次设计目标。
3. **启动性能影响**：仅一次 append 写入，影响可忽略。
