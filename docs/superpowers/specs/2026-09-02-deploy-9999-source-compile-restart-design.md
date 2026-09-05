# 部署机 9999：源码编译 + 增量安装 + 精准/全量重新编译

- **日期**: 2026-09-02
- **状态**: accepted（/1-brainstorming 总体设计审批：approve_sync_conflocal，2026-09-02）
- **迭代**: deploy-9999-source-compile-restart
- **作者**: cursor
- **意图**: `docs/intents/platform/deploy_9999_source_compile_restart.intent.md`
- **拟 ADR**: ADR-0056（批准后写入；局部修订 ADR-0052「精准编译重启仅源码树开发工具」一句）
- **python_api_approval**: not_applicable（无新增 Python HTTP 接口；扩展既有 Go runAll）

## 背景

ADR-0052 把交付拆成三平面：源码仓 / 配置仓 `daydaymoney-deploy` / 产物。本机已 clone-run：业务进程从部署根 exec，9999 对外是 `http://192.168.1.10:9999/`。

`DEPLOY_MODE=1` 时 `resolveBuildCommand` 返回空字符串。因此：

| 按钮 | 源码树语义 | 当前部署树实际 |
|------|------------|----------------|
| 精准编译重启 | 登记服务 compile-then-swap | 无 `build_command` → 等价「只重启」或空转 |
| 全部重新编译 | 全部可编译服务只编不杀 | 全部被 skip（`noBuild`） |

日常补救是 `.daydaymoney-deploy-seed/update.sh`：

```bash
git pull
rm -rf artifacts/
cp -r /tmp/ram-work/deploy-binaries artifacts
rm -rf conf-local
cp -r /tmp/ram-work/conf-local .
./scripts/up.sh
./runAll/run.sh
```

痛点：整树拷贝、整棵机密覆盖、`up.sh` 全量装配、再拉一次 runAll。改一个 Go 服务也要走完整 bootstrap。

源码侧已有 `scripts/precise-compile.sh`（只编译、写入 `deploy-binaries/`、不重启）。缺的是**部署 9999 把这条链接上**，并做成增量。

## 当前架构理解（口头确认）

根据现行架构稿与仓库：

- 共有 2 个 **current** 视图：`enterprise-landscape`、`application-integration`（**v126** ✅）。
- 业务层：租户成员 / 项目 OAuth / 自动运行（v126）；平台运维 clone-run 在 **v124** 已交付后被后续业务迭代叠在 current 上。
- 应用层：Go 微服务、taskFE、taskGateway、taskEvents、**runAll `:9999`**（`DEPLOY_MODE=1` 时 `working_dir` 映射到 `$DEPLOY_ROOT`）。
- 技术层：同机 Docker 基础设施；运行时 `conf/` 来自 `daydaymoney-deploy`；机密 `conf-local/`；产物 `artifacts/` → `bin/`。
- 上次交付版本 **v126**（2026-09-01 23:10）。

📋 架构版本历史（近端）：

- v126 (2026-09-01) ✅ current — 项目 L2 种下自动运行评论
- v125 (2026-09-01) 📦 — ImageMarket 恢复厂商申请入口
- v124 (2026-08-31) 📦 — 新节点 clone-run
- v123 / ADR-0052 — 二进制部署与独立配置仓

本次迭代在 **v126** 上补交付控制面（源码树 ↔ 部署 9999），不改业务 API。

## 🔍 Trace 日志分析

无 traceId。本需求为运维控制面，非单次请求排障。

## 🕸️ Code Review Graph 分析

```
CRG unavailable: `.code-review-graph/graph.db` 不存在。
```

设计依据静态源码：

| 符号 | 文件 | 影响 |
|------|------|------|
| `resolveBuildCommand` | `runAll/src/service_build.go` | `DEPLOY_MODE` 清空编译 |
| `Runner.PreciseRestart` | `runAll/src/precise_restart.go` | 现网 compile-then-swap |
| `Runner.BuildAll` | `runAll/src/runner.go` | 只编不杀；完成后清登记 |
| `precise-compile.sh` | `runAll/scripts/precise-compile.sh` | 源码编译 → `deploy-binaries/` |
| `install-local-artifacts.sh` | `runAll/scripts/install-local-artifacts.sh` | artifacts → bin / unpack；copy-then-mv |
| `update.sh` | `.daydaymoney-deploy-seed/update.sh` | 全量同步（将被降为应急） |

