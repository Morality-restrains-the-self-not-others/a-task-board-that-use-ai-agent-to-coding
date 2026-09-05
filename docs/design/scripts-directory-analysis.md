# scripts/ 目录脚本职能分析

> 头脑风暴产物 — 设计文档
> 日期: 2026-06-05
> 状态: 待审批

---

## 一、目录概览

```
scripts/
├── conf-*            # 配置管理 (4 个脚本 + 1 个共享库)
├── docker-*          # Docker 环境与基础设施 (11 个脚本)
├── runall-local-*    # 本地日志采集 (1 个脚本)
├── remote-compose-*  # 远程 Docker 栈辅助 (1 个脚本)
├── migrate_*         # 一次性数据迁移 (1 个脚本)
├── ci/               # CI/CD 检查 (5 个脚本)
└── hooks/            # Git pre-commit 钩子模板 (6 个脚本 + README)
```

总计 **23 个功能脚本** + **1 个共享 Python 库** + **1 个 README**。

---

## 二、按职能分类详解

### 类别 A：配置管理（5 个文件）

| 文件 | 语言 | 职能 | 调用方 |
|------|------|------|--------|
| [conf_lib.py](scripts/conf_lib.py) | Python | **共享库**：repo_root 定位、YAML 加载/写入、deep_merge、kebab→camelCase 转换、generated_header 生成 | 被 conf-sync.py、migrate_port_config_to_conf.py 引用 |
| [conf-read.py](scripts/conf-read.py) | Python | **配置读取 CLI**：读取 monorepo 下 `conf/<app>/` 的 YAML 配置，支持 dot.path 访问，输出 JSON 或单值。支持 `snapshot-json`（全量快照）、`git-oauth`、`domain-events` 等特殊查询 | 被 shell 脚本、Vite、Playwright 调用 |
| [conf-sync.py](scripts/conf-sync.py) | Python | **配置同步引擎**：读取 `conf/<app>/sync.manifest.yaml`，按 `from / to / pick` 指令从源 YAML 提取字段，生成带 `# GENERATED` 头的派生配置片段 | 被 conf-sync-all.sh 批量调用 |
| [conf-sync-all.sh](scripts/conf-sync-all.sh) | Bash | **批量同步入口**：遍历 `conf/*/*/` 下所有包含 `sync.manifest.yaml` 的目录，逐一调用 `conf-sync.py` | 开发时手动执行、CI 验证 |
| [migrate_port_config_to_conf.py](scripts/migrate_port_config_to_conf.py) | Python | **一次性迁移**：将旧版 `task2app/conf/port_config.json` 拆解为新 `conf/<app>/config.yaml` 布局（django/vue/ai-provider/git-oauth 等 16 个 app），生成 `git-oauth/providers/*.yaml` 和 `domain-events/*/config.yaml` | 由迁移流程执行 |

**设计意图**：将单一大 JSON (`port_config.json`) 拆散为按 app 分目录的 YAML，并通过 `sync.manifest.yaml` + `pick` 机制实现字段级派生（如 vue 配置从 django 配置中挑选 `apiBaseUrl`），避免重复维护。

**关键数据流**：
```
conf/<app>/config.yaml  ──(sync.manifest.yaml)──▶  conf/<app>/xxx.generated.yaml
     ▲                                                       │
     │ conf-read.py 读取                                      │ 被应用消费
     │                                                       ▼
shell/Vite/Playwright                                运行时配置
```

---

### 类别 B：Docker 环境与基础设施（11 个脚本）

这是 scripts/ 中最大的一组，围绕 **远程 Docker（ZCPU 云主机）+ SSH 隧道** 的架构设计。

