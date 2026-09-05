# 二进制部署 + 独立配置仓（与源码隔离）

- **日期**: 2026-08-30
- **状态**: accepted（/1-brainstorming 总体设计审批：approve_full，2026-08-30）
- **迭代**: binary-deploy-config-repo
- **作者**: cursor
- **意图**: `docs/intents/platform/binary-deploy-config-repo.intent.md`
- **拟 ADR**: ADR-0052（批准后写入 `docs/adr/`）
- **python_api_approval**: not_applicable（无新增 Python/HTTP 接口）

## 背景

当前运行时与源码绑在同一棵树上：

| 现状 | 后果 |
|------|------|
| `conf/` 在源码 monorepo（元规则 42 SSOT） | clone 源码即获得部署拓扑、端口、公网 URL |
| `confload.FindMonorepoRoot()` 依赖 `conf/base.yaml` / `dataMigrate/` / `.gitmodules` | 进程假设自己活在源码仓里 |
| runAll `working_dir: taskAuth` + `./bin/taskAuth` | 二进制与 Go 模块目录同处；部署机必须有源码子目录 |
| ADR-0027：重启只 exec last-good | 开发机已「不在 start 时编译」，但产物仍长在源码树里 |
| `dockerInfra/*/run.sh` 用 `WORKSPACE_ROOT/conf/infra/...` | 配方脚本与源码仓根绑定 |

用户确认的目标：

1. **部署机不 clone 源码仓** — 只放编译产物 + 配置仓。
2. **整棵运行时 `conf/` 迁出源码仓** — 含 `runAll.yaml`、`base.yaml`、各服务 `config.yaml`；源码仓只留 schema/示例。
3. **范围**：Go 业务二进制、taskFE 静态包、taskEvents worker、Docker 基础设施配方。

密钥未在问卷中勾选：**默认生产密钥不进配置仓**（主机 `config.local.yaml` / KMS）。配置仓本身私有，可含非密钥运行时参数。

## 成功标准

1. 一台只有 `$DEPLOY_ROOT/{conf,bin,dockerInfra,frontend}`、没有 `.gitmodules` 和业务 `*.go` 的机器，能用 runAll 拉起已钉版本的服务。
2. 改端口/域名/内存只提交配置仓；改业务逻辑只提交源码仓并走产物发布。
3. 升级失败时磁盘 last-good 二进制继续服务（ADR-0027 在无源码环境下仍成立）。
4. 开发者本机仍可 `./build.sh`；通过 `CONF_ROOT` 指向配置仓 clone，不必把生产 conf 写回源码仓。

## 当前架构理解（口头确认）

根据现行架构稿与仓库：

- 共有 2 个 **current** 视图：`enterprise-landscape`、`application-integration`（**v121** ✅，意见与建议链接）。
- 应用层：Go 微服务（taskAuth / taskBill / taskTaskService / …）、taskFE（Docker nginx 静态，ADR-0022）、taskGateway（APISIX）、taskEvents intent worker、runAll `:9999`。
- 技术层：同机 Docker 基础设施（MySQL / Redis / Kafka / GitLab）、日志 Loki/Grafana；配置 SSOT 在源码仓 `conf/<area>/<app>/`。
- 上次交付版本 **v121**（2026-08-30）。

📋 架构版本历史（近端）：

- v121 (2026-08-30) ✅ current — 意见与建议链接
- v120 (2026-08-30) 📦 — 任务/项目内容历史版本
- v118 (2026-08-29) 📦 — Git OAuth 资源使用标记

本次迭代在 **v121** 上增加交付拓扑（源码仓 / 配置仓 / 产物库），不改业务 API。

## 🔍 Trace 日志分析

无 traceId。本需求为交付拓扑，非线上请求排障。

## 🕸️ Code Review Graph 分析

```
CRG unavailable for Go: graph Languages = javascript, typescript, python, bash; no runAll/confload/taskAuth Go nodes.
```

