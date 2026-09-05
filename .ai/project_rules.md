# Trae 提示词索引

## 基本信息

- 版本：5.139.0
- 创建日期：2026-01-27
- 最后修改：2026-09-02
- 维护者：Trae AI 团队

## 元规则导航（读我优先）


| 角色            | 路径                                                                             | 说明                                                                            |
| ------------- | ------------------------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| **Cursor 常驻** | `[.cursor/rules/ai-rules-loader.mdc](../.cursor/rules/ai-rules-loader.mdc)` | 按任务类型加载 `.ai`；**文件/目录级** `{path}.ai.md`、`ai.md` 伴读；**执行原则** 1～14 条为会话级硬约束                                           |
| **本索引（权威目录）** | 本文 `project_rules.md`                                                          | 全库规则 **金字塔总表**：各一级分类下的细则条目 + 维护流程                                             |
| **索引沿革归档**    | `[project_rules_CHANGELOG_ARCHIVE.md](./project_rules_CHANGELOG_ARCHIVE.md)` | **业务规则**沿革（2026-04 及更早）；正文「变更日志」仅保留必要摘要 |
| **逐条落地**      | 本文所在目录 `[.ai/](./)` 下 `00`～`11` 子目录                                      | 具体规范正文；修改某域代码时以对应 `00_*.md` 为入口继续下钻                                           |
| **Agent 技能包** | `[.claude/skills/](../.claude/skills)`                                      | 规划、意图框定、简化加固等可执行 `SKILL.md`（路径以本仓库为准）                                         |


**冲突时**：以本文件文末「规则冲突处理」为框架；**专项合规**（支付 KYC、Git 禁止 `--no-verify` 等）以 `.ai` 对应专章为准且优先于一般性 AI 工作流建议；`ai-rules-loader` 执行原则与本文一致时叠加适用。

## 核心概念

### 金字塔原理组织

本提示词索引采用金字塔原理进行组织，从一般到具体，层次分明：

1. **一级分类**：职能范畴（如技术实现、前端开发、测试质量等）
2. **二级分类**：具体领域（如技术实现中的代码规范、架构设计等）
3. **三级分类**：详细规则（具体的提示词规则和实践指南）

### 提示词管理目标

- 提供结构化的提示词索引，便于快速定位和使用
- 确保提示词的一致性和质量
- 支持高效的提示词维护和更新
- 指导用户输入的意图分类和处理

## 规则加载与使用

### 层级（目录 / 文件）

- **目录级**：修改某目录或其子树时，优先加载该目录下 `ai.md`，并查看同目录 `.ai/` 子目录。
- **文件级**：修改某文件时，加载与其同名的 `.ai.md`（如改 `index.js` 则读 `index.js.ai.md`）。
- **优先级**：`文件名.ai.md` > 目录 `ai.md` > 目录 `.ai/`（若均不存在则跳过）。

### 会话流程（与 `ai-rules-loader` 一致）

1. **意图识别** → **职能范畴（一级）** → **领域（二级）** → **匹配细则（三级）** → **按需打开 `.ai` 下对应 `00_*.md` 及子文件**。
2. **Companion 伴读**：读取或修改源文件时，若存在同名 `{path}.ai.md` 或目录 `ai.md`，须与源文件一并加载（详见 `ai-rules-loader.mdc` 执行原则第 11 条）。
3. **按需加载**：勿一次性读完全部 `.ai`；按当前任务域加载。
4. **文件命名**：`.ai` 内提示词文件使用 `{序号}_{主题}.md`，与本索引「一级分类」目录对应。

## 规则添加与维护流程

### 1. 规则分类（金字塔结构）

- **一级 / 二级 / 三级** 定义见上文 **「核心概念」**；`**00`～`11` 目录与职责的唯一对照表**见下文 **「提示词目录结构」**（含 `00_start` 起点类别），新增规则时先定位目录再落 `.ai` 专文，并视需在 **「总提示词索引」** 增加摘要条目。

### 2. 添加与维护（合并流程）

1. **定位**：按金字塔确定层级与 `.ai` 目标文件（对照「提示词目录结构」树）。
2. **落地**：写入 `.ai` 专文，遵循统一格式（描述 / 适用场景 / 优先级 / 细则）。
3. **同步索引**：若规则对用户可见度较高，在本文件「总提示词索引」补充摘要条目。
4. **一致性**：全局变更时一并更新所有触及的 `.ai` 专文与本索引，避免目录树、索引与仓库脱节。

### 3. 避免的错误

- 不把**仅存在于变更日志**的约束当作唯一来源：新增或变更规则须写入正文「总提示词索引」对应小节或 `.ai` 专文；变更日志只记录**索引文件本身的版本沿革摘要**。
- 不在本索引中堆砌与目录 `.ai` 重复的长篇细则（索引以定位与摘要为主）。
- 不混放不同职能范畴的规则；不创建空目录；遵守 `{序号}_{主题}.md` 命名。

## 规则文件结构化要求

### 1. 统一文件结构

所有规则文件（`ai.md` 和 `.ai.md`）必须遵循以下金字塔结构：

```markdown
# 规则文件

## 基本信息
- 版本：1.0.0
- 创建日期：YYYY-MM-DD
- 最后修改：YYYY-MM-DD
- 维护者：维护者名称/团队

## 规则分类

### 核心规则（一级分类）
> 影响代码质量和安全性的关键规则，必须严格遵守

#### 规则领域（二级分类）
##### 规则名称（三级分类）
- 描述：规则详细说明
- 适用场景：明确规则适用的文件类型或场景
- 优先级：高

### 最佳实践（一级分类）
> 提升开发效率和代码可维护性的建议

#### 规则领域（二级分类）
##### 规则名称（三级分类）
- 描述：规则详细说明
- 适用场景：明确规则适用的文件类型或场景
- 优先级：中

### 风格指南（一级分类）
> 统一代码风格和格式的规范

#### 规则领域（二级分类）
##### 规则名称（三级分类）
- 描述：规则详细说明
- 适用场景：明确规则适用的文件类型或场景
- 优先级：低

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- YYYY-MM-DD：版本 X.X.X - 本文件沿革摘要（细则须在上文「规则分类」或外链 `.ai` 维护）
```

## 提示词目录结构（金字塔结构）

> **维护时的目录映射以本节树为准**，避免与过时列表分叉。

```
.ai/ (一级分类：职能范畴)
├── 00_start/                  # 起点提示词
│   ├── 00_start.md
│   └── 01_session_continuation.md  # 会话中断与「继续」请求
├── 01_project_constraints/    # 项目规范与约束提示词
├── 02_documentation/          # 文档与流程管理提示词
├── 03_technical_implementation/ # 技术实现要求提示词
├── 04_frontend_development/   # 前端开发规范提示词
├── 05_testing_quality/        # 测试与质量保障提示词
├── 06_execution_monitoring/   # 执行与监控提示词
├── 07_togaf_project_management/ # TOGAF项目管理提示词
│   ├── 01_adm_phases/         # 二级分类：TOGAF ADM阶段提示词
│   ├── 02_core_concepts/      # 二级分类：TOGAF核心概念提示词
│   ├── 03_ai_implementation/  # 二级分类：AI在TOGAF中的实现提示词
│   └── 04_project_application/ # 二级分类：TOGAF项目应用提示词
├── 08_prompt_management/      # 提示词管理指引
├── 09_failure_experience/      # 失败案例经验库
│   ├── 01_compilation_errors/  # 二级分类：编译错误
│   │   └── 01_export_syntax_error.md # 三级分类：export 语法错误
│   ├── 02_runtime_errors/      # 二级分类：运行时错误（可扩展）
│   ├── 03_performance_issues/  # 二级分类：性能问题（可扩展）
│   └── 04_security_issues/     # 二级分类：安全问题（可扩展）
├── 10_software_design_philosophy/ # 软件设计哲学提示词
│   └── 01_core_principles/     # 二级分类：核心原则
└── 11_ai_development/          # AI开发规范提示词
    └── 00_ai_development.md    # AI开发规范
```

## 总提示词索引（金字塔结构）

### 一级分类：起点

- 详细内容请参考：[起点提示词](./00_start/00_start.md)
- 会话接续（网络中断、`继续`）：[会话中断与「继续」请求](./00_start/01_session_continuation.md)；Cursor 元规则 `.cursor/rules/ai-rules-loader.mdc` 中已同步援引

### 一级分类：项目规范与约束

- 详细内容请参考：[项目规范与约束提示词](./01_project_constraints/00_project_constraints.md)

#### 测试邮件地址规则

- **描述**：测试邮件只能使用 [contact@daydaymoney.com](mailto:contact@daydaymoney.com)，不允许使用其他邮箱地址，尤其是 example.com 后缀的邮件地址
- **适用场景**：当编写或修改测试代码、邮件模板或任何涉及测试邮件发送的场景
- **优先级**：高

#### Git 提交规范

- **描述**：禁止使用 `--no-verify` 标志提交代码，确保所有测试通过后再进行提交。即使测试失败与当前更改没有关系，也禁止使用 `--no-verify` 标志。当出现测试错误时，应该去修复错误，实在无法修复的错误，应该提示用户处理，然后中断执行。此规则为禁止忽略的核心规则，必须严格遵守。**随机单测遗留债**：每次提交须跑随机单元测试；遗留失败适用下方「提交时随机单元测试与遗留债 10% 修复」。
- **适用场景**：所有 Git 提交操作
- **优先级**：高
- **规则类型**：禁止忽略

#### 密钥禁止硬编码在源码中

- **描述**：**禁止**把 API key、口令、Token、私钥、client secret 写成业务源码字面量。必须存放在 gitignored 的 `conf-local/`、环境变量或 KMS/CI secret store。禁止写入已跟踪 `conf/` YAML。禁止提交 `.env`（example 除外）。测试凭据只读环境变量，禁止 `env || '真实口令'`。高置信泄露（PEM、AKIA、GitHub PAT、Stripe live key）即使在测试中也阻断。例外须 `Secret-Hardcode-OK:`。已进入已推送历史则走「禁止重写 Git 历史」条：废弃并重新生成。
- **适用场景**：新增认证/支付/云 SDK/OAuth/Webhook；编写 Playwright 或本地调试脚本；pre-commit 报 hardcoded secrets
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[密钥禁止硬编码在源码中](./01_project_constraints/62_no_hardcoded_secrets.md)；ADR-0046；Cursor：`.cursor/rules/no-hardcoded-secrets.mdc`（alwaysApply）；门禁：`db/scripts/ci/check_no_hardcoded_secrets.py`；自测：`db/scripts/ci/test_check_no_hardcoded_secrets.py`；约束索引第 57 条

#### 机密参数仅允许放在 conf-local

- **描述**：**`conf/` 只放非机密参数。** 密钥、secretId、secretKey、Token、client secret、私钥 PEM 只放 gitignored 的 `conf-local/`（与 `conf/` 同相对路径）。加载器深合并。禁止再维护 HOST_SECRETS 登记册。已跟踪 `conf/` YAML 不得含非空机密键。
- **适用场景**：新增云/支付/OAuth/SMS/COS/TLS/OIDC 密钥；改 `confload` / `conf_loader` / `conf-sync`；clone-run 放置 `secrets/conf-local/`
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[机密参数仅允许放在 conf-local](./01_project_constraints/63_conf_local_secrets_only.md)；ADR-0054；Cursor：`.cursor/rules/conf-local-secrets.mdc`（alwaysApply）；门禁：`db/scripts/ci/check_conf_local_secrets.py`；自测：`db/scripts/ci/test_check_conf_local_secrets.py`；约束索引第 58 条

#### 所有进程加载 conf 必须叠加 conf-local

- **描述**：**凡运行时读取 `conf/**/*.yaml` 必须深合并 `conf-local/<同相对路径>`。** 禁止 `os.ReadFile` / `yaml.safe_load` / `readFileSync` 只读 tracked `conf/`。禁止加载 `config.local.yaml`。走 Go `confload` 或 Python `overlay_conf_file`。自定义合并须 `Conf-Local-Overlay-OK:`。补第 58 条：58 管落点与契约，本条强制所有进程实际走加载器。
- **适用场景**：新增/修改 `loadConfig`、`config.go`、进程启动读 YAML；改 `confload` / `conf_loader`；评审发现直读 `conf/` YAML
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[所有进程加载 conf 必须叠加 conf-local](./01_project_constraints/64_conf_local_overlay_all_processes.md)；ADR-0054；Cursor：`.cursor/rules/conf-local-overlay-all-processes.mdc`（alwaysApply）；门禁：`db/scripts/ci/check_conf_local_overlay.py`；自测：`db/scripts/ci/test_check_conf_local_overlay.py`；约束索引第 59 条

#### 禁止重写 Git 历史清理私密信息；泄露凭据须废弃并重新生成

- **描述**：**禁止**使用 `git filter-repo` / `git filter-branch` / `git replace` 或 force-push 覆盖共享 ref 来「清除」已推送历史中的私密信息（密钥、密码、Token、Cookie、证书、连接串）——已分发副本（协作者 clone、CI 缓存、gitService 服务端对象、备份）无法被重写删除，且泄露凭据在重写后仍有效。私密信息一旦进入已推送历史视为**已泄露**：立即废弃并在相关系统**重新生成、部署替换**，再补防再犯（gitignore / secret 扫描）。允许例外：从未推送、从未离开本机的提交整理（rebase/squash/amend）、`filter-repo --analyze` 只读分析、删除整个分支/仓库。Agent 收到「清历史 / 删提交中的密钥」类请求必须拒绝并说明正确处置。pre-commit（v1.4.0+）检测 `.git/filter-repo/`、`refs/original/*`、`refs/replace/*` 遗留标记命中即阻断提交
- **适用场景**：任何「清理 git 历史」「删除提交中的密钥」「filter-repo / filter-branch」请求；密钥/凭据泄露事件响应与安全审计；pre-commit 报出历史重写标记
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[禁止重写 Git 历史清理私密信息](./01_project_constraints/61_no_git_history_rewrite.md)；Cursor：`.cursor/rules/no-git-history-rewrite.mdc`（alwaysApply）；门禁：`.githooks/pre-commit`（v1.4.0）；复查：`scripts/lib/check_git_history_rewrite.sh --scan-all`；约束索引第 56 条

#### 提交时随机单元测试与遗留债 10% 修复

- **描述**：每次 `git commit` 须经 pre-commit 执行 monorepo 随机单元测试；发现遗留失败 N 个文件时，同次至少修复 `max(1, ceil(N×0.10))` 个并复跑通过，余债写入 `.learnings/UNIT_TEST_DEBT.md`；暂存相关单测须 100% 通过；禁止 `--no-verify` / 删测例凑配额
- **适用场景**：所有提交的 pre-commit；Agent 代提与撞上随机遗留失败时
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[提交时随机单元测试与遗留债 10% 修复](./01_project_constraints/28_commit_random_unit_test_debt_fix.md)；Cursor：`.cursor/rules/commit-random-unit-test-debt-fix.mdc`（alwaysApply）；门禁：`db/scripts/ci/run_commit_random_unit_tests.py`；约束索引第 29 条

#### 已合入 feat 分支自动清理

- **描述**：`feat/*` 合入并推送 `main` 后（含内容已在 main、仅 SHA 不同的等价落地），必须删除本地与各远端同名分支，并拆除 `{repo}-wt/` 与额外元仓 worktree；Agent 须自动执行 `cleanup_stale_worktrees.py --apply --shipped-branch feat/<name>` 与 `delete_merged_feat_branches.py --apply`，禁止仅建议「择机删除」
- **适用场景**：merge/PR 合入 main、goal-mode「提交到 main 并推送」、ship 交付收尾、旁路 commit 把同一功能写入 main、周维护 cron
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[已合入 feat 分支自动清理](./01_project_constraints/21_merged_feat_branch_cleanup.md)

#### 服务监听禁止仅绑 127.0.0.1

- **描述**：所有 HTTP/TCP 服务 listen/bind 须为 **`0.0.0.0`**（或等价全网卡）；**禁止**仅绑 `127.0.0.1`/`localhost`。Docker APISIX（`host.docker.internal`）与边缘 nginx 依赖宿主机非 loopback 可达；不得以「安全加固」改回 loopback-only。同机客户端 URL 仍可用 `http://127.0.0.1:PORT`
- **适用场景**：新建/迁移/hardening 任意后端与 sidecar、修改 `conf/**/config.yaml` 或 provider `host`、审查 listen 相关 PR
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[服务监听地址禁止仅绑 127.0.0.1](./01_project_constraints/22_service_listen_host_not_localhost_only.md)；根摘要：[`.ai.md`](../.ai.md)

#### 服务仅读本目录配置；跨服务配置须 sync

