# conf 子仓非配置资产收口

- **日期**: 2026-09-01
- **状态**: accepted（/1-brainstorming 总体设计审批：approve，2026-09-01）
- **迭代**: conf-non-config-purge
- **作者**: cursor
- **意图**: `docs/intents/platform/conf-non-config-purge.intent.md`
- **既有 ADR**: [ADR-0052](../../adr/0052-binary-deploy-config-repo.md)（运行时 conf 与源码隔离；配方在 gitService / daydaymoney-deploy，不在 YAML 树）；不新开 ADR
- **python_api_approval**: not_applicable（无新增 Python/HTTP 接口）
- **架构更新**: 不需要（无服务/数据流/基础设施组件增删改）

## 当前架构理解

根据现行架构稿：

- 共有 2 个视图：`enterprise-landscape`、`application-integration`。
- 业务层：租户成员、镜像市场申请认证、SSO 进门户。
- 应用层：taskFE、taskGateway APISIX、taskAuth、taskAiProvider、Go 微服务群、runAll `:9999`。
- 技术层：同机 Docker 基础设施；运行时 conf 在 `conf/` 子仓，机密在 `conf-local/`（ADR-0054）；部署平面目标是 `daydaymoney-deploy`（ADR-0052）。
- 上次更新：**v125** ✅ current（2026-09-01 14:30）— ImageMarket 恢复厂商申请入口。

📋 架构版本历史（近端）：

- v125 (2026-09-01) ✅ current — ImageMarket 恢复厂商申请入口
- v124 (2026-08-31) 📦 — clone-run（PEM 入 conf-local）
- v123 (2026-08-31) 📦 — conf-local 机密收口

本次只收口 `conf.git` 跟踪面，不改运行时拓扑。**不写 target 架构文件。**

## 背景：为什么 conf 里会有非配置文件

`conf/` 不是「纯 YAML 文件夹」，而是独立 Git 子仓 `github.com/task2money/conf.git`（`.gitmodules`）。三条互不相关的决策叠在一起，把非配置资产带进了这个仓：

| # | 决策 | 时间 | 带进来的东西 |
|---|------|------|----------------|
| 1 | 全子仓 `.githooks` 入库分发（git-hooks-version-control v13） | 2026-08-06 | `.githooks/**`（placeholder pre-commit、commit-msg、bug-fix checker、random_test_runner、session_lock）；`deploy_settings` 还往每个子仓写 `.claude/settings.json`（本机 exclude，不入库） |
| 2 | conf-sync / 配置表征测试与 YAML 并置 | 2026-06 起 | 各 app `sync.sh`（应保留）、`scripts/ci/check_conf_sync.sh`、两份 `test_*.py` |
| 3 | 历史遗留 / 错放 | 2026-06～08 | `com.user.ramsync.plist`（写死 `/Users/task2app/gitClone/ramDisk/`）；`taskChromePlugin/generateKey.sh`；`infra/git-service-tencent-sh-1/docker-compose.yml`（区域配方被当成 conf SSOT） |

`value-stream.yaml`、`runAll.yaml`、`LICENSE` **不是本次要清的对象**：前两者是 YAML 配置；后者是独立 Git 仓的法律文件。

盘点（`conf.git` 已跟踪，2026-09-01）：**155** 个文件，其中 YAML/YML **116**；其余为 md/sh/py/plist/hooks。

## 🔍 Trace 日志分析

无 `data-traceId` / 运行时错误。本次是仓库内容分类问题，不查 Loki。

## 🕸️ Code Review Graph 分析

- **图状态**: `.code-review-graph/graph.db` 存在（114 nodes / 1012 edges，2026-09-01，branch main）。
- **MCP**: `CRG unavailable: no code-review-graph MCP namespace in this session`。
- **CLI**: `code-review-graph search` 参数与 runbook `--brief` 不兼容；爆炸半径改用 `git ls-files` + `rg`。
- **结论**: 触达面是 meta 分发器、CI 钩子门禁、gitService 区域部署脚本、两份 conf 内 Python 表征测试，无业务服务调用链。

## 成功标准

