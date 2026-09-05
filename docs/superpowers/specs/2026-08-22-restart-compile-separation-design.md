# 全部重启与编译分离：禁止对未完成符号依赖的脏树编译

- **日期:** 2026-08-22
- **状态:** accepted（总体设计审批：approve_full，2026-08-22）
- **作者:** cursor
- **相关:** `.ai/09_failure_experience/02_runtime_errors/110_task_status_changed_dlt_cloud_refused_retry_backoff_reset.md`；OPT-20260822-002；约束 42；OPT-20260810-005（kill-first）
- **ADR:** [ADR-0027](../../adr/0027-restart-compile-separation.md)
- **架构视图:** **不新增** v94 四件套（无新服务/无新 Rel_Flow；仅修订既有 runAll 生命周期语义）

## 当前架构理解

根据 current 架构设计稿（v93 ✅ current，2026-08-21 16:35，迭代 image-container-skill-list）：

- 共有 2 个 current 视图：`enterprise-landscape`、`application-integration`
- 业务层：厂商登记镜像、租户浏览市场、创建任务/评论（与本次无关）
- 应用层：taskAiProvider / taskCloudService / taskFE / taskTaskService 等；**runAll :9999** 是进程编排器（不在 v93 局部图中展开，但托管全部 `conf/runAll.yaml` 服务，含 `task-events-*`）
- 技术层：MySQL / Redis / Kafka / Loki；领域事件消费者由 `taskEvents/run.sh` 启停
- 上次交付版本：v93；另有未交付 target 积压（v89/v90 等）

📋 架构版本历史（最近）：

- v93 (2026-08-21 16:35) ✅ current — 镜像容器技能列表
- v92 (2026-08-20 21:40) ✅ previous — 多渠道推荐码

本次需求在此基础上**只改 runAll 生命周期与 taskEvents start 脚本**，不新增组件。

## 问题现象

全部重启（`POST /api/restart-all`）进行到 start 阶段时，`task-events-task-status-changed-2-fanout-work-panel-sse` 报 `READINESS_TIMEOUT`，stderr：

```
consumer/retry_wait.go:16:16: undefined: broker.DelayForAttempt
```

用户判断：重启只应拉起**已有二进制**；编译产物应是原子替换；Go 改完应先验证符号依赖。这三条都成立。现场**不是**「半截 ELF 被 exec」。

## 🔍 问题分析（非 torn binary）

### 用户三条约束 vs 现场

| 用户约束 | 现场实际 | 是否命中 |
|---------|---------|---------|
| 重启只执行二进制重启 | `RestartAll` = `StopAll` + `StartAll`。多数 Go 服务 `start_command` 是 `./bin/foo`（不编译）。**task-events 的 `start_command` 是 `bash run.sh start …`，内部强制 `build_intent`（`go build`）** | 命中 |
| 编译生成二进制应原子 | `go build -o $out` 失败时**不覆盖**旧产物（Go 写临时文件再 rename）。fanout 二进制当时仍在磁盘上 | 原子写已成立；空窗来自**先杀进程** |
| Go 修改完成应验证符号依赖 | TDD 先写入调用 `broker.DelayForAttempt` 的 `retry_wait.go`，函数尚未进 `broker/retry.go`。并发 restart-all 编译了**不一致源码** | 命中 |

### 因果链

```
Agent TDD 半步：retry_wait.go 引用 DelayForAttempt（定义尚未写入）
        ↓ 并发
restart-all stop 阶段：kill 旧 fanout 进程（端口空）
        ↓
restart-all start 阶段：start_command → run.sh start → build_intent
        ↓
go build 失败：undefined: broker.DelayForAttempt（set -e，不启动）
        ↓
旧进程已死 + 新进程未起 → READINESS_TIMEOUT / connection refused
        ↓
放大：TASK_STATUS_CHANGED 释放路径 20s 内打满 DLT（见失败经验 110）
```

`go build` 的原子 rename **救不了**这件事：编译从未成功，而旧进程在 stop 阶段已经被杀掉。磁盘上的 last-good 二进制被 `set -e` 挡住，根本没被 exec。

