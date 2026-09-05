# runAll 显式生命周期命令（YAML 四条命令）设计

**日期：** 2026-05-31  
**状态：** 已实施（Increment 1–3 + 部分 4；见 plan Task 11 文档项）  
**决策：** 用户选择 **方案 B** — 每个服务在 YAML 中显式配置 **`start_command` / `stop_command` / `build_command`（可选）**（可指向各自脚本）；**重启**由 runner 编排为 **先 `stop_command` 再 `start_command`**（不配置 `restart_command`）。runAll runner 作为薄执行层，不再依赖命令推断与 SIGTERM 特例。

---

## 1. 背景与问题

### 1.1 现状

runAll Web UI（`:9999`）提供：启动、关闭、重启、编译、启动/关闭本组。当前实现混合了多种路径：

| 操作 | 当前实现 | 问题 |
|------|----------|------|
| 启动 | YAML `command` → `exec sh -c` | 与关闭/重启语义割裂 |
| 关闭 | 优先 `SIGTERM` 跟踪进程；对含 `run.sh` 的服务**推断** `bash run.sh stop` | 推断不完整（如 docker-kafka 曾出现 UI 已停、容器仍运行） |
| 重启 | 可选 `build_command` 推断 → 杀进程 → 再跑 `command` | 不调用服务侧 `restart` 脚本 |
| 编译 | `build_command` 或从 `build.sh` / `go build` / 解析 `run.sh` 推断 | UI「可编译」规则不透明 |

部分服务已有良好脚本契约（`dockerInfra/*/run.sh`、`AiMonitor/run.sh`、`taskEvents/run.sh`），部分服务仅有内联命令（`saas-backend`）或裸二进制（`taskAuth/run.sh` 无 stop）。

### 1.2 目标

- **每个 UI 按钮对应一条显式命令**（通常指向该服务目录下的脚本）。
- **可复现：** 开发者可在 `working_dir` 内手动执行与 UI 相同的命令排障。
- **可测试：** runner 单测验证「给定四条命令 → 调用正确 shell」；脚本单测验证业务副作用（容器 down、端口释放等）。

### 1.3 非目标（本阶段不做）

- 不改变 DAG / `depends_on` / 级联启停策略。
- 不引入远程/SSH 执行。
- 不统一所有服务为单一 `run.sh` 入口（允许 `scripts/start.sh` 等多文件，由 YAML 指向即可）。

---

## 2. 方案对比（在 B 框架内的实现变体）

| 变体 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| **B1 扁平四字段（推荐）** | 服务级 `start_command` / `stop_command` / `restart_command` / `build_command` | 与现有 `build_command` 一致；易 grep；runAll.yaml 一眼可读 | 字段略多 |
| **B2 `lifecycle:` 块** | 四条命令嵌套在 `lifecycle:` 下 | 配置分组清晰 | 与「显式四条」字面略远；迁移改动面更大 |
| **B3 仅脚本路径** | YAML 只写 `start_script: run.sh`，子命令固定 | 最短 YAML | 灵活性差；taskEvents 等多参数域不适用 |

**推荐 B1**：与用户选择的「YAML 显式四条命令」一致，且与现有 `build_command` 字段自然扩展。

---

## 3. 配置模型

### 3.1 Service 结构（新字段）