- **描述**：业务进程运行时**禁止**直读其它服务的 `conf/<area>/<other-app>/`；需要他方配置键时须经本目录 `sync.manifest.yaml` / conf-sync 生成 GENERATED 片段，再用 `confload.ReadAppFragment`（或等价仅打开本目录路径的 API）读取。`conf/base.yaml` 寻址模板为唯一约定例外
- **适用场景**：新增/修改 `loadConfig`、`confload`、`conf/**/sync.manifest.yaml`、跨服务配置依赖、评审跨 `conf_app` 的 YAML 路径
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[服务仅读本目录配置；跨服务配置须 sync](./01_project_constraints/29_service_own_conf_directory_only_via_sync.md)；Cursor：`.cursor/rules/service-own-conf-directory-sync.mdc`（alwaysApply）；约束索引第 30 条

#### 人工可改配置项统一落在 conf/<area>/<app>/

- **描述**：端口、内存上限、并发、公网 URL、数据目录、功能开关等需人工调整的配置，**SSOT 必须**在对应服务的 `conf/<area>/<app>/config.yaml`（本机覆盖 `conf-local/`）。`docker-compose.yml` / `run.sh` 等只消费 conf（读 conf → 导出 env），禁止把长期默认可调值只写在 compose/脚本里。与「仅读本目录 conf」互补：本条管编辑落点，第 30 条管运行时读边界
- **适用场景**：新增/搬迁可调参数、修改 `docker-compose*.yml` 默认值、用户询问「配置改哪里」、infra 资源配额（如 GitLab `memLimit`）
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[人工可改配置项统一落在 conf](./01_project_constraints/47_conf_app_human_editable_config_ssot.md)；Cursor：`.cursor/rules/conf-app-config-ssot.mdc`（alwaysApply）；约束索引第 42 条

#### NFR 路径分片键与可伸缩性审视

- **描述**：执行 `/5-nfr` 时须枚举增量路径；无分片 ID → 评估可伸缩性需求；有分片 ID → 判定是否合适分片键。NFR 文档须含「路径分片键审视」表，未通过禁止进入 `/6-ddd`
- **适用场景**：`/5-nfr`、编写/修改 `*-nfr-clarification.md`、新 API/路由/Kafka 键设计评审
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[NFR 路径分片键与可伸缩性审视](./01_project_constraints/48_nfr_path_shard_id_scalability.md)；Cursor：`.cursor/rules/nfr-path-shard-id-scalability.mdc`（alwaysApply）；技能：`.claude/skills/5-nfr/`；约束索引第 43 条

#### NFR 幂等性审视

- **描述**：执行 `/5-nfr` 时须枚举副作用路径；写明重复触发源、业务重复边界、幂等键与重放语义；键须与边界同粒度。禁止默认用 `company_id`/`user_id` 作事件消费键。NFR 文档须含「幂等性审视」表，未通过禁止进入 `/6-ddd`
- **适用场景**：`/5-nfr`、编写/修改 `*-nfr-clarification.md`、写 API / Kafka 消费 / Webhook / timer 设计评审
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[NFR 幂等性审视](./01_project_constraints/53_nfr_idempotency.md)；Cursor：`.cursor/rules/nfr-idempotency.mdc`（alwaysApply）；技能：`.claude/skills/5-nfr/`；约束索引第 48 条；门禁 `db/scripts/ci/check_nfr_idempotency_table.py`

#### 事件消费者幂等消费

- **描述**：Kafka / 领域事件消费必须走 `taskEvents` 共享 `IdempotentDispatchService`（at-least-once + 幂等 = effectively-once）。禁止业务服务自建无幂等 consume loop；幂等键须与业务重复边界同粒度；重复投递须 Ack 空操作并打 `idempotency skip`
- **适用场景**：新增/修改 `taskEvents` intent、handler、消费 runner；业务服务出现 Kafka Reader；重复投递 / SSE 被吞 / 副作用执行两次
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[事件消费者幂等消费](./01_project_constraints/54_event_consumer_idempotency.md)；ADR-0015；Cursor：`.cursor/rules/event-consumer-idempotency.mdc`（alwaysApply）；约束索引第 49 条；门禁 `db/scripts/ci/check_event_consumer_idempotency.py`

#### GitLab 禁止自行注册、仅允许 SSO

- **描述**：平台 GitLab（含全部区域实例）禁止用户经 `/users/sign_up` 自行注册；Web 登录仅允许 taskAuth OIDC SSO；`signupEnabled` / `passwordAuthWeb` / `passwordAuthGit` 必须为 `false`。禁止以排障为由打开注册或账密
- **适用场景**：改 `conf/infra/git-service*`、`gitService` compose/run.sh/OmniAuth、新增 GitLab 区域、GitLab「登不上」排障
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[GitLab 禁止自行注册、仅允许 SSO](./01_project_constraints/55_gitlab_sso_only_no_self_signup.md)；ADR-0016；Cursor：`.cursor/rules/gitlab-sso-only-no-self-signup.mdc`（alwaysApply）；约束索引第 50 条；门禁 `db/scripts/ci/check_gitlab_sso_only_no_self_signup.py`

#### 禁止链接点击拦截

- **描述**：导航控件须使用真实 `<a href>`（或整页 `location.href`）；禁止 `@click.prevent` + `router.push` 冒充链接；鉴权/租户失败禁止静默 `replace` 回用户来源页（须弹窗确认）
- **适用场景**：Navbar/侧栏/个人资料等跨页导航、租户页鉴权失败回退、任何「打开某个地址」的 UI
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[禁止链接点击拦截](./01_project_constraints/49_no_link_click_interception.md)；Cursor：`.cursor/rules/no-link-click-interception.mdc`（alwaysApply）；约束索引第 44 条

#### 前端按钮点击须有防重放设计

- **描述**：副作用按钮必须同步门闩 + pending 态 + 短窗 debounce；写请求须带点击意图级 `Idempotency-Key`（同意图重试回传同一键）。禁止只靠下一帧 `disabled` 或单独 debounce。前端锁不替代服务端幂等
- **适用场景**：所有前端写按钮（建单/支付/启停机器/提交表单等）、新增 Vue/React 点击处理
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[前端按钮点击须有防重放设计](./01_project_constraints/57_frontend_button_anti_replay.md)；ADR-0020；Cursor：`.cursor/rules/frontend-button-anti-replay.mdc`（alwaysApply）；参考 `taskFE/app/src/utils/clickGuard.js`；约束索引第 52 条；门禁 `db/scripts/ci/check_frontend_button_anti_replay.py`

#### 逻辑回退须经审批；禁止保留无用旧逻辑

- **描述**：替换/重构后默认删除旧函数、旧分支、注释掉的旧实现。保留或恢复旧逻辑必须在本会话获得人类批准；`/goal` 零交互不覆盖。未批准按删除执行。批准后代码注释与 commit message 须含 `Logic-Rollback-OK:` 及移除日期
- **适用场景**：所有业务源码修改、重构、git revert、双路径/兼容层、注释掉旧代码、为让测试变绿而撤回目标语义
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[逻辑回退须经审批](./01_project_constraints/58_logic_rollback_requires_approval.md)；ADR-0021；Cursor：`.cursor/rules/logic-rollback-requires-approval.mdc`（alwaysApply）；约束索引第 53 条；门禁 `db/scripts/ci/check_logic_rollback_approval.py`

#### 前端无必要禁止 Teleport

- **描述**：Vue `<Teleport>` / React `createPortal` 默认禁止；UI 在哪棵子树展示就在哪棵子树挂载。仅当祖先 overflow/transform 导致浮层被裁剪或错位且无法就地解决时才允许，并须标注 `Teleport-OK`。禁止把功能卡 Teleport 到远亲 `data-testid` 槽
- **适用场景**：所有前端 Vue/React 组件；新增浮层/下拉/模态/跨区域搬 DOM
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[前端无必要禁止 Teleport](./01_project_constraints/50_no_unnecessary_vue_teleport.md)；Cursor：`.cursor/rules/no-unnecessary-vue-teleport.mdc`；约束索引第 45 条

#### 业务服务禁止进程内轮询 / 循环

- **描述**：任意业务 HTTP/RPC 服务不得用 `time.NewTicker` / sleep 循环自唤醒做周期工作。周期工作须由专门的 `taskEvents` timer worker（独立 runAll 进程）HTTP 调用一次性 API；即时工作须由 HTTP、Kafka、Webhook、显式 CLI 等外界触发。前端禁止无用户操作、无 SSE 的后台 API 轮询
- **适用场景**：新建/修改服务启动路径、对账/过期/回收/扫描、runAll 新进程、前端状态刷新
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[业务服务禁止进程内轮询 / 循环](./01_project_constraints/51_no_service_internal_poll_loop.md)；ADR-0011；Cursor：`.cursor/rules/no-service-internal-poll-loop.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_no_service_internal_poll_loop.py`；约束索引第 46 条

#### runAll 健康检查端口必须等于进程监听 SSOT

- **描述**：runAll 探活地址必须等于该进程实际 listen 的 SSOT。禁止从相邻服务复制 `health_check.url` 后只改 `name`。task-events intent 端口只在 `conf/events/domain-events/<event>/config.yaml` 定义；`intent_registry.go`、`conf/runAll.yaml`、Prometheus file_sd 必须同一数字。新 `groupId` 必须有 runAll 条目（存量缺口只减不增）
- **适用场景**：新增/修改 runAll 服务、task-events intent、domain-events 端口、health_check URL、Prometheus runall-health-targets
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[runAll 健康检查端口必须等于进程监听 SSOT](./01_project_constraints/52_runall_health_port_ssot.md)；ADR-0012；Cursor：`.cursor/rules/runall-health-port-ssot.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_task_events_runall_health_ports.py`；约束索引第 47 条

#### 迁移/重构——测试先行

- **描述**：凡识别到用户指令包含迁移或重构意图（关键词：迁移/migrate、重构/refactor、重写/rewrite、搬迁/relocate、替换/replace、升级/upgrade、抽取/extract、合并/merge、拆分/split；或语义模式如「把 X 移到 Y」「用 Y 重新实现 X」等），Agent 必须在执行代码改动**之前**先建立行为基线——优先扩展现有测试，若无则写 characterization test（基线测试），确保改动前后外部行为一致。迁移/重构完成后须用同一测试集验证通过
- **适用场景**：所有满足触发条件的用户指令；不确定时默认触发；排除修复单 bug、新增独立功能、纯格式化、单文件 < 20 行有效变更且不改接口签名
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[迁移/重构——测试先行](./01_project_constraints/30_migration_refactoring_test_first.md)；Cursor：`.cursor/rules/migration-refactoring-test-first.mdc`（alwaysApply）；约束索引第 31 条

#### 子仓库优先提交

- **描述**：本仓库为 meta-repo，包含 37 个 git submodule。当一次变更同时涉及子仓库（submodule 目录内）和主仓库（meta-repo 层级）时，**必须先提交并推送每个有变更的子仓库**，再提交并推送主仓库。禁止在子仓库未推送的情况下提交/推送主仓库，避免主仓库记录的 submodule commit 指针指向远端不存在的 commit。Agent 在提交前须通过 `git submodule status` 或 `git diff --submodule` 自动识别有变更的子仓库，确保优先处理
- **适用场景**：所有同时触及 submodule 目录与 meta-repo 层级的变更；Agent 自动提交、/goal 交付、/ship 收尾等自动化流程
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[子仓库优先提交规则](./01_project_constraints/32_submodule_commit_order.md)；Cursor：`.cursor/rules/submodule-commit-order.mdc`（alwaysApply）；约束索引第 32 条

#### 数据库 ID 字段雪花算法生成（元规则）

- **描述**：所有**新建数据库表**的主键 `id` 字段**必须**使用 Snowflake（雪花算法）生成，禁止使用数据库自增主键（`AUTO_INCREMENT`/`SERIAL`/`BIGSERIAL`）或 UUID。统一算法参数：epoch=2020-01-01（`1577836800000`），41 位时间戳 + 10 位机器 ID + 12 位序列号。Python 调用 `core.utils.snowflake.generate_snowflake_id()`（`SnowflakeIDField` 自动处理），Go 复用 `shareLib/snowflake`（`snowflake.GenerateID()` / `snowflake.GenerateIDString()`）。机器 ID 通过 `MACHINE_ID` 环境变量注入（0–1023）。与 [ID 字段字符串传输规范](../.ai/03_technical_implementation/11_id_field_string_transit.md) 联合生效（Snowflake 管生成，string 管传输）。存量表迁移时应当切换并触发测试先行规则
- **适用场景**：所有新建数据库表、新建后端服务、存量表迁移/重构；monorepo 内全部服务（Django / Go / 其他）
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略
- **详细内容**：[数据库 ID 字段雪花算法生成规范](./01_project_constraints/35_snowflake_id_generation.md)；约束索引第 33 条

#### 应用启动禁止使用环境变量 Proxy

- **描述**：业务/基础设施进程及其出站 HTTP 客户端**不得**继承或依赖 `HTTP(S)_PROXY` / `ALL_PROXY`；此类环境变量**仅**供开发者本机工具做网络加速。`conf/runAll.yaml` 的 `use_proxy` 须为 `false`；启动脚本应 unset；生产依赖 Client 应显式直连
- **适用场景**：新增/改造服务启动脚本、runAll 代理开关、出站 HTTP Client、排障 `proxyconnect` / 死本地 SOCKS
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[应用启动禁止使用环境变量 Proxy](./01_project_constraints/23_app_startup_no_env_proxy.md)；根摘要：[`.ai.md`](../.ai.md)

#### 请求报错展示须带 data-traceId

- **描述**：因网络/HTTP/RPC 请求失败而在前端展示的错误 UI，其错误展示 DOM 节点**必须**设置属性 **`data-traceId`**，值为该次失败请求的 traceId（优先响应头 `X-Trace-Id`）；确无时省略、禁止伪造；纯前端校验不得挂载。Agent 从快照/用户指令**读取**标注时须大小写不敏感（`data-traceid`、`traceid`/`TraceId` 等）；**写入**仍固定字面 `data-traceId`。**排障硬门禁**：有 `data-traceId` 时须**优先**查 Loki/Grafana 并按时间线重建整条错误路径，禁止未查日志就凭文案猜根因或直接改代码（短规范：`.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`）
- **适用场景**：新增/修改错误 toast、inline 错误、错误页/弹层、统一错误处理与请求封装；Playwright 断言与 Agent 排障取证
- **优先级**：高
- **规则类型**：禁止忽略（改动报错展示时；Agent 排障遇 `data-traceId` 时）
- **详细内容**：[请求报错展示须带 data-traceId](./01_project_constraints/24_frontend_error_data_trace_id.md)；Cursor：`.cursor/rules/frontend-error-data-trace-id.mdc`

#### 会话结束优化建议 Todo

- **描述**：会话收尾时须将可执行优化建议编号写入 [`.learnings/OPTIMIZATION_TODOS.md`](../.learnings/OPTIMIZATION_TODOS.md)（`OPT-YYYYMMDD-NNN`，初始 `pending`）；某条落地验证后标记 `completed`（填 `Completed` / `Completion-Note`）。禁止只在聊天中写建议不落盘；与 `LEARNINGS.md` 分工为「待办 vs 经验」
- **适用场景**：任务最终答复、`/goal` Final Overview、ship 收尾、用户明确结束会话且有可沉淀建议
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[会话结束优化建议 Todo](./01_project_constraints/25_session_end_optimization_todo.md)；Cursor：`.cursor/rules/session-optimization-todo.mdc`

#### HTML head 须标明提供页面的服务

- **描述**：入口 HTML 的 `<head>` **必须**声明 `<meta name="trae-service" content="<serviceId>" />`，`content` 优先与 monorepo 顶层目录名一致，便于多域名 / 网关转发下确认页面归属
- **适用场景**：新增/修改 Vite `index.html`、Django SPA 模板、onlineService 静态页、Chrome 扩展入口 HTML；新建托管 UI 的服务
- **优先级**：高
- **规则类型**：禁止忽略（入口 HTML）
- **详细内容**：[HTML head 须标明提供页面的服务](./01_project_constraints/26_frontend_head_trae_service.md)；Cursor：`.cursor/rules/frontend-head-trae-service.mdc`；门禁：`db/scripts/ci/check_frontend_head_trae_service.py`

#### 行数门禁触发后须自动削减目标文件

- **描述**：单源文件默认 **> 500 行**、或 pre-commit/CI 行数门禁失败并点名路径时，Agent **必须立即自动**将全部目标文件拆分削减到 **≤ 阈值** 并复跑验收；禁止只写「后续拆分」、禁止询问是否拆分、禁止继续向超标文件堆功能。CI 遗留净增长容差不免除削文件义务
- **适用场景**：任意源码超标；`check_frontend_component_line_limit` 等门禁失败；用户/评审点名超标文件
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[行数门禁触发后须自动削减目标文件](./01_project_constraints/27_source_file_line_limit_auto_reduce.md)；Cursor：`.cursor/rules/source-file-line-limit-auto-reduce.mdc`（alwaysApply）；约束索引第 20 条

#### ORM 优先使用（元规则）

