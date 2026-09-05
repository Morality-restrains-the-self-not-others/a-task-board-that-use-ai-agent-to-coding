# 新节点克隆 daydaymoney-deploy：只拷 conf-local 是否足够

- **日期**: 2026-08-31
- **状态**: accepted（/1-brainstorming 总体设计审批：repin_release，2026-08-31）
- **迭代**: daydaymoney-deploy-new-node-clone
- **作者**: cursor
- **意图**: `docs/intents/platform/daydaymoney-deploy-new-node-clone.intent.md`
- **既有 ADR**: [ADR-0052](../../adr/0052-binary-deploy-config-repo.md)、[ADR-0054](../../adr/0054-conf-local-secrets-only.md)（本迭代不新开选型 ADR）
- **python_api_approval**: not_applicable（无新增 Python/HTTP 接口）

## 结论（先答用户问题）

**不够。** 对当前 `/tmp/ram-work/.daydaymoney-deploy-seed` 的加载逻辑，单独 `git clone` 到新节点后**只手动拷 `conf-local/` 不能拉起栈**。

分两层：

| 层 | 现状 | 只拷 conf-local |
|----|------|-----------------|
| **seed 仓脚本（磁盘上的 `.daydaymoney-deploy-seed`）** | `scripts/up.sh` 仍按 **`*.local.yaml` / `secrets/conf/`** overlay；`.gitignore` **没有** `/conf-local/`；README 仍指向已撤销的 `HOST_SECRETS.md` | **加载器不会吃这份 overlay**（除非你碰巧把文件放到它还在找的旧路径） |
| **源码仓已交付的 v123 目标**（`runAll/scripts/up-from-config-repo.sh` + `confload`） | YAML 只合 `conf/<rel>` → `conf-local/<rel>`；网关/OIDC PEM 仍在 `taskGateway/certs/`、`db/task-auth/` | 拷 conf-local **覆盖 YAML**；PEM 要另拷，除非本修订把它们迁进 conf-local |

v123 企业景观里「新机器只拷 conf-local」是**机密平面**的口号，不是完整 clone-run 清单。本修订把网关 TLS 与 OIDC 签名 PEM 也收进 conf-local，使**机密面真正一棵树**；产物、`INFRA_HOST`、Docker 仍在机密面之外。

## 背景

用户要判断：把 `daydaymoney-deploy`（本机 seed 即 `.daydaymoney-deploy-seed`）克隆到一台新机器后，是否只要把现网 `conf-local/` 拷过去就能跑。

对照物：

- **seed 仓**：独立 git 仓，最后提交 `ca59859`（2026-08-31 10:42）仍导出 HOST_SECRETS 登记册。
- **源码仓**：同日 17:33 已把 `up-from-config-repo.sh` 改成 rsync `secrets/conf-local/` + `taskGateway`/`db` PEM；`export-deploy-payload.sh` 会写 `/conf-local/` 进 `.gitignore`。**尚未 re-export 进 seed。**

## 当前架构理解

根据现行架构稿：

- 共有 2 个视图：`enterprise-landscape`、`application-integration`。
- 业务层：平台运维；「新机器只拷 conf-local」。
- 应用层：runAll `:9999`、`confload` / `conf_loader`（两步合并）。
- 技术层：`daydaymoney-deploy` 非机密 `conf/` + 主机 `conf-local/` overlay；产物走 GitHub Release。
- 上次更新：**v123 ✅ current** — conf-local 机密收口；**v122 🎯 target 积压** — 二进制部署（未 ship）。

📋 架构版本历史（近端）：

- v123 (2026-08-31 17:35) ✅ current — conf-local 唯一 overlay
- v122 (2026-08-30) 🎯 target — 二进制部署与独立配置仓（积压）
- v121 (2026-08-30) ✅ — 意见与建议链接

本次需求是**澄清 clone-run 边界**并修正 v123 口号，不改业务 API。批准后新增 **v124**，`@based_on: v123`。

## 🔍 Trace 日志分析