```yaml
- name: docker-kafka
  working_dir: dockerInfra/kafka
  start_command: "bash run.sh managed"
  stop_command: "bash run.sh stop"
  build_command: ""                    # 空 = UI 不显示「编译」
  launch_mode: detach                  # 见 3.3
  health_check: { ... }
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `start_command` | 是 | UI「启动」、DAG 层启动、组启动 |
| `stop_command` | 是 | UI「关闭」、组关闭；**必须**由脚本完成真实停机（含 `compose down`） |
| `build_command` | 否 | 空则 UI「编译」禁用；非空则仅编译不重启进程 |
| `launch_mode` | 否 | `attach`（默认）或 `detach`；替代对 command 字符串的启发式判断 |
| `restart_command` | 否 | **已废弃，不配置。** UI「重启」由 runner 组合 `stop_command` → `start_command`（见 §4.4） |

### 3.2 弃用与兼容（一个发布周期）

| 旧字段 | 处理 |
|--------|------|
| `command` | 若 `start_command` 为空，加载时复制 `command` → `start_command` 并打 warn 日志；下一主版本删除 `command` |
| `build_command` | 已存在则保留；与新区一致 |
| runner 推断 | `composeStopShellCommand`、`resolveBuildCommand`、`extractGoBuildFromRunScript` 等在兼容期作为 **fallback**；四条命令齐全后移除 |

### 3.4 配置校验（strict，默认开启）

**已确认：** 迁移期不采用 warn 模式；缺必填生命周期命令时 **直接拒绝加载配置**。

`LoadConfig` / `validate()` 对每个服务检查：

| 字段 | strict 要求 |
|------|-------------|
| `start_command` | 必填（或由 `command` 别名填充后仍不得为空） |
| `stop_command` | 必填 |
| `build_command` | 可选（空 = 不可编译） |
| `restart_command` | 若存在则加载时 **warn 并忽略**（向后兼容旧草稿）；不得依赖 |

失败示例：

```text
service "docker-kafka": stop_command is required (lifecycle strict mode)
```

- 默认 **始终 strict**；仅当显式设置环境变量 `RUNALL_LIFECYCLE_STRICT=0` 时降级为 warn（供本地临时调试，**不推荐**提交到仓库）。
- strict 失败发生在 **runAll 进程启动前**，避免 UI 显示可关闭但后端无法执行 `stop_command` 的半配置状态。

### 3.3 `launch_mode: detach`

用于 Docker Compose / `run.sh managed` 等「启动脚本很快退出、服务由后台容器/进程托底」的场景：

- `attach`：启动进程退出且健康检查未通过 → 失败（现状非 detach 行为）。
- `detach`：启动进程退出后 **继续** 按 `health_check` 探测直至成功或超时（替代 `isDetachLaunchCommand(svc.Command)` 字符串匹配）。

---

## 4. Runner 行为（薄执行层）

### 4.1 统一执行器

新增 `runLifecycleCommand(ctx, svc, kind)`：

- `working_dir` 内执行 `sh -c "<command>"`
- 继承 `env`
- 合并 stdout/stderr 到服务日志
- **启动**（`start_command`）仍走现有 `startAndCheck`：跟踪进程、PID、健康检查、持续监控
- **停止 / 编译 / 重启（脚本阶段）** 走 **一次性** 命令：等待脚本退出码；不依赖 `SIGTERM` 杀 launch 进程作为唯一手段

### 4.2 各 API 映射

| UI / API | 行为 |
|----------|------|
| `POST /api/start` | `start_command` + 健康检查 + 监控 |
| `POST /api/stop` | 停止监控 → `stop_command`（**不再**以 `stopProcess` 为主路径）→ 可选：若仍有跟踪进程则 SIGTERM 兜底 → 验证不可达（见 4.4） |
| `POST /api/restart` | 见 §4.4（先 `stop_command`，再 `start_command`；可选先 `build_command`） |
| `POST /api/build` | 仅 `build_command`；不改变运行中进程（与现行为一致） |

组级 `start-group` / `stop-group`：仍按拓扑顺序调用单服务 `stopFn` / `startFn`，无需新字段；**关闭本组**语义见 §4.6。

### 4.3 停止路径（修复类问题的核心）

```
stopService:
  1. 策略检查（依赖、ownership）— 不变
  2. stopMonitoring
  3. runLifecycleCommand(stop_command)   # 主路径
  4. stopProcess()                         # 兜底：清理仍被跟踪的 launch shell
  5. ensureServiceNotReachable             # 探针仍通 → StatusFailed
```

**删除**对 `composeStopShellCommand` 的硬编码推断作为主路径；docker 栈的 `stop` 必须写在 YAML `stop_command` 中。

### 4.4 重启语义（已确认：先停止，再启动）

**已确认：** UI「重启」不由单独 `restart_command` 脚本表达；runner **固定顺序** 组合已有命令：

```
restartService:
  1. 策略检查（ownership 等）— 不变
  2. stopMonitoring
  3. （可选）若配置了 build_command 且服务可编译 → 执行 build_command
       — 与现网一致：编译失败则中止，**不**执行 stop，保留旧进程
  4. 执行 stop_command（同 stopService 主路径，含 stopProcess 兜底）
  5. 执行 start_command（startAndCheck + 健康检查 + 恢复监控）