设计依据源码：`shareLib/confload/load.go`（`FindMonorepoRoot`）、`conf/runAll.yaml`（`working_dir` + `start_command`）、ADR-0027 / ADR-0022 / ADR-0035、`dockerInfra/mysql/run.sh`（`WORKSPACE_ROOT/conf`）。

影响半径（静态）：

- `confload.FindMonorepoRoot` 的全部调用方（各 Go 服务 `config.go`）
- runAll `build_command` / `start_command` / `working_dir`
- 各 `dockerInfra/*/run.sh`、`gitService/run.sh`、`taskFE/app/scripts/runall-lifecycle.sh`
- 元规则 42 / 29 / `conf/ai.md`

## 方案（选定）

**三平面隔离**，部署机只组装后两面：

```
┌─────────────┐     CI publish      ┌──────────────────┐
│ 源码仓       │ ─────────────────► │ 产物库            │
│ code/tests  │   binaries+dist+   │ GitHub Packages   │
│ conf.example│   dataMigrate.tgz  │ Packages / SHA    │
│ dataMigrate │                    └────────┬─────────┘
└─────────────┘                             │ deploy-sync
                                            ▼
┌─────────────┐  git clone/pull    ┌──────────────────┐
│ 配置仓       │ ─────────────────► │ $DEPLOY_ROOT      │
│ conf/**     │                    │ conf/  (runtime)  │
│ releases.yaml│                    │ bin/   (gitignored│
│ dockerInfra │                    │ dockerInfra/      │
│ frontend nginx│                   │ frontend/releases │
└─────────────┘                    └──────────────────┘
```

### 1. 源码仓（本 monorepo）留下什么

| 留下 | 迁出 |
|------|------|
| 全部服务源码、测试、CI | 运行时 `conf/**`（含 `runAll.yaml`、`base.yaml`） |
| `conf.example/`：schema + 本地开发默认骨架 + 注释「勿当生产 SSOT」 | `dockerInfra/` 配方（compose / run.sh / load_*_conf.py） |
| `dataMigrate/**/*.sql`（仍随代码评审） | 生产 `gitService` 启动配方（仅 compose/run.sh；**不**搬 `gitlab-ce/` 源码树） |
| 开发用 `build.sh`、精准编译重启 | — |

CI（源码仓 tag / `main`）：交叉编译或本机构建 → 上传：

- 每个 Go 服务一个 ELF（`taskAuth`、`taskBill`、`taskEvents` worker、`runAll`、…）
- `taskFE` `dist` tarball（hashed 资产，ADR-0022）
- `dataMigrate-<gitsha>.tar.gz`（9999 初始化用）

坐标建议：**GitHub Packages**（Generic / 或 GHCR 仅当以镜像发布前端/辅助镜像），路径如 `https://github.com/task2money/daydaymoney-deploy/packages` 或源码仓下的 generic package `daydaymoney/<service>/<gitsha>`。不把二进制 commit 进配置仓 Git 历史；**不**用租户 GitLab Package Registry 存平台自身产物（权限面与产品 Git 混在一起）。

### 2. 配置仓（新建 Git 仓库）

建议名：**`daydaymoney-deploy`**（私有，托管在**平台 GitHub**，与工程源码仓同侧，例如 `github.com/task2money/daydaymoney-deploy`）。**不**放在租户/产品 GitLab（`gitService`）：那是业务 Git 托管，与交付配置隔离目标相反。

**一个仓、按环境分目录**（`envs/dev/`、`envs/prod/`），避免每环境一个仓导致 schema 漂移。

