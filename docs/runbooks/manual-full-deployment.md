# 全栈手动部署手册（Manual Full Deployment）

> **用途**：不依赖 runAll UI，按依赖序**手工部署整个服务栈**；也用于 runAll 一键拉起后的状态核对与故障排查。
>
> **适用环境**：单机 Linux（生产 LAN IP `10.2.150.68`）。多节点部署时通过环境变量 `INFRA_HOST` 覆盖（约束 39）。
> **更新日期**：2026-08-24（依据 `conf/runAll.yaml`、`conf/base.yaml`、各子仓 `build.sh`/`run.sh` 逐一核对）。
> **相关文档**：`docs/runbooks/new-developer-environment-setup.md`（新环境 Git 钩子激活）、`dataMigrate/README.md`（迁移链路）。

---

## 1. 架构总览

```
                        ┌──────────────────────────────────────────┐
                        │  runAll 编排器 UI :9999（最外层，可选一键）  │
                        └──────────────────────────────────────────┘
   ┌──────────────┬──────────────┬───────────────┬──────────────────────────┐
   │ 基础设施层    │ 代码托管层     │  平台层         │  事件/增值层               │
   │ docker-mysql │ gitlab :8012 │ 业务 Go 服务    │ taskEvents 意图(~40个)     │
   │ docker-redis │ gitlab SH-1  │ taskGateway    │ go-relay :8797            │
   │ docker-kafka │ (远程 :8014)  │ (APISIX :18081)│ value-stream :9998        │
   │ portainer    │              │ taskSSE :8798  │ taskFE nginx :4000        │
   │ AiMonitor    │              │                │                           │
   └──────────────┴──────────────┴───────────────┴──────────────────────────┘
```

**地址 SSOT**：`conf/base.yaml`（`scheme`/`baseDomain`/`subdomains.*`/`infraHost`）。所有服务配置经 `conf/<app>/config.yaml` 引用，`${INFRA_HOST}` 默认 `10.2.150.68`。

### 1.1 端口总览

| 端口 | 服务 | 容器/进程 | 说明 |
|------|------|-----------|------|
| 3306 | MySQL | `docker-mysql-mysql-1` | 12 个业务库，用户 `taskapp/taskapp123` |
| 6379 | Redis | `redis-redis-1` | `redis:7.0-alpine` |
| 2181 / 9092 / 9093 / 18080 | Zookeeper / Kafka / Kafka 内部 / Kafka UI | `kafka-*` | advertised 地址 = `INFRA_HOST:9092` |
| 9000 / 9443 | Portainer | `portainer-portainer-1` | HTTP/HTTPS |
| 3000 / 3100 / 9090 / 3200 / 4317 / 4318 | Grafana / Loki / Prometheus / Tempo / OTEL(gRPC) / OTEL(HTTP) | `aimonitor-*` | 可观测性栈 |
| 9100 / 9115 | node-exporter / blackbox-exporter | `aimonitor-*` | 指标采集 |
| 8012 / 22 | GitLab CE HTTP / SSH | `gitlab` | 主 GitLab |
| 8014 | GitLab SH-1（远程 1.117.67.121） | 远程容器 | 上海区域 |
| 18081 / 18444 / 9180 | APISIX HTTP / TLS / Admin | `taskgateway-apisix-1` | taskGateway |
| 8002 | taskGitOauth | `bin/taskGitOauth` | Git OAuth 换票 |
| 8003 | taskAuth | `bin/taskAuth` | 认证 |
| 8004 | taskBill | `bin/taskBill` | 计费 |
| 8010 | taskAiProvider | `bin/taskAiProvider` | AI 提供商代理 |
| 8011 | taskAgentSupport | `bin/taskAgentSupport` | Agent 支持 |
| 8013 | taskAIEndPoint | `bin/taskAIEndPoint` | AI 端点 |
| 8014 | taskContainerGateway | `bin/taskContainerGateway` | 容器网关 |
| 8015 | taskCredentialService | `bin/taskCredentialService` | 容器令牌 |
| 8016 | taskProjectService | `bin/taskProjectService` | 项目域 |
| 8017 | taskTaskService | `bin/taskTaskService` | 任务域 |
| 8018 | taskCloudService | `bin/taskCloudService` | 云资源域 |
| 8019 | taskAIComment | `bin/taskAIComment` | AI 评论 |
| 8020 | taskTenantService | `bin/taskTenantService` | 组织域 |
| 8025 | taskReferral | `bin/taskReferral` | 推荐奖励 |
| 8798 | taskSSE | `node src/server.mjs` | SSE 推送（Node） |
| 4000 | taskFE | `taskfe-nginx` | 前端静态站（nginx） |
| 8797 | go-relay | `bin/go_relayToTrae` | Relay 桥接 |
| 9998 | value-stream | `bin/valueStream` | 价值流 UI |
| 9999 | runAll UI | `bin/runAll` | 编排器自身 |