1. `conf.git` 已跟踪文件符合下文 **允许清单**；门禁 `db/scripts/ci/check_conf_tracked_allowlist.py` 绿。
2. `scripts/deploy_repo_random_precommit.sh` **不再**把模板钩子 / `.claude/settings.json` 写入 `conf/`。
3. CI `check_subrepo_random_precommit_hooks.py` 对 `CONFIG_REPOS=conf` 豁免全套 REQUIRED_HOOKS；若保留薄 pre-commit，只允许 allowlist 内那一支。
4. 迁出文件在新位置有调用方更新，旧路径无引用。
5. git-oauth `client_id` 实时校验在 **conf 子仓提交时仍会跑**（不得因删模板钩子而丢失 OPT-20260807-041a）。
6. 不新增 Python HTTP 接口；不改业务运行时行为。

## 允许清单（`conf.git` 已跟踪）

| 类别 | 模式 | 理由 |
|------|------|------|
| 运行时 / 产品 YAML | `**/*.{yaml,yml}` | 元规则 42 / 47 SSOT |
| 同步声明 | `**/sync.manifest.yaml` | 已含在 YAML；列此强调 |
| 同步入口 | `**/sync.sh` | 规则 29：各 app 目录内薄封装，只 `exec conf-sync.py` |
| Companion | `ai.md`、`**/*.ai.md` | Companion 元规则 |
| 仓身份 | `README.md`、`LICENSE`、`COMMERCIAL.md`、`.gitignore` | 独立 Git 仓必需 |
| 机密骨架示例 | `**/*.example` | 非机密、指引 overlay |
| **唯一允许的可执行钩子** | `.githooks/pre-commit`（手写，见下） | 配置质量门禁；禁止模板全家桶 |

**禁止**（已跟踪即失败）：`.py`、`.plist`、`docker-compose*.yml`、`generateKey.sh`、`.githooks/commit-msg`、`.githooks/lib/**`、`.githooks/install.sh`、`.githooks/HOOK_VERSION`、`.claude/**`、其它非上表扩展名。

运行时噪音（已 gitignore 或应补 ignore）：`logs/`、`.runall/`、`.pytest_cache/`。`.pytest_cache/` 补进 `conf/.gitignore`。

## 迁出对照

| 现状（conf） | 目标 | 动作 |
|--------------|------|------|
| `.githooks/commit-msg`、`check_bug_fix_commit_msg.sh`、`lib/*`、`install.sh`、`HOOK_VERSION` | 删除 | 分发器把 `conf` 列入 `SKIP_HOOK_REPOS` |
| `.claude/settings.json` | 不再分发到 conf | `deploy_settings` 跳过 `SKIP_HOOK_REPOS` |
| `.githooks/pre-commit`（placeholder + oauth live check） | **改写成 20 行内手写钩子**，只调 `db/scripts/ci/check_git_oauth_client_id_live.py` | 见「配置质量钩子」 |
| `com.user.ramsync.plist` | 删除 | 本机 launchd 配方，路径写死旧开发机；与 conf SSOT 无关。若仍要 ramsync，落 meta `scripts/ramsync/`（本迭代不复活） |
| `infra/git-service-tencent-sh-1/docker-compose.yml` | `gitService/docker-compose.tencent-sh-1.yml` | 配方归 gitService（与本机 `gitService/docker-compose.yml` 对称）；可调键仍只改 `conf/infra/git-service-tencent-sh-1/config.yaml`。更新 `deploy_tencent_sh_1_from_infra.sh` 的 `SSOT_COMPOSE` |
| `taskChromePlugin/generateKey.sh`（+ companion） | `taskChromePlugin/scripts/generateKey.sh` | 从 `conf-local/taskChromePlugin/key_pkcs8.pem` 导出公钥，不是 conf 值 |
| `auth/git-oauth/test_provider_website_isolation.py` | `db/scripts/ci/test_git_oauth_provider_website_isolation.py` | 表征测试归 CI |
| `infra/git-service/test_config_resource_keys.py` | `db/scripts/ci/test_git_service_config_resource_keys.py` | 同上；更新 `55_gitlab_sso_only_no_self_signup.md` 验收命令 |
| `scripts/ci/check_conf_sync.sh` | `db/scripts/ci/check_conf_sync.sh` | CI 脚本归 db/scripts；更新 `conf/README.md` |