| 文件 | 语言 | 职能 | 层级 |
|------|------|------|------|
| [docker-env.sh](scripts/docker-env.sh) | Bash | **环境变量与共享函数**：定义 context 名称、SSH 主机/用户/地址、隧道端口组（核心/观测/Git 三组），提供 `docker_env_*` 系列辅助函数 | 基础设施层（被其他 docker-*.sh source） |
| [docker-desktop-helper.sh](scripts/docker-desktop-helper.sh) | Bash | **Docker Desktop 管理辅助**：检测 Daemon 就绪、读取 Desktop 内存配置、内存不足警告。注意：`try_start_desktop` 已禁用（避免与 SSH 隧道端口冲突） | 本地辅助层 |
| [docker-install-cli.sh](scripts/docker-install-cli.sh) | Bash | **CLI 安装**：通过 Homebrew 安装 docker CLI（不含 Desktop），清理 Docker.app 断链，初始化 context 并切换到远程 | 初始化层 |
| [docker-context-init.sh](scripts/docker-context-init.sh) | Bash | **Context 初始化**：确保远程 (`zcpu-remote`) 和本地 (`desktop-linux`) 两个 Docker context 已创建 | 初始化层 |
| [docker-setup.sh](scripts/docker-setup.sh) | Bash | **一键设置入口**：初始化 context → 切换到远程 → 追加 `.zshrc` 集成（快捷命令别名） | 用户入口层 |
| [docker-shell.sh](scripts/docker-shell.sh) | Bash | **Shell 集成**：定义 `dr`/`dl`/`ds`/`dt`/`dru`/`dlu`/`dti` 快捷别名和函数，终端启动时自动切换到远程 context | 用户入口层 |
| [docker-use-remote.sh](scripts/docker-use-remote.sh) | Bash | **切换到远程**：重置 SSH 复用连接 → 确保远程 context → 切换 → 打印状态 | 操作层 |
| [docker-use-local.sh](scripts/docker-use-local.sh) | Bash | **切换到本地**（已弃用）：打印警告引导用户使用远程方案 | 操作层（deprecated） |
| [docker-status.sh](scripts/docker-status.sh) | Bash | **状态查询**：显示当前 context、Daemon 可用性、运行容器、SSH 隧道状态 | 操作层 |
| [docker-tunnel-remote.sh](scripts/docker-tunnel-remote.sh) | Bash | **SSH 隧道管理**：`start`（后台启动端口转发，支持 `--core` 仅转发 Redis/Kafka） / `stop` / `status`。转发端口包括 Redis(6379)、Kafka(9092/9093)、Kafka-UI(18080)、Grafana(3000)、Loki(3100)、OTEL(4317/4318)、GitLab HTTP(8012)/SSH(2222) | 操作层 |
| [docker-uninstall-desktop.sh](scripts/docker-uninstall-desktop.sh) | Bash | **Docker Desktop 卸载**：停止进程 → 运行卸载程序 → 删除 `/Applications/Docker.app` → 安装 brew docker CLI → 切换到远程 context | 清理层 |

**架构设计意图**：
```
┌──────────────────────────────┐      SSH Tunnel       ┌──────────────────────────┐
│   Mac 开发机（本机）           │ ◄────────────────── ▶ │   ZCPU 云主机（远程）      │
│                              │   端口转发              │                          │
│  docker CLI (brew)           │   6379  Redis          │  docker daemon            │
│  docker context → zcpu-remote│   9092/9093 Kafka      │  ├─ Redis                 │
│  无 Docker Desktop           │   3000  Grafana        │  ├─ Kafka                 │
│                              │   3100  Loki           │  ├─ GitLab                │
│  scripts/docker-tunnel-*.sh  │   4317/4318 OTEL       │  ├─ Grafana/Loki          │
│  scripts/runall-local-*.sh   │   8012/2222 GitLab     │  └─ AiMonitor             │
└──────────────────────────────┘                        └──────────────────────────┘
```

快捷命令速查：
| 别名 | 命令 | 作用 |
|------|------|------|
| `dr` | `docker-remote` | 切换到远程 Docker |
| `dl` | `docker-local` | 切换到本地 Docker |
| `ds` | `docker-dctx` | 查看 Docker 状态 |
| `dt` | `docker-tunnel` | SSH 隧道 start/stop/status |
| `dru` | `docker-remote-up` | 切换远程 + 启动 infra |
| `dlu` | `docker-local-up` | 停止隧道 + 切换到本地 |
| `dti` | `docker-infra-remote` | 远程 infra 管理 |