---

## 2. 前置条件

### 2.1 工具链

| 工具 | 版本要求 | 本机实测 |
|------|---------|---------|
| Go | 1.24+（runAll 硬性要求；服务编译需 `otel_enabled` tag） | go1.26.0 |
| Node.js | 20+（taskSSE / taskFE 构建） | v22.22.1 |
| Python3 | 3.8+（conf-sync / 迁移脚本） | 3.14.4 |
| Docker | 24+，含 `docker compose`（v2） | 本地 unix socket |
| lsof | 进程/端口管理 | 系统自带 |

### 2.2 仓库与子模块

```bash
git clone <ram-work 远端> && cd ram-work
git submodule update --init --recursive     # 检出全部子仓
bash runAll/scripts/install-hooks-all.sh    # 激活全部 git hooks（门禁）
bash runAll/scripts/install-hooks-all.sh --check   # 校验全绿
```

### 2.3 环境变量

```bash
# 关键：基础设施地址 SSOT（conf/base.yaml infraHost；默认即 10.2.150.68）
export INFRA_HOST=10.2.150.68
```

- `INFRA_HOST` 决定 Kafka advertised 地址、Redis/MySQL 连接地址、可观测性 URL。
- **多节点部署**：仅需覆盖 `INFRA_HOST`，勿改各服务配置（约束 39）。
- 所有启动脚本会自动清除 HTTP(S) 代理环境变量（约束 23），无需手动处理。

### 2.4 构建前提示

- Go 服务构建脚本均为 `./build.sh`（输出 `bin/<svc>`，带 `-tags otel_enabled`），需在对应子仓目录执行。
- 前端构建耗时较长（vite 产物 + 原子切换），构建阶段与启动阶段分离（镜像下载收敛到编译阶段，见 4.5/8 节）。

---

## 3. 部署顺序（依赖 DAG）

| 阶段 | 内容 | 说明 |
|------|------|------|
| 一 | 基础设施（MySQL → Redis → Kafka → Portainer → AiMonitor → promtail） | MySQL/Redis/Kafka 无相互依赖可并行；promtail 依赖 Loki |
| 二 | GitLab CE（主 + SH-1 远程） | 依赖 Redis |
| 三 | **数据库初始化（迁移）** | 必须先于业务服务；业务进程启动**不再自动迁移**（约束 40） |
| 四 | 平台服务（Go 微服务 + APISIX 网关 + taskSSE） | 按服务依赖序 |
| 五 | 前端 taskFE（nginx 静态站） | 依赖网关 |
| 六 | 事件意图 taskEvents（约 40 个） | 依赖 Redis/Kafka + 平台服务 |
| 七 | go-relay / value-stream | 独立 |
| 八 | runAll 编排器（最外层） | 最后启动；**禁止**把 runAll 自身登记为托管服务（会递归） |

---

## 4. 阶段一：基础设施

> 所有命令在 monorepo 根目录执行。健康检查脚本返回 0 为健康；Docker 中间件数据持久化在各自 `dockerInfra/<svc>/data*` 与 compose volume 中，`stop` 不删数据（`down --volumes=false`）。

### 4.1 MySQL（3306）

```bash
bash dockerInfra/mysql/run.sh start       # 支持 start|stop|restart|status|logs
bash dockerInfra/mysql/health.sh && echo OK   # 健康探针（runAll 同款）
```

