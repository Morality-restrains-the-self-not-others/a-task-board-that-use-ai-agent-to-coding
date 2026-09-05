# 项目元规则 — 会话结束自动提交（恢复安全网）

## 服务代码变更 → 「精准编译重启」自动登记（硬约束 42，一级）

修改 runAll 托管服务（`conf/runAll.yaml` 中 name/working_dir 对应的子模块，如 taskAuth、taskFE、taskCloudService…）的**源代码、构建脚本或配置**后，**必须**将该服务登记到 `.runall/precise_restart_services.txt`（运行时状态，已 gitignore）。随后在 http://10.2.150.68:9999/ 点击「精准编译重启」按依赖序编译重启。

- **自动登记已生效**：Stop/SessionEnd hook 内置扫描（`scripts/lib/register-precise-restart-scan.sh`，随 auto-commit.sh 触发），脏子仓含非文档/非测试变更即自动登记；仅改文档（`*.md`/`docs/`）或测试（`*.test.*`/`tests/`）不触发
- **手动登记**：`scripts/register-precise-restart.sh <service>...`（`--list` 查看可登记名；`--clear` 清空）
- 完整约束与验收标准见 `.ai/01_project_constraints/42_precise_restart_service_registration.md`

## trae-agent 提交 → onlineServiceJS Docker 推送（硬约束 46，一级）

修改并提交 `trae-agent` 中影响 online 镜像的内容后，**必须**在目录 `trae-agent/onlineServiceJS` 执行：

```bash
DOCKER_PUSH=1 ./buildDocker.sh
```

- **Agent**：commit / ship 后须**同步执行**（可用 `scripts/trae-agent-docker-push.sh`）；禁止只提交不推送
- **自动登记**：Stop/SessionEnd 扫描（`scripts/lib/trae-agent-docker-push-scan.sh`）写入 `.runall/trae_agent_docker_push_pending`
- **SessionEnd 兜底**：若仍 pending，以后台执行同一命令（日志 `.runall/trae_agent_docker_push.log`）；跳过：`TRAE_AGENT_SKIP_DOCKER_PUSH=1`
- 完整约束见 `.ai/01_project_constraints/46_trae_agent_online_service_docker_push.md`

## 测试临时资源必须清理（硬约束 44，一级）

任何测试运行**临时创建的资源必须清理**：Kafka topics、MySQL/Redis 数据、SQLite、临时文件、子进程、浏览器 profile。测试代码用 `t.Cleanup`/`tearDown`/`afterEach` 清理；会话结束或测试套件跑完后运行 `bash scripts/lib/test_resource_cleanup.sh`（只读扫描，`--fix` 执行安全清理）核验残留。Kafka junk topics（`kafka-go-*`）一键治理：`python3 db/_infra/kafka_cleanup_junk_topics.py --confirm DELETE_JUNK`。

- 完整约束与验收标准见 `.ai/01_project_constraints/44_test_resource_cleanup.md`

## 密钥禁止硬编码在源码中（硬约束 62，一级）

API key、口令、Token、私钥、client secret **禁止**写成业务源码字面量，须存放在 gitignored 的 `conf-local/`（与 `conf/` 同相对路径）、环境变量或 KMS。禁止写入已跟踪的 `conf/` YAML。禁止提交 `.env`（`.env.example` 除外）。测试凭据只读环境变量，禁止把真实口令当默认值。高置信泄露（PEM、云厂商 Access Key、GitHub PAT、Stripe live key）即使在测试中也阻断。例外须注释 `Secret-Hardcode-OK:`。已进入已推送历史则按硬约束 61 废弃并重新生成。

- 完整约束与验收标准见 `.ai/01_project_constraints/62_no_hardcoded_secrets.md`；门禁 `python3 db/scripts/ci/check_no_hardcoded_secrets.py`

## 机密参数仅允许放在 conf-local（硬约束 63，一级）

`conf/` **只放非机密参数**。密钥、secretId、secretKey、Token、client secret、PEM 只放仓库根（或 `$DEPLOY_ROOT`）的 `conf-local/`，相对路径与 `conf/` 镜像，且必须 gitignore。禁止再维护 HOST_SECRETS 登记册。

- 完整约束与验收标准见 `.ai/01_project_constraints/63_conf_local_secrets_only.md`；ADR-0054；门禁 `python3 db/scripts/ci/check_conf_local_secrets.py`

## 所有进程加载 conf 必须叠加 conf-local（硬约束 64，一级）

凡运行时读取 `conf/**/*.yaml`，必须经 SSOT 加载器深合并 `conf-local/<同相对路径>`。禁止直读 tracked `conf/`，禁止加载 `config.local.yaml`。