```
daydaymoney-deploy/
  README.md
  releases.yaml              # 产物版本钉（服务 → gitsha / package URL）
  envs/<env>/
    conf/
      base.yaml
      runAll.yaml
      <area>/<app>/config.yaml
      <area>/<app>/sync.manifest.yaml
    dockerInfra/             # 从本仓迁出的配方
    gitService/              # 仅 run.sh + compose，拉官方/私有镜像
    frontend/                # nginx compose + conf 引用
  scripts/
    deploy-sync.sh           # 按 releases.yaml 拉产物到 $DEPLOY_ROOT/bin
    conf-sync.sh             # 现有 conf-sync 逻辑，ROOT=$DEPLOY_ROOT
  .gitignore                 # bin/、config.local.yaml、volumes、logs
```

`releases.yaml` 示例：

```yaml
# 与 envs/<env> 可同提交或分文件
artifacts:
  taskAuth:     { sha: "abc1234", package: "github://task2money/daydaymoney-deploy/taskAuth@abc1234" }
  taskBill:     { sha: "abc1234", package: "github://task2money/daydaymoney-deploy/taskBill@abc1234" }
  taskEvents:   { sha: "abc1234", package: "github://task2money/daydaymoney-deploy/taskEvents@abc1234" }
  taskFE:       { sha: "abc1234", package: "github://task2money/daydaymoney-deploy/taskFE@abc1234" }
  runAll:       { sha: "abc1234", package: "github://task2money/daydaymoney-deploy/runAll@abc1234" }
  dataMigrate:  { sha: "abc1234", package: "github://task2money/daydaymoney-deploy/dataMigrate@abc1234" }
```

钉的粒度：**同一发布批次共用一个 gitsha**（整栈一致）；允许热修单服务单独钉（须在 PR 说明兼容性）。

### 3. 部署根 `$DEPLOY_ROOT`

环境变量 **`DEPLOY_ROOT`**（目录）+ **`CONF_ROOT`**（默认 `$DEPLOY_ROOT/conf`，可与 DEPLOY_ROOT 分离）。

```
$DEPLOY_ROOT/
  conf/                 # 配置仓 checkout 的 envs/<env>/conf
  bin/taskAuth          # deploy-sync 写入，gitignore
  bin/taskEvents/...
  dockerInfra/
  frontend/releases/<sha>/  + html → 当前 sha（ADR-0022 symlink）
  dataMigrate/          # 从 tarball 展开，不入库
  var/logs  volumes/    # 不入库
  conf/**/config.local.yaml  # 密钥，gitignore
```

runAll：

```bash
./bin/runAll --config "$CONF_ROOT/runAll.yaml"
```

`runAll.yaml` 的 `working_dir` 全部改为 `$DEPLOY_ROOT`（或相对配置仓根），`start_command` 为 `$DEPLOY_ROOT/bin/taskAuth`，**删除部署环境的 `build_command`**（或 no-op 并文档化为「禁止在部署机编译」）。

开发机保留源码仓内精准编译重启；9999 部署模式提供「按钉拉取并 swap」，语义对齐 ADR-0027 compile-then-swap，只是 compile 换成 download。

### 4. `confload` 根定位

将 `FindMonorepoRoot` 升级为 **`FindConfigRoot`**：

1. `CONF_ROOT` 若设置且含 `base.yaml` → 用之  
2. `DEPLOY_ROOT/conf/base.yaml`  
3. 现有向上查找 `conf/base.yaml`（兼容源码树开发）  
4. **不再**把 `.gitmodules` / `dataMigrate/` 当作「必须有源码仓」的成功条件（可作次要 hint，但有 `base.yaml` 即足够）

各服务运行时读边界仍守元规则 29：只读 `$CONF_ROOT/<area>/<app>/`。

`dockerInfra/*/run.sh`：`WORKSPACE_ROOT` 改为 `DEPLOY_ROOT`（`CONF_ROOT` 推导 conf 路径），禁止再假设「上两级是 monorepo」。

### 5. Schema 迁移（9999）

- SQL 真源仍在**源码仓** `dataMigrate/`（评审与代码同 PR）。
- 部署机：`deploy-sync` 展开与二进制同 sha 的 tarball 到 `$DEPLOY_ROOT/dataMigrate/`。
- `db/registry.yaml`：**迁配置仓**（哪几个库、脚本入口），因它描述「这台环境要 init 谁」，不是业务代码。
- 应用进程仍禁止启动时迁移（元规则 40）。