无 traceId。本需求为部署加载链分析，非单次请求排障。

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：Nodes 114 / Edges 1012 / Files 18；Languages javascript, typescript, python, bash；branch `main` |
| 关键发现 | 图未索引 Go `shareLib/confload` / seed 仓。爆炸半径依源码：`FindConfigRoot`（`root.go`）、`MergeConfLocal`（`overlay.go`）、`up-from-config-repo.sh` `overlay_manual_secrets`、seed 过期的 `scripts/up.sh` |
| 决策影响 | seed 与源码脚本漂移是主因；即使对齐，conf-local 也只覆盖「与 `conf/` 同相对路径」的 overlay |

`CRG unavailable for Go confload and .daydaymoney-deploy-seed: graph Languages = js/ts/python/bash; no shareLib/confload nodes.`

## 当前加载链（实测）

### 1. 配置根

`confload.FindConfigRoot`：`CONF_ROOT` → `DEPLOY_ROOT` → 向上找 `conf/base.yaml`。返回的是**含 `conf/` 的目录**（clone 根），不是 `conf/` 本身。

`MergeConfLocal(root, rel)` 读的是：

```
$DEPLOY_ROOT/conf-local/<area>/<app>/config.yaml
```

`up.sh` 写的 `cutover.env` 把 `CONF_ROOT=$DEPLOY_ROOT/conf`，解析后 root 仍是 `$DEPLOY_ROOT`，因此 **conf-local 必须放在 clone 根下、与 `conf` 符号链接并列**，不能放进 `envs/current/conf-local/`。

### 2. YAML 合并（源码仓 / 未来二进制）

```
1. conf/<app>/config.yaml          （seed 里已有，git 跟踪）
2. conf-local/<app>/<同名文件>     （gitignore，须手工放置）
不读 config.local.yaml
```

同规则还覆盖：Go `ReadAppFragment`、Python `conf_loader`、`gitService/scripts/load_gitservice_config.py`、`db/load` 的 `conf-local/db/registry.yaml` 只覆盖 MySQL `password`。

### 3. seed 仓 `scripts/up.sh`（与源码漂移）

当前 seed overlay：

1. 若 `SECRETS_DIR` 或 `secrets/` 下存在 `conf/` **或** `taskGateway` **或** `db` → **整树 rsync 到 `$DEPLOY_ROOT/`**（旧 HOST_SECRETS 树）。
2. 否则只把目录里的 `*.local.yaml` 拷进 **`$DEPLOY_ROOT/conf/`**。
3. **不识别 `conf-local/`**。

因此：在新节点把现网 `conf-local/` 直接放到 clone 根，**seed 的 `up.sh` 不会 rsync 它**；若二进制已是 v123 `confload`，进程自己会读到这份目录——但前提是你**已经**把目录放到 `$DEPLOY_ROOT/conf-local/`，而不是 `secrets/conf-local/` 且没跑对齐后的 `up.sh`。

### 4. 产物

`releases.yaml` 钉 GitHub Release `deploy-20260831`（17 个 ELF + `taskEvents-bin.tar.gz` + `taskFE-dist.tar.gz`）。`up.sh` 需要 `gh` + `GITHUB_TOKEN`（或 `ARTIFACTS_DIR`）才能 `deploy-sync`。**seed 不带 `bin/`。**

该 Release 打于 08:09–08:58，**早于** 17:33 的 confload 两步合并。新节点若只用这批 ELF，即使用上 conf-local，**旧二进制仍可能去读 `config.local.yaml`**。须重新发布含 `MergeConfLocal` 的 ELF，或新节点用本机构建产物。

## 只拷 conf-local 仍缺的清单

### A. 机密平面（与 conf 镜像 — 拷 conf-local 能覆盖）

本机 `conf-local/` 相对路径与 `conf-local.example/` 同构（PayPal/Stripe/SSO/SMS/微信/OAuth/COS/runAll env 等）。**这一块拷过去即可**，加载器（v123）会合并。

### B. 网关 / OIDC PEM（修订：迁入 conf-local）

当前读取点：