- **配置 SSOT**：`conf/infra/mysql/config.yaml`（账号/连接数/InnoDB 参数）。
- **冷启动初始化**：`dockerInfra/mysql/init/01-create-databases.sql` 创建 12 个业务库（`task_auth`/`task_bill`/`task_budget`/`task_referral`/`git_oauth`/`ai_provider`/`task_project`/`task_task`/`task_cloud`/`task_ai_comment`/`task_tenant`/`container`）与应用用户 `taskapp`。
- **数据目录**：`dockerInfra/mysql/data`。启动时自动检测损坏空壳目录并重建（旧目录挪为 `data.corrupt.<ts>`）。
- **验证**：`bash dockerInfra/mysql/run.sh status`；`mysql -h127.0.0.1 -utaskapp -ptaskapp123 -e 'SHOW DATABASES;'`。

### 4.2 Redis（6379）

```bash
bash dockerInfra/redis/run.sh start
bash dockerInfra/redis/health.sh && echo OK
```

- **配置 SSOT**：`conf/infra/redis/config.yaml`。
- **验证**：`redis-cli -h 127.0.0.1 ping` → `PONG`。

### 4.3 Kafka（9092 / UI 18080）

```bash
bash dockerInfra/kafka/run.sh start       # INFRA_HOST 缺省即 10.2.150.68
bash dockerInfra/kafka/health.sh && echo OK
```

- **配置 SSOT**：`conf/infra/kafka/config.yaml`；advertised 地址 = `INFRA_HOST:9092`（PLAINTEXT_HOST）。
- **健康判定必须 broker-first**（`health.sh`）：Kafka UI（:18080）可能假健康——UI 存活不代表 broker 存活（FE-20260721-KAFKA-UI-FALSE-HEALTHY）。
- **验证**：`docker compose -f dockerInfra/kafka/docker-compose.yml ps`；`curl -s http://127.0.0.1:18080`。

### 4.4 Portainer（9000）

```bash
bash dockerInfra/portainer/run.sh start
curl -s http://127.0.0.1:9000/api/status | head -1   # 200 OK
```

- 可选依赖；失败不阻断整体（`on_failure: skip`）。

### 4.5 AiMonitor 可观测性栈（Grafana/Loki/Prometheus/Tempo/OTEL）

```bash
bash AiMonitor/run.sh pull      # 镜像下载收敛到编译阶段（先 pull，避免启动卡下载超时）
bash AiMonitor/run.sh start     # 支持 start|managed|stop|pull
```

- **依赖**：Redis 就绪后启动。
- **验证**：
  ```bash
  curl -s http://127.0.0.1:3000/api/health   # Grafana
  curl -s http://127.0.0.1:3100/ready        # Loki（/ready 为就绪探针）
  curl -s http://127.0.0.1:9090/-/healthy    # Prometheus
  ```

### 4.6 promtail-local（本地日志投递）

```bash
bash runAll/scripts/runall-local-promtail.sh up    # 依赖 Loki 就绪（脚本内含 90s 等待）
```

- Loki 冷启动窗口约 4-5 分钟，期间探针短暂 retrying 属预期，Loki 就绪后自动转 healthy。

---

## 5. 阶段二：GitLab CE（gitService）

### 5.1 主 GitLab（8012）

```bash
bash gitService/run.sh start          # start|stop|managed；stop --clean 完全清理容器（保留 GITLAB_HOME 数据）
```

- **配置 SSOT**：`conf/infra/git-service/config.yaml`。
- **依赖**：Redis（OIDC 会话）。
- **验证**：`curl -sI http://127.0.0.1:8012/users/sign_in | head -1`（200/302）。
- **root 初始密码**：`docker exec -it gitlab grep 'Password:' /etc/gitlab/initial_root_password`。
- **注意**：首次启动容器初始化可达数分钟（健康检查超时窗口 900s）。

### 5.2 上海 SH-1 GitLab（远程 1.117.67.121:8014）

```bash
bash gitService/scripts/runall_ssh_sh_gitlab.sh start   # 经 ssh Host=sh 操作远程
```

- 远程独立实例；**禁止**用本机 8014 检查（本机 8014 是 task-container-gateway）。
- **验证**：`curl -sI http://1.117.67.121:8014/users/sign_in | head -1`。

---

## 6. 阶段三：数据库初始化与迁移（业务库）

> ⚠️ **必须**先于业务服务启动。业务进程启动**不自动迁移**（约束 40）；新增 SQL 放入 `dataMigrate/<service>/` 后需重新执行本阶段。

### 方式 A（推荐）：runAll UI 初始化

