# 设计：移除本机容器模拟启动（mock_run_container / go_run_container），容器启动统一走云端机器

- **日期**: 2026-08-09 22:25
- **作者**: claude
- **状态**: 🎯 target（设计待审批）
- **迭代**: remove-local-mock-container-startup
- **架构版本**: v17 / v69（基于 v16/v68）

## 1. 背景与目标

当前系统存在**两条容器启动路径**：

| 路径 | 载体 | 说明 |
|------|------|------|
| 🧪 模拟启动（**本次删除**） | `mock_run_container/`（Python Flask :8796）+ `go_run_container/`（Go 重写） | 在**本机**执行 `docker pull` + `docker run` 模拟容器运行，供任务详情页 `mockContainerStart=true` 时联调 |
| ☁️ 云端机器启动（**保留并成为唯一路径**） | taskCloudService start-vm + 阿里云 ECS | 真实云端机器，`cloud_server_events` 状态机 + CloudServerConfig 可达性登记 |

**业务目标**：去掉本机容器镜像模拟启动能力，后续所有容器启动统一走**启动云端机器**的方式。

**用户已确认的范围决策**：
1. **全套删除** — 包括 taskFE「模拟启动」面板/useServerConfigMockRun/mockRun tab/路由 flag 透传，以及全部 mock-start 系列 playwright 测试
2. **comment_csc_bootstrap mock/空平台分支 → 报错拦截** — platform 为空或 `mock` 时直接返回错误「请配置云平台」，不自动触发任何启动；存量 `mock-` instance_id 历史行不动
3. **USE_IN_MEMORY_CLOUD（Aliyun SDK 层内存 mock）保留** — 属云端路径内部测试设施，不在本次范围

## 2. 现状分析：模拟启动链路全貌

### 2.1 入口与调用链

```
taskFE ?mockContainerStart=true (isRouteMockStartQuery)
  → useServerConfigSections: activeServerSection='mockRun'
  → ServerConfigMockRunPanel（选镜像 → 提取 UserData 环境变量 → 启动/停止/日志）
  → /api/cloud/compute/mock-run-container/tenant_id/{t}/workspace_id/{w}/task_id/{task}/{start|stop|status|env-defaults}
  → taskContainerGateway handleMockRunContainer（authorizeContainerRequest 鉴权）
      ├─ start: forwardToMockRun POST /v1/jobs（body: image/env/comment_id）
      │           → mock_run_container docker pull + docker run
      │           → pollMockRunJob 轮询日志 → emitMockRunSSE（server_status_update）
      │           → cloudInternalPost /api/internal/runtime-session/open/ (runtime_source=mock_run)
      ├─ env-defaults → taskCloudService /api/internal/mock-run/env-defaults（UserData 变量解析）
      ├─ resolve-image → taskCloudService /api/internal/mock-run/resolve-image
      └─ stop/status → mock_run_container /v1/jobs/{id}/stop、/v1/tasks/{id}/container/status
```

### 2.2 其他挂接点

- **taskCloudService comment_csc_bootstrap**：评论级 CSC 的 platform 为空/`mock` 时，写 `mock-{容器名}` instance_id 并**异步触发网关 mock-run start**（`postCommentCSCMockRun`）登记 server_url；云平台分支走 `postCommentCSCStartVM`（start-vm-auto）
- **taskCloudService server_release_reconcile**：泄漏 server_url 清理时调 `stopMockRunSidecarForReconcile`（POST /v1/stop）
- **taskCloudService terminal_release_migrate_provision**：`startMigratedContainer` 优先 relay 路径，mock-run-container 为回退路径
- **runAll 编排**：`container-stack.go-run-container`（working_dir=go_run_container，conf_app=mock/mock-run-container，health /health）
- **conf 配置**：`conf/mock/mock-run-container/config.yaml`；runAll scripts `mock_run_container` 端口映射
- **taskFE 路由**：`taskDetailEditing.js` mockContainerStart 参数透传；`useTaskDetail.js` isMockStartRoute；`playwrightTaskDetailUrl.js` 测试 URL 注入

## 3. 变更设计

### 3.1 删除清单（全套）

**A. 运行器服务与编排（删除）**