| 文件 | 谁读 | 缺失行为 |
|------|------|----------|
| `taskGateway/certs/dev-gateway.pem` + `-key.pem` | APISIX bind-mount `./certs`；`conf/gateway/task-gateway/config.yaml` `tls.certFile` | `run.sh setup_tls` **openssl 现签**，CN 写死 `10.2.150.89` |
| `db/task-auth/oidc_signing_key.pem` | `taskAuth` `defaultSigningKeyPath()` | **现生成一把新 RSA** 并写入该路径 → GitLab SSO / 已发 token 全部失效 |

微信支付 PEM、Chrome 插件私钥已在 conf-local。网关与 OIDC 是机密面最后两块「树外文件」。

**规范落点（与 `conf/` 镜像）：**

```
conf-local/gateway/task-gateway/dev-gateway.pem
conf-local/gateway/task-gateway/dev-gateway-key.pem
conf-local/auth/task-auth/oidc_signing_key.pem
```

`conf-local.example/` 放同名空/占位文件（或 0 字节 + README 说明禁止真值）。

**运行时（不让 Docker/APISIX 直读 conf-local 绝对路径）：**

1. **网关** — `taskGateway/run.sh setup_tls`：若 conf-local 两文件都在，**拷到** `taskGateway/certs/` 再 compose（bind-mount 不变）。`DEPLOY_MODE=1` 且 conf-local 缺 PEM → **失败退出**，禁止再 openssl 现签（避免新节点静默拿到错误 SAN）。开发机无 overlay 时仍可生成，但 SAN 必须用 `INFRA_HOST`，禁止写死 `10.2.150.89`。
2. **OIDC** — `defaultSigningKeyPath` 改为 `FindConfigRoot()/conf-local/auth/task-auth/oidc_signing_key.pem`。`DEPLOY_MODE=1` 缺失 → **启动失败**（禁止 mint 新钥）。开发机缺失可生成到 conf-local（gitignore）。兼容：若旧路径 `db/task-auth/oidc_signing_key.pem` 存在且 conf-local 没有，启动时复制进 conf-local 并打 warn（一次性；不双写长期）。
3. **up.sh** — 删除对 `secrets/taskGateway`、`secrets/db` 的 rsync；只 overlay `conf-local/`。

迁完后：**机密面 = 只拷 conf-local 一棵树**（YAML + 全部 PEM）。

### C. 新节点必填（按来源分类）

**脚本会自动拉（不必手拷 ELF）** — `./scripts/up.sh`：

1. 若没有 `bin/runAll`：用 `gh release download` 按 `releases.yaml` 里 `runAll` 的 `github://…@tag` 引导下载。
2. 默认 `SYNC_ARTIFACTS=1`：再 exec `runAll -command deploy-sync`，把其余 ELF 和 `taskEvents-bin.tar.gz` / `taskFE-dist.tar.gz` 拉到 `bin/` 并 unpack。

前置：`gh` 已登录，或环境变量 `GITHUB_TOKEN`（私有仓 Release）。失败时脚本 **log 后继续**（`deploy-sync skipped or failed (last-good kept)`），不会硬退出——新机器没有 last-good，看起来「up.sh 成功了」但 `bin/` 仍空。也可设 `ARTIFACTS_DIR` 走本地文件，跳过 GitHub。

当前钉 `deploy-20260831` **早于** confload 两步合并；自动拉到的仍是旧 ELF（还会读 `config.local.yaml`）。新节点要用 v123 加载行为，须先打一版含 `MergeConfLocal` 的 Release，或暂时 `ARTIFACTS_DIR` 指向本机构建产物。

**须手拷**

| 项 | 说明 |
|----|------|
| `conf-local/` | 机密面（修订后含网关/OIDC PEM）；gitignore，Release 里没有 |

**须环境 / 本机能力（脚本不代劳）**