```

| 要点 | 说明 |
|------|------|
| 与「四条命令」关系 | 对外仍是 **三条显式脚本命令**（start / stop / build）；restart 为 runner 编排，**无需** YAML 第四字段 |
| detach 服务 | 步骤 4 `stop_command` 负责 `compose down`；步骤 5 `start_command` 负责 `managed`/`up -d`；`launch_mode: detach` 用于启动探针 |
| 服务侧 `run.sh restart` | 可保留供 **CLI 手工** 使用；runAll **不**调用，避免与 runner 双轨 |

**不再使用** YAML `restart_command`（配置中出现则 warn 并忽略）。

### 4.5 编译

- `build_command` 非空 → `buildable: true`
- 移除对 `go.mod` / `run.sh` 行扫描的推断（兼容期保留 fallback）

### 4.6 组级「关闭本组」（已确认：best-effort 继续）

**已确认：** 某一服务 `stop_command`（或 `StopService` 策略/ownership 检查）失败时，**仍继续**按拓扑逆序停止同组其余服务。

```
stopGroup:
  errors := []
  for serviceName in stopOrder:
    if err := stopFn(serviceName); err != nil:
      errors.append(serviceName, err)
      continue                    # 不 return，继续下一项
  if len(errors) > 0:
    return aggregatedError(errors)  # API 仍返回失败，但组内已尽力全停
  return nil