启动 runAll（见阶段八）后访问 `http://10.2.150.68:9999/` → 点击「**初始化全部数据库**」。

链路：`POST /api/dev/init-databases` → `InitAllDatabases()` → `db/registry.yaml`（数据库注册表）→ 逐库 `db/<db>/migrate.sh` → `db/scripts/apply_datamigrate.sh <db> <dataMigrate_dir>` → 幂等执行 SQL（`data_migrate_log` 追踪表 + 稳定存在性检查 + SQL 幂等，三层保障）。

### 方式 B（手动逐库）

```bash
bash db/task-auth/migrate.sh        # 单库迁移（含 Go seed，表级幂等）
bash db/task-bill/migrate.sh
bash db/task-tenant/migrate.sh
bash db/task-project/migrate.sh
bash db/task-task/migrate.sh
bash db/task-cloud/migrate.sh
bash db/task_budget/migrate.sh
bash db/task-ai-comment/migrate.sh
bash db/task-referral/migrate.sh
bash db/git-oauth/migrate.sh
bash db/container/migrate.sh
bash db/ai-provider/migrate.sh
```

- 注册表：`db/registry.yaml`（库名/归属服务/迁移脚本路径/顺序）。
- 执行器：`db/scripts/apply_datamigrate.sh <database_name> <dataMigrate_dir>`（幂等，可重复执行）。
- 迁移脚本含 `CREATE PROCEDURE` 时使用 `DELIMITER //` 双路径（mysql CLI 与 Go 实现各自处理）。

---

## 7. 阶段四：平台 Go 服务

### 7.1 通用构建启动模式

```bash
cd <service-dir> && ./build.sh                    # → bin/<svc>（-tags otel_enabled）
# 后台启动（前台调试直接运行 ./bin/<svc> 即可）
setsid -f nohup ./bin/<svc> >../logs/<svc>-console.log 2>&1 </dev/null
# 停止（按端口）
bash -c 'lsof -ti:<port> | xargs kill -9 2>/dev/null || true'
```

- 服务从 monorepo 根 `conf/<conf_app>/config.yaml` 读配置（`conf/base.yaml` 为锚点）；`${INFRA_HOST}` 由环境变量注入。
- 启动前**务必**已执行阶段三（约束 40：进程与迁移解耦）。

### 7.2 服务清单（目录 / conf_app / 端口 / 健康检查 / 依赖）

| 服务目录 | conf_app | 端口 | 健康检查 | 依赖 |
|---------|----------|------|---------|------|
| taskAuth | auth/task-auth | 8003 | `GET /api/health/` | MySQL |
| taskBill | billing/task-bill | 8004 | `GET /api/health/` | MySQL |
| taskReferral | task-referral | 8025 | `GET /api/health/` | taskAuth |
| taskGitOauth | auth/git-oauth | 8002 | `GET /api/health/` | MySQL, GitLab |
| taskAiProvider | ai/ai-provider | 8010 | `GET /api/health/` | taskAuth |
| taskAgentSupport | ai/task-agent-support | 8011 | `GET /api/health/` | — |
| taskAIEndPoint | ai/task-ai-endpoint | 8013 | `GET /api/health/` | — |
| taskContainerGateway | gateway/task-container-gateway | 8014 | `GET /api/health/` | — |
| taskProjectService | taskProjectService | 8016 | `GET /api/health` | taskAuth |
| taskTenantService | taskTenantService | 8020 | `GET /api/health` | taskAuth, taskProjectService |
| taskTaskService | taskTaskService | 8017 | `GET /api/health` | taskProjectService, taskAuth, taskBill |
| taskCloudService | taskCloudService | 8018 | `GET /api/health` | taskProjectService, taskTaskService, taskAuth, taskGitOauth |
| taskCredentialService | container/task-credential-service | 8015 | `GET /health` | taskGitOauth, taskTaskService |
| taskAIComment | taskAIComment | 8019 | `GET /api/health` | taskAuth, taskTaskService, taskSSE, taskCloudService, taskCredentialService, Kafka |

### 7.3 特殊平台服务

**taskGateway（APISIX，HTTP 18081 / TLS 18444 / Admin 9180）**

```bash
cd taskGateway && bash run.sh start        # docker compose 起 APISIX；stop 同理
curl -s http://127.0.0.1:9180/apisix/admin/routes -H 'X-API-KEY: ...' | head   # Admin API
```