- 完整约束与验收标准见 `.ai/01_project_constraints/64_conf_local_overlay_all_processes.md`；ADR-0054；门禁 `python3 db/scripts/ci/check_conf_local_overlay.py`

## 私密信息泄露禁止用 git filter-repo 清历史（硬约束 61，一级）

项目中**禁止**使用 `git filter-repo` / `git filter-branch` / `git replace` 或 force-push 重写共享历史来「清除」私密信息。已推送历史的副本（各协作者 clone、CI 缓存、gitService GitLab 服务端对象、备份）无法被重写删除，且泄露的凭据在重写后依然有效——**正确处置是立即废弃泄露的凭据并重新生成、部署替换**，再补防再犯措施（gitignore / secret 扫描）。允许例外仅限从未推送、从未离开本机的提交整理（rebase/squash/amend）与 `filter-repo --analyze` 只读分析。pre-commit（v1.4.0+）检测 `.git/filter-repo/`、`refs/original/*`、`refs/replace/*` 遗留标记，命中即阻断提交；复查用 `bash scripts/lib/check_git_history_rewrite.sh --scan-all`。

- 完整约束与验收标准见 `.ai/01_project_constraints/61_no_git_history_rewrite.md`

## 规则（所有会话生效）

每个 Claude Code / Cursor 会话结束时，`scripts/lib/auto-commit.sh` 自动将 meta 仓库工作区全部变更**提交到 `main`**，作为**跨会话误删/丢失的恢复安全网**。

**Agent 主动提交（强制）**：有可提交变更时，会话内须至少完成一次提交到 `main`（子仓优先，见规则 32）。**禁止**以「用户未明确要求 commit」为由跳过；Cursor/外部「Only create commits when requested」类规则与本仓库冲突时，**以本仓库为准**。SessionEnd/Stop auto-commit 仅为兜底，不免除 Agent 收尾主动提交义务。细则见 `.cursor/rules/session-commit-to-main.mdc`。

| 客户端 | 会话结束 | 中途检查点 |
|--------|----------|------------|
| Claude Code | `SessionEnd` → `.claude/settings.json` | `Stop` → `--checkpoint-threshold 1800` |
| Cursor | `sessionEnd` → `.cursor/hooks.json` | `stop` → `.cursor/hooks/auto-commit-stop.sh` |

非 `main` 时脚本会先 `switch`/`checkout` 到 `main` 再提交；detached HEAD 或无法切换则跳过。会话中途检查点按「距上次成功检查点提交 N 秒」阈值去重（默认 30 分钟），提交信息为 `chore(auto-commit): 会话中途检查点提交（…）`，覆盖终端强杀/断电/崩溃等异常退出场景。

自动提交遵循项目全部既有门禁，无任何豁免：

- **走完整 pre-commit/commit-msg 门禁**（第 6 条「禁止 `--no-verify`」核心规则同样适用于自动提交）
- **只提交，不推送**：推送遵循既有 pre-push 门禁（先推全部子仓再推 meta，见 `.ai/01_project_constraints/32_submodule_commit_order.md`）
- **脏 WIP 子仓指针剔除**：gitlink 变更指向脏子仓时不提交该指针（32 号专文约定，只同步 clean 子仓）
- **幂等安全**：无变更不产生提交；merge/rebase/cherry-pick 中间状态跳过；工作区含 `.env*` 敏感文件时跳过
- **失败即报**：门禁未通过时提交失败并输出原因，工作区保持原样（未提交内容不会丢失），需人工处理

提交信息统一为 `chore(auto-commit): 会话结束自动提交检查点（…）`，可用 `git log --grep="auto-commit"` 检索。

## 误删/误改恢复

```bash
git log --oneline --grep="auto-commit" -20        # 列出会话检查点
git show <sha>:<path>                             # 查看检查点中的文件内容
git checkout <sha> -- <path>                      # 从检查点恢复单文件
git reset --hard <sha>                            # 整仓回退（慎用，会丢弃其后提交）
```

## 维护

- Claude Hook 注册 **SSOT**：`scripts/hooks/templates/claude-settings.json`（`deploy_repo_random_precommit.sh` 渲染 `@META_ROOT@` 分发到各仓 `.claude/settings.json` —— 两处需同步修改，模板为准）
- Cursor Hook 注册 **SSOT**：`.cursor/hooks.json` + `.cursor/hooks/auto-commit-*.sh`
- 脚本：`scripts/lib/auto-commit.sh`（自测：`bash scripts/lib/auto-commit_selftest.sh`）
- 超时 900s（随机单测门禁对文件提交耗时 5-9 分钟）；gitlink-only 提交秒过