---

### 类别 C：日志采集（1 个脚本）

| 文件 | 语言 | 职能 |
|------|------|------|
| [runall-local-promtail.sh](scripts/runall-local-promtail.sh) | Bash | **本地 Promtail 容器管理**：在本机 Docker（desktop-linux context）运行 Promtail，tail `$RUNALL_LOG_ROOT` 日志目录，推送至远程 Loki (`http://<host>:3100/loki/api/v1/push`)。支持 `up/down/status/reset` 子命令 |

**设计意图**：用 sidecar 容器采集 runAll 写入的本地日志文件，通过 HTTP push 到远程 Loki，实现 A2（Application → Observability）日志链路。

---

### 类别 D：远程 Docker 栈辅助（1 个脚本）

| 文件 | 语言 | 职能 |
|------|------|------|
| [remote-compose-helper.sh](scripts/remote-compose-helper.sh) | Bash | **远程 Linux 版 helper**：`docker-desktop-helper.sh` 的极简替代，不含 Desktop 检测逻辑。通过 sync 推送到远程 Linux 主机执行 |

**设计意图**：Mac 和 Linux 环境共享相同的函数签名（`docker_helper_compose_cmd`、`docker_helper_ensure_daemon` 等），但实现不同——Mac 版可读取 Desktop 配置，Linux 版仅检查 daemon 可用性。

---

### 类别 E：CI/CD 检查（5 个脚本）

| 文件 | 语言 | 职能 | 检查项 |
|------|------|------|--------|
| [ci/check_ddd_bdd_compliance.py](scripts/ci/check_ddd_bdd_compliance.py) | Python | **DDD/BDD 合规总入口**：协调运行 4 项检查——Python DDD/BDD（委托到 task2app）、Go DDD 合规、conf 同步状态、valueStream Go 测试 | 4 合 1 |
| [ci/check_go_ddd_compliance.py](scripts/ci/check_go_ddd_compliance.py) | Python | **Go DDD 合规检查**：检查 domain 层不引入 infrastructure/application 包、aggregate 不依赖 service/repository、repository 包定义了至少一个 interface。支持 `--module-root` 指定模块、`--forbid-external` 禁止外部依赖 | domain import / aggregate boundary / repository interface |
| [ci/check_taskgateway_routes.sh](scripts/ci/check_taskgateway_routes.sh) | Bash | **API 路由一致性检查**：验证 `apisix/apisix.yaml` 与 `routes/routes.yaml` + `conf/task-gateway/config.yaml` 保持一致 | routes 一致性 |
| [ci/check_conf_sync.sh](scripts/ci/check_conf_sync.sh) | Bash | **配置同步漂移检测**：将 conf/ 目录拷贝到临时目录 → 重新执行 `conf-sync-all.sh` → 逐文件对比，确保已提交的 conf/ 与 sync 输出一致 | conf 同步状态 |
| [ci/smoke_taskgateway.sh](scripts/ci/smoke_taskgateway.sh) | Bash | **冒烟测试**：先跑路由一致性检查 → 启动 taskGateway → 轮询 health endpoint（最多 30 次 × 2s）→ 确认健康 | taskGateway 健康 |

**CI 检查链路**：
```
check_ddd_bdd_compliance.py (总入口)
├── Python DDD/BDD check  →  task2app/scripts/ci/check_ddd_bdd_compliance.py
├── Go DDD compliance      →  scripts/ci/check_go_ddd_compliance.py
├── conf sync check        →  scripts/ci/check_conf_sync.sh
└── valueStream Go tests   →  go test ./... (TestLoadProductionValueStream / TestDesignDocFieldNames)

smoke_taskgateway.sh (冒烟)
├── check_taskgateway_routes.sh
└── taskGateway health check
```

---

### 类别 F：Git Hooks 模板（6 个脚本 + README）