| # | 位置 | 内容 |
|---|------|------|
| 1 | `mock_run_container/` 目录 | Python Flask 实现（server.py / bootstrap.py / start.sh / scripts / README） |
| 2 | `go_run_container/` 目录 | Go 重写实现（src/ / build.sh / start.sh / build_test.go / README） |
| 3 | `conf/runAll.yaml` | `container-stack` 下 `go-run-container` 服务条目 |
| 4 | `conf/mock/mock-run-container/config.yaml` | 端口 8796 配置 |
| 5 | `runAll/scripts/conf_lib.py` / `conf_loader.py` / `migrate_port_config_to_conf.py` | `mock_run_container` 映射条目 |
| 6 | `runAll/src/service_build_test.go` | `TestResolveBuildCommand_FromGoBuildInCommand` 中的 `go_run_container` 样例命令（改样例为其他服务） |

**B. taskContainerGateway（删除）**

| # | 位置 | 内容 |
|---|------|------|
| 7 | `src/mock_run_handlers.go` | 整个文件（handleMockRunContainer / Start / Stop / Status / EnvDefaults、parseMockRunPath、forwardToMockRun、pollMockRunJob、mockRunContainerName*、emitMockRunSSE） |
| 8 | `src/handlers.go` | `/cloud/compute/mock-run-container/` 路由分发分支 |
| 9 | `src/mock_run_handlers_test.go` | 配套单测 |
| 10 | 配置 | `MockRunContainerURL` / `MockRunContainerSecret` 配置项及读取逻辑 |

**C. taskCloudService（删除 + 改造）**

| # | 位置 | 内容 |
|---|------|------|
| 11 | `src/mock_run_env_defaults.go` | `handleInternalMockRunEnvDefaults` + `handleInternalMockRunResolveImage` |
| 12 | `src/main.go` | `/api/internal/mock-run/env-defaults`、`/api/internal/mock-run/resolve-image` 路由注册 |
| 13 | `src/comment_csc_bootstrap.go` | **改造**：platform 为空/`mock` 分支改为**报错拦截**（见 3.2）；删除 `postCommentCSCMockRun`、`mock-run` mode 分支、`buildCommentMockContainerName` 相关 |
| 14 | `src/server_release_reconcile.go` | 删除 `stopMockRunSidecarForReconcile` 及 mock-run stop 调用 |
| 15 | `src/terminal_release_migrate_provision.go` | `startMigratedContainer` 删除 mock-run 回退路径，仅保留 relay 路径 |

**D. taskFE（删除）**

| # | 位置 | 内容 |
|---|------|------|
| 16 | `src/components/ServerConfigMockRunPanel.vue` | 模拟启动面板组件 |
| 17 | `src/composables/taskDetail/useServerConfigMockRun.js` | mock-run composable |
| 18 | `src/composables/taskDetail/useServerConfigSections.js` | `mockRun` tab、isMockStartEnabled、route flags |
| 19 | `src/utils/serverConfigRouteHelpers.js` | `isRouteMockStartQuery` |
| 20 | `src/composables/useTaskDetail.js` | `isMockStartRoute` computed |
| 21 | `src/composables/taskDetail/taskDetailEditing.js` | `mockContainerStart` 参数透传 |
| 22 | `src/components/ServerConfig.logic.vue` | mockRun 面板接线、props 传递 |

**E. 测试与文档（删除/改写）**

| # | 位置 | 内容 |
|---|------|------|
| 23 | `taskFE/tests/` | 全部 mock-start / mock-run-container 系列 playwright 测试：`TaskDetail.mock-run-container-tab`、`TaskDetail.mock-start-hello-world-git-commit-push`、`TaskDetail.mock-start-yanghui-git-commit-push`、`TaskDetail.mock-start-layer-graph-ready-gate`、`TaskDetail.mock-start-container-heartbeat`、`TaskDetail.custom-mock-start-hello-world-git-commit-push` 等（约 8 个）；`playwrightTaskDetailUrl.js` 中 mockContainerStart 注入 |
| 24 | `conf/value-stream.yaml` | `relay-precheck-token-ssot.remove-dual-write-hack` 字段描述中「（mockStart 保留）」注释移除 |
| 25 | 本文档引用的 API 文档 | go_run_container / mock_run_container README（随目录删除） |

### 3.2 comment_csc_bootstrap 报错拦截（改造）

当前行为（platform 为空或 `mock`）：
```
写 instance_id = "mock-{容器名}" → upsert CSC → 异步触发网关 mock-run start 登记可达性
```

改为：
```
platform 为空或 "mock" → 记录错误日志 event=comment_csc_bootstrap_platform_unsupported
                         → 返回错误「评论级容器启动需要配置云平台（platform 缺失或为 mock）」
                         → 不写 instance_id、不触发任何启动
                         → CSC 保持 starting/binding，由现有 reachability / release 机制收敛
```

