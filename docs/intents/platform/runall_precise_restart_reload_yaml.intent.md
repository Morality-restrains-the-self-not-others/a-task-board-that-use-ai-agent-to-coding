# Intent: 精准编译重启前重载 runAll.yaml

## 背景与目标

runAll 进程启动时把 `conf/runAll.yaml` 读入内存。之后 YAML 新增的 `task-events-*` 等服务会被自动登记脚本写入 `.runall/precise_restart_services.txt`，但精准编译重启只查内存配置，于是报 `unknown service (not in runAll config)`。

目标：点击「精准编译重启」时，先从磁盘重载 YAML，再解析登记名，使新服务可编译重启，无需为此单独重启 runAll。

## 范围与边界

- **范围内**：`Runner.PreciseRestart` 在解析登记名之前重载磁盘配置；新服务写入 StatusStore（不覆盖已有运行状态）；真正不在磁盘 YAML 中的名字仍失败并保留登记。
- **范围外**：不在每次 GET `/api/precise-restart/registrations` 轮询时重载（避免 2s 轮询反复 `LoadConfig`）；不处理 YAML 中删除服务的卸载。

## 约束与风险

- 重载失败（YAML 非法 / DAG 环）须保留原内存配置并打 warn，不得中断已知服务的重启。
- `StatusStore.Init` 会把已有服务打回 pending，禁止用于热加载；只能 Ensure 缺失名。
- 热替换仍须使用含 skip-orphan 标记的 `bin/runAll`。

## 验收标准

1. 内存配置无某服务、磁盘 YAML 有该服务时，精准编译重启不再报 `unknown service`。
2. 磁盘 YAML 也没有的登记名仍报 unknown，并作为 failed 保留在登记文件。
3. 重载失败时已知服务仍按原配置继续重启。
4. 新服务在 StatusStore 中以 pending 出现，且不把已有 healthy 服务打回 pending。

## 实施计划

1. `reloadConfigFromDisk` + `StatusStore.EnsureNames`
2. `PreciseRestart` 解析前调用
3. 单元测试覆盖过期内存 / 非法 YAML / 真未知名
4. 热替换 runAll 使现网加载含 `task-events-sse-message-2-persist-job-execution-event` 的 YAML

## 业务意图 → 事件对照

> 运维编排器热加载配置，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 精准编译重启前重载 runAll.yaml | — | — | — | — | 运维编排器配置热加载，不产生业务领域事件 |

## 变更记录

- 2026-08-16：新增。根因：taskEvents 脏树自动登记磁盘 YAML 新消费者，而 :9999 进程内存配置未包含该服务。