### 6. Docker / GitLab

- 配方（compose、`run.sh`、从 conf 导出 env）在配置仓。
- 镜像继续从 Registry 拉（GitLab-CE 官方镜像或已有私有镜像）；**配置仓不含 `gitService/gitlab-ce/` 源码**。
- 数据目录（MySQL datadir、`GITLAB_HOME`）落 `$DEPLOY_ROOT/volumes/` 或 conf 里已有的绝对路径，不进 Git。

### 7. 密钥（默认）

| 内容 | 位置 |
|------|------|
| 端口、域名模板、memLimit、非密钥开关 | 配置仓 `config.yaml` |
| DB 口令、云 AK、支付密钥、OIDC client secret | 主机 `config.local.yaml` 或 KMS；gitignore |
| 配置仓 ACL | 仅运维/发布角色；与源码仓开发者权限分离 |

### 8. 本地开发

| 角色 | 怎么跑 |
|------|--------|
| 写业务代码 | clone 源码仓；`CONF_ROOT` 指向配置仓的 `envs/dev`；`./build.sh` 产物可直接 exec |
| 只调运行时参数 | 只改配置仓 PR |
| 新环境引导 | 从源码 `conf.example/` 复制到配置仓 `envs/<new>/`，再填真实值 |

过渡期（expand/contract）：源码仓 `conf/` 与配置仓可双写一段时间，门禁警告「运行时 SSOT 已迁出」；截止日期后源码仓只留 `conf.example/`。

### 9. 明确不做

- 不用 Helm/K8s 替换 runAll（本迭代仍是主机 + Docker + 二进制）。
- 不把二进制推进配置仓 Git 历史。
- 不在部署机安装 Go 工具链作为启动依赖。
- 不为本次引入业务 Kafka 事件。

## Alternatives Considered

### A. 源码仓保留 conf 骨架，配置仓只 overlay

- **Pros:** 与元规则 42 改动小。
- **Cons:** 两份 YAML 合并规则复杂；部署机仍可能误用源码默认；隔离目标弱。
- **Why rejected:** 用户选择整棵运行时 conf 迁出。

### B. 部署机仍 clone 源码，仅禁止编译

- **Pros:** 路径改动小（ADR-0027 已接近）。
- **Cons:** 源码与配置不隔离；磁盘与权限暴露全部代码。
- **Why rejected:** 用户选择部署机无源码树。

### C. 配置与产物都放进配置仓 Git LFS

- **Pros:** 一次 clone 齐。
- **Cons:** Git 不适合 ELF；回滚/权限/体积差。
- **Why rejected:** 产物走 Package Registry，配置仓只钉坐标。

## 领域概念清单（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| Bounded Context | **交付/运维**（非租户业务域）；与 Auth/Bill/Task 正交 |
| DeployRoot | 部署机组装根，不含源码 |
| ConfigRepository | 运行时 conf + 配方 + 版本钉 |
| ArtifactPin | `releases.yaml` 中服务 → sha/URL |
| ReleaseBatch | 一次 CI 发布的一组同 sha 产物 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 部署机仅用二进制 + 独立配置仓运行 | — | — | — | 运维交付拓扑，不产生业务领域事实 |

## 价值流影响

- **现有流**：`runall-*`（cascade、precise-restart、build-all、group-build）需增加「部署模式：无 build_command / 按钉拉取」。业务流（user-auth、task-management 等）**不改字段**。
- **新流（建议，step 4 切片）**：`platform-delivery` / `binary-deploy-config-isolation` — 步骤：发布产物 → 钉版本 → deploy-sync → runAll start（无源码）。
- **字段**：无业务表变更。配置平面新增 `releases.yaml` 键，不是 `<service>.<table>.<column>`。
- **测试**：新增 `confload` 根定位与 deploy-sync 单测；runAll 启动路径回归「缺二进制不编译」。