### 与 kill-first 的关系

OPT-20260810-005 把单服务 `restartService` 改成 **先停再编**：构建失败时服务停机。约束 42 正文仍写「编译 → 停止 → 启动」，与代码 **stop → build → start** 漂移。

`RestartAll` **不走** `restartService`，但 task-events 把编译藏进了 `start_command`，效果等价于「全量停机后再对脏树编译」。

精准编译重启对 task-events 还会 **编两次**：`restartService` 跑 `build_command`（`run.sh build`），随后 `start_command` 再 `build_intent`。

## 🕸️ Code Review Graph 分析

CRG 图存在（`.code-review-graph/graph.db`，108 节点 / 17 文件，语言 javascript/typescript/python/bash，提交 `3a6d8f0a3b70`）。

`CRG unavailable for this change: 图未索引 Go（runAll/src、taskEvents）。` 爆炸半径按源码检索：

| 符号/入口 | 位置 | 影响 |
|----------|------|------|
| `Runner.RestartAllWithActor` | `runAll/src/runner_restart_all.go` | StopAll → StartAll，不调用 `BuildService` |
| `Runner.startService` | `runAll/src/runner.go` | 只跑 `start_command`，不跑 `build_command` |
| `Runner.restartService` | `runAll/src/runner.go` | kill-first：stop → `runBuild(build_command)` → start |
| `build_intent` / `start_intent` | `taskEvents/run.sh` | `start` **总是先 compile**；`set -euo pipefail` |
| `DelayForAttempt` | `taskEvents/broker` + `consumer/retry_wait.go` | 已合入；事故窗口是写入顺序 |

## 决策（拟锁定，待审批）

### D1. 重启与编译分离

| 操作 | 允许编译？ | 行为 |
|------|-----------|------|
| 全部重启 / 启动全部 / 单服务启动/停止 | **否** | 只 stop/start **磁盘上已有二进制**（last-good） |
| 全部重新编译 / 分组编译 / 单服务编译 | **是** | 只写磁盘，不杀进程 |
| 精准编译重启 | **是** | 见 D2：先原子编译，成功后再切进程 |

### D2. 编译原子 + 先编后切（纠正 kill-first 在「需要新二进制」路径上的空窗）

1. 编译输出到 `bin/foo.new`（或 Go 默认临时文件），成功后再 `rename` 到 `bin/foo`。失败则 last-good 不变。
2. **精准编译重启**：保持旧进程服务，直到新二进制 rename 成功；再 stop + start 新文件。编译失败 → 旧进程继续、登记保留、标记 failed（与 taskFE `skip_stop_on_restart` 热替换同构，推广到非 detach 的 Go 进程）。
3. 约束 42 与 `precise_restart.go` 注释改为与代码一致：**编译 → 停止 → 启动**（废除该路径上的 kill-first）。
4. 单服务「↻ 重启」（不带编译意图）与全部重启一样：**不编译**。需要新代码走精准编译重启或先点编译再重启。

### D3. start 脚本禁止隐式编译

`taskEvents/run.sh` 的 `start_intent` **删除** `build_intent` 调用。没有二进制则明确失败（「先 build 或精准编译重启」），不得在 start 时对工作树 `go build`。

举一反三：所有 `start_command` 含 `go build` / `./build.sh` 的服务同样拆开（检索 `conf/runAll.yaml` 的 start vs build）。

### D4. Go 修改完成 = 模块可编译（符号依赖门）

TDD 允许测试失败（红灯断言），**不允许**模块 `go build` 失败（`undefined:` 等）。

智能体写 Go 时：

1. 同一批写入必须同时包含新符号的定义与引用（禁止跨工具调用留下 `undefined` 窗口）。
2. 每批 Go 源码落地后、让出控制权或登记精准重启之前，对该模块执行 `go test`/`go build`（编译通过即可；测试可红）。
3. 写入约束：修订 `.ai/01_project_constraints/42_precise_restart_service_registration.md` 或新增短约束；失败经验 111。