爆炸半径：runAll Go 编排 + 上述脚本；不改业务服务 HTTP。

## 问题分析

同一宿主机上两棵树：

```
/tmp/ram-work                    源码仓（Go toolchain、.runall 登记）
  deploy-binaries/               gitignored 产物暂存
.daydaymoney-deploy-seed 或 DEPLOY_ROOT
  artifacts/  bin/  conf/  conf-local/
  bin/runAll                     对外 :9999
```

`update.sh` 把「配置仓更新 + 机密同步 + 产物同步 + 编排器重启」绑成一次。日常编码只需要后半段的**产物增量 + 进程 swap**。

## 方案（选定）

**部署侧 runAll 作为编排器**，同机 `SOURCE_ROOT` 子进程编译，增量安装后再按按钮语义决定是否重启。不新开 HTTP 服务、不改按钮 URL。

```
浏览器  http://192.168.1.10:9999/
            │  既有 POST /api/precise-restart
            │  既有 POST /api/build-all
            ▼
     runAll (DEPLOY_MODE=1)
            │
            ├─ 读 $SOURCE_ROOT/.runall/precise_restart_services.txt
            ├─ unset DEPLOY_MODE; bash $SOURCE_ROOT/scripts/precise-compile.sh …
            ├─ 编译成功后 rsync conf-local/（增量，--delete）
            ├─ 增量 install-local-artifacts（仅变化文件 copy-then-mv）
            └─ 精准：restartService（无 build_command）
               全量编译：到此结束（不杀进程；YAML 须后续重启才进已运行进程）
```

### 1. 配置

在部署根 **`conf-local/`** 增加（gitignore，不进配置仓 Git）：

```yaml
# conf-local/core/runall/config.yaml  或  cutover.env 等价项
# 推荐 cutover.env 一项，避免再开 conf app 目录：
#   export SOURCE_ROOT=/tmp/ram-work
```

选定：**`SOURCE_ROOT` 环境变量**（由 `cutover.env` / `conf-local` overlay 的 runAll env 注入）。理由：`cutover.env` 已是 `DEPLOY_MODE`/`DEPLOY_ROOT` 的 SSOT，避免第三套 YAML。缺省时 `DEPLOY_MODE=1` 的两个按钮 **fail-fast**。

源码树 9999（若仍存在）行为不变：继续本地 `build_command`。本机应以部署 9999 为唯一 :9999。

### 2. 精准编译重启（`DEPLOY_MODE=1`）

保持 ADR-0027 compile-then-swap，把「compile」换成「源码编译 + 增量安装」：

1. 重载磁盘 `runAll.yaml`（现有 `precise_restart_reload.go`）。
2. 解析登记：路径改为 **`$SOURCE_ROOT/.runall/precise_restart_services.txt`**（不是部署根 `.runall/`）。Agent/hook 仍写源码仓。
3. 空登记 → 现网提示，return。
4. `exec`：`SOURCE_ROOT/scripts/precise-compile.sh <names…>`，cwd=源码根，**子进程 unset `DEPLOY_MODE`**，`META_ROOT=$SOURCE_ROOT`。
5. 编译失败 → 不 rsync、不 install、不 stop、登记保留、SSE failed。
6. **`rsync -a --delete`** `$SOURCE_ROOT/conf-local/` → `$DEPLOY_ROOT/conf-local/`（审批 `approve_sync_conflocal`：与 `update.sh` 同语义，但只传变化文件；同机假设下无 dest-only 机密）。
7. 增量安装：`$SOURCE_ROOT/deploy-binaries/` → `$DEPLOY_ROOT`。对登记服务映射到的产物做 size+mtime（或 sha）跳过；变化项 `install_over`。
8. 按依赖序 `restartService`（`resolveBuildCommand` 仍为空，只 stop→start→health）。已重启的服务会读到新 conf-local。
9. 成功项从**源码仓**登记文件清除；写 consumed-at 水位线（路径也在 `SOURCE_ROOT/.runall/`）。