## 🏛️ 架构变更影响

- **迭代版本**: v122 🎯 target
- **迭代名称**: binary-deploy-config-repo
- **作者**: cursor
- **设计日期**: 2026-08-30 15:31
- **新增文件**（每个视图四类伴生格式，**缺一不可**）:
  - 🆕 `docs/architecture/v122-enterprise-landscape-20260830-1531-cursor.puml`
  - 🆕 `docs/architecture/v122-application-integration-20260830-1531-cursor.puml`
  - 🆕 `docs/architecture/v122-enterprise-landscape-20260830-1531-cursor.diff.archimate`（增量变迁：v121→v122 变更元素 + Plateau/Gap/WP 链）
  - 🆕 `docs/architecture/v122-application-integration-20260830-1531-cursor.diff.archimate`（增量变迁视图）
  - 🆕 `docs/architecture/v122-enterprise-landscape-20260830-1531-cursor.full.archimate`（全量拓扑：变迁后完整架构）
  - 🆕 `docs/architecture/v122-application-integration-20260830-1531-cursor.full.archimate`（全量拓扑视图）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v121-*-*.puml` (current)
- **变更明细**: 🟢 daydaymoney-deploy / GitHub Packages / DEPLOY_ROOT / deploy-sync 🟡 runAll + confload 🔴 源码仓 conf 运行时 SSOT

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量模型 — Plateau v121（Current 基线）+ Plateau v122（Target）+ Gap + WorkPackage 链 + 本次 🟢/🟡/🔴 变更元素；视图 `架构变迁 v121→v122 — binary-deploy-config-repo` + 变更目标拓扑（须含 `sourceConnection` 连线） |
| **`.full.archimate`** | 全量模型 — 变迁后完整架构（v121 意见链接业务保留 + 交付拓扑 + 变更标注）；视图 `v122 全量拓扑`（可导入 Archi 打开看全量；须含 `sourceConnection` 连线） |

## 权限影响分析

见 `docs/superpowers/specs/2026-08-30-binary-deploy-config-repo-permission-analysis.md`。无新租户 API；配置仓/Packages ACL 为工程运维面。绿灯。

## 角色与权限建模

不新增租户角色。`deploy_ops` 仅工程 GitHub 写配置仓 + 部署机只读拉包。

## 安全审查结论

密钥不进 Git；配置仓私有且不放在租户 GitLab；9999 不增加远程设 `CONF_ROOT` 的接口。

1. **P0** `FindConfigRoot` + `CONF_ROOT`/`DEPLOY_ROOT`（行为兼容现网）
2. **P1** 建配置仓，复制当前 `conf/` + docker 配方；本机 runAll 改指向；源码仓 conf 仍在（双写）
3. **P2** CI 发布 Generic Package；`releases.yaml` + `deploy-sync.sh`；部署模式去掉 `build_command`
4. **P3** 源码仓 `conf/` → `conf.example/`；门禁禁止把生产 conf 提交回源码仓
5. **P4** 独立部署机验收：无源码 clone 全绿 — **2026-08-30 本机 `/tmp/ram-deploy` 已过**：无 `.gitmodules` / 服务 `*.go`；`taskAuth` exe=`$DEPLOY_ROOT/bin/taskAuth` cwd=部署根；80/81 healthy（`git-service` 仍 empty，ADR-0047）。跨机 GitHub Release 见 OPT-20260830-019。

## 元规则 / ADR 回写（P3 必须）

- 更新 `.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`：运行时 SSOT = 配置仓
- 更新 `conf/ai.md` companion：本树为例（example）或删除
- ADR-0052 accepted；ADR-0027 补充「部署机 download-then-swap」
- 约束 42 精准编译重启：**仅源码工作树**；部署机对应「拉产物重启」