不采用「runAll 只编 git HEAD」：精准编译重启的存在就是为了脏 WIP；根因是 **重启路径不该编**，以及 **Agent 不得留下不可编译树**。

## 备选（拒绝）

| 方案 | 拒绝原因 |
|------|---------|
| 只把 `go build -o` 改成显式 tmp+mv | 事故不是 torn ELF；stop 后编译失败仍空窗 |
| 保持 kill-first，仅加更长 readiness | 脏树编译失败时永远起不来 |
| 全部重启也编译（当前 task-events 行为） | 与「重启=二进制」冲突，放大 DLT |
| 源码 flock 等 Agent 写完 | 编辑器无法可靠参与；不如禁止 start 编译 + 可编译门 |

## 业务意图 → 事件对照

运维编排，不改变租户/任务/计费事实。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 全部重启只拉起 last-good 二进制 | — | runAll RestartAll | — | 运维进程生命周期，无对应 MQ |
| 精准编译重启先原子编译再切进程 | — | runAll PreciseRestart | — | 同上 |
| start 不再隐式 go build | — | taskEvents/run.sh | — | 同上 |

## Domain Concept Inventory

- **Bounded Context:** 平台运维编排（runAll），非业务域
- **Key Entities:** ManagedService、last-good Binary、RegistrationFile
- **Aggregates:** 单服务生命周期（stop/build/start 不得交叉污染）
- **Domain Events:** 无（见上表例外）

## 🐍 Python 新增接口

不触发。无新 Django/Flask endpoint。

## Value Stream 影响

现有流（`conf/value-stream.yaml`）：

- `runall-cascade-lifecycle` / `runall-global-start-stop-all`：start/restart **不再编译**
- `runall-build-all-progress` / `runall-build-group-progress`：仍是唯一全量编译入口
- `runall-precise-restart-reload-yaml`：精准重启语义改为 compile-then-swap；新增步骤建议：`runall-restart-does-not-compile`、`runall-precise-restart-compile-then-swap`、`task-events-start-no-build`

无新业务域；字段仍是 `runall.runtime.*` / 各服务 `runtime.lifecycle_status`。

## 🏛️ 架构变更影响

- **不创建** v94 `.puml` / `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **原因:** 无 Application_Component 增删，无跨服务 Rel_Flow 变更；runAll 内部启停语义用 ADR-0027 + 约束 42 记录
- 批准后：ADR-0027 proposed→accepted（随实现）；修订约束 42；失败经验 111

## 实施切片（批准后 /8-build）

1. 表征测试：RestartAll/StartAll **不**调用 `runBuild`；task-events fixture 的 start 脚本在无 binary 时失败且不 invoke go
2. `taskEvents/run.sh`：`start_intent` 去掉 `build_intent`；`build` 子命令保持
3. `restartService`：无「编译意图」时跳过 `runBuild`；精准重启走 compile-then-swap
4. `conf/runAll.yaml` 审计：start_command 不得再包编译
5. 单测：编译失败时旧进程仍 healthy（精准路径）；restart-all 不读 Go 源码
6. 文档：约束 42、失败经验 111、OPT-20260810-005 在编译路径上被 ADR-0027 部分取代

## 风险

| 风险 | 缓解 |
|------|------|
| 全部重启后仍跑旧二进制，开发者以为代码已生效 | UI 文案：重启 ≠ 编译；改代码必须精准编译重启 |
| 纠正 kill-first 后旧进程残留（OPT-20260810-005 原痛点） | 编译成功后再 `forceFreshStart` 强杀端口监听者；保留端口扫尾 |
| task-events 冷启动无 bin | build-all / 精准重启 / 显式 `run.sh build`；start 给出明确错误 |
| Agent 仍跨文件半写入 | D4 + 失败经验；不把 runAll 当类型检查器 |

## 澄清待定（审批选项）

1. 单服务 ↻ 是否也禁止编译？（推荐：是，与全部重启一致）
2. 是否保留「编译失败则停机」作为显式开关？（推荐：否，编译失败必须保留 last-good 进程）