`runAll` 自身若在登记中：允许安装新 ELF，**本轮不 restart 编排器**（避免把自己杀掉导致 SSE 中断）。进度注明「runAll 已安装，需手动重启编排器」。

### 3. 全部重新编译（`DEPLOY_MODE=1`）

与现网 BuildAll 对齐：**只让磁盘变新，不杀进程**。

1. `precise-compile.sh --all`
2. 编译成功后同样 `rsync -a --delete` conf-local
3. 增量安装全部变化产物
4. 正常完成则清源码仓精准登记（与现 `clearPreciseRestartRegistrationsAfterFullRebuild` 同语义，文件改指 `SOURCE_ROOT`）
5. `SyncMonorepoConf` 在 `DEPLOY_MODE` **跳过**（运行时 git 跟踪的 `conf/` 属配置仓，禁止从源码 `conf/` 覆盖部署 `conf/`；机密只走 conf-local rsync）

需要新进程生效时：再点「精准编译重启」（若仍有登记）或「全部重启」（只 exec last-good，此时 last-good 已是刚装上的 inode）。

### 4. 增量安装（相对 `update.sh` 的效率）

| 现状 `update.sh` | 本方案 |
|-----------------|--------|
| `rm -rf artifacts/` 再整树 `cp` | 按文件 size+mtime 跳过 |
| `rm -rf conf-local/` 再整树 `cp` | `rsync -a --delete`（只传变化文件） |
| `git pull` + `up.sh` | 不执行 |
| `./runAll/run.sh` 再拉编排器 | 编排器 PID 保持 |

`install-local-artifacts.sh` 增加 `copy_if_changed`（与 `collect-deploy-binaries.sh` 同策略）。可选：精准路径只处理映射表内文件，避免因 collect 扫描而 `mv` 未改 ELF。

服务 → 产物（最小集）：

| 登记名 / working_dir | 产物 |
|----------------------|------|
| `task-auth` / `taskAuth` | `bin/taskAuth` ← ELF `taskAuth` |
| `taskFE` | `taskFE-dist.tar.gz` → `taskFE/app/public` |
| `task-events-*` | `taskEvents-bin.tar.gz` → `taskEvents/bin` |
| 其它 Go `name` | `$DEPLOY_ROOT/bin/<working_dir 末段或 ELF 名>` |

`precise-compile.sh` 末尾仍全量 collect（内部已 skip unchanged）；瓶颈在 compile 与 unpack，不在 `git pull`/`up.sh`。

### 5. `update.sh` 降级

保留为**应急全量装配**（新节点、机密轮换、配置仓配方变更），README 写明：日常编码用 9999 两按钮。禁止 Agent 会话把 `update.sh` 当默认发布。

### 6. 不做什么

- 不在部署树恢复 `build_command`。
- 不引入 SSH/远程编译 Agent（本机 `SOURCE_ROOT` 足够；跨机是后续迭代）。
- 不 `git pull` 配置仓、不跑 `up.sh`、不重启 runAll 编排器。conf-local **会**随编译成功 rsync（用户审批）。
- 不新开 Python 接口；不新开独立「编译微服务」。

## Alternatives Considered

### Alternative 1: 在部署 9999 直接 `go build`（把源码再拷回部署根）

- **Pros:** 按钮逻辑少改
- **Cons:** 打破 ADR-0052「部署机无源码」
- **Why rejected:** 用户要的是分离后的效率，不是撤回分离

### Alternative 2: 继续只用 `update.sh`，脚本内做 rsync --checksum

- **Pros:** 零 Go 改动
- **Cons:** 仍 `up.sh` + 重启 runAll；9999 按钮仍空转；与约束 42 的「去页面点按钮」不一致
- **Why rejected:** 用户明确要 192.168.1.10:9999 执行那两个操作

### Alternative 3: 源码树另开 :9998 编译，部署 9999 只 download-then-swap

