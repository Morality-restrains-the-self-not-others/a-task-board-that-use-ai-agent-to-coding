# conf-local 机密收口（唯一 overlay + 历史泄露轮换）

- **日期**: 2026-08-31
- **状态**: accepted（/1-brainstorming 总体设计审批：approve，2026-08-31）
- **迭代**: conf-local-secrets-completion
- **作者**: cursor
- **意图**: `docs/intents/platform/conf-local-secrets-completion.intent.md`
- **既有 ADR**: [ADR-0054](../../adr/0054-conf-local-secrets-only.md) accepted；本迭代补齐未完成落地，不新开选型 ADR
- **python_api_approval**: not_applicable（无新增 Python/HTTP 接口）
- **续接**: 先前会话已抽已跟踪 `conf/` YAML 到 `conf-local/`、撤销 HOST_SECRETS（OPT-20260831-010/011/012）；[Conf-local key transfer status](39f49182-1a72-4f2b-bfe7-521a7ab0be15) 未完成收口

## 背景

ADR-0054 规定：**`conf/` 只放非机密骨架；机密只放 gitignored `conf-local/`，相对路径镜像。** 门禁 `check_conf_local_secrets.py` 目前对已跟踪 YAML 为绿。

未完成且构成现网风险的是加载器**仍会读** `config.local.yaml` / `*.local.yaml`，且排在 conf-local **之后**：

```
ReadAppConfig / load_app_config 当前顺序（有问题）
  1. conf/<app>/config.yaml
  2. conf-local/<app>/config.yaml    （真密钥）
  3. conf/<app>/config.local.yaml    （抽取后留下 key: ""）← 再合并，空串抹掉真值
```

**批准后的目标顺序只有两步**（2026-08-31 用户修订：不再依赖 `config.local.yaml`）：

```
1. conf/<app>/config.yaml
2. conf-local/<app>/config.yaml
```

`deepMergeMaps` 用空字符串覆盖已有值。本机仍有 7 个 gitignored `*.local.yaml`，机密键均为 `""`。只要加载器还读它们，**conf-local 的真值就会被抹掉。** 修法是停止读取，而不是再给 `.local.yaml` 留一条非机密 overlay。

（值未写入本文。对比用 SHA-256 前缀：空串为 `e3b0c44298fc`；`conf-local` 对应键均非空且与 leftover 不等。）

残留文件（仅列路径与键名）：

| 残留文件 | 空机密键 | 处置 |
|----------|----------|------|
| `conf/core/sms/config.local.yaml` 等纯空机密文件 | `access_key_*` 等 | 删除；加载器不再读 |
| `conf/auth/task-auth/config.local.yaml` | 公众号 token/secret | 删除；`appId` 若仍要本机覆盖则写入 `conf-local/auth/task-auth/config.yaml` |
| `conf/ai/ai-provider/config.local.yaml` | COS 密钥 | 删除；`backend` / CORS 写入 `conf-local/ai/ai-provider/config.yaml` |
| GitLab `runAllStartEnabled` 等非机密本机开关 | — | 写入对应 `conf-local/<app>/config.yaml`，不进已跟踪 `conf/` |

另外：`conf/infra/git-service/gitLabRootPwd.md` **仍被 conf 子仓跟踪**，HEAD 含 root 口令表（OPT-20260830-017）。这不是历史问题，是当前树明文。

`conf-local.example/` 只有 README，冷启动仍不知道要拷哪些相对路径——HOST_SECRETS 登记册撤销后，这个缺口没补上。

## 成功标准