- 配置 SSOT：`conf/gateway/task-gateway/config.yaml`（`httpPort`/`tlsPort`/`adminPort`/upstream `host.docker.internal`）。
- 启动脚本自动清理宿主机遗留 `nginx.pid`/`worker_events.sock`（EACCES 修复），并将容器内日志 chmod a+rw 供宿主机截断。
- 依赖：taskAuth、taskGitOauth。

**taskSSE（Node，8798）**

```bash
cd taskSSE && bash run.sh start            # 无 node_modules 自动 npm install
curl -s http://127.0.0.1:8798/health
```

- 配置 SSOT：`conf/gateway/task-sse/config.yaml`（transport: redis）。依赖 Redis。

### 7.4 配置同步（conf → 子仓）

`conf/<area>/<app>/` 为配置 SSOT（部分一级目录如 `conf/task-referral/`）。修改后同步到对应子仓：

```bash
bash runAll/scripts/conf-sync-all.sh       # 遍历含 sync.manifest.yaml 的 conf 目录执行 sync.sh
```

- 单目录同步：`python3 runAll/scripts/conf-sync.py <name>`（如 `task-referral`）。
- 同步日志：`conf/logs/conf-sync.log`。

---

## 8. 阶段五：前端 taskFE（nginx 静态站，4000）

```bash
cd taskFE/app
bash scripts/runall-lifecycle.sh build     # vite 原子构建 → public/releases/<ts>/，切换 public/html symlink
bash scripts/runall-lifecycle.sh start     # docker compose up taskfe-nginx（--wait）
# nginx 配置变更热生效（静态站常驻，精准重启只切 symlink）：
docker exec taskfe-nginx nginx -s reload
bash scripts/runall-lifecycle.sh stop
```

- 配置 SSOT：`conf/frontend/vue/config.yaml`（`nginxImage`/`host`/`port`/`memLimit`/`releaseKeep`）。
- `public/html/index.html` 不存在时拒绝启动——必须先 build。
- 验证：`curl -s http://127.0.0.1:4000/ | head -3`。

---

## 9. 阶段六：事件意图 taskEvents（约 40 个，Redis + Kafka 消费者）

```bash
cd taskEvents
# 单个意图：build → start（stop 同理）
bash run.sh build <event>/<intent>
bash run.sh start <event>/<intent>
bash run.sh stop  <event>/<intent>
# 示例
bash run.sh build billing_transaction_created/1_process_billing_transaction
bash run.sh start billing_transaction_created/1_process_billing_transaction
```

- 意图路径（canonical）：`INTENT_PATHS` 列表见 `taskEvents/run.sh` 头部；端口读 `conf/events/domain-events/<event>/config.yaml`（SSOT）。
- 每个意图为独立 Go 二进制：`bin/<event>/<intent>/task-events-<event>-<intent>`（`go build -tags otel_enabled ./cmd/<event>/<intent>`）。
- 健康检查：readiness `GET /api/health/ready`，liveness `GET /api/health/`。
- **全量拉起**：推荐用 runAll（阶段八）按 `depends_on` DAG 自动处理 40 个意图的依赖顺序与健康检查；手工逐个 start 时按「事件订阅无强依赖」原则可并行，但以下意图有服务依赖，须先就绪：
  - `user_created/0_create_company` → taskTenantService
  - `task_status_changed/2_fanout_work_panel_sse` → taskSSE
  - `workspace_machine_idle/1_recycle_idle_nodes`、`cloud_csc_reconcile/1_sweep` → taskCloudService
  - `queued_auto_run_scan/1_dispatch` → taskTaskService
  - `task_comment_image_mentioned/1_start_vm_for_at_mention` → taskCloudService/taskTaskService/taskProjectService/taskAIComment
  - `container_migrate_await_ready`、`task_graceful_shutdown_await` → taskCloudService/taskContainerGateway/Kafka
  - `task_post_expiry_scan/1_expire_posts`、`member_joined/1_create_default_git_identity` → taskTaskService
  - `billing_referral_settle_scan`、`billing_profit_sharing_scan` → taskBill
  - `referral_code_expiry_scan` → taskReferral
  - `user_account_deletion_execute_scan` → taskAuth

---

## 10. 阶段七：container-stack 与 value-stream

