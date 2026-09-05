# relay status-push timeout（Go relay）设计

## 背景与问题

在任务详情页执行「直接启动 + 刷新状态」后，`go_relayToTrae` 日志出现连续超时：

- `[relayToTrae] status push failed: ... context deadline exceeded`
- `[relayToTrae] 5 consecutive missed ACKs, unregistering task ...`

该问题导致 relay 误判任务下线并注销注册，前端状态无法持续收敛。

## 目标

修复 `go_relayToTrae` 的 status-push 网络出站策略，避免被系统代理环境劫持导致超时，从而消除误注销。

## 根因分析

- 当前 Go 实现的 status-push 使用默认 `http.Client` 行为。
- 默认行为会读取 `HTTP_PROXY/HTTPS_PROXY` 等环境变量。
- 在本地/混合网络场景下，status-push 请求可能经过不稳定代理链路，出现等待响应头超时。
- 连续超时会被当作 missed ACK 计数，达到阈值后触发注销。

## 设计方案（已批准）

1. 对 Go relay 的后端出站调用统一采用 no-proxy transport（`Transport.Proxy = nil`）。
2. 覆盖两类关键链路：
   - `status-push`（`go_relayToTrae/src/push.go`）
   - token-exchange / refresh-access（`go_relayToTrae/src/token.go`，保持一致性）
3. 保持协议与业务语义不变：
   - 不修改 `seq/ack/status` 契约
   - 不修改 missed-ack 阈值逻辑
   - 不新增数据库字段

## 非目标

- 不调整 Django `TASK_API_ENDPOINT_ORIGIN` 选择策略
- 不引入新的重试退避策略
- 不改变 ACK 判定规则

## 验收标准

1. 在存在代理环境变量时，status-push 仍可直连目标 API，不再出现连续 `context deadline exceeded`。
2. 不再出现因该类超时触发的 `5 consecutive missed ACKs` 误注销。
3. Go 单测通过，且新增 no-proxy 行为测试覆盖关键路径。

## 风险与回滚

- 风险：若某部署强依赖代理访问 API，禁用代理后可能无法连通。
- 缓解：仅作用于 relay 的后端状态/换票调用链路，影响范围小；必要时可通过回滚该改动恢复原行为。

## 领域概念清单（用于后续 DDD 输入）

- Bounded Context
  - Relay Runtime（relay 进程与任务注册）
  - Relay Status Convergence（status-push 到任务状态收敛）
- Key Entities
  - `RegisteredTask`（任务注册状态、ACK 计数）
  - `relayState`（进程运行态与状态快照）
- Candidate Aggregates
  - `RegisteredTask` 作为 ACK 计数与注销决策一致性边界
- Domain Events
  - `status_push_ok`
  - `status_push_failed`
  - `task_unregistered_after_missed_ack`