1. 加载器**只**合并 `conf/<rel>` → `conf-local/<rel>`；**不读** `config.local.yaml` / `*.local.yaml`。
2. 本机残留 `conf/**/*.local.yaml` 删除（gitignore 可保留以防误提交）。
3. 抽取脚本只往 `conf-local/` 写，不再改写/生成 `.local.yaml`。
4. 已跟踪树无 `*Pwd.md` / 非空机密 Markdown；门禁覆盖 YAML **与** 这类文件。
5. `conf-local.example/` 有与已知机密路径同构的 **键名骨架**（值一律空/占位，禁止真密钥）。
6. 文档与 OPT 行动项不再把机密写入 `config.local.yaml`。
7. 曾进入 Git 的凭据按第 56 条 **废弃 + 重新签发**，新值只进 `conf-local/`（执行需 OPS 窗口；设计含清单与验证，不在本步改 live 口令）。

## 当前架构理解

根据现行架构稿：

- 共有 2 个视图：`enterprise-landscape`、`application-integration`。
- **v121** ✅ current — 意见与建议链接（taskFE / taskBill）。
- **v122** 🎯 target（未交付）— 二进制部署 + `daydaymoney-deploy`；ADR-0052 正文仍写主机密钥放 `config.local.yaml`。
- 应用层：Go 微服务 + taskFE + APISIX + taskEvents + runAll `:9999`。
- 技术层：同机 Docker 基础设施；运行时 conf 正在迁配置仓；机密面本应是主机 `conf-local/`。

📋 架构版本历史（近端）：

- v122 (2026-08-30) 🎯 target — 二进制部署与独立配置仓（积压）
- v121 (2026-08-30) ✅ current — 意见与建议链接
- v120 (2026-08-30) 📦 — 任务/项目内容历史版本

**警告**：已有 v122 target 积压。本迭代不改 v122 文件；批准后新增 **v123**，`@based_on: v122`，把 `conf-local` 画进技术层并修正「密钥在 config.local.yaml」的表述。

本次需求将在此基础上收口机密平面，不改业务 API。

## 🔍 Trace 日志分析

无 traceId。本需求为配置/密钥治理，非单次请求排障。

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：Nodes 114 / Edges 1012 / Files 18；Languages javascript, typescript, python, bash；branch `main` |
| 关键发现 | 图未索引 Go `shareLib/confload`。爆炸半径依源码：`ReadAppConfig` / `ReadAppFragment`（`load.go` 42–65、274–287 行）、Python `conf_loader.load_app_config`、`extract_conf_secrets_to_local.py`、`up-from-config-repo.sh overlay_manual_secrets`、`check_conf_local_secrets.py` |
| 决策影响 | 加载器必须停止读 `.local.yaml`；Go + Python + gitService 三处；残留文件删除以免误导 |

`CRG unavailable for Go confload: graph has no shareLib/confload nodes.`

## 方案（选定）

### A. 运行时：只合两层（P0）

```
1. conf/<app>/config.yaml          （及 ReadAppFragment 的 conf/<app>/<name>）
2. conf-local/<app>/config.yaml    （同相对路径；机密与本机非机密覆盖都在这里）
```

Go `ReadAppConfig` / `ReadAppFragment`、Python `load_app_config`、以及 `gitService/scripts/load_gitservice_config.py` **删除**对 `config.local.yaml` / `*.local.yaml` 的 merge。单测：磁盘上即使存在带空密钥或真值的 `config.local.yaml`，合并结果仍只来自 conf + conf-local。

### B. 本机残留清理（P0）

- 删除本机 `conf/**/config.local.yaml` 与 `*.local.yaml`（加载器已不读；留着只会误导）。
- 其中仍需要的非机密键（公众号 `appId`、COS `backend`/CORS、GitLab `runAllStartEnabled`）迁入对应 `conf-local/<rel>`。
- 抽取脚本：只写 `conf-local/`；不要再产出 `.local.yaml`。

### C. 门禁与 example 骨架（P1）

- `check_conf_local_secrets.py`：conf 子仓已跟踪 `*Pwd.md`、含 `Password:` 的运维备忘 → 失败。
- `conf-local.example/`：按现网 `conf-local/` **相对路径**生成键名骨架（值 `""` 或 `replace-me`），README 说明拷到 `$DEPLOY_ROOT/conf-local/`。禁止把真值写进 example。
- `up-from-config-repo.sh`：只 rsync `secrets/conf-local/`（或 `SECRETS_DIR/conf-local/`）到 `$DEPLOY_ROOT/conf-local/`；**禁止**再把 `*.local.yaml` 拷进 `conf/`。