所有钩子遵循统一策略：**暂存文件相关测试优先 → 随机抽测（30%）→ 任一失败即阻止提交**。

| 文件 | 适用仓库 | 测试机制 | 状态 |
|------|----------|----------|------|
| [hooks/pre-commit-go](scripts/hooks/pre-commit-go) | go_relayToTrae, go_run_container, runAll, valueStream, taskAuth, taskBill, taskEvents | `go test -count=1 ./...` 按包 | active |
| [hooks/pre-commit-django](scripts/hooks/pre-commit-django) | gitOauth | `python3 manage.py test <app>` | active |
| [hooks/pre-commit-jest](scripts/hooks/pre-commit-jest) | DaydaymoneyGrafana | `npm run test:ci` / `npx jest` | active |
| [hooks/pre-commit-node](scripts/hooks/pre-commit-node) | taskSSE | `node --test *.test.mjs` | active |
| [hooks/pre-commit-python-scripts](scripts/hooks/pre-commit-python-scripts) | AiMonitor | `python3 test_*.py` | active |
| [hooks/pre-commit-noop](scripts/hooks/pre-commit-noop) | gitService, mock_run_container | 跳过（暂无测例） | placeholder |

**安装方式**：复制对应模板到子仓库 `scripts/hooks/pre-commit` → 软链到 `.git/hooks/pre-commit`，或配置 `core.hooksPath`。

---

## 三、依赖关系图

```
conf_lib.py ─────────────────────────────────────────────┐
   │                                                      │
   ├── conf-sync.py ─── conf-sync-all.sh ─── ci/check_conf_sync.sh
   ├── conf-read.py                                       │
   └── migrate_port_config_to_conf.py                     │
                                                          │
docker-env.sh ────────────────────────────────────────────┤
   │                                                      │
   ├── docker-context-init.sh                             │
   ├── docker-use-remote.sh                               │
   ├── docker-use-local.sh (deprecated)                   │
   ├── docker-status.sh                                   │
   ├── docker-tunnel-remote.sh                            │
   ├── docker-uninstall-desktop.sh                        │
   └── runall-local-promtail.sh                           │
                                                          │
docker-desktop-helper.sh ─── docker-setup.sh              │
                                                          │
docker-shell.sh ─── ~/.zshrc 集成                         │
                                                          │
remote-compose-helper.sh ─── (sync → 远程 Linux)          │
                                                          │
ci/check_ddd_bdd_compliance.py ───────────────────────────┤
   ├── task2app/scripts/ci/check_ddd_bdd_compliance.py    │
   ├── ci/check_go_ddd_compliance.py                      │
   ├── ci/check_conf_sync.sh                              │
   └── valueStream Go tests                               │
                                                          │
hooks/* ─── 各子仓库 .git/hooks/pre-commit                │
```

---

## 四、每个脚本的服务归属与消费者分析

### 问题 1：这些脚本都是给哪些服务用的？

#### A. 配置管理脚本

| 脚本 | 消费者（服务/调用方） | 调用方式 |
|------|----------------------|----------|
| **conf-read.py** | **taskSSE** | `taskSSE/src/config.mjs:67,78` — `execSync('python3 scripts/conf-read.py ...')` 获取运行时配置 |
| | **Vite (vue 前端)** | 构建时通过 shell 调用 `conf-read.py vue apiBaseUrl` 注入前端配置 |
| | **Playwright** | E2E 测试通过 `conf-read.py snapshot-json` 获取全量配置快照 |
| | **开发者** | CLI 查询：`conf-read.py django port`、`conf-read.py git-oauth <key>` |
| **conf-sync.py** | **8 个 conf 子目录** | `conf/core/django/sync.sh`、`conf/frontend/vue/sync.sh`、`conf/auth/git-oauth/sync.sh`、`conf/auth/task-auth/sync.sh`、`conf/events/domain-events/sync.sh`、`conf/ai/ai-provider/sync.sh`、`conf/gateway/task-sse/sync.sh` 各自调用 `conf-sync.py <app-name>` |
| **conf-sync-all.sh** | **所有 conf 消费者** + **CI** | 遍历 `conf/*/*/` 全部子目录批量同步；`ci/check_conf_sync.sh` 用它验证一致性 |
| **conf_lib.py** | **conf-sync.py** + **migrate_port_config_to_conf.py** | Python import 共享库 |
| **migrate_port_config_to_conf.py** | **一次性迁移**（已执行完毕） | 从 `task2app/conf/port_config.json` → `conf/<app>/config.yaml` 的历史迁移 |