- `triggerCommentCSCStartBootstrapAsync` 仅保留 `start-vm-auto` 分支；`postCommentCSCStartBootstrap` 的 `mock-run`/`mock` mode 分支删除
- 存量 `instance_id LIKE 'mock-%'` 历史行**不动**（只读历史数据；泄漏清理逻辑仍按 server_url 判定，mock 行无真实 server_url 不受影响）

### 3.3 云端机器启动（保留，成为唯一路径）

不做行为变更，仅确认其完整性：

```
taskFE 硬件面板 / taskTaskService auto_run / queued_schedule
  → taskCloudService POST /api/cloud/compute/start-vm/{...}（或 start-vm-auto）
  → cloud_server_events 状态机（pending → processing → success/error，pruneFailedStartEvents 去重）
  → taskEvents CLOUD_SERVER_STARTED handler → Aliyun ECS RunInstances（VMStarter 抽象，测试可 mock）
  → CloudServerConfig 登记 instance_id / public_ip / server_url（reachability 闭环）
  → FE 轮询 server-runtime-status 呈现
```

### 3.4 不涉及的边界

- **USE_IN_MEMORY_CLOUD**（Aliyun SDK 内存 mock，value-stream `cloud-server-image` 等步骤使用）**保留** — 用户已确认属云端路径内部设施
- **go_relayToTrae / relay-to-trae 路径**（:8797）**保留** — 与 mock-run-container 无关，是另一条真实容器交互链路
- **云端启动行为、cloud_server_events 状态机、计费** — 不做变更
- Django saas-backend 已退役（v57），无 Python 侧 mock 路径残留

## 4. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 评论级容器启动（原 mock 平台） | — | — | — | **本次为删除/拦截**：platform mock/空 → 报错拦截，不产生启动意图，无对应事件 |
| 移除 mock 平台启动能力 | MockStartDeprecated | — | — | **纯删除变更，无运行时事件**；注释/文档记录即可 |
| 云端容器启动 | CLOUD_SERVER_STARTED | taskCloudService compute_start_vm_native | taskEvents aliyun ECS RunInstances | 既有事件，本次不新增 |

> 说明：本设计以**删除**为主，不引入新的业务意图；唯一的行为变化（mock 平台报错拦截）是「拒绝」，不是「产生事实」，故不产生新事件。

## 5. 🕸️ Code Review Graph 分析

```
CRG unavailable: 图过期（2026-08-06 构建，12 文件 93 节点，仅 JS/TS/Python 覆盖），
受本次变更影响的主体（taskContainerGateway/taskCloudService Go 代码、runAll 编排）无图覆盖；
taskFE 侧已用代码图谱（codegraph）手动核验调用链（ServerConfig.logic.vue → useServerConfigMockRun → mock-run-container API）。
```

## 6. 🐍 Python 新增接口清单与 Go 替代评估

**触发判定**：本设计**不新增**任何 Python HTTP 接口；相反，**删除** Python 侧车 `mock_run_container`（Flask :8796）的既有接口（/v1/jobs、/v1/jobs/{id}、/v1/jobs/{id}/stop、/v1/tasks/{id}/container/status）。

- 按「Python 服务新增接口 — 额外审批门」第 1 节，**不触发**（无新 path + method；仅删除既有接口，无 Swagger deprecated 义务 — mock_run_container 无 Swagger）
- `python_api_approval: not_applicable`

## 7. Value Stream Impact

受影响的既有价值流（`conf/value-stream.yaml`）：

| 流 | 步骤 | 影响 |
|----|------|------|
| `relay-precheck-token-ssot` | `remove-dual-write-hack` | 字段描述移除「（mockStart 保留）」注释（不改变字段语义 — cloud_cloudserverconfig.container_access_token 在 relay 路径已停止写入；mock 路径随本次删除） |
| `cloud-server-image` / 云平台相关流 | 各步骤 | 不涉及（USE_IN_MEMORY_CLOUD 保留） |
| 任务协作启动类流（auto_run / queued / runtime） | 各步骤 | 不涉及行为变更 — 云端启动路径已是唯一真实启动路径 |

**无需新增价值流条目** — 本变更是删除型变更，映射到既有流的描述更新。

## 8. 🏛️ 架构变更影响