```bash
# go-relay（8797）
cd go_relayToTrae && ./build.sh && setsid -f nohup ./bin/go_relayToTrae >../logs/go-relay-console.log 2>&1 </dev/null
curl -s http://127.0.0.1:8797/health

# value-stream（9998）
cd valueStream && ./build.sh && setsid -f nohup ./bin/valueStream --config ../conf/value-stream.yaml --ui-port :9998 >../logs/value-stream-console.log 2>&1 </dev/null
curl -sI http://127.0.0.1:9998 | head -1
```

---

## 11. 阶段八：runAll 编排器（最外层，UI 9999）

```bash
cd runAll && ./run.sh        # 自动 build（校验 skip-orphan 能力标记）→ setsid nohup 分离启动
# UI: http://127.0.0.1:9999   日志: logs/runall-console.log
```

- **环境变量**：`INFRA_HOST=127.0.0.1 bash runAll/run.sh` 可覆盖（开发环境）；`RUNALL_SKIP_BUILD=1` 跳过编译；`RUNALL_FOREGROUND=1` 前台。
- **CLI 命令**：`./run.sh -command build-all`（同步全量编译，非零退出码）；`doctor`/`takeover` 需前台。
- ⚠️ **禁止对生产配置执行 `-command doctor`**：doctor 是破坏性 preflight，会杀掉仍在运行的托管服务进程（恢复用 UI `/api/restart-all`）。
- **热替换**：新二进制必须含 `Verified skip-orphan capability`（`grep -aF 'skip orphan port cleanup' bin/runAll`），旧二进制会把托管服务当孤儿强杀；禁止反向替换。
- **编排范围**：runAll 按 `conf/runAll.yaml` 的 `depends_on` 拓扑排序启动全部托管服务并健康检查（UI 每 2s 刷新）；托管服务含阶段四/五/六/七全部服务与基础设施。
- ⚠️ **禁止**将 runAll 自身登记为被编排服务（递归拉起）。

### 11.1 常用运维端点（runAll UI 能力）

| 动作 | 位置 | 说明 |
|------|------|------|
| 初始化全部数据库 | UI「初始化全部数据库」 | 阶段三方式 A |
| 精准编译重启 | UI「精准编译重启」 | 按依赖序编译重启登记的脏服务（`.runall/precise_restart_services.txt`） |
| 全量恢复 | `POST /api/restart-all` | doctor 误杀后的恢复手段 |

---

## 12. 全栈验证清单

```bash
# 1) 容器期望状态（对比 §1.1 端口表）
docker ps --format '{{.Names}}\t{{.Ports}}'

# 2) 端口监听核对（应包含下列全部）
ss -tlnp | grep -E ':(3306|6379|9092|3000|3100|8012|8002|8003|8004|8010|8011|8013|8014|8015|8016|8017|8018|8019|8020|8025|8797|8798|18081|4000|9998)\b' | awk '{print $4}' | sort -u

# 3) 健康端点抽查（路径按服务精确匹配；8002-8025 已验证全部 200）
#    注：/api/health 不带尾斜杠会 307 重定向，按表内路径带斜杠；8015 走 /health
for p in 8002 8003 8004 8010 8011 8013 8014 8016 8017 8018 8019 8020 8025; do
  printf '%s: ' "$p"; curl -s -o /dev/null -w '%{http_code}\n' "http://127.0.0.1:$p/api/health/"
done
printf '8015: '; curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8015/health
curl -s -o /dev/null -w 'taskFE: %{http_code}\n' http://127.0.0.1:4000/health
curl -s -o /dev/null -w 'grafana: %{http_code}\n' http://127.0.0.1:3000/api/health
curl -s -o /dev/null -w 'loki: %{http_code}\n' http://127.0.0.1:3100/ready
curl -s -o /dev/null -w 'gitlab: %{http_code}\n' http://127.0.0.1:8012/users/sign_in

# 4) 端到端冒烟（可选）
cd taskFE/app && npx playwright test --config playwright.verify.config.js   # 或按 e2e-tests/ 脚本执行
```

---

## 13. 回滚与重置