#### B. Docker 环境脚本

| 脚本 | 消费者 | 调用方式 |
|------|--------|----------|
| **docker-env.sh** | **6 个 docker-* 脚本** + **runall-local-promtail.sh** | `source` 引入：`docker-use-remote.sh`、`docker-use-local.sh`、`docker-status.sh`、`docker-tunnel-remote.sh`、`docker-context-init.sh`、`docker-uninstall-desktop.sh` |
| **docker-shell.sh** | **开发者的 `~/.zshrc`** | `docker-setup.sh` 将 `source scripts/docker-shell.sh` 写入 `~/.zshrc`，提供 `dr/dl/ds/dt/dru/dlu/dti` 快捷别名 |
| **docker-setup.sh** | **开发者（首次初始化）** | 手动执行：初始化 context → 切换远程 → 写入 `.zshrc` |
| **docker-tunnel-remote.sh** | **所有需要访问远程容器端口的场景** | 通过 `dt` 别名或直接执行，转发 Redis(6379)、Kafka(9092/9093)、Grafana(3000)、Loki(3100)、OTEL(4317/4318)、GitLab(8012/2222) |
| **docker-install-cli.sh** | **新开发者机器初始化** | `docker-setup.sh` 提示执行 |
| **docker-context-init.sh** | **docker-setup.sh** + **docker-install-cli.sh** | 被调用以创建远程/本地 context |
| **docker-use-remote.sh** | **开发者** (`dr` 别名) | 切换 Docker context 到 `zcpu-remote` |
| **docker-use-local.sh** | **已废弃** | 打印警告引导至远程方案 |
| **docker-status.sh** | **开发者** (`ds` 别名) | 查看当前 context、隧道状态、运行容器 |
| **docker-uninstall-desktop.sh** | **开发者（迁移用）** | 从 Docker Desktop 迁移到远程 Docker CLI 方案 |
| **docker-desktop-helper.sh** | **docker-setup.sh** | 提供 Docker Desktop 检测、内存警告等 Mac 端辅助函数 |

#### C. CI 检查脚本