| 项 | 说明 |
|----|------|
| `INFRA_HOST` | `cutover.env` 与 `conf/base.yaml` 默认 **`10.2.150.68`**；不改则中间件地址指向旧机 |
| Docker + 镜像 + 端口 | MySQL/Redis/Kafka/GitLab/APISIX |
| `./runAll/run.sh` | source `cutover.env`（`CONF_ROOT` / `DEPLOY_MODE=1`）；勿先手 source |
| 9999 `INIT_ALL` | 新空库；datadir 不是 clone 的一部分 |
| clone-as-root 路径 | `runAll.yaml` `file_root: /tmp/ram-work/logs`；`AiMonitor/.env` 同类；仅当 `DEPLOY_ROOT != CONFIG_REPO` 时 layout 会 sed |

### D. 新独立环境 vs 同环境副本

- **同环境另一台机器**（同一域名、同一批 OAuth client）：拷完整 `conf-local/`（含 PEM），并改 `INFRA_HOST`；OAuth 回调仍指向原域名则还要 DNS/反代。还要 C 节产物与 Docker。
- **全新环境**：不能原样拷现网 conf-local。须从 `conf-local.example/` 填新密钥、新 `BASE_DOMAIN`、各 Git OAuth `client_secret`、新 GitLab、新支付商户。现网 conf-local 含本机 GitLab / daydaymoney 拓扑。

## 方案（选定）

YAML 两步合并不改（v123 已对）。本迭代把树外 PEM 收进 conf-local，并对齐 seed。

### P0 — 网关 / OIDC PEM 迁入 conf-local

1. 本机把现网 `taskGateway/certs/*.pem`、`db/task-auth/oidc_signing_key.pem` 拷到上表 conf-local 路径（真值不进 git）。
2. `taskGateway/run.sh`：优先从 conf-local staging 到 `certs/`；`DEPLOY_MODE` 缺文件失败。
3. `taskAuth` `defaultSigningKeyPath`：conf-local；`DEPLOY_MODE` 缺文件失败；旧路径一次性迁入。
4. `up-from-config-repo.sh` 去掉 `taskGateway`/`db` 目录 rsync；单测改断言只出现 `conf-local/`。
5. `conf-local.example/gateway/task-gateway/*.pem` 与 `auth/task-auth/oidc_signing_key.pem` 占位（空或 `replace-me` 文本头，禁止真密钥）。
6. 门禁：已跟踪树仍无 `*.pem`；`extract_conf_secrets_to_local.py` 若扫到 `certs/*.pem` / `oidc_signing_key.pem` 则提示迁 conf-local。

### P0 — 对齐 seed 与源码仓（关闭漂移）

1. 在源码仓执行 `bash scripts/seed-daydaymoney-deploy.sh`（或 `export-deploy-payload.sh`）刷新 `.daydaymoney-deploy-seed`：`scripts/up.sh`、`.gitignore` 含 `/conf-local/`、`secrets.example/README.md` 来自 `daydaymoney-secrets.example.md`、**删除** `HOST_SECRETS.md`。
2. 提交并推送配置仓（seed 是独立 git）。
3. 门禁已有：`test_export_deploy_payload.py` 断言无 HOST_SECRETS、gitignore 含 `/conf-local/`。

### P0 — 新节点清单写入 seed README（可执行）

克隆后最小步骤（PEM 迁入 conf-local 之后）：

```
git clone <daydaymoney-deploy>
rsync -a <src>/conf-local/  ./conf-local/
export INFRA_HOST=<本机可达 IP>
export GITHUB_TOKEN=...          # gh auth 亦可；脚本由此自动 deploy-sync
./scripts/up.sh                  # layout + 拉 Release 产物（写出 cutover.env）
# 起 Docker 基础设施 → ./runAll/run.sh（会 source cutover.env）→ 空库则 9999 INIT_ALL
```