| 场景 | 操作 |
|------|------|
| 单服务回滚 | 按端口杀进程 → 恢复旧二进制 → 重新 start（`lsof -ti:<port> \| xargs kill -9`） |
| 全栈优雅关闭 | runAll 运行中：`kill -TERM <runAll pid>`（反向 DAG 逐个 SIGTERM→SIGKILL）；未运行：按阶段逆序逐组 stop |
| MySQL 数据重置（测试环境，⚠️ 破坏性） | `bash db/_infra/mysql-reset.sh`（清空重建全部业务库，随后重跑阶段三） |
| Redis 清空（测试环境） | `bash db/_infra/redis-flush.sh` |
| Kafka junk topics 治理 | `python3 db/_infra/kafka_cleanup_junk_topics.py --confirm DELETE_JUNK`（kafka-go-* 前缀） |
| MySQL 数据备份/恢复 | 走 ramsync 备份体系（`backup_mysql_dump` + cron；容器数据目录不直接打包，见 ramwork 运维约定） |
| 前端版本回退 | `public/releases/` 保留历史版本，改 symlink `public/html → releases/<前版本>` 后 `docker exec taskfe-nginx nginx -s reload` |

---

## 14. 故障排查速查表

| 症状 | 根因/排查 |
|------|----------|
| 服务健康检查超时 | 未执行阶段三（业务库为空）→ 先初始化数据库再启动 |
| Kafka 连接失败 | `INFRA_HOST` 未导出 → advertised 地址错误；`docker compose -f dockerInfra/kafka/docker-compose.yml ps` 确认 broker 而非仅 UI 存活 |
| 服务间调用 502/超时 | 依赖服务未起（看 §7.2 依赖列）；APISIX upstream 用 `host.docker.internal`（勿手写 bridge IP） |
| AiMonitor 启动卡住 | 镜像未 pull → 先 `bash AiMonitor/run.sh pull`（启动阶段不拉镜像） |
| taskFE 拒绝启动 | `public/html/index.html` 缺失 → 先 `bash scripts/runall-lifecycle.sh build` |
| APISIX EACCES / 残留 pid | 脚本自动清理；手动 `rm -f taskGateway/logs/nginx.pid taskGateway/logs/worker_events.sock` |
| GitLab 页面打不开 | NO_PROXY 未含 localhost/127.0.0.1（有代理时） |
| runAll 页面异常/服务假死 | 日志 `logs/runall-console.log`；Loki 冷启动窗口内 retrying 属预期 |
| 端口被占 | `lsof -ti:<port>` 定位；停旧进程后再启动（不要换端口） |
| 迁移未生效 | `data_migrate_log` 幂等：已执行的 step 不重复；新增 SQL 必须重新触发初始化 |

---

## 15. 注意事项与已知问题

### 15.1 已知配置不一致（2026-08-24 核对）

- `conf/runAll.yaml` 中 `task-events-user-account-deletion-execute-scan-1-execute-due` 的 `TASK_AUTH_INTERNAL_URL` 为 `:8001`，而 taskAuth 实际监听 **:8003**（`conf/auth/task-auth/config.yaml`、runAll stop_command、当前监听均一致为 8003）。该意图的账户删除扫描会连错端口；已登记 OPT 跟踪修正。

### 15.2 硬约束引用

- `.ai/01_project_constraints/23_app_startup_no_env_proxy.md` — 启动脚本自动清除代理环境变量
- `39_network_topology_aware_config.md` — `INFRA_HOST` 拓扑感知配置
- `40_app_process_independent_of_db_migrate.md` — 业务进程不自动迁移
- `42_precise_restart_service_registration.md` — 服务代码变更后须登记精准编译重启
- `44_test_resource_cleanup.md` — 测试临时资源必须清理
- `46_trae_agent_online_service_docker_push.md` — trae-agent 提交后须推送 onlineServiceJS 镜像
- `32_submodule_commit_order.md` — 子仓优先提交，meta 后置

### 15.3 非 runAll 托管的独立部署件

- **trae-agent onlineServiceJS 镜像**（在线工作区）：`cd trae-agent/onlineServiceJS && DOCKER_PUSH=1 ./buildDocker.sh`（改动影响 online 镜像时，约束 46）。
- **taskChromePlugin**：Chrome 浏览器扩展（DevTools 抓包建任务），开发者手动加载，不参与服务部署。
- **trae-agent 服务端**（trae_agent/server）：独立 Python 服务，按 `trae-agent/README.md` 部署，不属于 runAll 托管范围。