| 脚本 | 检查的服务 | 说明 |
|------|-----------|------|
| **ci/check_ddd_bdd_compliance.py** | **task2app** (Python DDD/BDD) + **valueStream** (Go DDD) + **conf/** (sync) + **valueStream** (Go tests) | 4 合 1 总入口 |
| **ci/check_go_ddd_compliance.py** | **valueStream** | 检查 `valueStream/domain/**/*.go` 的 DDD 合规性 |
| **ci/check_taskgateway_routes.sh** | **taskGateway** + **APIsix** | 验证路由配置一致性 |
| **ci/check_conf_sync.sh** | **conf/** 全部目录 | 验证 conf sync 后无漂移 |
| **ci/smoke_taskgateway.sh** | **taskGateway** | 路由检查 + health endpoint 冒烟 |

#### D. 日志采集

| 脚本 | 消费者 | 说明 |
|------|--------|------|
| **runall-local-promtail.sh** | **runAll** (日志产出方) + **AiMonitor** (Promtail compose 文件) + **Loki** (日志消费方) | 跨 3 个服务的胶水脚本 |

#### E. Git Hooks 模板

| 模板 | 目标仓库 | 安装状态 |
|------|----------|----------|
| **pre-commit-go** | go_relayToTrae, go_run_container, runAll, valueStream, **taskAuth**, **taskBill**, **taskEvents** | 需手动安装 |
| **pre-commit-django** | gitOauth | 需手动安装 |
| **pre-commit-jest** | DaydaymoneyGrafana | 需手动安装 |
| **pre-commit-node** | **taskSSE** | 需手动安装 |
| **pre-commit-python-scripts** | AiMonitor | 需手动安装 |
| **pre-commit-noop** | gitService, mock_run_container | 占位符 |

---

## 五、为什么这些脚本没有放在各自的服务目录中？

这是核心的设计决策问题。每个脚本留在 `scripts/` 而非服务目录，各有原因：

### 模式 1：跨多个服务操作（无法归属）

| 脚本 | 涉及服务 | 如果放在某个服务下会怎样？ |
|------|---------|--------------------------|
| `conf-read.py` | taskSSE + Vite + Playwright + 开发者 CLI | ❌ 放在 taskSSE 下，Vite 构建时就要 `../taskSSE/scripts/conf-read.py`，产生反向依赖 |
| `conf-sync.py` / `conf-sync-all.sh` | 8 个 conf 子目录 | ❌ 放在任一 conf 子目录都不对——它是 conf/ 全域的工具 |
| `ci/check_ddd_bdd_compliance.py` | task2app + valueStream + conf | ❌ 放在 task2app 下，Go 项目却要依赖 Python 项目 |
| `runall-local-promtail.sh` | runAll + AiMonitor + Loki | ❌ 放在任一服务下，都要跨服务引用 compose 文件 |

**核心原因**：这些脚本是 **monorepo 级别的编排工具**，它们在多个服务之间做胶水工作。如果放在任一服务目录，会产生**反向依赖**（服务 A 的脚本被服务 B 调用）。

### 模式 2：开发者环境工具，不属于任何服务

| 脚本 | 说明 |
|------|------|
| `docker-env.sh` | 定义 SSH 主机、Docker context 名称、端口转发列表——这些是开发环境的**基础设施参数**，不属于业务代码 |
| `docker-shell.sh` | 被注入 `~/.zshrc`，提供交互式终端别名（`dr/dl/ds/dt`） |
| `docker-setup.sh` | 一次性开发环境初始化 |
| `docker-install-cli.sh` | Homebrew 安装，CI/Local 都可能用 |
| `docker-tunnel-remote.sh` | SSH 端口转发——纯运维工具 |
| `docker-context-init.sh` | Docker context 创建——纯环境配置 |

**核心原因**：这些是 **开发环境基础设施**，类比 `.devcontainer/` 或 `Makefile`。它们不参与任何服务的编译或运行，只影响开发者本机环境。放在服务目录下反而让人以为它属于该服务的业务逻辑。

### 模式 3：一次性数据迁移

| 脚本 | 说明 |
|------|------|
| `migrate_port_config_to_conf.py` | 从旧 `task2app/conf/port_config.json` → 新 `conf/<app>/` 目录。它天然是跨目录的——读源在 task2app，写目标在 conf/。放在任一目录都不合适。 |

**核心原因**：迁移的本质是 **从旧结构到新结构的桥**，执行完毕后源和目标的关联就断了。它是"一次性手术刀"，不是任何服务的持久组件。

### 模式 4：模板需要中性位置

| 模板 | 说明 |
|------|------|
| `hooks/pre-commit-go` | 被 7 个 Go 仓库**复制**安装。如果模板放在 `go_relayToTrae/scripts/` 下，其他仓库安装时就要 `cp ../go_relayToTrae/scripts/hooks/pre-commit ...`——不仅路径奇怪，而且会让 `taskAuth` 依赖 `go_relayToTrae` 的存在 |

**核心原因**：模板的"源"应该在所有消费者都能平等访问的中性位置。`scripts/hooks/` 是这个 monorepo 的**统一钩子注册表**。

### 模式 5：CI 脚本与所检查服务的关系

| 脚本 | 检查服务 | 为什么不在那个服务下？ |
|------|---------|----------------------|
| `ci/check_go_ddd_compliance.py` | valueStream | ✅ **实际上可以放** valueStream/。放在 scripts/ci/ 是因为它是 DDD 合规总检查链路中的一环（被 `check_ddd_bdd_compliance.py` 串联调用），且未来可能检查更多 Go 服务 |
| `ci/check_taskgateway_routes.sh` | taskGateway | ✅ **也可以放** taskGateway/。放在 scripts/ci/ 是因为它被 `smoke_taskgateway.sh` 引用，CI 视角的检查集中管理更方便 |
| `ci/check_conf_sync.sh` | conf/ 全域 | ❌ 跨所有 conf 子目录，只能在上层 |

**核心原因**：这些 CI 脚本 **可以放在各自服务下**（taskGateway、valueStream），但目前放在 `scripts/ci/` 是为了 **CI 检查的集中可发现性**——一个人看 `scripts/ci/` 就知道全部 CI 门禁。如果分散在各服务目录，需要逐个翻找。

### 模式 6：历史遗留（可以考虑移动的）

| 脚本 | 当前 | 建议 |
|------|------|------|
| `ci/check_go_ddd_compliance.py` | scripts/ci/ | 可考虑移到 `valueStream/scripts/ci/`，在 `scripts/ci/check_ddd_bdd_compliance.py` 中更新路径引用 |
| `ci/check_taskgateway_routes.sh` | scripts/ci/ | 可考虑移到 `taskGateway/scripts/ci/`，`smoke_taskgateway.sh` 和 CI 总入口更新路径 |
| `ci/smoke_taskgateway.sh` | scripts/ci/ | 同上，和 taskGateway 服务放在一起更自然 |
| `docker-use-local.sh` | scripts/ | 已废弃，建议移除 |

---

## 六、决策框架

总结脚本放置的通用原则：

```
脚本是否操作 ≥2 个服务？      → Yes → scripts/（monorepo 级工具）
脚本是否是开发者环境配置？     → Yes → scripts/（基础设施）
脚本是否要被多个仓库复制安装？ → Yes → scripts/hooks/（中性模板源）
脚本是否是跨结构的一次性迁移？ → Yes → scripts/（临时手术刀）
脚本是否只服务单一服务？       → Yes → <service>/scripts/（就近原则）✅
脚本是否是 CI 门禁检查？       → Yes → 可以在 scripts/ci/（集中可发现），也可以在 <service>/scripts/ci/（就近原则），两种都可以
```

当前 `scripts/` 中有约 **3-4 个脚本**（Go DDD 合规检查、taskGateway 路由检查/smoke、已废弃的 docker-use-local.sh）可以考虑移到对应服务目录下。其余脚本的 monorepo 级定位是合理的。

---

## 七、设计原则总结（来自脚本分析）

1. **远程优先**：Docker 守护进程运行在远程 ZCPU 云主机，本地仅保留 CLI + SSH 隧道
2. **配置分散化**：从单一大 JSON 迁移到按 app 分目录的 YAML，通过 manifest + pick 机制派生
3. **CI 门禁分层**：DDD 合规（架构约束）→ 配置同步（一致性）→ 路由检查（API 契约）→ 冒烟测试（可用性）
4. **钩子统一策略**：所有 pre-commit 模板共享同一逻辑——相关优先 + 随机抽测兜底
5. **环境自适应**：Mac 端和 Linux 端使用不同的 helper 实现（`docker-desktop-helper.sh` vs `remote-compose-helper.sh`），但保持函数签名一致

---

## 八、价值流影响

本次为分析/文档任务，不涉及代码变更，无价值流影响。

---

## 九、待确认项

1. `runall-remote-docker.sh` 在 `docker-shell.sh:27` 中被引用，但 scripts/ 目录中未找到——确认该文件是否在 `runAll/` 或其他目录中？
2. `docker-use-local.sh` 已标记 deprecated，是否计划在下个版本中移除？
3. `hooks/pre-commit-noop` 覆盖的 gitService、mock_run_container 仓库是否有添加测例的计划？