- **描述**：所有数据库操作**必须优先使用 ORM 库**（Django ORM / GORM），禁止在业务代码中直接编写原始 SQL 语句。CRUD 全部走 ORM 查询构造器；复杂查询优先用 ORM 高级特性（F 表达式、Subquery、CTE、Scopes 等）。原始 SQL 仅限例外场景（Migration DDL、聚合分析、批量操作、数据库专有特性、报表导出、dataMigrate），且**必须**封装在仓储层、使用参数化查询、返回类型安全对象、附带原由注释与测试覆盖。存量裸 SQL 遇功能修改时须同步改为 ORM 或符合封装要求
- **适用场景**：所有数据库操作（增删改查、聚合、关联查询）；Django / Go 后端服务；代码审查中的 SQL 相关检查；存量裸 SQL 整改
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略
- **详细内容**：[ORM 优先使用规范](./01_project_constraints/37_orm_priority_over_raw_sql.md)；约束索引第 35 条

#### 文件修改后格式验证（元规则）

- **描述**：修改任何文件后，必须使用对应格式验证工具确认文件格式有效。若缺乏验证工具，必须先安装再验证。覆盖 Go（gofmt + go vet）、Python（py_compile + ruff）、JavaScript/TypeScript/Vue（eslint）、YAML（yaml.safe_load + yamllint）、JSON（jq）、Shell（bash -n + shellcheck）、Dockerfile（hadolint）、Markdown（markdownlint）、SQL（sqlfluff）、HTML/CSS（prettier）等全部文件类型。网络不可用时使用语言内建机制（py_compile、bash -n、node --check 等）作为 fallback，并在 commit message 中注明
- **适用场景**：所有文件的新建、修改、批量修改；Agent 自动修改与人工修改均适用
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略
- **详细内容**：[文件修改后格式验证](./01_project_constraints/38_file_format_validation.md)；约束索引第 36 条；Cursor：`.cursor/rules/file-format-validation.mdc`（alwaysApply）

#### 网络拓扑感知服务配置（元规则）

- **描述**：所有服务配置必须面向未来多节点拆分设计。基础设施服务（Redis、Kafka、MySQL）地址**禁止**硬编码 `127.0.0.1` / `localhost`，必须使用 `${INFRA_HOST}` 或 `${subdomains.xxx}` 模板变量；服务按网络可达性分为三层（内网基础设施层 / 外部服务平台层 / 业务网关层）；`conf/base.yaml` 须为每类基础设施注册子域条目；同节点 loopback 调用须注释标注「拆分时改 xxx」。触及即改（不强制一次性全部迁移）
- **适用场景**：新增/修改基础设施配置（`conf/infra/**`）；新增服务或中间件连接；修改 `conf/base.yaml`；架构评审中的网络拓扑讨论
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略
- **详细内容**：[网络拓扑感知服务配置](./01_project_constraints/39_network_topology_aware_config.md)；约束索引第 37 条；Cursor：`.cursor/rules/network-topology-aware-config.mdc`（alwaysApply）

#### ADR 架构决策记录（元规则）

- **描述**：凡涉及架构层面的决策（新技术选型、架构模式引入、基础设施变更、跨服务协议约定、全局数据库 Schema 约定、安全/合规架构），**必须**在 `docs/adr/` 创建 Architecture Decision Record (ADR)，使用标准模板（Context → Decision → Alternatives → Consequences），遵循 4 位序号命名（`NNNN-title.md`）和生命周期（proposed → accepted → deprecated → superseded）。存量重要决策（Snowflake ID、Kafka 迁移、单库单表所有权、ORM 优先、外键禁止）在触及相关代码时回溯补写 ADR。PR 涉及新增依赖/新服务/全局 Schema 变更/基础设施替换时，Reviewer 须检查是否附带 ADR
- **适用场景**：技术栈选型、架构模式变更、基础设施迁移、跨服务协议制定、数据库全局约束、安全合规架构决策、已被既有规范充分定义以外的设计决策
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略（满足触发条件时）
- **详细内容**：[docs/adr/README.md](../docs/adr/README.md)；模板：[docs/adr/template.md](../docs/adr/template.md)；Cursor：`.cursor/rules/adr-architecture-decisions.mdc`（alwaysApply）

#### 租户逻辑资源组 RBAC（元规则）

- **描述**：租户控制台与租户域 API 的授权须遵循 v72 **页面组 ⊃ UI 组件区域**模型（ADR-0003）。新页面/区块/API 必须种子化 `auth_resource_group`/`auth_resource_member`，BE `RequireRegion`、FE `hasRegion`/`hasPage`，PDP 注入 `region:*`/`page:*`；禁止仅前端隐藏、禁止「页面=资源组」、禁止混用业务 `tenant_resource_group_assignment`
- **适用场景**：租户侧栏/页面、页内分区、租户 HTTP API 鉴权、访问管理、成员/组角色、People* / settings / billing 等租户控制台改动
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略（触发条件满足时）
- **详细内容**：[租户逻辑资源组 RBAC](./01_project_constraints/45_tenant_logical_rbac_resource_groups.md)；约束索引第 40 条；Cursor：`.cursor/rules/tenant-logical-rbac-resource-groups.mdc`（alwaysApply）

#### trae-agent onlineServiceJS Docker 推送（元规则）

- **描述**：修改并提交 `trae-agent` 中影响 online 镜像的内容后，必须在 `trae-agent/onlineServiceJS` 执行 `DOCKER_PUSH=1 ./buildDocker.sh`。Stop/SessionEnd 自动登记 pending；SessionEnd 后台兜底；Agent 提交后须同步执行
- **适用场景**：trae-agent / onlineServiceJS 源码、Dockerfile、依赖变更并 commit / ship / 交付
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略
- **详细内容**：[trae-agent onlineServiceJS Docker 推送](./01_project_constraints/46_trae_agent_online_service_docker_push.md)；约束索引第 41 条；Cursor：`.cursor/rules/trae-agent-docker-push.mdc`（alwaysApply）

#### 超级管理员创建规则

- **描述**：创建超级管理员必须使用 `Saas_project/scripts/init/01_02_create_admin_ruandao.py` 脚本
- **适用场景**：当需要创建超级管理员用户时
- **优先级**：高

#### 文件修改范围限制

- **描述**：所有文件修改操作必须限制在项目根目录内，不得修改项目根目录以外的文件。
- **适用场景**：所有文件修改、创建、删除操作
- **优先级**：高
- **规则类型**：禁止忽略

#### 异域部署：前端与 API 分域名

- **描述**：用户访问的 Web 前端与 HTTP(S) API 可部署在**不同域名**；设计鉴权、会话、CORS、Cookie、`SameSite`、OAuth/支付回调、上传与 WebSocket、CSP、E2E 基地址等时，**禁止**默认前后端同站；须显式按跨源场景评估并实现可配置基 URL 与允许源列表
- **适用场景**：新功能与接口、登录与 SSO、文件与实时通道、开放 CORS、网关与多环境部署、Playwright 等浏览器自动化
- **优先级**：高
- **详细内容**：[异域前端与 API 域名分离](./01_project_constraints/17_cross_domain_frontend_api.md)

#### 静态资源引用须带内容 Hash 查询参数

- **描述**：通过 URL 引用且可能被浏览器或网关/CDN 缓存的前端静态资源（模板 `<script>`/`<link>`、绝对路径动态 `import`、开发态 `/utils/` 等），**必须**在 URL 上附带与**文件内容**绑定的 hash query（如 `?h=<摘要>`），避免命中过期 JS/CSS 引发模块导出缺失等运行时错误；优先走打包器 `import` 模块图
- **适用场景**：Django/SPA 壳资源注入、异域静态域、nginx 长 `max-age`、联调域经反向代理缓存 Vite 源码路径
- **优先级**：高
- **详细内容**：[静态资源缓存击穿（hash query）](./01_project_constraints/18_static_resource_cache_bust_query.md)

#### 单库 / 单表单服务数据所有权

- **描述**：任意数据库（或逻辑库）及其中任意数据表，运行时只允许由 **一个拥有服务** 直连访问（读与写均适用）。若多个服务需要同一库/表，须采用 **服务转发**（经拥有服务 API/RPC/领域事件）或 **将数据表迁移出去**，禁止共享连接串、跨服务复用 Model/DAO 或旁路 SQL
- **适用场景**：新服务拆分、新表设计、跨进程读写既有表、monorepo 多进程持久化、设计/DDD 限界上下文与仓储落点、发现多服务共用库或表时的整改
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[单库/单表单服务数据所有权](./01_project_constraints/19_single_service_data_ownership.md)

#### 新增服务 / 接口优先落 Go

- **描述**：凡新增业务 HTTP/RPC 接口或独立后端服务，默认实现为 **Go**。须先评估并入职责匹配的**现有 Go 服务**；无匹配限界上下文时**新建 Go 服务**。禁止将 Django/Flask 等 Python 进程新增 endpoint 作为默认方案；Python 仅可为可审计例外（须 Go 替代评估与设计门禁）。机器可读对照与 CI：`db/api_route_ownership.yaml`、`db/scripts/ci/check_django_new_api_routes.py`；存量迁出节奏见 `docs/superpowers/specs/2026-07-05-task2app-api-go-split-brainstorm-design.md`（不强制一次迁完）
- **适用场景**：新功能设计、新公网/服务间 API、新 sidecar、从 Django 拆出限界上下文、接口落点选型与架构评审
- **优先级**：高
- **规则类型**：禁止忽略
- **详细内容**：[新增服务/接口优先落 Go](./01_project_constraints/20_go_service_first_apis.md)；人读对照：[api-route-to-owner](../docs/architecture/api-route-to-owner.md)

#### 路径配置集中化规则

- **描述**：所有路径的设置必须仅在项目根目录的 `paths.conf` 文件中约定，并使用绝对路径；其他模块需要设置路径时应从该文件或项目提供的路径加载模块（如 `core.paths_loader`）获取
- **适用场景**：所有涉及文件路径、目录路径的配置（日志目录、静态资源、数据库路径、配置文件路径等）
- **优先级**：高
- **详细内容**：请参考 [路径配置规范](./01_project_constraints/04_paths_configuration.md)

#### 迁移前环境激活规则

- **描述**：执行数据库迁移命令前，必须先在仓库根目录执行 `source activate_env.sh` 激活对应环境，禁止在未激活环境下执行迁移
- **适用场景**：执行 `python manage.py makemigrations`、`python manage.py migrate` 等数据库迁移命令时
- **优先级**：高

#### 支付与 KYC 合规规则

- **描述**：为支付能力搭建 KYC 体系时，须满足身份层级与限额绑定、数据最小化、KYC 状态可审计、卡/敏感支付数据不进自有库与日志、第三方集成验签与密钥管理、AML 与高风险场景扩展、测试与生产数据隔离等要求；合规以法务/监管结论为准，代码与文档须可追溯对齐。
- **适用场景**：收款、付款、充值、提现、订阅计费、钱包余额、退款、分账及任何与资金或支付能力相关的功能、接口、数据与第三方集成
- **优先级**：高
- **详细内容**：请参考 [支付与 KYC 合规约束](./01_project_constraints/16_payment_kyc_compliance.md)

#### 微信支付官方 Go SDK 规则

- **描述**：处理微信支付 APIv3（下单、查单、退款、分账、回调验签/解密等）时，必须调用仓内 `sdk/wechatpay-go`（module `github.com/wechatpay-apiv3/wechatpay-go`，服务 `go.mod` 须 `replace` 到该目录）；优先 `services/<产品>` 类型化 API。禁止社区 SDK、APIv2 XML、手写签名或裸 HTTP。
- **适用场景**：taskBill 及任何微信支付接入、改 `go.mod` wechatpay 依赖、Agent 实现支付/退款/分账
- **优先级**：高
- **详细内容**：请参考 [微信支付必须调用仓内 wechatpay-go SDK](./01_project_constraints/59_wechatpay_go_sdk.md)；ADR-0032；Cursor `.cursor/rules/wechatpay-go-sdk.mdc`

#### 模拟登录审计标识规则

- **描述**：模拟会话下日志/审计必须含操作者、被模拟用户、会话 ID；网关注入 `X-Impersonator-Id` 与 `X-Impersonation-Session-Id`；开始模拟须理由并写入收信箱。
- **适用场景**：改访问日志、forward-auth、模拟登录、用户收信箱
- **优先级**：高
- **详细内容**：请参考 [模拟登录必须在日志与审计中标识](./01_project_constraints/60_impersonation_audit_label.md)；ADR-0038；Cursor `.cursor/rules/impersonation-audit-label.mdc`

### 一级分类：文档与流程管理

- 详细内容请参考：[文档与流程管理提示词](./02_documentation/00_documentation.md)

#### 业务逻辑调整文档规则

- **描述**：当调整业务逻辑时，需要创建对应的 wsd 文件，且这些 wsd 文件需要按照金字塔结构放在 docs/flows 目录中，并且这些目录名和文件名都应该用中文命名。每次调整时，应该先调整 docs/flows 目录中的文件，等review后，再应用到项目代码中
- **适用场景**：当修改或调整系统业务逻辑时
- **优先级**：中

#### 项目功能结构维护规则

- **描述**：当增加或调整项目功能时，需要将新增或调整的功能对应的添加到 docs/ProjectFeature.opml 中，确保功能结构与实际代码实现保持一致，按照现有的模块分类和层级结构进行添加或调整
- **适用场景**：当增加或调整项目功能时
- **优先级**：中

#### 需求对照与变更留痕规则

- **描述**：执行实现或调整类工作前，对照既有需求文档（如 `docs/intents/`、相关流程与 `ProjectFeature.opml` 等）；若当前指令与既有记载不一致或构成需求变更，须在落地代码与测试的同时，将变更后的需求写回对应文档并保留简要变更记录（日期、差异、原因），避免实现与文档漂移。Cursor 元规则见仓库内 `.cursor/rules/ai-rules-loader.mdc` 中执行原则第 5 条。
- **适用场景**：由 AI 或人工执行的实现、改造、及可能与既有规格不一致的交付
- **优先级**：高
- **详细内容**：见 [文档与流程管理提示词](./02_documentation/00_documentation.md) 中「9. 需求对照与变更落地」

### 一级分类：技术实现要求

- 详细内容请参考：[技术实现要求提示词](./03_technical_implementation/00_technical_implementation.md)
- **API 与浏览器跨源**（前端与 API 不同域名时的 CORS、凭证、回调等）与 [异域前端与 API 域名分离](./01_project_constraints/17_cross_domain_frontend_api.md) 一并评估
- **多服务库/表直连**禁止；一库/一表仅一服务访问，跨服务须转发或迁表，见：[单库/单表单服务数据所有权](./01_project_constraints/19_single_service_data_ownership.md)
- **新增服务/接口默认落 Go**（扩展现有或新建）；Python/Django 为可审计例外，见：[新增服务/接口优先落 Go](./01_project_constraints/20_go_service_first_apis.md)
- **服务 listen 禁止仅绑 127.0.0.1**（须 `0.0.0.0`；APISIX/边缘可达），见：[服务监听地址禁止仅绑 127.0.0.1](./01_project_constraints/22_service_listen_host_not_localhost_only.md)
- **应用启动禁止环境 Proxy**（`use_proxy: false`；Client 直连；env proxy 仅开发加速），见：[应用启动禁止使用环境变量 Proxy](./01_project_constraints/23_app_startup_no_env_proxy.md)
- **服务仅读本目录 conf**（跨服务配置须 sync 片段，禁止运行时直读他服务 YAML），见：[服务仅读本目录配置；跨服务配置须 sync](./01_project_constraints/29_service_own_conf_directory_only_via_sync.md)
- **迁移/重构——测试先行**（识别迁移/重构指令后必须先写测试建立行为基线，确保改动前后一致），见：[迁移/重构——测试先行](./01_project_constraints/30_migration_refactoring_test_first.md)
- **Bug 修复后 Playwright 验收**（用户可感知 bug 修复后必须写 Playwright 测试覆盖回归），见：[Bug 修复 Playwright 验收规范](./01_project_constraints/36_bug_fix_playwright_acceptance.md)
- **ORM 优先使用**（数据库操作必须优先 ORM，原始 SQL 仅限例外且须仓储层封装），见：[ORM 优先使用规范](./01_project_constraints/37_orm_priority_over_raw_sql.md)

#### 视图HTML内容返回规范

- **描述**：视图中不应该直接 hardcode 返回 HttpResponse html 内容，也不要返回模版文件，而是重定向到前端渲染页面
- **适用场景**：所有需要返回 HTML 内容的视图函数
- **优先级**：高
- **规则**：
  - 避免在视图函数中直接拼接 HTML 字符串
  - 避免使用 Django 的模板系统渲染 HTML 内容
  - 对于需要返回 HTML 内容的请求，应该重定向到前端渲染页面
  - 使用 HttpResponseRedirect 进行重定向到前端路由
  - 前端路由应该负责渲染对应的页面内容

#### Django 函数式编程规范