明确：**不必手拷 bin/**。`up.sh` 在 `gh`/token 可用时自动拉 GitHub Release。手拷的只有 `conf-local/`。仍须本机 IP 与 Docker。

### P1 — 纠正架构口号

v124 enterprise-landscape：运维 note 改为「机密面只拷 conf-local（含网关/OIDC PEM）；另需 Release 产物、INFRA_HOST、Docker」。application-integration 增加 conf-local PEM staging → APISIX certs、taskAuth 读 conf-local 签名钥、以及「配置仓 clone → up.sh → GitHub Release」。不新增服务组件。

### 不做什么

- 不把 PEM 再塞进 git。
- 不把 MySQL datadir 当 clone 附件。
- 不新开 ADR（仍是 0052 交付面 + 0054 overlay）。

### P0 — 新 Release 钉（含 MergeConfLocal）

旧钉 `deploy-20260831` 的 ELF **没有**两步 conf-local 合并。本迭代构建并发布新 tag（建议 `deploy-20260831-conf-local` 或当日 `deploy-YYYYMMDD`），`releases.yaml` 全部 `github://…@tag` 改钉；`up.sh` 自动拉的就是能读 conf-local 的二进制。

## Domain Concept Inventory

| 概念 | 说明 |
|------|------|
| Bounded Context | 平台交付（daydaymoney-deploy），非业务域 |
| Key Entities | DeployRoot、ConfSkeleton、ConfLocalOverlay、ReleasePin |
| Candidate Aggregates | 一次 clone-run 装配（layout + overlay + deploy-sync） |
| Domain Events | 无（运维控制面） |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 新节点 clone-run 装配 | — | — | — | 运维交付，不产生业务领域事实 |
| 机密 overlay 合并 | — | — | — | 配置加载，无 MQ |

## 价值流影响

无业务 `value-stream.yaml` 步骤变更。属平台交付流（ADR-0052 clone-run）。Step 4 可标 `skipped_non_product` 或补一条 platform 运维流（非必须）。

## 🏛️ 架构变更影响

- **迭代版本**: v124 🎯 target
- **迭代名称**: daydaymoney-deploy-new-node-clone
- **作者**: cursor
- **设计日期**: 2026-08-31 18:17
- **新增文件**（每个视图四类伴生，缺一不可）:
  - 🆕 `docs/architecture/v124-enterprise-landscape-20260831-1817-cursor.puml`
  - 🆕 `docs/architecture/v124-application-integration-20260831-1817-cursor.puml`
  - 🆕 `docs/architecture/v124-enterprise-landscape-20260831-1817-cursor.diff.archimate`（增量变迁：v123→v124）
  - 🆕 `docs/architecture/v124-application-integration-20260831-1817-cursor.diff.archimate`
  - 🆕 `docs/architecture/v124-enterprise-landscape-20260831-1817-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v124-application-integration-20260831-1817-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v123-*-20260831-1703-cursor.puml` (current)
- **变更明细**: 🟢 新 Release 钉与 conf-local PEM / 🟡 up.sh、taskGateway、taskAuth / 🔴 secrets 树 overlay
- **变更明细**:
  - 🟡 [MODIFIED] 运维「新机器只拷 conf-local」→ 机密面只拷 conf-local（含网关/OIDC PEM）；另需 Release + INFRA_HOST + Docker
  - 🟡 [MODIFIED] taskGateway `setup_tls`、taskAuth 签名钥路径改读 conf-local
  - 🟢 [NEW] `conf-local/gateway/task-gateway/*.pem`、`conf-local/auth/task-auth/oidc_signing_key.pem`
  - 🔴 [DEPRECATED] `secrets/taskGateway`、`secrets/db` overlay；长期依赖 `db/task-auth/oidc_signing_key.pem` 作为 SSOT

## 验收

1. 对照本文「只拷 conf-local 仍缺的清单」人工走读 seed `up.sh` / 源码 `up-from-config-repo.sh`，条目一一有落点。
2. re-seed 后 `diff scripts/up.sh runAll/scripts/up-from-config-repo.sh` 仅 shebang 路径类差异（或零差异）。
3. 新节点实验（可选 OPS）：无源码树、有 conf-local + PEM + token + INFRA_HOST 时 `up.sh` 能 layout 且 `bin/runAll` 存在。

## 变更记录

- 2026-08-31：头脑风暴初稿；结论「只拷 conf-local 不够」
- 2026-08-31：用户问 — 产物由 `up.sh`/`deploy-sync` 自动拉 Release，不入手拷 ELF
- 2026-08-31：GitHub 下载失败 → 改为手拷 `deploy-binaries/` 到 `artifacts/`，`up.sh` 本地安装