```

| 要点 | 说明 |
|------|------|
| 与单服务 stop | 失败服务在 `stopService` 内仍标 `StatusFailed` 并写入 `error`；成功项为 `stopped` |
| API 响应 | `POST /api/stop-group`：任一失败 → HTTP 4xx/5xx + body 列出 **所有** 失败服务名与原因；UI `alert` 展示聚合信息 |
| 与「启动本组」 | **不**对称改动；`start-group` 仍遇错即停（依赖链未就绪时继续无意义） |
| 现状差异 | 当前 `stopGroup` 在首个 `err` 处 `return`（`runner.go`）；实施时改为 best-effort |

---

## 5. 服务脚本补齐计划（按组）

以下命令列为 **目标 YAML 草案**（实施时与 `port_config.json` 对齐端口）。

### 5.1 infrastructure

| 服务 | start | stop | build |
|------|-------|------|-------|
| docker-redis | `bash run.sh managed` | `bash run.sh stop` | — |
| docker-kafka | 同上 | 同上 | — |
| ai-monitor | `./run.sh managed` | `./run.sh stop` | — |

（重启由 runner 执行上表 stop → start，无需单独配置。）

`launch_mode: detach` 用于三项。

### 5.2 platform

| 服务 | 脚本工作 | 建议命令 |
|------|----------|----------|
| task-auth | 扩展 `taskAuth/run.sh` 增加 `start/stop/build`（`restart` 子命令可选，仅 CLI） | `bash run.sh start` 等 |
| task-bill | 新建或扩展 `taskBill/run.sh` | 同上 |
| git-oauth | 扩展 `gitOauth/run.sh` | 同上 |
| saas-backend | 新建 `task2app/scripts/runall-saas-backend.sh`（或 `run.sh` 子命令）封装 venv + runserver | 四条显式指向该脚本 |
| ai-provider | 已有 `Saas_Ai_Provider/run.sh` | `build` / `start-embedded` / `stop` / `restart` |
| task-agent-support 等 Go | `build.sh` + 二进制；新建 `run.sh` 包装 start/stop | `bash run.sh start` = build+run 等 |
| task-sse | 扩展 `taskSSE/run.sh` | 四条子命令 |
| taskFE | 新建 `task2app/front_project/app/scripts/runall.sh` | start=`npm run dev`，stop=杀端口/进程，build=`npm run build` |

### 5.3 domain-events

保持「每条服务不同参数」显式写在 YAML（已接近目标）：

```yaml
start_command: "bash run.sh start accounts"
stop_command: "bash run.sh stop accounts"      # 需在 taskEvents/run.sh 实现
build_command: "bash run.sh build accounts"
```

### 5.4 container-stack & value-stream

| 服务 | 说明 |
|------|------|
| go-run-container | 已有 `start.sh`；补 `stop.sh` / `restart.sh` 或统一 `run.sh` |
| go-relay | 新建 `run.sh` 包装 build + 二进制启停 |
| value-stream | 新建 `run.sh` 或 `scripts/lifecycle.sh` |

---

## 6. 领域概念清单（供 Step 5 DDD）

| 概念 | 类型 | 说明 |
|------|------|------|
| **ServiceLifecycleCommands** | 值对象 | 四条 shell 命令 + 可选 `launch_mode` |
| **ManagedService** | 实体 | 扩展：持有 LifecycleCommands，替代单一 `command` |
| **LifecycleExecutor** | 领域服务 | 在 `working_dir` 执行指定 kind 的命令并返回退出码 |
| **DetachLaunchPolicy** | 值对象 | `attach` / `detach`，决定启动进程退出后是否继续探针 |
| **LifecycleKind** | 枚举 | start / stop / restart / build |

Bounded Context：**本地开发编排（runAll）** — 与「业务 SaaS 域」相邻，通过脚本调用边界隔离。

---

## 7. 价值流影响

查阅 `value-stream.yaml`：

| 问题 | 评估 |
|------|------|
| 受影响 stream | **`runall-cascade-lifecycle`**（启停语义改为脚本驱动）、**`runall-docker-infra-split`**（验收改为显式 `stop_command`）、**`runall-global-start-stop-all`** |
| 新 stream？ | 可选增量 **`runall-explicit-lifecycle-commands`**：步骤「每服务四条命令配置齐全 + UI 停启一致性」 |
| 字段 | 无业务表字段；配置字段为 `runAll.yaml` 服务块（文档可用 `runall.service.lifecycle_commands` 描述性三段名） |
| 测试 | `runAll/src/runner_test.go`、`compose_lifecycle_test.go` 重构；`view_test/runall-stop-cascade.md` 更新；每服务脚本可加 smoke |
| 交叉依赖 | 依赖各子项目脚本就绪；**saas-backend**、**taskFE** 为关键路径 |

---

## 8. 错误处理与可观测性

- 脚本非零退出 → `StatusFailed`，`error` 含 stderr 尾部（截断 2KB）。
- `stop_command` 成功但探针仍通 → `StatusFailed`，提示检查 stop 脚本是否真正释放端口/容器。
- 日志行前缀：`[docker-kafka] stop_command:` 便于 Loki 过滤。
- 配置校验：启动 runAll 时缺少 `start_command` / `stop_command` → **拒绝加载**（strict 默认开启；见 §3.4）。

---

## 9. 验收标准

| # | 场景 | 期望 |
|---|------|------|
| AC1 | UI 关闭 docker-kafka | `stop_command` 执行；`docker ps` 无 kafka 容器；status `stopped` |
| AC2 | UI 重启 task-auth | 先 `stop_command` 再 `start_command`；端口 8003 短暂不可用后恢复 |
| AC3 | UI 编译 taskFE | 仅 `build_command`；`npm run dev` 进程不被杀 |
| AC4 | 手动在 `working_dir` 执行 YAML 四条命令 | 与 UI 行为一致 |
| AC5 | 配置缺 `stop_command` | strict 模式下 runAll 启动失败并指明服务名 |
| AC6 | Go 单测 | 不再依赖 `composeStopShellCommand` 推断通过（推断代码删除后） |
| AC7 | 关闭本组且一项 `stop_command` 失败 | 同组其余服务仍 `stopped`；API 返回聚合错误 |

---

## 10. 实施分期

| 阶段 | 内容 | 风险 |
|------|------|------|
| **P0** | `config.go` 四字段 + **strict 校验** + `command` 别名；**同 PR 内** `runAll.yaml` / `config.yaml` 全量补齐四条命令（否则 runAll 无法启动） | 中 |
| **P1** | infrastructure + docker 栈 YAML 与脚本对齐；`launch_mode` | 低（已部分修复 stop） |
| **P2** | platform 高流量服务（saas-backend、vue、task-auth）脚本化 | 中 |
| **P3** | domain-events `stop/restart` 子命令；Go 微服务统一 `run.sh` | 中 |
| **P4** | 删除推断逻辑 + `command` 字段；更新 `runAll.yaml.ai.md` / README | 低 |

---

## 11. 已确认与开放问题

### 已确认

| 项 | 决策 |
|----|------|
| **strict 默认** | 缺 `stop_command` 或 `start_command` → **直接拒绝启动** runAll；无 warn 迁移窗口 |
| **restart 语义** | runner **先** `stop_command`，**再** `start_command`；可选先 `build_command`（失败则不 stop）；**不**使用 YAML `restart_command` |
| **关闭本组** | **best-effort**：单服务 stop 失败仍继续停同组其它服务；API 返回聚合错误（见 §4.6） |

### 仍待确认

（无 — 设计决策已齐，可进入实施。）

---

## 12. 参考

- `docs/superpowers/specs/2026-05-28-docker-infra-no-repull-design.md` §4.3a（stop 脚本，已实现但未完全配置化）
- `docs/superpowers/specs/2026-05-31-runall-docker-infra-split-design.md` §4.5 compose 生命周期
- `runAll/src/compose_lifecycle.go`（待退役的推断逻辑）