- **描述**：规范Django开发中函数式编程的使用，提高代码可测试性
- **适用场景**：所有Django功能开发，特别是新增功能
- **优先级**：中
- **规则**：
  - 对于新增功能，除非是Django自身的类库（如视图类、模型类等），否则应采用函数式编程的方式实现
  - 函数应保持纯函数特性，输入相同参数应返回相同结果，避免依赖外部状态
  - 函数应职责单一，专注于完成特定任务
  - 函数应易于测试，可通过传入参数和断言返回值进行单元测试
  - 对于复杂业务逻辑，应拆分为多个小函数，每个函数负责一个具体功能
  - 避免在函数中直接操作全局状态或进行副作用操作，如需进行副作用操作应通过参数注入或回调函数处理
  - 函数命名应清晰反映其功能，便于理解和使用

#### 属性访问异常处理规范

- **描述**：规范属性不存在异常的处理方式，确保代码的健壮性和可维护性
- **适用场景**：所有涉及属性访问的代码开发
- **优先级**：高
- **规则**：
  - 当出现属性不存在之类的异常时，不应该采用try-catch的方案
  - 应该经过测试找到正确的属性来撰写代码
  - 对于不确定的属性，应使用适当的方法进行检查（如使用hasattr或getattr）
  - 对于复杂数据结构，优先使用glom库进行属性提取，避免直接访问可能不存在的属性
  - 确保代码在访问属性前进行充分的验证，避免运行时异常
  - 对于SDK调用，禁止使用hasattr或getattr，因为SDK的属性是确切的，应该直接使用正确的属性名称

#### 异常处理降级策略规范

- **描述**：规范异常处理的降级策略，确保异常能够被正确处理和传递
- **适用场景**：所有涉及异常处理的代码开发
- **优先级**：高
- **规则**：
  - 当出现异常时，不要采用hardcode的方式来作为降级策略
  - 应该直接抛出异常，让上层调用者决定如何处理
  - 异常信息应该清晰明确，包含足够的上下文信息
  - 对于可预期的异常，应该使用特定的异常类型，而不是通用的Exception
  - 确保异常能够被正确捕获和处理，避免静默失败

#### API 错误分层返回规范

- **描述**：对外错误响应必须同时兼顾终端用户体验和研发调试效率，返回友好文案并附带脱敏后的调试细节
- **适用场景**：所有 REST API、第三方 SDK 转发接口、云厂商接口适配层
- **优先级**：高
- **规则**：
  - 错误响应中应包含用户友好文案（如 `message`），可直接用于前端提示
  - 同时提供结构化调试字段（如 `error_detail`、`error_type`、`request_id`）用于研发定位
  - 调试字段必须脱敏，严禁返回密钥、Token、Cookie、完整账号等敏感数据
  - 推荐统一错误响应结构：`{"status":"error","message":"用户友好文案","error_detail":"调试细节","request_id":"..."}`

#### ID 字段字符串传输规范（元规则）

- **描述**：`id` 及 `*_id` 类业务标识在**进程运行**与**进程间交互**（HTTP、Kafka/Redis、env、日志等）中一律以 **string** 存在；**仅在入库**（ORM/SQL 写库）时转换为 DB 原生类型（`bigint`/`int64` 等）；出库后在 Repository 边界立即转回 string
- **适用场景**：monorepo 内全部服务（Django、Go 侧车、taskBill、taskEvents 等）；Snowflake 主键与外键 ID；动态外键双字段中的 ID 段
- **优先级**：高（核心规则）
- **规则摘要**：
  - 禁止 JSON number 传递 Snowflake ID；禁止在领域层用 `int`/`int64` 跨模块传递业务 ID
  - HTTP Header（`X-Tenant-Id` 等）、URL 路径段、事件 payload 均为 string
  - 非 ID 数值字段仍按 API 数字字符串规范处理
- **详细内容**：[ID 字段字符串传输规范](./03_technical_implementation/11_id_field_string_transit.md)

#### GitOAuth 接口 Swagger 管理界面规则

- **描述**：GitOAuth 相关接口必须在 Swagger/OpenAPI 管理界面中保持可见、可调试、可追踪，防止接口实现与文档脱节
- **适用场景**：所有 GitOAuth 相关接口的新增、修改、下线（包括 `accounts` 中 `gitoauth`/`git_oauth` 相关路径）
- **优先级**：高
- **规则**：
  - 新增 GitOAuth 接口时，必须同步创建对应 Swagger 条目（路径、方法、参数、请求体、响应体、错误码）
  - 修改 GitOAuth 接口契约（入参、出参、状态码、鉴权）时，必须同步更新 Swagger 管理界面
  - 下线或废弃 GitOAuth 接口时，必须同步在 Swagger 中标记废弃或移除
  - 交付前应确认 Swagger 管理界面中可检索并可调试对应接口

#### 未使用变量清理规范

- **描述**：规范未使用变量的处理方式，确保代码的整洁性和可维护性
- **适用场景**：所有代码开发场景
- **优先级**：中
- **规则**：
  - 及时删除未使用的变量：当发现变量不再被使用时，应立即从代码中删除
  - 避免冗余变量：在编写代码时，应避免创建不必要的变量
  - 代码审查检查：在代码审查过程中，应检查并移除未使用的变量
  - 使用静态分析工具：利用静态分析工具（如lint）检测未使用的变量
  - 保持代码简洁：通过删除未使用的变量，减少代码的认知负荷，提高代码可读性
  - 避免内存占用：删除未使用的变量可以减少内存占用，提高系统性能

#### Python 辅助 Web 服务并发与锁规范（元规则）

- **描述**：新建或修改本机/容器侧 **Python 辅助 Web 服务**（如 `relayToTrae`、`mock_run_container` 等 Flask + `threading.Lock` + 共享内存状态）时，必须遵守锁分层与持锁禁区，避免 **不可重入锁重入**、持锁阻塞 I/O 导致 HTTP 永久挂起（经代理表现为 500 或前端无响应）
- **适用场景**：`mock_run_container/server.py` 及未来同模式 `*/server.py` 侧车；Django 内手写 `threading` 管理子进程/SSE 时参考持锁禁区（原 Python `relayToTrae/` 已迁移至 `go_relayToTrae/`）
- **优先级**：高（核心规则）
- **规则摘要**：
  - `*_locked()` 表示调用方已持锁，函数体内禁止再次 `with _lock`
  - 持锁路径禁止调用内部会再次加锁的包装函数；持锁区内禁止网络/子进程等待/`join`
  - 后台线程：锁内快照、锁外 I/O、锁内写回
  - 必须提供死锁回归单测（典型 API 序列 + 线程超时），参考 `mock_run_container/` 同类测试
- **详细内容**：[Python 辅助 Web 服务并发与锁规范](./03_technical_implementation/10_python_sidecar_web_concurrency.md)

#### 服务端性能规范

- **描述**：处理服务端代码时需注意数据库查询等性能问题，避免 N+1 查询、过度 SELECT、缺少索引等导致的性能下降
- **适用场景**：所有涉及数据库查询、ORM 使用、原生 SQL 的服务端代码
- **优先级**：高
- **规则**：
  - **N+1 查询规避**：使用 `select_related()`、`prefetch_related()` 预加载关联数据，避免在循环中逐条查询
  - **SELECT 字段限制**：使用原生 SQL 时禁止 `SELECT `*，应显式列出所需字段；ORM 可利用 `values()`、`only()`、`defer()` 限制返回字段
  - **查询字段索引**：WHERE、ORDER BY、GROUP BY、JOIN 中频繁使用的字段应有对应索引，新增查询时评估并补充索引
  - **ORM 优先**：数据库操作优先使用 ORM 而非原始 SQL；约束详见 [ORM 优先使用规范](./01_project_constraints/37_orm_priority_over_raw_sql.md)（约束索引第 35 条）

#### 服务端开发领域驱动规范

- **描述**：服务端开发**必须**采用领域驱动设计（DDD）与领域事件的方式，降低业务逻辑与底层依赖库的耦合；不得以「暂时方便」为由在领域层直连 ORM、消息队列客户端或云 SDK 等基础设施实现
- **适用场景**：所有服务端代码开发，包括 API、业务逻辑、数据持久化等
- **优先级**：强制（违反视为不合规，CI 与 pre-commit 将拦截）
- **规则**：
  - **领域模型优先**：业务逻辑应围绕领域模型组织，而非围绕数据库表或框架结构组织
  - **领域事件驱动**：跨聚合、跨模块的协作优先通过发布/消费领域事件实现，而非直接调用
  - **业务意图必发事件**：业务意图被接受后，必须向消息队列投递对应业务/领域事件（纯查询等例外须在 `docs/intents/` 标注）；见 [意图驱动开发工作流规范](./08_prompt_management/01_intent_driven_development.md)
  - **解耦底层依赖**：业务层不应直接依赖具体的 ORM、消息队列、存储等实现，应通过领域接口或适配层隔离
  - **领域边界清晰**：按业务领域划分模块，同一领域内的逻辑内聚，领域间通过事件或明确接口通信
  - **基础设施隔离**：将数据库、消息队列、外部 API 等基础设施访问封装在适配器/仓储中，领域层仅依赖抽象
  - **进程间数据所有权**：跨服务禁止共享同一库/表的直连；一库或一表仅由一个拥有服务访问，他方经 API/领域事件协作或迁表划清 owner（见 `.ai/01_project_constraints/19_single_service_data_ownership.md`）
  - **新增接口落点**：新服务与新 HTTP/RPC 接口默认落 Go（先扩展现有 Go 服务，否则新建）；不得默认在 Django/Python 扩面（见 `.ai/01_project_constraints/20_go_service_first_apis.md`）
  - **自动化校验**：GitHub Actions 工作流 `.github/workflows/ddd-bdd-compliance.yml` 与脚本 `scripts/ci/check_ddd_bdd_compliance.py` 对 `Saas_project/**/domain/**/*.py` 做静态导入门禁；本地提交若安装项目 pre-commit，亦会执行同一检查（`--staged`）

#### Kafka Handler 新事件派发顺序规范

- **描述**：处理事件的 Handler 若需派发新事件（如 `send_event`），应将派发操作放在业务逻辑的**最后**执行，避免在逻辑中间派发导致嵌套消费、执行顺序难以理解
- **适用场景**：`Saas_project/core/kafka/handlers` 下的所有事件处理脚本
- **优先级**：中
- **规则**：
  - 先完成当前 Handler 的主逻辑（创建/更新数据等），再派发衍生事件
  - 使事件流保持线性可追踪，便于日志排查与调试
  - 详细内容请参考 [最佳实践 - Kafka消息队列事件处理](./03_technical_implementation/05_best_practices.md)

#### 开发模式消息队列选择规范

- **描述**：当 `conf/port_config.json` 中 `django.messageQueue.memory` 为 `true`（或未配置 `messageQueue` 时旧版顶层 `isDev` 为 `true`）时，领域事件队列应采用内存实现，而非 Kafka，避免 Kafka 消费者进程无法热重载导致的代码版本不一致问题
- **适用场景**：本地开发选用内存队列时（通过 `django.messageQueue` 或旧版 `isDev` 标识）
- **优先级**：高
- **规则**：
  - 开发模式下使用内存消息队列，生产模式继续使用 Kafka
  - 消息队列切换通过依赖注入或配置开关实现，业务代码无需感知底层类型
  - 内存消息队列实现应与 Kafka 保持相同的接口语义
  - 详细内容请参考 [最佳实践 - 开发模式消息队列选择规则](./03_technical_implementation/05_best_practices.md)

#### 数据库 Migration 字段顺序规范

- **描述**：规范后端数据库 migration 文件中字段的声明顺序，确保 ID 字段始终位于第一列
- **适用场景**：所有 Django 或类似框架的 migration 文件（如 `migrations/*.py` 中的 `CreateModel`、`AddField` 等操作）
- **优先级**：中
- **规则**：
  - ID 字段第一列：在 `migrations.CreateModel` 的 `fields` 列表中，`id` 字段（或主键字段）必须作为第一个字段声明
  - 继承模型特殊处理：对于继承自 `AbstractUser`、`AbstractBaseUser` 等基类的模型，自定义的 `id` 字段应放在 `fields` 列表的最前面
  - 一致性原则：所有新建或修改的 migration 文件均应遵守此顺序，便于阅读和 diff 比较
  - 编写与审查：创建 migration 时注意调整字段顺序；对 AI 生成或迁移工具自动生成的 migration，应在提交前检查并调整

#### 数据库 Model 与 Migration 对应规范

- **描述**：确保每个 Model 都有对应的 migration 文件，保证数据库 schema 与模型定义同步
- **适用场景**：所有 Django 或类似框架的 Model 开发、模型新增或修改
- **优先级**：高
- **规则**：
  - 一一对应：每个 Model 类的创建和字段变更，必须在对应 app 的 `migrations/` 目录下有对应的 migration 文件记录
  - 新增 Model：新增 Model 后必须执行 `makemigrations` 生成 migration，不得遗漏
  - 修改 Model：修改 Model 字段后必须执行 `makemigrations` 生成 migration，不得直接修改已有 migration 文件内容
  - 合并迁移：禁止随意删除或合并已应用到生产环境的 migration 文件
  - 提交前检查：在提交涉及 Model 变更的代码前，应检查 `python manage.py makemigrations --check --dry-run` 无未生成的迁移

#### 模型字段 Null 禁止规范

- **描述**：模型中的字段除非有明确说明，否则禁止设为 Null（禁止使用 `null=True`）
- **适用场景**：所有 Django 或类似框架的 Model 字段定义
- **优先级**：高
- **规则**：
  - 默认禁止 Null：定义 Model 字段时，默认不得使用 `null=True`，应使用非空约束或提供默认值
  - 明确说明例外：若有业务需求必须允许 NULL，需在字段旁添加注释明确说明原因
  - 字符串字段：对于 CharField、TextField 等，应使用 `blank=True` 配合空字符串表示可选，而非 `null=True`
  - 外键可选：外键若允许为空，使用 `null=True, blank=True` 时，需在注释中说明该关系为可选的业务原因

#### 外键禁止与业务层关联规范

- **描述**：后端模型中除非明确约定，否则禁止使用 ForeignKey、OneToOneField、ManyToManyField 等外键字段；数据库表之间的关联应在业务层通过 ID 等字段自行维护
- **适用场景**：所有 Django 或类似框架的 Model 定义、新增或修改
- **优先级**：高
- **规则**：
  - 默认禁止外键：定义 Model 时，不得使用 ForeignKey、OneToOneField、ManyToManyField 等 ORM 级关联字段
  - 业务层维护关联：表与表之间的逻辑关联应通过业务层实现，使用普通字段（如 IntegerField、CharField 存储关联 ID），在业务逻辑中显式查询、校验和组装关联数据
  - 明确约定例外：若经团队或项目明确约定确需使用外键字段，须在 Model 或相关设计文档中注明原因和适用范围

#### 动态外键双字段规范

- **描述**：模型中涉及动态外键（即关联目标可能是多种不同模型之一）时，应拆分为两个字段：一个字段标记外键所属的模型，另一个字段存储外键的 ID
- **适用场景**：所有涉及多态关联、通用关联或动态外键的 Model 定义、新增或修改
- **优先级**：高
- **规则**：
  - 双字段结构：不得使用单一 ForeignKey 或 GenericForeignKey 直接引用；必须采用「模型标记字段 + ID 字段」的双字段设计
  - 模型标记字段：用于标识外键指向的模型，可存储 content_type、model_name、app_label+model 等
  - ID 字段：用于存储该模型下对应记录的 ID
  - 业务层解析：根据模型标记字段和 ID 字段在业务层中解析并获取关联对象

### 一级分类：前端开发规范

- 详细内容请参考：[前端开发规范提示词](./04_frontend_development/00_frontend_development.md)
- 平台设计规范技能包请参考：[平台设计规范技能包](./04_frontend_development/01_platform_design_skills.md)
- DESIGN.md 集成规范请参考：[DESIGN.md 集成规范](./04_frontend_development/02_design_md_integration.md)
- **前端与 API 分域名**时的跨源、鉴权与基地址约束见：[异域前端与 API 域名分离](./01_project_constraints/17_cross_domain_frontend_api.md)
- **静态资源 URL 缓存**须带内容 hash 查询参数，见：[静态资源缓存击穿（hash query）](./01_project_constraints/18_static_resource_cache_bust_query.md)
- **请求报错展示**须在错误 DOM 上带 **`data-traceId`**，见：[请求报错展示须带 data-traceId](./01_project_constraints/24_frontend_error_data_trace_id.md)
- **无必要禁止 Teleport**（就地渲染；仅 overflow/层叠无法解决时才允许），见：[前端无必要禁止 Teleport](./01_project_constraints/50_no_unnecessary_vue_teleport.md)

#### 平台设计技能包使用规则