- **Pros:** 两进程职责更干净
- **Cons:** 两个 UI、两套 SSE、端口与操作习惯分裂
- **Why rejected:** 用户指定现有 9999

### Alternative 4: 全部重新编译在部署模式也重启全部业务进程

- **Pros:** 点一次就生效
- **Cons:** 违背 ADR-0027 与现网 BuildAll「只编不杀」；全栈抖动大
- **Why rejected:** 保持按钮语义；生效用精准重启或全部重启

## 🏛️ 架构变更影响

- **迭代版本**: v127 🎯 target
- **迭代名称**: deploy-9999-source-compile-restart
- **作者**: cursor
- **设计日期**: 2026-09-02 13:35
- **新增文件**（每个视图四类伴生格式，**缺一不可**）:
  - 🆕 `docs/architecture/v127-enterprise-landscape-20260902-1335-cursor.puml`
  - 🆕 `docs/architecture/v127-application-integration-20260902-1335-cursor.puml`
  - 🆕 `docs/architecture/v127-enterprise-landscape-20260902-1335-cursor.diff.archimate`（增量变迁：v126→v127 变更元素 + Plateau/Gap/WP 链）
  - 🆕 `docs/architecture/v127-application-integration-20260902-1335-cursor.diff.archimate`（增量变迁视图）
  - 🆕 `docs/architecture/v127-enterprise-landscape-20260902-1335-cursor.full.archimate`（全量拓扑：v126 业务 + 本迭代控制面）
  - 🆕 `docs/architecture/v127-application-integration-20260902-1335-cursor.full.archimate`（全量拓扑视图）
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v126-*-20260901-2244-cursor.puml` (current)
- **变更明细**: 🟢 SOURCE_ROOT / precise-compile / 登记读源码仓；🟡 runAll 9999 + conf-local rsync；🔴 日常 update.sh

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量模型 — Plateau v126（Current）+ Plateau v127（Target）+ Gap + WorkPackage 链 + 本次 🟢/🟡/🔴 变更元素；视图 `架构变迁 v126→v127` + 变更目标拓扑（含 `sourceConnection`） |
| **`.full.archimate`** | 全量模型 — v126 业务拓扑 + 部署 9999 控制面；视图 `v127 全量拓扑` + 架构变迁链 |

## Domain Concept Inventory

| 概念 | 说明 |
|------|------|
| Bounded Context | 平台交付 / runAll 控制面（非业务域） |
| Key Entities | SourceRoot、DeployRoot、PreciseRestartRegistry、DeployArtifact |
| Candidate Aggregates | 一次「编译批次」（登记集合或 `--all`） |
| Domain Events | 无 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 部署 9999 编排源码编译并增量安装 | — | — | — | 运维控制面 |
| 精准路径重启已安装二进制 | — | — | — | ADR-0027 进程编排 |

## 价值流影响

无 `value-stream.yaml`（仓库根未跟踪业务价值流文件）。无租户产品步骤变更。属平台交付控制面；Step 4 可 `skipped_non_product`。

## 🐍 Python 新增接口清单与 Go 替代评估

不触发。接口落点为扩展现有 Go `runAll` 的既有 path。

## 验收

1. 单元/脚本：见测试意图 T1–T9。
2. 本机：登记 `task-auth`（或当前脏服务），在 `http://192.168.1.10:9999/` 点精准编译重启 → 仅这些服务 stop/start；`pgrep -a runAll` 的 9999 PID 不变；部署根 `conf-local/` 与 `$SOURCE_ROOT/conf-local/` 内容一致。
3. 全部重新编译走完后业务 PID 不变，磁盘 ELF mtime 更新；再全部重启才换进程。
4. 拔掉 `SOURCE_ROOT` 后按钮失败且错误节点带 `data-traceId`。

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-09-02 | 头脑风暴初稿；同机 SOURCE_ROOT；update.sh 降为应急 |
| 2026-09-02 | 审批 `approve_sync_conflocal`：编译成功后 rsync conf-local；仍不 git pull / 不重启 runAll |