### D. 文档（P1）

改正仍写「密钥/本机覆盖进 config.local.yaml」的落点：ADR-0052、ADR-0054「同目录 config.local.yaml」句、元规则 47/62、`scripts/seed-daydaymoney-deploy.sh`、`conf.example/runAll.deploy.yaml`、security-and-hardening 技能、BLOCK OPT（017 / 015-004 / 025-003）。本机覆盖一律 `conf-local/`。`.gitignore` 可继续忽略 `conf/**/config.local.yaml`，避免误提交残留文件，但加载器不消费它们。

### E. 历史泄露轮换（P1，OPS 窗口；禁止 filter-repo）

原则：第 56 条。进入过已推送历史的值视为已泄露。新值只写 `conf-local/`。

| # | 凭据 | 泄露面 | 轮换动作 | 写入 | 验证 | 既有 OPT |
|---|------|--------|----------|------|------|----------|
| 1 | GitLab root | **HEAD** `gitLabRootPwd.md` + 配置仓历史 | 实例改 root；删跟踪文件 | `conf-local/infra/git-service/root-password` 或密码管理器（不建 *Pwd.md） | 新口令可登录；旧口令失败；`git ls-files` 无该 md | OPT-20260830-017 |
| 2 | 阿里云短信 AccessKey | 曾跟踪 `conf/core/sms/config.yaml` 及 sync 片段 | RAM 禁用旧 Key、签发新 Key | `conf-local/core/sms/config.yaml` + 各 sync 镜像目录的 conf-local 片段（不要再写 `sms.local.yaml`） | 发一条验证码；Loki 无 InvalidAccessKey | OPT-20260825-003 |
| 3 | SSO JWT `ssoJwtSecret` | 曾跟踪 `conf/core/sso` 等 | 新随机 32 字节；taskAuth 与消费方同值 | `conf-local/core/sso/config.yaml` 及依赖该键的 conf-local 片段 | 新密钥 bridge 200；旧密钥 401 | OPT-20260806-062 同类；OPT-011 上下文 |
| 4 | Git OAuth `client_secret` | 曾跟踪 provider YAML | 各 IdP 控制台轮换 | `conf-local/auth/git-oauth/providers/*.yaml` 等 | 走一遍授权回调 | OPT-010 |
| 5 | PayPal / Stripe secret | 曾跟踪 billing YAML | 支付控制台轮换 | `conf-local/billing/paypal|stripe/config.yaml` | sandbox/live 一笔只读或小额 | OPT-010 / 011 |
| 6 | MySQL `db/registry.yaml` password | 曾跟踪非空口令 | 改 MySQL 用户口令 | `conf-local/db/registry.yaml` | 各服务连库成功 | OPT-20260831-011 迁出已做，轮换未做 |
| 7 | 微信 APIv3 / COS / 公众号 secret | 现网在 conf-local；是否进过 Git 需抽查 `git log -S` | **仅当**历史出现非空值才轮换；未进 Git 则不强制 | 只写 conf-local | 对应 live 路径 | OPT-20260815-004 改落点 |

轮换顺序建议：先 **(1) GitLab root**（HEAD 明文）与 **(2) 短信**（云 AK 可被滥用），再 SSO / OAuth / 支付 / MySQL（需停机或双密钥窗口）。

**禁止**：filter-repo、force-push 清历史、把新密钥写进 commit message / example / `config.local.yaml`。

Agent **不在无 OPS 窗口时改 live GitLab/云控制台**。实现阶段可先做 A–D；E 按上表在窗口内逐项勾选，勾完把对应 OPT 迁 completed。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 机密仅从 conf + conf-local 两层合并 | — | — | — | 运维/配置加载，不产生业务领域事实 |
| 废弃已泄露凭据并签发新值 | — | — | — | 密钥轮换是控制面操作；无 MQ。GitLab OIDC 轮换若走既有 `TENANT_GITLAB_OIDC_SSO_SECRET_ROTATED` 则复用该事件，不新造 |