- **描述**：在进行界面设计、设计评审或可访问性审计时，应加载并使用 [ehmo/platform-design-skills](https://github.com/ehmo/platform-design-skills) 中对应平台的规范。该技能包涵盖 Apple HIG、Material Design 3、WCAG 2.2，适用于 Web、iOS、Android 等多平台
- **适用场景**：Web 界面设计与评审、可访问性审计、多平台 UI 设计与合规检查
- **优先级**：中
- **规则**：
  - Web 前端：使用 `web` 技能（响应式、WCAG、性能、现代 CSS/HTML）
  - 安装：`npx skills add ehmo/platform-design-skills`
  - 技能在检测到平台相关任务时自动激活

#### DESIGN.md 统一视觉基线规则

- **描述**：在前端页面设计和实现时，采用项目级 `DESIGN.md` 作为统一视觉输入，并与平台设计技能包配合，兼顾风格一致性与可访问性合规。可执行技能：[`.claude/skills/design-md`](../.claude/skills/design-md/SKILL.md)
- **适用场景**：Web 页面设计、UI 重构、组件视觉统一、设计评审
- **优先级**：高
- **规则**：
  - 产品 SPA 以仓库根 [`DESIGN.md`](../DESIGN.md) 为 SSOT（紫曜 `#7241F2`），不要每次任务从 catalog 换品牌
  - 仅当明确要求新视觉语言时，才从 [awesome-design-md](https://github.com/VoltAgent/awesome-design-md) 选模板并本地化；禁止整份覆盖品牌文件
  - 使用 `DESIGN.md` 统一颜色、字体、间距、圆角、阴影和组件语义，避免页面间样式漂移
  - 使用 `platform-design-skills` / `web-design-guidelines` 进行可访问性和交互合规检查，尤其确保 WCAG 2.2 基线
  - 若视觉风格与可访问性冲突，优先满足可读性、可操作性与可访问性，再调整视觉参数
  - 门禁：`python3 db/scripts/ci/check_design_md.py`

#### 前端组件分类规范

- **描述**：规范前端组件的分类和实现方式，确保界面组件和逻辑组件的清晰分离
- **适用场景**：所有前端组件开发和修改
- **优先级**：高
- **规则**：
  - 界面组件（UI Components）：专注于展示逻辑，优先使用函数式组件实现，避免包含副作用
  - 逻辑组件（Logic Components）：负责业务逻辑处理，允许包含副作用（如API调用、状态管理等）
  - 组件拆分原则：将界面展示与业务逻辑分离，提高代码可维护性和复用性
  - 命名规范：
    - 界面组件：采用 `{Feature}.ui.vue` 格式命名，如 `Navbar.ui.vue`
    - 逻辑组件：采用 `{Feature}.logic.vue` 格式命名，如 `Auth.logic.vue`
    - 现有组件：在修改时逐步迁移，新组件必须遵循此规范
  - 项目修改原则：
    - 在项目修改过程中，必须对涉及的组件按照此原则进行调整
    - 每次修改至少确保一个组件符合规范
    - 建立组件迁移计划，分阶段完成所有组件的规范化
    - 定期检查规范执行情况，确保所有新代码都符合要求

#### 大型组件行数门禁（自动拆分）

- **描述**：单个前端组件文件超过 **500 行**（物理行数，含模板/样式/脚本）时，须在新建、大改或维护触及该文件时**自动拆分**为多个子组件或 composable 等边界清晰的单元，使各文件回落至阈值以内；**门禁触发后须当场削到 ≤ 阈值**（全局义务见约束专文）
- **适用场景**：所有前端组件源码（Vue/React/Svelte 等 SFC 及等价页面组件）
- **优先级**：高
- **规则**：
  - 阈值：**大于 500 行**即触发拆分义务
  - 触及文件时必须拆分，禁止在无拆分前提下继续堆叠功能
  - **门禁触发后强制削减**：失败输出点名的文件须立即拆分至 ≤ 阈值并复跑验收；见 [27_source_file_line_limit_auto_reduce.md](./01_project_constraints/27_source_file_line_limit_auto_reduce.md)
  - 详细度量与例外说明见：[前端开发规范提示词](./04_frontend_development/00_frontend_development.md) 中「大型组件行数门禁（自动拆分）」
  - **门禁脚本**：`task2app/scripts/ci/check_frontend_component_line_limit.py`（pre-commit 校验暂存组件；CI 与上述 workflow 注入的 diff 范围一致；CI 可对遗留超大文件启用 **`TASK2APP_FRONTEND_LEGACY_GROWTH_GRACE`** 净增长容差，详见脚本说明；容差不免除 Agent 削文件义务）

#### 无必要禁止 Teleport

- **描述**：Vue `<Teleport>` / React `createPortal` 默认禁止；UI 出现在哪棵子树就在哪棵子树挂载。仅当祖先 overflow/transform 导致浮层被裁剪或错位、且无法就地解决时才允许，并须 `Teleport-OK` 注释
- **适用场景**：所有前端组件；新增浮层、下拉、模态、跨区域搬 DOM
- **优先级**：高
- **规则**：
  - 禁止用 Teleport 把功能卡送到远亲 `[data-testid=…]` 槽
  - 禁止 `Teleport defer` 等待另一区域 `v-if`
  - 细则见：[前端无必要禁止 Teleport](./01_project_constraints/50_no_unnecessary_vue_teleport.md)

#### 前端按钮 className 规范

- **描述**：所有前端按钮必须添加 className 属性，以便于测试和自动化脚本的定位
- **适用场景**：所有前端按钮元素的开发和修改
- **优先级**：高
- **规则**：
  - 每个按钮元素必须设置唯一的 className 属性
  - className 命名应遵循语义化原则，清晰表达按钮的功能
  - className 命名应使用小写字母和连字符（kebab-case）格式
  - 避免使用动态生成的 className，确保按钮可以被稳定定位
  - 现有按钮在修改时应添加 className，新按钮必须遵循此规范

#### 前端错误提示与调试信息规范

- **描述**：前端展示错误时应优先保证用户可理解性，同时保留研发调试所需的细节
- **适用场景**：所有前端 API 调用、异常捕获和错误反馈交互
- **优先级**：高
- **规则**：
  - 用户可见错误提示应使用 toast 或全局消息组件展示友好文案
  - 禁止将原始堆栈或协议错误直接展示给用户
  - 后端返回的 `error_detail`、`request_id` 等调试信息应写入控制台日志或调试面板
  - toast 文案需简洁可执行，必要时提供下一步建议（如重试、检查网络、联系管理员）

#### 子模态框层级与背景规范

- **描述**：子模态框应位于主模态框之上，子模态框的背景遮罩应使用半透明黑色
- **适用场景**：所有包含主模态框与子模态框嵌套的前端开发
- **优先级**：高
- **规则**：
  - 通过 z-index 确保子模态框层级高于主模态框
  - 子模态框的背景遮罩使用半透明黑色（如 `rgba(0, 0, 0, 0.5)`）

### 一级分类：测试与质量保障

- 详细内容请参考：[测试与质量保障提示词](./05_testing_quality/00_testing_quality.md)

#### BDD 先测后写规则

- **描述**：撰写功能时**必须**按照 BDD（行为驱动开发）的方式进行；CI 无法回放「先后顺序」，因此对仓库采用**可执行近似**：同一 PR/提交中，凡变更业务实现代码（后端 `Saas_project/`、`taskAiProvider/` 下实现路径，或约定前端源码路径），**必须**同时变更至少一类测试资产（pytest、Playwright、Go `*_test.go`、`*.test.js` 等）。本地推荐严格保持「先写测例再写实现」的实际工序；当测例全部通过时，业务职能即视为能够正确跑通
- **适用场景**：所有新功能开发、业务逻辑实现、API 接口开发、领域服务开发
- **优先级**：强制（违反视为不合规，CI 与 pre-commit 将拦截）
- **规则**：
  - 第一步：编写测例（在实现业务逻辑之前，先根据需求/行为规格编写对应的测试用例）
  - 第二步：实现业务职能（测例编写完成后，再编写具体的业务代码，使测例从失败变为通过）
  - 第三步：测例通过即完成（当所有相关测例通过时，表明业务职能已正确实现）
  - 禁止先写业务再补测，应严格遵循「测例先行」的顺序；并与 CI「业务与测例同批变更」门禁一并遵守
  - **自动化校验**：`.github/workflows/ddd-bdd-compliance.yml` 调用 `scripts/ci/check_ddd_bdd_compliance.py`；细则见 `scripts/ci/README_DDD_BDD_COMPLIANCE.md`
- **详细内容**：请参考 [BDD 开发流程规范](./05_testing_quality/04_bdd_development_workflow.md)

#### 单元测试定位规则

- **描述**：当出现某个逻辑没有验通时，应该对这个逻辑的每个单独的一环进行单元测试，以便缩小处理的复杂度，快速定位到哪里出现了问题。等问题修复后，需要对整体逻辑进行一次验证，以确保整体逻辑没有被修改
- **适用场景**：当系统逻辑出现验证失败或异常时
- **优先级**：高

#### DDD测试原则

- **描述**：测试**必须**符合 DDD 原则；除非该测试专门验证外部依赖行为，否则不得绕过领域边界或把基础设施细节当作领域断言的依据
- **适用场景**：当编写或修改测试用例时
- **优先级**：强制

#### Playwright与Chrome DevTools MCP配合使用规则

- **描述**：使用 playwright 时需要配合 Chrome DevTools MCP 一起使用，以提高测试效率和调试能力
- **适用场景**：当使用 playwright 进行测试时
- **优先级**：高

#### Playwright 通过 9222 端口连接非沙盒 Chrome 规则

- **描述**：使用 Playwright 时，应通过 **9222** 端口的 CDP 连接已由外部启动的 Chrome（独立用户数据目录、非默认 `launch()` 隔离 Chromium），测试代码使用 `connectOverCDP('http://127.0.0.1:9222')` 等；浏览器需预先以 `--remote-debugging-port=9222` 启动（工作区根目录可参考 `runDebugChrome.sh`）。Linux 容器等场景如需关闭浏览器沙盒，启动 Chrome 时可附加 `--no-sandbox`，仍以 9222 为 CDP 端口
- **适用场景**：Playwright 端到端测试、智能体通过 Playwright 操作页面
- **优先级**：高
- **详细内容**：请参考 [核心测试规则](./05_testing_quality/01_core_testing_rules.md) 中的「Playwright 通过 9222 端口连接非沙盒 Chrome 规则」

#### Playwright 本地可见浏览器规则

- **描述**：运行本仓库 `@playwright/test` 用例时，本地与智能体环境应显示浏览器窗口；仅在 CI（已设置 `CI` 环境变量）等自动化场景使用无头。实现与细则见 `.ai/05_testing_quality/02_test_management_rules.md` 及 `front_project/app/playwright.config.js` 中的 `use.headless`
- **适用场景**：执行、编写或调试 `front_project/app/tests/playwright/` 下 Playwright 测试
- **优先级**：高

#### Playwright 测试环境变量登录规则

- **描述**：使用 Playwright 测试时，必须通过环境变量提供口令，禁止硬编码。**通用登录**：`PLAYWRIGHT_TEST_EMAIL`、`PLAYWRIGHT_TEST_PASSWORD`。**GitHub 绑定或相关流程**：`PLAYWRIGHT_TEST_GITHUB_EMAIL`、`PLAYWRIGHT_TEST_GITHUB_PASSWORD`。邮箱未设置时可用文档约定的测试身份；口令缺失则 skip 或失败。完整约定见 [`task2app/测试.ai.md`](../task2app/测试.ai.md)。约束第 57 条。
- **适用场景**：所有 Playwright 测试中需要账号登录或 GitHub 相关操作的场景
- **优先级**：高
- **详细内容**：请参考 [测试管理规则](./05_testing_quality/02_test_management_rules.md)

#### 测试短信号码目标规则

- **描述**：在测试（单元测试、集成测试、Playwright、手工联调等）中，凡需指定短信接收方号码的，**目标号码必须且仅能**取自项目根目录 `conf/port_config.test.json` 中 `django` 对象下的 `test_phone_numbers` 数组；不得向未列入该数组的手机号发送测试短信，也不得在测试代码或脚本中硬编码其他号码作为收件人。读取与校验约定可与 `Saas_project/tests/test_phone_config.py`（如 `load_django_test_phone_numbers`）保持一致。
- **适用场景**：验证码短信、通知短信、绑定手机、充值/账单等涉及短信投递的测试与调试
- **优先级**：高

#### 测试文件存储规范

- **描述**：所有测试文件必须放置在专门的目录中，避免与项目业务代码混合
- **适用场景**：所有测试文件的创建和管理
- **优先级**：高
- **规则**：
  - 后端单元测试：应放在`test`或`tests`目录中，按照模块或功能进行组织
  - 前端测试：应放在`front_project/tests`目录中，按照测试类型或功能进行组织
  - 集成测试：应放在专门的`integration_tests`目录中
  - 端到端测试：应放在专门的`e2e`或`end_to_end_tests`目录中
  - 测试目录结构应清晰反映测试类型和覆盖范围

#### 测试分层隔离规则

- **描述**：领域模型测试、基础设施测试与端到端测试应严格分开，不得混合在同一目录或文件中。领域模型测试放在 `tests/domain/` 或 `tests/{domain_module}/`，基础设施测试放在 `tests/infrastructure/` 或 `integration_tests/infrastructure/`，端到端测试放在 `e2e/`、`end_to_end_tests/` 或 `front_project/app/tests/playwright/`
- **适用场景**：所有测试文件的组织、创建和维护
- **优先级**：高

#### 测试时使用内存实现（依赖注入规范）

- **描述**：对 Kafka、Redis、数据库、邮件、云厂商 SDK 等外部依赖应采用依赖注入方式，测试时切换为内存/Mock 实现，以减少测试时间、无需启动外部服务。通过 `USE_IN_MEMORY_SERVICES` 或 `saas_project.settings_test` 启用；使用 `core.services` 的抽象接口与注册表
- **适用场景**：涉及外部依赖的后端开发、单元测试、测试环境配置
- **优先级**：高
- **详细内容**：请参考 [测试时使用内存实现](./05_testing_quality/03_in_memory_services_for_testing.md)

#### 测试文件命名规范

- **描述**：不同类型的测试文件需要使用不同的文件后缀，以便于识别和使用适当的测试运行器
- **适用场景**：所有测试文件的创建和命名
- **优先级**：高
- **规则**：
  - Playwright 端到端测试：使用 `.playwright.test.js` 后缀
  - 单元测试：使用 `.unit.test.js` 后缀
  - 集成测试：使用 `.integration.test.js` 后缀
  - 其他测试：使用 `.test.js` 后缀
  - 测试文件命名应清晰反映测试的功能和范围

#### 测试意图伴随文档规则

- **描述**：编写或修改测试时，必须为该测试文件配套 `${testFileName}.testIntent` 自然语言意图文档，确保后续测试被破坏时可据此恢复原始验证目标
- **适用场景**：所有测试文件（单元、集成、端到端、Playwright）的创建与修改
- **优先级**：高
- **规则**：
  - 同目录维护同名意图文件：测试文件 `foo.test.js` 对应 `foo.test.js.testIntent`
  - 意图内容必须使用自然语言说明：测试目标、关键断言、破坏信号、回归验证要点
  - 推荐使用统一模板：`docs/intents/00_索引/test_file_intent_template.testIntent`
  - 测试语义变化时必须同步更新 `.testIntent`，禁止仅修改测试代码不更新测试意图

#### 前端变更 Playwright 验证规则

- **描述**：当前端代码修改完成后，必须使用 Playwright 进行验证测试，确保修改后的功能正常工作。验证时应使用指定的测试账号：租户账号 ([contact@daydaymoney.com](mailto:contact@daydaymoney.com)/rgNodkdq8677!ci) 和系统管理员账号 ([author@example.com](mailto:author@example.com)/rgNodkdq8677!ci)。
- **适用场景**：所有前端代码修改完成后
- **优先级**：高
- **规则**：
  - 前端修改完成后，必须运行相关的 Playwright 测试用例进行验证
  - 验证时应使用提供的测试账号进行登录测试
  - 确保修改的功能在租户和系统管理员权限下都能正常工作
  - 测试完成后，应检查测试结果，确保所有测试通过

#### Bug 修复后 Playwright 验收规则（元规则）

- **描述**：凡修复用户可感知的 bug（前端交互、API 响应异常导致的前端错误展示、业务流程断裂等），修复后**必须**编写或扩展 Playwright 端到端测试用例覆盖该 bug 场景，并通过测试验证修复有效且无回归。纯后端/内部服务 bug（不影响用户可见行为）可用单元/集成测试代替，但仍须有自动化测试覆盖。与「迁移/重构——测试先行」互补：前者管迁移/重构，本条专管 bug 修复的回归验收。不确定时默认触发 Playwright。
- **适用场景**：所有 bug 修复（用户可感知的 bug 强制 Playwright；纯后端 bug 可用单元/集成测试代替）；Agent 排障修复、/goal 交付、手动修 bug 后的验收
- **优先级**：高（核心规则）
- **规则类型**：禁止忽略（用户可感知 bug）
- **规则**：
  - bug 修复完成后，必须编写或扩展现有 Playwright 测试覆盖该 bug 的原始复现场景
  - 测试须覆盖正常路径（防回归）+ 原始 bug 场景（验修复）
  - 纯后端 bug（不影响用户可见行为）：可用单元/集成测试代替，但 commit message 须说明为何未使用 Playwright
  - 测试通过后结果写入 `commitResult/` 对应目录
- **详细内容**：[Bug 修复 Playwright 验收规范](./01_project_constraints/36_bug_fix_playwright_acceptance.md)；约束索引第 34 条

#### 单元测试阿里云 Access Key 获取规则

- **描述**：撰写单元测试时，碰到需要使用 ALIYUN_ACCESS_KEY 的话，应该从数据库的表 cloud_cloudplatformauthorization 的 remark 为 [ljy080829@gmail.com](mailto:ljy080829@gmail.com) 的记录中获取
- **适用场景**：当编写涉及阿里云服务的单元测试时
- **优先级**：高

#### 测试文件可执行性规则

- **描述**：所有测试文件必须在同一目录下配备对应的 `fileName.sh` 脚本，运行该脚本即可直接执行该测试文件
- **适用场景**：所有测试文件的创建和管理
- **优先级**：高
- **规则**：
  - **统一要求**：每个测试文件旁应有一个同名的 `.sh` 脚本（如 `test_foo.py` 对应 `test_foo.sh`），脚本运行后直接执行该测试文件
  - **命名规范**：脚本名为 `{测试文件名去掉扩展名}.sh`，与测试文件置于同一目录
  - **后端测试**：创建 Python 单元测试（`test_*.py`）时，必须同时创建对应的 `test_*.sh` 脚本，脚本负责激活单元测试环境并调用 pytest 执行该测试文件
  - **前端测试**：创建 Playwright 等前端测试时，必须同时创建对应的 `fileName.sh` 脚本，脚本负责执行该测试文件，并把测试结果输出到 commitResult 中
  - 测试脚本应该能够直接运行（`./fileName.sh`），无需依赖外部测试运行器（虽然也可以通过 pytest/npx 等直接执行测试文件）
  - 确保测试脚本在不同环境中都能正常执行

### 一级分类：执行与监控

- 详细内容请参考：[执行与监控提示词](./06_execution_monitoring/00_execution_monitoring.md)

#### 解决方案验证规则

- **描述**：当发现解决方案后，应该仅保留解决方案的修改，而丢弃中间尝试过程的修改，这样以确保仅修改必要的内容，而不新增无用的内容，等丢弃中间尝试后，应该对解决方案做最终验证
- **适用场景**：当实现或修复系统功能时
- **优先级**：高

#### 测试执行状态记录规则

- **描述**：每次执行 ./.git/hooks/pre-commit 时，应该先清空 `根目录/commitResult`，然后把每个测试的执行状态和结果放在该目录中，以`testFileName`.log 为命名，执行结束后，如果执行结果为成功那么文件名重命名为 success.`testFileName`.log, 如果执行结果为失败，那么文件名重命名为 fail.`testFileName`.log
- **适用场景**：所有 Git 提交操作的 pre-commit 钩子执行
- **优先级**：高

#### 测试结果目录结构规则

- **描述**：commitResult 中的测试结果文件，需要按照测试文件的目录结构进行对应的存放
- **适用场景**：所有测试结果的存储和管理
- **优先级**：高
- **规则**：
  - 对于测试文件 `front_project/tests/components/LoginInputTest.playwright.test.js`，其测试结果文件应该存放在 `commitResult/front_project/tests/components/` 目录中
  - 对于测试文件 `front_project/tests/e2e/login.test.js`，其测试结果文件应该存放在 `commitResult/front_project/tests/e2e/` 目录中
  - 确保 commitResult 目录结构与测试文件的目录结构保持一致，便于查找和管理测试结果

#### 测试脚本退出码处理规则

- **描述**：所有测试脚本（包括Python脚本和shell脚本）必须在测试失败时返回非零的退出码，在测试成功时返回零的退出码。这确保pre-commit钩子能够正确判断测试结果并生成相应的日志文件名。
- **适用场景**：所有测试脚本的编写和修改
- **优先级**：高
- **规则**：
  - Python脚本：使用 `sys.exit(0)` 表示成功，`sys.exit(1)` 表示失败
  - Shell脚本：使用 `exit 0` 表示成功，`exit 1` 表示失败
  - 确保测试脚本在所有失败情况下都能正确返回非零退出码
  - 定期检查现有测试脚本，确保它们符合此规则

#### 目标明确与验证规则

- **描述**：每次输入指令时，都要明确目标是什么，然后在停止前需要核验是否完成目标，如果没有完成目标则需要继续执行下去，直到完成目标
- **适用场景**：所有任务执行和指令处理
- **优先级**：高
- **规则**：
  - 执行前澄清：在开始执行前，如果认为有需要提前了解的问题（如需求细节、技术约束、边界条件、潜在风险等），必须主动向使用者澄清，不得在假设基础上直接开始执行
  - 指令输入时：明确说明具体目标和预期结果
  - 执行过程中：定期检查进度和目标完成情况
  - 结束前验证：确保所有目标都已完成，未完成的目标需要继续执行
  - 目标调整：如果目标需要调整，应明确沟通并重新确认新目标

#### 预期问题前置处理规则

- **描述**：当能够预期某个问题将会发生时，应立即在当前执行阶段直接处理并消除该问题，不得等问题实际出现后再被动修复
- **适用场景**：所有需求实现、代码修改、测试验证、提交流程与自动化执行场景
- **优先级**：高
- **规则**：
  - 发现可预见风险（如必然报错、边界缺失、依赖缺口、校验缺失）时，直接纳入当前变更一并解决
  - 不以“先跑通主流程、后续再修”为理由推迟已识别的问题处理
  - 若风险可在当前上下文快速验证，应立即补充验证（测试、静态检查、最小复现）
  - 仅当受外部权限、环境或信息阻塞时，才允许暂缓处理，并需明确告知阻塞点与后续动作

### 一级分类：TOGAF项目管理

- 详细内容请参考：[TOGAF项目管理提示词](./07_togaf_project_management/00_togaf_project_management.md)

### 一级分类：提示词管理

- 详细内容请参考：[提示词管理指引](./08_prompt_management/00_prompt_management.md)

#### 意图驱动开发闭环规则

- **描述**：当用户输入功能意图时，必须先在 `docs/intents/` 中按金字塔结构创建/更新功能意图文档，再生成同名异后缀的测试意图文档，随后按功能意图实现代码，按测试意图生成测试并执行验证。服务端业务意图出现时，必须向消息队列投递对应业务/领域事件，并在意图文档中维护「意图 → 事件 → 发布点」对照。
- **适用场景**：所有功能开发、功能改造、功能修复任务
- **优先级**：高
- **详细内容**：请参考 [意图驱动开发工作流规范](./08_prompt_management/01_intent_driven_development.md)

### 一级分类：失败案例经验库

- 详细内容请参考：[失败案例经验库](./09_failure_experience/00_failure_experience.md)

### 一级分类：软件设计哲学

- 详细内容请参考：[软件设计哲学提示词](./10_software_design_philosophy/00_software_design_philosophy.md)

#### 软件设计哲学应用规则

- **描述**：将 John Ousterhout 的《软件设计的哲学》中的核心原则作为项目开发的指导思想，用于代码审查、架构讨论、API 设计、模块分解决策、重构指导、复杂性分析、命名和注释改进、错误处理策略设计等场景。
- **适用场景**：所有软件开发和设计活动，特别是代码审查、架构设计、API 设计、模块分解、重构等场景
- **优先级**：高
- **规则**：
  - 以管理复杂性为核心挑战，识别和消除复杂性的症状（变更放大、认知负荷、未知的未知）
  - 采用战略性编程而非战术性编程，投资 10-20% 的开发时间用于设计改进
  - 设计深度模块（简单接口，丰富实现），避免浅模块
  - 实施信息隐藏，避免信息泄露
  - 优先设计通用目的模块，分离通用和专用代码
  - 确保不同层级提供不同的抽象，避免直通方法和直通变量
  - 将复杂性向下拉，让模块内部吸收复杂性而不是推给调用者
  - 通过重新定义语义来消除错误，减少异常处理
  - 对重要设计决策进行双重设计（设计两次）
  - 编写有意义的注释，描述代码中不明显的信息
  - 使用精确、一致、信息丰富的命名
  - 保持系统一致性，相同的事物用相同的方式处理
  - 编写明显的代码，让读者能够快速理解代码行为和意图
  - 使用红旗快速参考来发现设计问题

### 一级分类：AI辅助软件开发规范

- 详细内容请参考：[AI辅助软件开发规范提示词](./11_ai_development/00_ai_development.md)

#### Agent 工作流技能（pskoett-ai-skills）

- **描述**：多步、长会话或高复杂度 AI 编码任务时，按需加载 [pskoett/pskoett-ai-skills](https://github.com/pskoett/pskoett-ai-skills) 规范；本仓库已将对应技能纳入 `.claude/skills/` 子目录并与项目路径对齐（钩子与示例命令使用 `./.claude/skills/...`，而非上游默认的 `./skills/...`）
- **适用场景**：中大型功能/重构、跨会话任务、需要规划访谈、意图约束、上下文健康度监控、交付前简化加固与持续学习沉淀时
- **优先级**：中
- **规则**：
  - 流水线与取舍见各技能 `SKILL.md`（如 `plan-interview`、`intent-framed-agent`、`context-surfing`、`simplify-and-harden`、`self-improvement`）； trivial 任务不必强行套全流程
  - CI/无头场景可选用 `self-improvement-ci`、`simplify-and-harden-ci`
  - `dx-data-navigator` 依赖 Salesforce DX Data Cloud 与有效凭证，无环境则勿调用
  - `agent-teams-simplify-and-harden` 用于多 Agent 并行审计类工作流，按需启用

#### 统一 Agent 交付工作流（吸收）

- **描述**：**单轨十步**（与 `0-auto-flow` / `.claude/skills/README.md` 编号一致）；上游 Superpowers/gstack 仅为概念映射；权威专文 [`.ai/11_ai_development/03_superpowers_workflow.md`](./11_ai_development/03_superpowers_workflow.md)。**实现类任务默认自动按该流程执行**，无需用户显式提及上游产品名；以 `.claude/skills/` 与 `.ai` 落实各步，**不**全局强制上游「每轮必先调 Skill 工具」。
- **适用场景**：新功能、跨模块改动、长会话、易返工；琐碎无行为语义变更可裁剪（见专文）
- **优先级**：中高（与核心约束冲突时以 `.ai/01_project_constraints/` 等为准）
- **规则**：
  - 专文含十步主链、**systematic-debugging** / **verification-before-completion** 与 `.ai/09`、`diagnose`、测试/CI 的映射（「调试与完成门槛」节）
  - 权威专文：[`.ai/11_ai_development/03_superpowers_workflow.md`](./11_ai_development/03_superpowers_workflow.md)
  - [`.ai/11_ai_development/04_gstack_sprint_alignment.md`](./11_ai_development/04_gstack_sprint_alignment.md) 仅为旧链接重定向，勿双轨维护

#### Matt Pocock Skills 实践吸收

- **描述**：系统化吸收开源合集 [mattpocock/skills](https://github.com/mattpocock/skills)（*Skills For Real Engineers*：小而可组合、易改编、与模型无关）中的工程向技能要点。上游按 slash 命令组织（如 `/tdd`、`/diagnose`）；**本仓库不要求**执行 `npx skills@latest add mattpocock/skills` 或安装上游插件，行为约束以本文与 `.ai` / `.claude/skills/` 为准。若需逐字对照上游流程，可自行克隆该仓库阅读 `skills/` 下各 `SKILL.md`。
- **适用场景**：非琐碎功能、跨模块改动、长会话、易出现需求偏差；或缺少可运行证据（类型检查、测试、最小复现）即宣称完成的情况
- **优先级**：中（与 `.ai/01_project_constraints/` 等核心约束冲突时以核心约束为准）
- **上游四类失败模式与对策（README 框架）**：
  1. **人与 Agent 不一致（misalignment）** → 动工前做结构化对齐（grilling），消化决策树分支与验收标准。
  2. **表述冗长、概念漂移** → 使用项目统一术语（共享语言 / ubiquitous language）；术语与意图见 `docs/intents/`、`.ai/02_documentation/`。
  3. **代码不可用、缺反馈环** → 以垂直切片 TDD、静态检查、自动化测试与可重复复现为速度上限；疑难 bug 走规范诊断闭环。
  4. **复杂度失控（ball of mud）** → 持续关注模块深度与接缝（seam）；大改前明确触及范围，必要时做架构深化审视而非盲目堆代码。
- **上游工程技能 ↔ 本项目落地（名称仅供对照）**：

  | 上游技能                            | 作用                 | 本项目落地                                                                                                                                                                                                                                     |
  | ------------------------------- | ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
  | `grill-me`                      | 非代码场景下穷尽追问计划       | 与 `plan-interview`、重大需求前的结构化澄清同用                                                                                                                                                                                                          |
  | `grill-with-docs`               | 对齐计划并随会话更新术语与 ADR  | 冲突术语当场指出；决策写入 `.ai/02_documentation`、`docs/intents/` 或团队约定 ADR 路径；**一问一答**、能查代码则先查代码                                                                                                                                                      |
  | `tdd`                           | 红-绿-重构、垂直切片        | `.ai/05_testing_quality/`；**禁止「一次写完全部测试再写实现」**（水平切片）；采用 **tracer bullet**：单测 → 最小实现 → 再下一测；好测试断言**可观察行为与公共接口**，避免与实现细节强耦合                                                                                                                 |
  | `diagnose`                      | 硬 bug / 性能回归诊断     | `.ai/09_failure_experience/`；核心：**先构造快速、确定性的 pass/fail 反馈环**（失败测试、脚本、curl、Playwright 等优先），无可靠复现则明确列出已尝试手段并向用户索取环境/日志，**不空转假设**；若输入/快照含 **`data-traceId`（或可提取 traceId）**，须**优先**查 Loki/Grafana 重建全链路路径，再形成假设（见 `.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`），**禁止**未查日志凭文案改代码；3～5 个**可证伪**假设并排序，仪器化一次只变一个变量；闭环 *reproduce → minimise → hypothesise → instrument → fix → regression-test* |
  | `zoom-out`                      | 陌生代码抬升抽象层          | 说明相关模块、调用方与数据流，**使用项目术语**                                                                                                                                                                                                                 |
  | `improve-codebase-architecture` | 找「加深」机会（深模块、可测性）   | 与 `.ai/10_software_design_philosophy/` 一致；上游词汇可内化：**模块 / 接口（含不变量与错误语义）/ 实现 / 深度 / seam / 适配器**；**删除检验**（删掉该模块复杂度是消失还是分散到调用方）                                                                                                              |
  | `to-prd`                        | 将对话整理为 PRD / issue | 与 `docs/intents/`、任务说明文档及（若使用）GitHub Issue 结合；大改前先明确触及模块                                                                                                                                                                                  |
  | `to-issues`                     | 计划拆为可抓取工单          | **垂直切片**：每条贯穿各层、可演示或可验证；区分需人介入（HITL）与可自主完成（AFK）；与意图/测试意图拆分一致                                                                                                                                                                              |
  | `triage`                        | 工单状态机分诊            | 若团队使用 issue 跟踪：遵循类别与状态角色；上游要求 AI 评论带免责声明，可按团队规范采纳                                                                                                                                                                                         |
  | `setup-matt-pocock-skills`      | 初始化 issue 与文档布局    | **可选**；本仓库以现有 `.ai`、`docs/intents/` 为准                                                                                                                                                                                                    |
  | `caveman`                       | 极简通信控 token        | 仅用户明确要求或会话过长时；**不得**省略安全、合规与验收信息                                                                                                                                                                                                          |
  | `write-a-skill`                 | 新技能结构与渐进披露         | 新增仓库级技能时参考 `.claude/skills/` 既有 `SKILL.md` 结构，并遵循 [.ai/11_ai_development/02_agent_skills_authoring.md](./11_ai_development/02_agent_skills_authoring.md)（`description` 触发、`scripts/`/`references/` 分层、MCP 工具摘要）                                                                                                                                                                                             |
  | `git-guardrails-claude-code`    | 拦截危险 git 操作        | 精神一致：遵守本仓库 Git 与 pre-commit 规则；禁止 `--no-verify` 等绕过见上文「Git 提交规范」                                                                                                                                                                          |

- **规则（精炼执行清单）**：
  - **对齐**：重大或易偏差任务在编码前澄清未决分支与验收标准；可与 `plan-interview`、`intent-framed-agent` 组合。
  - **共享语言**：代码命名与文档使用已定义术语；重要决策落到文档或 ADR。
  - **反馈闭环**：小步交付，以类型检查、测试、本地复现为完成判据；TDD 与诊断要点见上表。
  - **架构与切片**：计划拆成垂直切片；读陌生代码先 zoom-out；定期审视模块深度，避免浅层透传模块泛滥。
  - **Token 效率**：仅在用户明确要求或会话过长时使用极简沟通；安全与合规信息不减损。
- **Cursor 对齐**：`.cursor/rules/ai-rules-loader.mdc` 中「执行原则」第 6～8 条（6～7 为 Matt Pocock 系，8 为 Heavy Thinking）与本节及下节呼应；任务类型表已索引。

#### HeavySkill（Heavy Thinking）实践要点

- **描述**：吸收 [HeavySkill](https://arxiv.org/abs/2605.02396)（*Heavy Thinking as the Inner Skill in Agentic Harness*；仓库示例路径可参考本地克隆 `HeavySkill/skill/heavyskill.md`）中的**推理放大**套路：将难题拆为 **Stage 1 并行推理**（多条彼此独立的推理轨迹）与 **Stage 2 顺序审议**（对轨迹做批判性综合，而非简单投票）。与 Matt Pocock 系的工程闭环（TDD、诊断、对齐）互补：HeavySkill 侧重**可核验的正确性推理**（数学/算法/严密逻辑），而非日常 CRUD 式编码。
- **适用场景**：竞赛级数学或 STEM、复杂逻辑推导、算法正确性极敏感、单一思路不放心且错误代价高时。**不适用**：简单事实检索、闲聊、改动路径显而易见的琐碎编辑、纯信息查阅。
- **优先级**：中（默认按需启用；与进度/成本冲突时以交付与核心约束为准）
- **Stage 1（并行推理）要点**：
  - 在同一问题上生成 **K 条彼此独立**的完整推理（Harness 内常用 **K=3～5**；切忌互相抄答案）。
  - 鼓励 **多样化解法**（如代数 vs 几何、枚举校验 vs 闭式推导），提高 Stage 2 可比对的信息量。
- **Stage 2（顺序审议）要点**：
  - 归纳**答案分布**，评估各链路的逻辑完备性与漏洞，**交叉验证**不同路径是否指向同一结论。
  - **审议不等于多数票**：共识只是信号；少数派若论证更严谨可能更可信；**亦可能全员皆错**，须准备在审议阶段独立重推。
  - 合成最终结论时保持与用户一致的**语言与格式**（如 STEM 约定、代码块等）。
- **Harness 落地（有能力时）**：若环境支持并行子 Agent / `Task` 等多路执行，可在单轮消息中派发 K 路独立推理，**由当前会话执行审议与定稿**（审议步骤不宜再外包给并行 Agent，以免丢失全局批判视角）。无并行工具时，可在上下文中**分轮显式扮演多条独立思路**再进入审议，原则不变。
- **可选迭代**：极难题可在 Stage 2 产出后将其作为补充轨迹再次审议，通常 **2～3 轮**为上限，避免无限膨胀。
- **与 Matt Pocock / 本仓库流程的关系**：复杂 bug 的根因推理可借鉴 HeavySkill；**工程验收仍以可运行证据为准**（测试、复现脚本），HeavySkill 不能替代 `diagnose` 中的确定性反馈环。

#### AI自动开发工具使用规范

- **描述**：规范AI自动开发工具的使用，提高开发效率和代码质量
- **适用场景**：所有使用AI辅助开发的场景
- **优先级**：高
- **规则**：
  - 选择合适的AI开发工具，根据任务类型和需求选择最适合的工具（如Trae、GitHub Copilot、Tabnine、Cursor、Codeium等）
  - 建立AI工具使用指南，包括工具的安装、配置和使用方法
  - 制定AI生成代码的审查流程，确保生成的代码符合项目规范
  - 建立AI工具使用的最佳实践，包括提示词编写、代码生成和代码审查
  - 定期评估AI工具的效果，及时更新工具版本和使用方法
  - 确保AI工具的使用符合公司的安全和合规要求
  - 利用多模型切换功能，根据不同场景选择最适合的AI模型
  - 实施代码安全扫描，确保AI生成的代码没有安全漏洞
  - 探索使用最新的AI开发助手，如Claude 3、GPT-4o等，提高开发效率
  - 建立AI开发工具的使用反馈机制，持续优化使用流程
  - 集成AI开发工具到CI/CD流程中，自动化代码生成和审查

#### Prompt工程规范

- **描述**：规范AI提示词的编写，提高AI生成内容的质量和准确性
- **适用场景**：所有使用AI模型进行内容生成的场景
- **优先级**：高
- **规则**：
  - 建立提示词编写指南，包括提示词的结构、格式和内容要求
  - 采用结构化的提示词格式，包括任务描述、输入示例、输出格式等
  - 提供足够的上下文信息，确保AI能够理解任务的具体要求
  - 使用明确的指令和约束，指导AI生成符合要求的内容
  - 定期更新和优化提示词，基于实际使用效果进行调整
  - 建立提示词库，存储和管理常用的提示词模板
  - 实施Skills Engineering，构建长期记忆和工作流程
  - 探索使用Agent Info和System Prompt等最新提示词技术
  - 设计分层提示词结构，包括系统级提示词、任务级提示词和上下文提示词
  - 利用少样本学习（Few-shot Learning）技术，提供高质量的示例
  - 实施提示词版本控制，跟踪提示词的演变和效果
  - 建立提示词评估机制，量化提示词的效果和质量

#### 生成式AI开发指南

- **描述**：规范生成式AI模型的开发和使用，确保模型的质量和安全性
- **适用场景**：所有使用生成式AI模型的开发场景
- **优先级**：高
- **规则**：
  - 选择合适的生成式AI模型，根据任务需求和性能要求进行选择
  - 建立模型评估标准，评估模型的生成质量、准确性和安全性
  - 实施模型输出的审核机制，确保生成内容的质量和安全性
  - 制定生成式AI的使用规范，包括内容使用范围、版权处理等
  - 定期监控模型的性能和输出质量，及时发现和解决问题
  - 探索使用最新的生成式AI模型，如Claude 3、GPT-4o、Gemini等
  - 实施生成式AI模型的安全防护措施，防止生成有害内容
  - 建立生成式AI模型的使用监控和审计机制
  - 探索使用RAG（检索增强生成）技术，提高生成内容的准确性和可靠性
  - 建立生成式AI模型的伦理审查流程，确保模型使用符合伦理规范
  - 制定生成式AI模型的内容过滤和 moderation 机制

#### AI生成代码质量保证规范

- **描述**：确保AI生成代码的质量和安全性
- **适用场景**：所有使用AI生成代码的开发场景
- **优先级**：高
- **规则**：
  - 实施严格的代码审查流程，确保AI生成代码符合项目规范
  - 对AI生成的代码进行安全性扫描，检测潜在的安全漏洞
  - 确保AI生成代码符合代码风格和质量标准
  - 对AI生成的代码进行单元测试，确保功能正确性
  - 建立AI生成代码的质量评估机制，持续改进生成质量
  - 定期培训开发人员，提高AI工具的使用效果和代码审查能力

#### 设计稿转代码规范

- **描述**：规范使用AI工具将设计稿转换为代码的流程，提高前端开发效率
- **适用场景**：前端开发中使用设计稿转代码工具的场景
- **优先级**：中
- **规则**：
  - 选择支持设计稿转代码的AI工具，如Trae的Figma设计稿转代码功能
  - 确保设计稿的质量和完整性，便于AI准确理解设计意图
  - 建立设计稿转代码的审查流程，确保生成的代码符合项目规范
  - 优化生成的代码，确保响应式布局和交互效果
  - 定期评估设计稿转代码的效果，持续改进流程

#### 多模型协作规范

- **描述**：规范使用多个AI模型协作完成开发任务的流程，提高开发效率和质量
- **适用场景**：需要使用多个AI模型协作的开发场景
- **优先级**：中
- **规则**：
  - 根据任务类型选择合适的AI模型组合，如Claude 3用于代码生成，GPT-4o用于创意内容
  - 建立模型间的协作流程，确保信息传递的准确性和一致性
  - 实施模型输出的验证机制，确保协作结果的质量和可靠性
  - 定期评估多模型协作的效果，持续优化模型组合和协作流程
  - 确保多模型协作符合公司的安全和合规要求
  - 探索使用模型编排工具，自动化多模型协作流程
  - 建立模型协作的错误处理和回退机制
  - 设计模型协作的信息传递格式，确保模型间的有效沟通
  - 实施多模型协作的监控和日志记录机制
  - 建立模型协作的性能评估指标，量化协作效果

## 规则冲突处理

- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志

> 仅记录**本索引文件**近期摘要。**业务规则**沿革（2026-04 及更早）见 [project_rules_CHANGELOG_ARCHIVE.md](./project_rules_CHANGELOG_ARCHIVE.md)；**本文件 5.92.x 元规则整理**细目见该文件 **「索引文件沿革」** 节。亦可查 Git。业务规则以正文及 `.ai` 为准。

- 2026-09-02：版本 5.139.0 - 落地 DESIGN.md 视觉基线：仓库根 `DESIGN.md` + 技能 `.claude/skills/design-md`；awesome-design-md 仅作可本地化模板；门禁 `db/scripts/ci/check_design_md.py`；No-ADR: covered by existing rule `02_design_md_integration.md`
- 2026-09-01：版本 5.138.0 - 新增「所有进程加载 conf 必须叠加 conf-local」元规则；专文 `.ai/01_project_constraints/64_conf_local_overlay_all_processes.md`；`00_project_constraints.md` 第 59 条；Cursor `.cursor/rules/conf-local-overlay-all-processes.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_conf_local_overlay.py`；禁止直读 tracked YAML / `config.local.yaml`；No-ADR: covered by existing ADR-0054
- 2026-08-31：版本 5.137.0 - 撤销 HOST_SECRETS 登记册；第 58 条改为「机密参数仅允许放在 conf-local」；专文 `.ai/01_project_constraints/63_conf_local_secrets_only.md`；ADR-0054 取代 ADR-0053；Cursor `.cursor/rules/conf-local-secrets.mdc`；门禁 `db/scripts/ci/check_conf_local_secrets.py`；`conf/` 仅非机密，密钥镜像放 `conf-local/`
- 2026-08-31：版本 5.136.0 - 新增「主机密钥必须登记在 HOST_SECRETS 册」元规则（已被 5.137.0 / ADR-0054 取代）
- 2026-08-27：版本 5.135.0 - 新增「密钥禁止硬编码在源码中」元规则；专文 `.ai/01_project_constraints/62_no_hardcoded_secrets.md`；`00_project_constraints.md` 第 57 条；ADR-0046；Cursor `.cursor/rules/no-hardcoded-secrets.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_no_hardcoded_secrets.py`；密钥须落 conf / config.local.yaml / 环境变量 / KMS，禁止源码字面量
- 2026-08-24：版本 5.134.0 - 新增「禁止重写 Git 历史清理私密信息」元规则；专文 `.ai/01_project_constraints/61_no_git_history_rewrite.md`；`00_project_constraints.md` 第 56 条；Cursor `.cursor/rules/no-git-history-rewrite.mdc`（alwaysApply）；门禁 `.githooks/pre-commit` v1.4.0 检测 `.git/filter-repo/`、`refs/original/*`、`refs/replace/*`；复查 `scripts/lib/check_git_history_rewrite.sh --scan-all`；泄露凭据须废弃 + 重新生成
- 2026-08-23：版本 5.133.0 - 新增「模拟登录必须在日志与审计中标识」元规则；专文 `.ai/01_project_constraints/60_impersonation_audit_label.md`；`00_project_constraints.md` 第 55 条；ADR-0038；Cursor `.cursor/rules/impersonation-audit-label.mdc`（alwaysApply）；开始模拟须理由 + 用户收信箱
- 2026-08-22：版本 5.132.0 - 新增「微信支付必须调用仓内 wechatpay-go SDK」元规则；专文 `.ai/01_project_constraints/59_wechatpay_go_sdk.md`；`00_project_constraints.md` 第 54 条；ADR-0032；Cursor `.cursor/rules/wechatpay-go-sdk.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_wechatpay_go_sdk.py`；依赖须 `replace` 到 `sdk/wechatpay-go`，禁止社区 SDK / APIv2 / 裸 HTTP
- 2026-08-19：版本 5.131.0 - 新增「逻辑回退须经审批」元规则；专文 `.ai/01_project_constraints/58_logic_rollback_requires_approval.md`；`00_project_constraints.md` 第 53 条；ADR-0021；Cursor `.cursor/rules/logic-rollback-requires-approval.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_logic_rollback_approval.py`；默认删除旧逻辑，保留须人类审批 + `Logic-Rollback-OK`
- 2026-08-19：版本 5.130.0 - 新增「前端按钮点击须有防重放设计」元规则；专文 `.ai/01_project_constraints/57_frontend_button_anti_replay.md`；`00_project_constraints.md` 第 52 条；ADR-0020；Cursor `.cursor/rules/frontend-button-anti-replay.mdc`（alwaysApply）；参考 `taskFE/app/src/utils/clickGuard.js`；门禁 `db/scripts/ci/check_frontend_button_anti_replay.py`；写按钮须同步锁 + Idempotency-Key
- 2026-08-18：版本 5.129.0 - 新增「GitLab 禁止自行注册、仅允许 SSO」元规则；专文 `.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`；`00_project_constraints.md` 第 50 条；ADR-0016；Cursor `.cursor/rules/gitlab-sso-only-no-self-signup.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_gitlab_sso_only_no_self_signup.py`；平台 GitLab 仅 taskAuth OIDC，禁止公开注册与账密登录
- 2026-08-18：版本 5.128.0 - 扩展「已合入 feat 分支自动清理」：ship 必须拆除 `{repo}-wt/` / 额外元仓 worktree；等价落地（内容已在 main、SHA 不同）不得留 feat；入口 `runAll/scripts/cleanup_stale_worktrees.py --apply --shipped-branch`；SessionEnd `--scan` 写入 `.runall/stale_worktrees.txt`
- 2026-08-18：版本 5.127.0 - 新增「事件消费者幂等消费」元规则；专文 `.ai/01_project_constraints/54_event_consumer_idempotency.md`；`00_project_constraints.md` 第 49 条；ADR-0015；Cursor `.cursor/rules/event-consumer-idempotency.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_event_consumer_idempotency.py`；at-least-once + 幂等消费 = effectively-once
- 2026-08-18：版本 5.126.0 - 新增「NFR 幂等性审视」元规则；专文 `.ai/01_project_constraints/53_nfr_idempotency.md`；`00_project_constraints.md` 第 48 条；Cursor `.cursor/rules/nfr-idempotency.mdc`（alwaysApply）；`/5-nfr` 硬门禁 + `references/idempotency.md`；门禁 `db/scripts/ci/check_nfr_idempotency_table.py`
- 2026-08-16：版本 5.125.0 - 新增「runAll 健康检查端口必须等于进程监听 SSOT」元规则；专文 `.ai/01_project_constraints/52_runall_health_port_ssot.md`；`00_project_constraints.md` 第 47 条；ADR-0012；Cursor `.cursor/rules/runall-health-port-ssot.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_task_events_runall_health_ports.py`；禁止从相邻服务复制 health_check URL
- 2026-08-16：版本 5.124.0 - 新增「业务服务禁止进程内轮询 / 循环」元规则；专文 `.ai/01_project_constraints/51_no_service_internal_poll_loop.md`；`00_project_constraints.md` 第 46 条；ADR-0011；Cursor `.cursor/rules/no-service-internal-poll-loop.mdc`（alwaysApply）；门禁 `db/scripts/ci/check_no_service_internal_poll_loop.py`；周期工作只允许 taskEvents timer worker 或外界 HTTP/Kafka/Webhook 触发
- 2026-08-13：版本 5.123.0 - 新增「前端无必要禁止 Teleport」元规则；专文 `.ai/01_project_constraints/50_no_unnecessary_vue_teleport.md`；`00_project_constraints.md` 第 45 条；Cursor `.cursor/rules/no-unnecessary-vue-teleport.mdc`；默认就地渲染，仅 overflow/层叠无法解决时允许并须 `Teleport-OK`
- 2026-08-12：版本 5.122.0 - 新增「禁止链接点击拦截」元规则；专文 `.ai/01_project_constraints/49_no_link_click_interception.md`；`00_project_constraints.md` 第 44 条；Cursor `.cursor/rules/no-link-click-interception.mdc`（alwaysApply）；工作面板 /me/ 403/404 改弹窗确认禁止静默回弹 profile
- 2026-08-12：版本 5.121.0 - 新增「NFR 路径分片键与可伸缩性审视」元规则；专文 `.ai/01_project_constraints/48_nfr_path_shard_id_scalability.md`；`00_project_constraints.md` 第 43 条；Cursor `.cursor/rules/nfr-path-shard-id-scalability.mdc`（alwaysApply）；`/5-nfr` 硬门禁 + `references/path-shard-id-scalability.md`
- 2026-08-12：版本 5.120.0 - 新增「人工可改配置项统一落在 conf/<area>/<app>/」元规则；专文 `.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`；`00_project_constraints.md` 第 42 条；Cursor `.cursor/rules/conf-app-config-ssot.mdc`（alwaysApply）；companion `conf/ai.md`；git-service `memLimit` 等资源键样例落地
- 2026-07-29：版本 5.119.0 - 新增「网络拓扑感知服务配置」元规则；专文 `.ai/01_project_constraints/39_network_topology_aware_config.md`；`00_project_constraints.md` 第 37 条；Cursor `.cursor/rules/network-topology-aware-config.mdc`（alwaysApply）；三层网络拓扑分类（内网基础设施/外部平台/业务网关）；禁止硬编码 127.0.0.1 为基础设施地址；`conf/base.yaml` 基础设施子域注册；存量配置迁移指引
- 2026-07-28：版本 5.118.0 - 新增「文件修改后格式验证」元规则；专文 `.ai/01_project_constraints/38_file_format_validation.md`；`00_project_constraints.md` 第 36 条；Cursor `.cursor/rules/file-format-validation.mdc`（alwaysApply）；覆盖 10 种文件类型的格式化验证与静态分析工具对照表；确立「缺工具则安装」原则；网络不可用时使用语言内建 fallback
- 2026-07-27：版本 5.117.0 - 新增「ORM 优先使用」元规则；专文 `.ai/01_project_constraints/37_orm_priority_over_raw_sql.md`；`00_project_constraints.md` 第 35 条；强制所有数据库操作优先 ORM（Django ORM / GORM），原始 SQL 仅限例外并须仓储层封装 + 参数化 + 测试覆盖
- 2026-07-27：版本 5.116.0 - 新增「Bug 修复后必须用 Playwright 验收」元规则；专文 `.ai/01_project_constraints/36_bug_fix_playwright_acceptance.md`；`00_project_constraints.md` 第 34 条；用户可感知 bug 修复后强制写 Playwright 回归测试；纯后端例外可用单元/集成测试代替；与第 31 条（迁移/重构测试先行）互补
- 2026-07-26：版本 5.115.0 - 新增「数据库 ID 字段雪花算法生成」元规则；专文 `.ai/01_project_constraints/35_snowflake_id_generation.md`；`00_project_constraints.md` 第 33 条；明确 Snowflake 为 monorepo 全服务新建表主键唯一算法，参数统一（epoch=1577836800000, 41+10+12），与 ID 字符串传输规范联合生效
- 2026-07-22：版本 5.114.0 - 新增「服务仅读本目录配置；跨服务配置须 sync」元规则；专文 `.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md`；`00_project_constraints.md` 第 30 条；Cursor `.cursor/rules/service-own-conf-directory-sync.mdc`（alwaysApply）；companion `conf/ai.md`、`shareLib/confload/ai.md`；`ai-rules-loader` / `backend-technical` / 根 `.ai.md` 交叉引用
- 2026-07-20：版本 5.113.0 - 强化「有 `data-traceId` 须先日志检索重建全路径」：短规范 `.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`；同步 brainstorming / pua / logging-audit / webapp-testing / 8-build / ai-coding-discipline；元规则 24→1.1.0；失败经验库核心规则 4；`diagnose` 与 systematic-debugging 交叉引用
- 2026-07-19：版本 5.112.0 - 全量子仓复制 task2app 随机单测 pre-commit；模板 `db/scripts/hooks/templates/`；部署 `db/scripts/deploy_repo_random_precommit.sh`；专文 28 升 1.1.0；Cursor 元规则同步
- 2026-07-19：版本 5.111.0 - 新增「提交时随机单元测试与遗留债 10% 修复」元规则；专文 `.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md`；`00_project_constraints.md` 第 6/29 条；Cursor `.cursor/rules/commit-random-unit-test-debt-fix.mdc`（alwaysApply）；门禁 `db/scripts/ci/run_commit_random_unit_tests.py` + `.pre-commit-config.yaml`；余债 `.learnings/UNIT_TEST_DEBT.md`；`ai-rules-loader` 执行原则第 14 条
- 2026-07-19：版本 5.110.0 - 强化「行数门禁触发后须自动削减目标文件」元规则；专文 `.ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md`；`00_project_constraints.md` 第 20 条；Cursor `.cursor/rules/source-file-line-limit-auto-reduce.mdc`（alwaysApply）；同步前端「大型组件行数门禁」与 `ai-rules-loader` 执行原则第 13 条
- 2026-07-18：版本 5.109.0 - 统一 Agent 交付工作流改为**权威十步 SSOT**（`03_superpowers_workflow.md` 3.0.0）；旧同号技能薄重定向；`.claude/skills/README.md`；索引与 `ai-rules-loader`「七步」表述同步为十步
- 2026-07-17：版本 5.108.0 - 新增「HTML head 须标明提供页面的服务」元规则；专文 `.ai/01_project_constraints/26_frontend_head_trae_service.md`；`00_project_constraints.md` 第 28 条；Cursor `.cursor/rules/frontend-head-trae-service.mdc`；门禁 `db/scripts/ci/check_frontend_head_trae_service.py`；`04_frontend_development` / `ai-rules-loader` / `db-ownership.yml` 交叉引用
- 2026-07-17：版本 5.107.0 - 新增「会话结束优化建议 Todo」元规则（收尾写入编号清单、完成后标记）；专文 `.ai/01_project_constraints/25_session_end_optimization_todo.md`；清单 SSOT `.learnings/OPTIMIZATION_TODOS.md`；`00_project_constraints.md` 第 27 条；Cursor `.cursor/rules/session-optimization-todo.mdc`（alwaysApply）；`ai-rules-loader` 执行原则第 12 条；`00_start` / goal-mode Final Overview 交叉引用
- 2026-07-17：版本 5.106.0 - 新增「请求报错展示须带 data-traceId」元规则；专文 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md`；`00_project_constraints.md` 第 26 条；Cursor `.cursor/rules/frontend-error-data-trace-id.mdc`；`frontend.mdc` / `04_frontend_development` / `ai-rules-loader` / 可观测性 §1.2 交叉引用
- 2026-07-16：版本 5.105.0 - 新增「应用启动禁止使用环境变量 Proxy」元规则（业务进程不继承 HTTP(S)_PROXY；env proxy 仅开发加速）；专文 `.ai/01_project_constraints/23_app_startup_no_env_proxy.md`；`00_project_constraints.md` 第 25 条；根 `.ai.md` 1.2.0；Cursor `.cursor/rules/app-startup-no-env-proxy.mdc`（alwaysApply）；失败经验 `24_provider_registry_proxyconnect_1234.md`；`ai-rules-loader` / `backend-technical` / `conf/runAll.yaml.ai.md` 交叉引用
- 2026-07-15：版本 5.104.0 - 新增「服务监听禁止仅绑 127.0.0.1」元规则（全服务须 `0.0.0.0`）；专文 `.ai/01_project_constraints/22_service_listen_host_not_localhost_only.md`；`00_project_constraints.md` 第 24 条；根 `.ai.md` 1.1.0；Cursor `.cursor/rules/service-listen-host.mdc`（alwaysApply）；失败经验 `13_apisix_host_docker_internal_localhost_bind.md`；`ai-rules-loader` / `backend-technical` / `conf/runAll.yaml.ai.md` 交叉引用
- 2026-07-15：版本 5.103.0 - 强制「业务意图出现 → 向消息队列投递对应业务事件」；同步 `.ai/08` 意图驱动、`09_domain_driven_design`、Kafka 最佳实践，以及 brainstorming/DDD/plans/build/review/ship/ai-coding-discipline 等技能硬门禁
- 2026-07-14：版本 5.102.0 - 新增「已合入 feat 分支自动清理」元规则；专文 `.ai/01_project_constraints/21_merged_feat_branch_cleanup.md`；`00_project_constraints.md` 第 23 条；脚本 `runAll/scripts/delete_merged_feat_branches.py`；Cursor `.cursor/rules/merged-feat-branch-cleanup.mdc`；合入推送 main 后 Agent 必须自动删本地/远端 `feat/*`
- 2026-07-13：版本 5.101.1 - **技能 SSOT 归一**：仓库根 `.claude/skills/` 为唯一技能目录；`task2app/.agents` 与 `task2app/.claude` 改为指向根 `.claude` 的符号链接；修复原指向不存在 `.agents`/`.trae` 的坏链；历史文档 `task2app/.ai/` 字面路径改为 `.ai/`
- 2026-07-13：版本 5.101.0 - **规则 SSOT 归一**：仓库根 `.ai/` 为唯一细则目录；`task2app/.ai` 改为指向根 `.ai` 的符号链接；本索引迁入 `.ai/project_rules.md`；失败案例 `02_runtime_errors` 编号去重规整
- 2026-07-13：版本 5.100.1 - 落地「接口→Go 服务」机器可读对照 `db/api_route_ownership.yaml`、人读 `docs/architecture/api-route-to-owner.md`、CI `db/scripts/ci/check_django_new_api_routes.py`（经 runAll DDD wrapper）；明确存量 Django 公网 API 迁出节奏仍以 `2026-07-05-task2app-api-go-split-brainstorm-design.md` 为准；`20_go_service_first_apis` 升至 1.1.0
- 2026-07-13：版本 5.100.0 - 新增「新增服务 / 接口优先落 Go」元规则；专文 `.ai/01_project_constraints/20_go_service_first_apis.md`；`00_project_constraints.md` 第 22 条；Cursor `.cursor/rules/go-service-first-apis.mdc`；落点顺序为扩展现有 Go → 新建 Go → Python 可审计例外；`project_rules` / `ai-rules-loader` / `backend-technical.mdc` / brainstorming 门禁交叉引用同步
- 2026-07-14：版本 5.99.2 - known-debt 清零（跨服务直连改 HTTP）；Go SSOT 库写入 `db/registry.yaml`；独立 GHA `.github/workflows/db-ownership.yml`（ownership 扫描 + smoke）
- 2026-07-10：版本 5.99.1 - 落地「表→owner」对照表 `db/table_ownership.yaml`、人读 `docs/architecture/table-to-owner.md`、CI `db/scripts/ci/check_single_service_db_ownership.py`（经 runAll DDD wrapper）；存量跨服务直连登记为 known-debt 警告
- 2026-07-10：版本 5.99.0 - 新增「单库 / 单表单服务数据所有权」元规则；专文 `.ai/01_project_constraints/19_single_service_data_ownership.md`；`00_project_constraints.md` 第 21 条；跨服务须服务转发或迁表；`project_rules` 项目规范 / DDD 索引交叉引用；`ai-rules-loader`、`backend-technical.mdc`、`09_domain_driven_design.md` 同步
- 2026-06-01：版本 5.98.0 - 新增「ID 字段字符串传输」元规则；专文 `.ai/03_technical_implementation/11_id_field_string_transit.md`；明确 ID 仅在入库转 DB 类型，进程内与进程间一律 string；`02_api_specifications`、`CLAUDE.md`、`backend-technical.mdc`、`ai-rules-loader` 同步
- 2026-05-17：版本 5.97.0 - 新增「Python 辅助 Web 服务并发与锁」元规则；专文 `.ai/03_technical_implementation/10_python_sidecar_web_concurrency.md`；归纳 relayToTrae 锁重入死锁（`/v1/stop`、`register`→`/v1/start`）；`ai-rules-loader`、`backend-technical.mdc` 同步
- 2026-05-17：版本 5.96.0 - 新增「静态资源引用须带内容 Hash 查询参数」元规则；专文 `.ai/01_project_constraints/18_static_resource_cache_bust_query.md`；`00_project_constraints.md` 第 19 条；`project_rules` 项目规范 / 前端索引交叉引用；`ai-rules-loader`、`frontend.mdc`、`.ai/04` 同步
- 2026-05-16：版本 5.95.0 - 新增「异域部署：前端与 API 分域名」元规则；专文 `.ai/01_project_constraints/17_cross_domain_frontend_api.md`；`00_project_constraints.md` 第 18 条；`project_rules` 项目规范 / 技术实现 / 前端索引交叉引用；`.ai/04`、`.ai/03` API 专文指针；`ai-rules-loader` 项目规范行增补
- 2026-05-14：版本 5.94.0 - 新增「预期问题前置处理规则」：当可预见问题时必须前置处理，禁止等待问题出现后再被动修复；同步 `ai-rules-loader` 执行原则
- 2026-05-13：版本 5.93.9 - 新增「GitOAuth 接口 Swagger 管理界面规则」：要求 GitOAuth 接口在新增/变更/下线时同步维护 Swagger/OpenAPI 管理界面；同步新增 `.cursor/rules/gitoauth-swagger.mdc`
- 2026-05-11：版本 5.93.8 - **统一交付单轨**：合并原 Superpowers / gstack 两节为「统一 Agent 交付工作流」；`03_superpowers_workflow.md` 2.0.0（默认自动执行、七步表合并 gstack）；`04` 重定向
- 2026-05-11：版本 5.93.7 - 新增「gstack Sprint 与本国工作流程（可选对标）」节；专文 `.ai/11_ai_development/04_gstack_sprint_alignment.md`；`03_superpowers_workflow.md` 1.1.1 增加 gstack 命名指针
- 2026-05-11：版本 5.93.6 - Superpowers 专文增至 1.1.0：索引节补充「调试与完成门槛」指针（*systematic-debugging* / *verification-before-completion* ↔ `.ai/09`、`diagnose`、测试/CI）
- 2026-05-11：版本 5.93.5 - 新增「Superpowers 式 Agent 交付工作流（吸收）」节并指向 `.ai/11_ai_development/03_superpowers_workflow.md` 与 `docs/workflows/superpowers/`（来源 [obra/superpowers](https://github.com/obra/superpowers)）；`ai-rules-loader` 任务表增补 `03_superpowers_workflow.md`
- 2026-05-11：版本 5.93.4 - 「Matt Pocock Skills 实践吸收」表 `write-a-skill` 一行链接至 `.ai/11_ai_development/02_agent_skills_authoring.md`（吸收公开 Agent Skills 编写与 MCP 工具要点）
- 2026-05-11：版本 5.93.3 - 前端大型组件行数门禁阈值自 **1000 行**收紧为 **500 行**（触及文件须自动拆分）；同步 `.ai/04_frontend_development/00_frontend_development.md`、`scripts/ci/check_frontend_component_line_limit.py` 默认上限
- 2026-05-10：版本 5.93.2 - 前端行数门禁脚本：diff 模式对遗留超大文件按基线对比净增长判定；workflow 注入 `TASK2APP_FRONTEND_LEGACY_GROWTH_GRACE`；索引与 `.ai` 同步
- 2026-05-10：版本 5.93.1 - 前端规范：单组件文件 **>1000 行**须在触及文件时自动拆分；索引「一级分类：前端开发规范」新增条目；`.ai/04_frontend_development/00_frontend_development.md` 增至 2.7.0；接入 `scripts/ci/check_frontend_component_line_limit.py`、`pre-commit` 暂存校验与 `.github/workflows/ddd-bdd-compliance.yml` 步骤
- 2026-05-08：版本 5.93.0 - DDD/BDD 升为**强制级**：新增 `.github/workflows/ddd-bdd-compliance.yml` 与 `scripts/ci/check_ddd_bdd_compliance.py`（领域层静态导入门禁 +「业务与测例同批变更」近似门禁）；`scripts/hooks/pre-commit` 增加 `--staged` 校验；同步 `.ai` 专文
- 2026-05-07：版本 5.92.4 - 元规则整理：5.92.0～5.92.3 沿革迁入归档「索引文件沿革」；blockquote 区分业务沿革与索引沿革；合并维护流程；去除文末重复指针
- 2026-05-07：版本 5.91.0 - 吸收 HeavySkill（Heavy Thinking）两阶段推理要点至「一级分类：AI辅助软件开发规范」；同步 `.cursor/rules/ai-rules-loader.mdc` 任务表与执行原则第 8 条
- 2026-05-06：版本 5.89.0 - 扩展「Matt Pocock Skills 实践吸收」：对照上游工程类技能（README 四失败模式、`diagnose`/`tdd`/`grill-with-docs`/`zoom-out`/`improve-codebase-architecture`/`to-issues`/`triage` 等）增补技能对照表与可执行要点；同步 `.cursor/rules/ai-rules-loader.mdc` 执行原则第 7 条
- 2026-05-06：版本 5.88.0 - 吸收 [mattpocock/skills](https://github.com/mattpocock/skills) 工程实践至「一级分类：AI辅助软件开发规范」下「Matt Pocock Skills 实践吸收」；同步更新 `.cursor/rules/ai-rules-loader.mdc` 任务表与执行原则第 6 条

---

**注**：使用提示词时按任务从 `.ai` 按需加载。全局变更须同步触及相关 `.ai` 专文与本索引目录，避免索引与仓库脱节。