- **迭代版本**: v17 / v69 🎯 target（基于 v16/v68）
- **迭代名称**: remove-local-mock-container-startup
- **作者**: claude
- **设计日期**: 2026-08-09 22:25
- **新增文件**（两个视图各四类伴生格式）:
  - 🆕 `docs/architecture/v17-enterprise-landscape-20260809-2225-claude.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`
  - 🆕 `docs/architecture/v69-application-integration-20260809-2225-claude.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`
- **已有文件（未修改）**: v16/v68 current 文件保持不动
- **变更明细 (enterprise-landscape vs v16)**:
  - 🔴 [DEPRECATED v17] `mock_run_container`（Python Flask 侧车 :8796）— 本机容器模拟启动运行器，删除
  - 🔴 [DEPRECATED v17] `go_run_container`（Go 侧车 :8796）— mock_run_container Go 重写，删除
  - 🔴 [DEPRECATED v17] taskContainerGateway `mock-run-container` 代理面 — start/stop/status/env-defaults 四个动作
  - 🔴 [DEPRECATED v17] taskCloudService `/api/internal/mock-run/*`（env-defaults / resolve-image）— 删除
  - 🔴 [DEPRECATED v17] taskFE「模拟启动」面板 / mockRun tab / mockContainerStart 路由 flag — 删除
  - 🟡 [MODIFIED v17] taskCloudService comment_csc_bootstrap — mock/空平台分支改为**报错拦截**，不再触发 mock-run
  - 🟡 [MODIFIED v17] taskCloudService server_release_reconcile — 移除 mock-run sidecar 停止
  - 🟡 [MODIFIED v17] taskCloudService terminal_release_migrate_provision — startMigratedContainer 仅保留 relay 路径
- **变更明细 (application-integration vs v68)**:
  - 🔴 [DEPRECATED v69] mock-run-container 数据流（FE → gateway → mock_run_container → SSE）
  - 🔴 [DEPRECATED v69] internal mock-run API 数据流（gateway → taskCloudService env-defaults/resolve-image）
  - 🟡 [MODIFIED v69] 评论级 CSC 启动链 — 仅保留 start-vm-auto 分支

## 9. 测试影响

| 类别 | 动作 |
|------|------|
| taskFE playwright（mock-start 系列约 8 个） | **删除**（面板已删，无测试对象） |
| taskContainerGateway `mock_run_handlers_test.go` | **删除**（文件随 handlers 删除） |
| taskCloudService mock_run_env_defaults 相关单测 | **删除/改写**（如有，随 internal API 删除） |
| taskCloudService comment_csc_bootstrap 单测 | **改写**：mock 平台断言改为「报错拦截 + 不触发启动」 |
| runAll `service_build_test.go` | 样例命令改为非 go_run_container 服务（如 go_relayToTrae） |
| 云端启动既有测试（start-vm / CLOUD_SERVER_STARTED / USE_IN_MEMORY_CLOUD） | **保留**，回归验证 |

## 10. 风险与回滚

| 风险 | 缓解 |
|------|------|
| 评论级容器在无云平台配置时启动失败（原本 mock 可本地跑） | 报错信息明确提示「请配置云平台」；CSC 保持 starting 由现有 release/reconcile 收敛 |
| 存量 `mock-` instance_id 历史行 | 只读历史数据不动；泄漏清理按 server_url 判定，mock 行不受影响 |
| gateway 删除路由后旧前端版本 404 | FE 与 gateway 同版本发布；404 返回明确错误信息 |
| 回滚 | git 恢复删除文件 + runAll.yaml 条目即可；mock_run_container 与 go_run_container 保留在 git 历史中可随时恢复 |

## 11. 交付检查清单

- [ ] 删除 A1-A5、B7-B9、C11-C12、D16-D22、E23 文件/条目
- [ ] C13 报错拦截改造 + 单测
- [ ] C14/C15 移除 mock-run 挂接
- [ ] runAll.yaml 移除 go-run-container 服务并验证 `runAll` 编排可用
- [ ] taskFE 删除 mockRun tab 后路由/构建无残留引用（grep mockContainerStart / mockRun 零残留）
- [ ] 架构 v17/v69 四类伴生文件生成 + 验证（PlantUML / ArchiMate / Archi load / Mermaid）
- [ ] VERSION_HISTORY.md 追加 v17/v69 条目
- [ ] 云端启动回归（start-vm + CLOUD_SERVER_STARTED + USE_IN_MEMORY_CLOUD 路径）