## Domain Concept Inventory

- **Bounded Contexts**: 配置加载（confload）/ 部署主机密钥平面；非业务域。
- **Key Entities**: 无新业务实体。`conf-local/<rel>` 是主机 overlay（机密 + 本机非机密）；加载器不再读取 `config.local.yaml`。
- **Candidate Aggregates**: 无。
- **Domain Events**: 无新增。若租户 GitLab OIDC client secret 轮换，沿用已有 `tenant_gitlab_oidc_sso_secret_rotated`。

## 价值流影响

无产品价值流变更（`value-stream.yaml` 无对应 stream）。影响面是部署/配置平面，与 v122 二进制部署同一类。`/4-value-stream` 可跳过或只记「无 fields 变更」。

## 🐍 Python 新增接口清单与 Go 替代评估

不触发。无新 HTTP 接口；仅改 `confload` / `conf_loader` / CI / 脚本。

## 🏛️ 架构变更影响

- **迭代版本**: v123 🎯 target
- **迭代名称**: conf-local-secrets-completion
- **作者**: cursor
- **设计日期**: 2026-08-31 17:14
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v123-enterprise-landscape-20260831-1703-cursor.puml`
  - 🆕 `docs/architecture/v123-application-integration-20260831-1703-cursor.puml`
  - 🆕 `docs/architecture/v123-enterprise-landscape-20260831-1703-cursor.diff.archimate`（增量变迁：v122→v123）
  - 🆕 `docs/architecture/v123-application-integration-20260831-1703-cursor.diff.archimate`
  - 🆕 `docs/architecture/v123-enterprise-landscape-20260831-1703-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v123-application-integration-20260831-1703-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v122-*-20260830-1531-cursor.puml`（target 积压）
  - `docs/architecture/v121-*-20260830-0922-cursor.puml`（current）
- **变更明细**: 🟢 conf-local 为唯一 overlay；🟡 confload 只合 `config.yaml` + `conf-local`；🔴 加载路径废弃 `config.local.yaml` / `*.local.yaml`；🔴 `gitLabRootPwd.md`

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v122 → Gap（空 overlay 抹密钥 + HEAD 明文）→ WP → Plateau v123；目标拓扑含 conf-local 加载链（`sourceConnection`） |
| **`.full.archimate`** | v122 全量 + v123 机密平面；含意见链接业务拓扑与二进制部署链 |

> 老文件未被修改。目标架构将在 `/10-ship` 执行时切换为 current。

## 实施切片（批准后 /8-build）

1. 红：存在 `config.local.yaml` 时仍被 merge 的表征测试。
2. 绿：Go/Python/gitService 加载器只合 conf + conf-local；旁路 `.local.yaml` 不影响结果。
3. 删除本机残留 `.local.yaml`；非机密本机键迁 `conf-local/`；改抽取脚本。
4. 门禁 + example 骨架 + 文档/OPT 落点。
5. 从 conf 子仓删除 `gitLabRootPwd.md`（**须先轮换**或与轮换同一窗口，避免删文件后运维只剩已泄露口令）。
6. OPS 窗口执行轮换表 1–6，逐项验证后标记 OPT。

## 验收命令（实现后）

```bash
python3 db/scripts/ci/test_check_conf_local_secrets.py
python3 db/scripts/ci/check_conf_local_secrets.py
cd shareLib/confload && go test -count=1 -run 'TestReadAppConfig|TestIgnoresConfigLocalYaml|TestMergeConfLocal' .
python3 -m pytest runAll/scripts/tests/test_conf_local.py -q
git -C conf ls-files | rg -i 'Pwd\.md' ; test $? -eq 1
```

轮换项另用各控制台验证，不写入仓库。
