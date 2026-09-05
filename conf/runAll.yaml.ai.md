# conf/runAll.yaml 规则文件

> 本文件为 `conf/runAll.yaml` 的 AI 协作规则，编排语义以 [runAll/README.md](../runAll/README.md) 为准。

## 基本信息

- 版本：1.4.0
- 创建日期：2026-05-28
- 最后修改：2026-08-27
- 维护者：Trae AI
- 关联实现：`runAll/`（Go 多服务编排器，DAG 拓扑排序 + HTTP 健康检查 + Web UI）

## 文件定位

| 路径 | 用途 |
|------|------|
| `conf/runAll.yaml`（本文件目标） | monorepo `conf/` 下编排入口；`working_dir` 相对**仓库根**（`conf/` 上级目录）；`host`/`port` 从 `conf/<app>/config.yaml` 自动读取 |
| `runAll/config.yaml` | `runAll/` 目录内示例/开发入口；`working_dir` 相对 **runAll/**；`runAll/run.sh` 默认指向 `../conf/runAll.yaml` |

> **2026-06-03 重要变更：** 根目录 `runAll.yaml` 已删除，统一迁移到 `conf/runAll.yaml`。服务的 `host`/`port` 通过 `conf_app` 映射从 `conf/<app>/config.yaml` 自动读取，不再硬编码 IP。基础设施服务使用 `${INFRA_HOST}` 占位符，值从 `conf/docker-infra/config.yaml` 自动解析。

## 启动方式

```bash
cd runAll
# 推荐：编译后 setsid nohup 独立会话（日志 ../logs/runall-console.log）
./run.sh
# 调试前台（会成为 screen/tty 作业，可能被 SIGTERM 带走，勿在 screen -S run 里用）：
./bin/runAll --config ../conf/runAll.yaml
./bin/runAll --config ../conf/runAll.yaml --daemon # 仅拉起服务后退出
```

**前置条件：** 远程 Docker（`cpu-remote` context + SSH 到 `zcpu`）、`task2app` 虚拟环境、`taskFE/app` 已 `npm install`、相关 Go 子项目可 `build.sh`。Mac **无需** Docker Desktop；基础设施由 `scripts/runall-remote-docker.sh` 经 runAll 拉起。

## 规则分类

### 核心规则

> 影响编排正确性、端口一致性与生产安全，必须严格遵守

#### 禁止递归编排 runAll（一级）

- **描述：** 不得将 `runAll` 自身写入 `conf/runAll.yaml` 的 `services`。否则会出现 runAll → runAll → … 无限递归拉起。
- **依据：** `conf/runAll.yaml` 文件尾注释；runAll 仅作为**最外层** orchestrator 由人工或 CI 单独启动。
- **优先级：** 高

#### 服务 host 禁止仅绑 127.0.0.1（一级，禁止忽略）

- **描述：** `conf/<app>/config.yaml`（及 OAuth provider 等）中的服务监听 `host` **必须**为 **`0.0.0.0`**。禁止改为 `127.0.0.1`/`localhost`。APISIX（Docker `host.docker.internal`）与边缘 nginx 依赖全网卡监听；runAll 健康检查会对 `0.0.0.0` 归一为 `127.0.0.1` 探测，二者不冲突。
- **依据：** 仓库根 `.ai.md`；细则 `.ai/01_project_constraints/22_service_listen_host_not_localhost_only.md`
- **优先级：** 高

#### 应用启动禁止环境 Proxy（一级，禁止忽略）

- **描述：** 本文件顶层 **`use_proxy` 必须为 `false`**（默认）。runAll 在 false 时会从子进程环境剥离 `HTTP(S)_PROXY` / `ALL_PROXY` 等。环境 Proxy **仅**供开发者本机工具加速，**禁止**为翻墙把全局 `use_proxy` 改为 `true` 并提交。单服务若确需临时代理，须书面例外且不得污染全局默认。
- **依据：** 仓库根 `.ai.md`「应用启动与环境 Proxy」；细则 `.ai/01_project_constraints/23_app_startup_no_env_proxy.md`；实现 `runAll/src/lifecycle_exec.go`（`stripProxyEnvVars`）
- **优先级：** 高

#### 端口与 monorepo conf 一致（一级）

- **描述：** 平台服务通过 `conf_app` 字段映射到 `conf/<app>/config.yaml`，runAll 自动读取 `host` + `port`，与 `health_path` 拼接为健康检查 URL。改端口只需改 `conf/<app>/config.yaml`。
- **适用场景：** 新增/修改服务、调整 `runserver`/`npm` 监听端口
- **优先级：** 高

| 服务名 | conf_app | health_path | 说明 |
|--------|----------|-------------|------|
| `task-auth` | task-auth | `/api/health/` | conf/task-auth/config.yaml → host:8003 |
| `task-bill` | task-bill | `/api/health/` | conf/task-bill/config.yaml → host:8004 |
| `task-git-oauth` | — (显式 url) | — | conf/task-git-oauth/ 无独立 config.yaml |
| `ai-provider` | ai-provider | `/api/health/` | conf/ai-provider/config.yaml → host:8010 |
| `saas-backend` | django | `/api/health/` + `/api/live/` | conf/core/django/config.yaml → host:8001 |
| `task-gateway` | task-gateway | `/api/health/` | conf/task-gateway/config.yaml → host:8080 (httpPort) |
| `task-agent-support` | task-agent-support | `/api/health/` | conf/task-agent-support/config.yaml → host:8011 |
| `task-ai-endpoint` | task-ai-endpoint | `/api/health/` | conf/task-ai-endpoint/config.yaml → host:8013 |
| `task-container-gateway` | task-container-gateway | `/api/health/` | conf/task-container-gateway/config.yaml → host:8014 |
| `task-project-service` | taskProjectService | `/api/health` | conf/taskProjectService/config.yaml → host:8016 |
| `task-task-service` | taskTaskService | `/api/health` | conf/taskTaskService/config.yaml → host:8017；创建任务经 Django internal 校验 + task-bill 扣费 |
| `task-cloud-service` | taskCloudService | `/api/health` | conf/taskCloudService/config.yaml → host:8018 |
| `task-ai-comment` | taskAIComment | `/api/health` | conf/taskAIComment/config.yaml → host:8019；ai-comments 公网真源 + instruct_worker |
| `task-credential-service` | container/task-credential-service | `/health` | conf/container/task-credential-service/config.yaml → host:0.0.0.0:8015（边缘 nginx 可达；同机调用走 127.0.0.1） |
| `task-sse` | task-sse | `/health` | conf/task-sse/config.yaml → host:8798 |
| `taskFE` | vue | `/health` | conf/vue/config.yaml → host:4000 |
| `go-relay` | relay-to-trae | `/health` | conf/relay-to-trae/config.yaml → host:8797 |

**基础设施服务（显式 url/tcp，使用 `${INFRA_HOST}` 占位符）：**

| 服务名 | 探活方式 | 说明 |
|--------|----------|------|
| `docker-redis` | `exec: bash dockerInfra/redis/health.sh` | Compose Redis 容器健康（避免宿主机 :6379 假阳性） |
| `docker-kafka` | `exec: bash dockerInfra/kafka/health.sh` | **Broker** 就绪（compose `kafka` running + `kafka-topics`）；禁止仅探 Kafka UI `:18080`（UI 可在 broker Exited 后仍 Up；OPT-20260721-007） |
| `docker-portainer` | `url: http://${INFRA_HOST}:9000/api/status` | Portainer |
| `ai-monitor` | `url: http://${INFRA_HOST}:3000/api/health` | AiMonitor 自有约定（3000）；`build_command: bash AiMonitor/run.sh pull` — 镜像下载收敛到「编译」阶段，启动 `--pull never` 不拉镜像（2026-08-05） |
| `promtail-local` | `exec: docker inspect … aimonitor-promtail`（Linux）/`aimonitor-promtail-local`（Mac） | 依赖 `ai-monitor`；**禁止**用 Loki `/ready` 代替容器探活（Loki 就绪但 Promtail 缺失时热替换会假 healthy）；`launch_mode: detach`；清空可观测栈后须 `runall-local-promtail.sh up` |
| `git-service` | `url: http://${INFRA_HOST}:8012/users/sign_in` | 现网 GitLab（组 `gitlab-regions`） |
| `git-service-tencent-sh-1` | `url: http://1.117.67.121:8014/users/sign_in` | 上海 Host sh 独立实例；禁止探本机 :8014（task-container-gateway） |
| `task-events-*` (34个) | `url + liveness_url` 使用 `${HOST}` 占位符 | domain-events 意图消费者（含 sse_message persist） |

#### conf_app 使用规则（一级）

- **描述：** 有 `conf_app` 的服务必须同时配置 `health_path`（或 `liveness_path`）；禁止同时指定 `conf_app` 和显式 `health_check.url`。
- **`conf_app` 值必须是 `conf/` 下的目录名**（如 `django`、`task-auth`），该目录内必须有 `config.yaml`，且 `config.yaml` 包含 `host` 和 `port`（或 `httpPort`）字段。
- **`remote_docker.host`** 自动从 `conf/docker-infra/config.yaml` 读取（YAML 中不写）；`${INFRA_HOST}` 模板在加载时自动替换。
- **`${HOST}`** 占位符用于无法使用 `conf_app` 但有端口号的服务（如 task-events-*），值从第一个 `conf_app` 服务的 host 派生。
- **优先级：** 高

#### DAG 与 depends_on（一级）

- **描述：** 执行顺序**仅**由 `depends_on` 决定，`groups` 仅作逻辑分组。同层服务并行启动；下一层须等当前层全部健康检查通过。
- **GitLab 可插拔（ADR-0048）：** 除 `git-service*` 自身外，禁止任何服务 `depends_on: git-service*`。本机实例默认不启动（ADR-0047）；OAuth/OIDC 运行时 HTTP 对齐。
- **本文件依赖图：**

```text
Level 0（并行）: docker-redis, docker-kafka, docker-portainer, ai-monitor

Level 0b: promtail-local  ← depends_on: [ai-monitor]
          git-service  ← depends_on: [docker-redis]（组 gitlab-regions；现网 INFRA）
          git-service-tencent-sh-1  ← 无 INFRA 依赖；ssh Host sh compose（组 gitlab-regions）


Level 1: task-auth, task-bill, task-git-oauth（仅 docker-mysql，不等 git-service）, task-sse, go-run-container, go-relay, value-stream

Level 2: saas-backend  ← depends_on: [task-auth, task-git-oauth, docker-redis, task-sse]

Level 3: task-gateway, ai-provider, taskFE, task-agent-support, task-ai-endpoint,
         task-container-gateway, task-project-service, task-events-*（16 个）
         ← task-project-service 亦依赖 saas-backend（workspace-access 等 Django internal）

Level 4: task-task-service  ← depends_on: [task-project-service, task-auth, saas-backend, task-bill]

Level 5: task-cloud-service  ← depends_on: [task-project-service, task-task-service, task-auth, saas-backend]
         task-credential-service（container-stack）← depends_on: [saas-backend, task-git-oauth, task-task-service]

Level 6: task-ai-comment  ← depends_on: [task-auth, task-task-service, task-sse, task-cloud-service,
         task-credential-service, docker-kafka]；POST 校验与 instruct 编排真源，CloudServerConfig 经 :8018
```

- **描述：** `domain-events-intents` 组内 16 个 `task-events-*` 可经 Web UI「启动本组」一键拉起；`PlanStartGroup` 会自动包含上游 `docker-redis`、`saas-backend` 等依赖。
- **优先级：** 高

#### 配置校验（一级）

启动前 runAll 会校验（见 README）：

- 全局 `name` 唯一
- `depends_on` 引用存在的服务名
- 无环依赖
- `on_failure` 仅为 `exit` 或 `skip`
- 每项必须有 `start_command`/`stop_command` 与健康检查探针（`url`/`tcp`/`conf_app+health_path`）
- `conf_app` + 显式 `url` 不可同时指定
- `conf_app` 值不可包含路径分隔符（防遍历）
- `conf_app` 指向的 `conf/<app>/config.yaml` 必须存在且包含有效 `host` + `port`

- **优先级：** 高

#### on_failure 策略（一级）

| 服务 | on_failure | 含义 |
|------|------------|------|
| `docker-redis`, `docker-kafka` | `exit` | 基础设施不可用 → 停止全部 |
| 其余所有服务 | `skip` | 失败记日志并继续其余服务 |

- **描述：** 非关键路径用 `skip`；核心基础设施用 `exit`。调整前评估下游 `depends_on` 是否会长时间等待失败依赖。
- **优先级：** 高

#### 必填字段与 health_check（一级）

| 字段 | 必填 | 说明 |
|------|------|------|
| `name`, `start_command`, `stop_command` | 是 | Lifecycle strict 模式强制 |
| 健康检查探针 | 是 | 三者之一：`url`、`tcp`、或 `conf_app`+`health_path` |
| `conf_app` | 否 | 指定后 `health_path` 必填，runAll 从 `conf/<app>/config.yaml` 读 host:port 拼接 URL |
| `health_path` / `liveness_path` | `conf_app` 存在时必填 | 以 `/` 开头，如 `/api/health/` |
| `depends_on` | 否 | 默认 `[]` |
| `on_failure` | 否 | 默认 `exit` |
| `working_dir` | 否 | 相对 **YAML 所在目录的上级目录**（即仓库根）；因为 YAML 在 `conf/` 下，路径与根目录对齐 |
| `env` | 否 | 追加到当前环境；可使用 `${INFRA_HOST}` / `${HOST}` 占位符 |
| `build_command` | 否 | 重启/UI 构建前可选执行 |

健康检查默认：`timeout` 30s、`retries` 10；本文件多数服务已加大 timeout/retries 以适配冷启动。探针为 HTTP GET，`2xx/3xx` 为健康；退避见 README（`backoff.initial/max/multiplier`）。

- **优先级：** 高

### 最佳实践

> 提升本地开发与可维护性

#### 配置 SSOT：conf/ 为权威来源（一级）

- **描述：** `host`/`port` 以 `conf/<app>/config.yaml` 为权威来源。新增服务或改端口时，只需编辑对应的 `conf/<app>/config.yaml`，runAll 自动生效。禁止在 `conf/runAll.yaml` 中重复声明 host/port。
- **优先级：** 高

#### build_command 使用（二级）

- **描述：** `build_command` 在 Web UI 重启或显式构建时先于 `start_command` 执行；日常 `command` 启动不自动跑 build（如 `go-run-container` 用 `start.sh managed --skip-build`）。
- **适用：** `taskFE`、`go-run-container`、`go-relay`、`value-stream`
- **优先级：** 中

#### 健康检查 URL 选择（一级，禁止忽略）

- **描述：** 探活 `host:port` **必须**等于该进程实际 listen 的 SSOT。平台服务优先使用 `conf_app` + `health_path`，禁止再手写一份端口。基础设施服务使用 `${INFRA_HOST}` 占位符，勿直接写裸 IP。
- **task-events-\*：** 端口 SSOT 是 `conf/events/domain-events/<event>/config.yaml` 的 `intents.<intent>.port`。`health_check.url` 与 `liveness_url` 必须使用**同一**端口；**禁止**复制相邻 `task-events-*` 的 health 块后只改 `name` / `start_command`。Prometheus `AiMonitor/prometheus/file_sd/runall-health-targets.json` 必须同步。
- **依据：** `.ai/01_project_constraints/52_runall_health_port_ssot.md`；ADR-0012；案例 FE-20260816-EVENTS-FANOUT-HEALTH-PORT（18048 被拷成 18056 → `READINESS_TIMEOUT`）
- **验收：** `python3 db/scripts/ci/check_task_events_runall_health_ports.py`
- **优先级：** 高

#### 显式生命周期命令（一级）

- **描述：** 每个服务须配置 `start_command`、`stop_command`；可选 `build_command`、`launch_mode: detach`。缺 `stop_command` 时 runAll **拒绝启动**（strict，仅 `RUNALL_LIFECYCLE_STRICT=0` 调试）。UI「重启」= runner 先 `stop_command` 再 `start_command`；「关闭本组」单项失败仍继续停其余服务。
- **废弃：** `command` 仅作 `start_command` 别名；`restart_command` 忽略。
- **优先级：** 高

#### 关闭与信号（二级）

- **描述：** `Ctrl+C` / `SIGTERM` 按**反向 DAG** 停服：先停依赖方，5s 后 `SIGKILL`，并信号整进程组。单服务 UI 关闭主路径为 `stop_command`（含 `compose down`），`SIGTERM` 仅兜底。
- **优先级：** 中

#### Web UI（二级）

- **描述：** 前台模式默认 `http://localhost:9999`（`--ui-port` 可改）；2s 刷新状态；健康/失败服务可点 ↻ 重启（先 stop 再 start，可选 `build_command`）。
- **优先级：** 低

#### 精准编译重启（二级）

- **描述：** 页头「精准编译重启」按钮读取登记文件 `<仓库根>/.runall/precise_restart_services.txt`，**先从磁盘重载本文件**（使启动后新增的 `task-events-*` 等服务可被解析），再按依赖序对登记服务逐个执行 restart（编译→停止→启动→健康检查；无 `build_command` 的服务跳过编译），完成后清空登记文件；失败的服务保留以便重试。智能体编程会话修改服务代码后须登记（`scripts/register-precise-restart.sh`，细则 `.ai/01_project_constraints/42_precise_restart_service_registration.md`）。
- **API：** `GET/POST /api/precise-restart/registrations`、`POST /api/precise-restart[/register|/cancel]`、`GET /api/precise-restart/progress`（SSE）。
- **优先级：** 低

### 风格指南

#### YAML 维护（二级）

- **描述：** 文件头注释保持「与 conf/<app>/ 端口一致、用法、前置条件」；新增服务按 `groups` 分组并补全 `depends_on`。
- **描述：** `working_dir` 使用相对仓库根的路径（如 `task2app`、`taskGitOauth`），勿混用 `../`。
- **优先级：** 低

## 服务清单（当前 conf/runAll.yaml）

| 分组 | 服务 | conf_app | 说明 |
|------|------|----------|------|
| infrastructure | docker-redis | — | TCP 探活 `${INFRA_HOST}:6379` |
| infrastructure | docker-kafka | — | exec 探活 broker：`dockerInfra/kafka/health.sh`（非 UI `:18080`） |
| infrastructure | docker-portainer | — | Portainer `${INFRA_HOST}:9000` |
| infrastructure | ai-monitor | — | Grafana `${INFRA_HOST}:3000`（`on_failure: skip`） |
| infrastructure | promtail-local | ai-monitor | Promtail → Loki；规范路径 `logs/<service>.log`（`on_failure: skip`） |
| gitlab-regions | git-service | infra/git-service | 现网 GitLab `${INFRA_HOST}:8012`（可插拔，其它服务不得 depends_on；组级 `skip_start_all`） |
| gitlab-regions | git-service-tencent-sh-1 | infra/git-service-tencent-sh-1 | 上海 Host sh GitLab `1.117.67.121:8014`；9999 经 `runall_ssh_sh_gitlab.sh` 远程 compose（组级 `skip_start_all`） |
| platform | task-auth | task-auth | `taskAuth` |
| platform | task-bill | task-bill | `taskBill` |
| platform | task-git-oauth | — (显式 url) | `taskGitOauth` | `depends_on: [docker-mysql]`（不得等 git-service，ADR-0048） |
| platform | saas-backend | django | `task2app` | `depends_on: [docker-redis, task-sse, ...]` |
| platform | task-gateway | task-gateway | `taskGateway` |
| platform | task-agent-support | task-agent-support | `taskAgentSupport` |
| platform | task-ai-endpoint | task-ai-endpoint | `taskAIEndPoint` |
| platform | task-container-gateway | task-container-gateway | `taskContainerGateway` |
| platform | task-project-service | taskProjectService | `taskProjectService` | `depends_on: [task-auth, saas-backend]` |
| platform | task-task-service | taskTaskService | `taskTaskService` | `depends_on: [task-project-service, task-auth, saas-backend, task-bill]` |
| platform | task-cloud-service | taskCloudService | `taskCloudService` | `depends_on: [task-project-service, task-task-service, task-auth, saas-backend]` |
| platform | task-ai-comment | taskAIComment | `taskAIComment` | `depends_on: [task-auth, task-task-service, task-sse, task-cloud-service, task-credential-service, docker-kafka]` |
| platform | task-sse | task-sse | `taskSSE` | `depends_on: [docker-redis]` |
| platform | taskFE | vue | `taskFE/app` |
| platform | ai-provider | ai-provider | `task2app/Saas_Ai_Provider` |
| domain-events-intents | task-events-* (34个) | — (`${HOST}` 占位符) | `taskEvents` | `depends_on: [docker-redis, ...]` |
| container-stack | go-relay | relay-to-trae | `go_relayToTrae` |
| container-stack | task-credential-service | container/task-credential-service | `taskCredentialService` | `depends_on: [saas-backend, task-git-oauth, task-task-service]` |
| value-stream | value-stream | — (`${HOST}` 占位符) | `valueStream` UI `:9998` |

Redis/Kafka/AiMonitor/git-service 在本机（`10.2.150.89`）运行，`${INFRA_HOST}` 占位符在 runAll 加载时自动从 `conf/docker-infra/config.yaml` 解析。

## 规则冲突处理

1. 核心规则 > 最佳实践 > 风格指南
2. 本文件（`conf/runAll.yaml.ai.md`）> 通用脚本类 `.ai.md`（针对 conf/runAll.yaml 的编排语义）
3. 用户显式指令 > 本规则文件
4. 端口/主机以 monorepo `conf/<app>/config.yaml` 为**权威来源**；`conf/runAll.yaml` 通过 `conf_app` 自动引用，不再重复声明

## 变更日志

- **2026-08-28：** `gitlab-regions` 组级 `skip_start_all: true` — start-all 不碰 GitLab；面板单启/按组启动仍可用（OPT-20260827-022 / ADR-0048）。
- **2026-08-27：** ADR-0048 — 其它服务不得 `depends_on git-service*`；`task-git-oauth` 仅等 `docker-mysql`。本机 GitLab 默认不启动（ADR-0047）。
- **2026-08-18：** `gitlab-regions` 组 — 现网 `git-service` 从 infrastructure 迁出；`git-service-tencent-sh-1` 探活 Host sh `:8014`，启停经 `runall_ssh_sh_gitlab.sh`
- **2026-08-16：** 版本 1.4.0 — 「健康检查 URL 选择」升为一级：task-events 探活端口必须等于 domain-events YAML SSOT；禁止从相邻条目复制 health_check；对齐 ADR-0012 / 约束索引第 47 条

- **2026-07-21：** `docker-kafka` 探活改为 `exec: bash dockerInfra/kafka/health.sh`（broker running + `kafka-topics`）；禁止仅探 Kafka UI `:18080`（OPT-20260721-007 / FE-20260721-KAFKA-UI-FALSE-HEALTHY）
- **2026-07-15：** 版本 1.2.0 — 新增一级规则「服务 host 禁止仅绑 127.0.0.1」（须 `0.0.0.0`；对齐根 `.ai.md` / `.ai/01_project_constraints/22_service_listen_host_not_localhost_only.md`）
- **2026-07-06（c）：** Phase 3 CloudServerConfig 切流 — `task-cloud-service` 增加 `depends_on: [saas-backend]`（Django internal）；`task-ai-comment` 补齐 `task-credential-service`、`docker-kafka`，移除对 `saas-backend` 硬依赖（校验/instruct 仅 Go）；`conf/taskAIComment/config.yaml` 显式声明 `taskCloudService :8018`
- **2026-07-06（b）：** `task-project-service` 增加 `depends_on: [saas-backend]`；新增 `task-ai-comment`（:8019）
- **2026-07-06：** Phase 2d 任务域 — 补充 `task-project-service`（:8016）、`task-task-service`（:8017）、`task-cloud-service`（:8018）conf_app 映射与 DAG；`task-task-service` 增加 `depends_on: [saas-backend, task-bill]`（Django internal 校验与扣费）
- **2026-06-03：** 配置统一迁移 — 根目录 `runAll.yaml` 删除，统一到 `conf/runAll.yaml`；新增 `conf_app` + `health_path` 自动 URL 拼接；`remote_docker.host` 从 `conf/docker-infra/config.yaml` 自动读取；新增 `${HOST}` 占位符支持；12 个平台服务使用 conf_app 映射，基础设施服务使用 `${INFRA_HOST}` 占位符
- **2026-06-03：** 本文件从根目录移入 `conf/`，路径引用全面更新
- **2026-06-01：** git-service 纳入 runAll（远程 CPU GitLab + 隧道 8012/2222）；本机可卸 Docker Desktop
- **2026-06-01：** infrastructure 改为远程 Docker + SSH 隧道（`ssh-tunnel` / `runall-remote-docker.sh`）；删除 `runAll-apps-local.yaml`
- **2026-06-01：** G4.5 新增 `domain-events-bin` 组（15 个事件级二进制，8020–8034）；与 `domain-events` 域级组不可同时运行
- **2026-05-31：** 显式 `start_command` / `stop_command` / `build_command`；strict 校验；stopGroup best-effort；重启先停后启
- **2026-06-01：** `docker-infra` 拆为 `docker-redis`（`dockerInfra/redis`，TCP 6379）与 `docker-kafka`（`dockerInfra/kafka`，HTTP 18080）；platform 链默认 depends_on `docker-redis`
- **2026-05-28：** docker-infra 改用 `run-infra.sh`（pull/up 分离，重启不重复拉镜像；stop 时 compose down）
- **2026-05-28：** 版本 1.0.0 — 初始创建；依据 `runAll/README.md` 整理 DAG、校验、health_check、on_failure 与 port_config 端口对照表