`value-stream.yaml` **留在 conf**：它是价值流/测试编排的 YAML SSOT（`valueStream` 工具消费），不是 shell/钩子/配方。

## 配置质量钩子（conf 唯一非 YAML 可执行文件）

子仓提交发生在 `conf.git` 内部，meta `.githooks` **看不到** YAML diff。OPT-20260807-041a（git-oauth `client_id` 对 GitHub Apps API 实时校验）必须留在 conf 提交路径。

手写 `.githooks/pre-commit` 职责：

1. Session Hub 锁校验可省略（配置仓无随机单测；锁由 meta 提交覆盖）。
2. 暂存含 `auth/git-oauth/` 或 `auth/task-credential/` 的 YAML 时，调用 `$META_ROOT/db/scripts/ci/check_git_oauth_client_id_live.py --all`（与现逻辑相同：网络失败 SKIP）。
3. **禁止** 再 `source random_test_runner.sh`、禁止 commit-msg 模板。

`core.hooksPath=.githooks` 仍可指向这一支，便于子仓 commit 触发。CI 对 conf：检查「若存在 `.githooks/pre-commit` 则不得存在 commit-msg / random_test_runner」。

## 分发器与 CI 变更

`scripts/deploy_repo_random_precommit.sh`：

```bash
SKIP_HOOK_REPOS="conf"   # 配置仓：不部署模板钩子与 .claude/settings.json
```

`pick_template` 里现有 `conf|dockerInfra|...` 的 placeholder 分支删除 `conf`；`deploy_one` 对 SKIP 直接 return。

`db/scripts/ci/check_subrepo_random_precommit_hooks.py`：对 `SKIP_HOOK_REPOS` 走豁免分支（不要求 REQUIRED_HOOKS 三件套）。

新增 `db/scripts/ci/check_conf_tracked_allowlist.py` + `test_check_conf_tracked_allowlist.py`（表征：允许 YAML/sync.sh；拒绝 `.plist` / `docker-compose.yml` / `.py`）。挂到现有 repo-quality-gates 或 conf 相关 CI 入口。

## Domain Concept Inventory

配置仓纯度是平台控制面，不是业务限界上下文。无新实体/聚合。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 收紧 conf.git 跟踪面 | — | — | — | 仓库布局/门禁，无领域事实变更，无对应 MQ 事件 |

## 价值流影响

不改 `conf/value-stream.yaml` 中任何 stream/step。无新字段、无测试文件路径因业务功能而变（仅 CI 脚本路径）。**不新增 value stream。**

## 🏛️ 架构变更影响

- **不创建** v126 target 文件。
- 无 🟢/🟡/🔴 运行时组件。
- 文档：`conf/ai.md`、`conf/README.md`、元规则 47 增「跟踪允许清单」一句；ADR-0052 不改正文（配方本就应在 gitService）。

## 实施切片（批准后）

1. 允许清单门禁（红）→ 实现（绿）→ 再迁文件。
2. 迁 Python 测试与 `check_conf_sync.sh`；改引用。
3. 迁 tencent-sh-1 compose + 更新 deploy 脚本/测试。
4. 迁 `generateKey.sh`；删 plist。
5. SKIP_HOOK_REPOS + 瘦身 conf `.githooks`；停分发 `.claude`。
6. 更新 `conf/ai.md` / README / 规则 47 验收命令。

## 风险

| 风险 | 缓解 |
|------|------|
| conf 单独 commit 失去 oauth live check | 保留手写 pre-commit |
| CI 仍要求 conf 全套 hooks | 同步改 `check_subrepo_random_precommit_hooks.py` |
| tencent-sh-1 远端 compose 漂移 | deploy 脚本改路径后跑 `diff` 模式对照 |
| Agent 再次往 conf 丢钩子 | 允许清单门禁阻断；分发器 SKIP |
