# 禁止重写 Git 历史清理私密信息；泄露凭据须废弃并重新生成（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-24
- 维护者：Trae AI 团队
- 适用范围：meta 仓库及全部子仓的 Git 历史与协作流程；gitService 平台 GitLab 及全部区域实例
- 架构决策：无（流程 / 安全治理规则，非架构决策，不写 ADR）
- 约束索引：第 56 条

## 核心原则

**禁止**使用 `git filter-repo`、`git filter-branch`、`git replace`、`git rebase --onto` 等重写历史的手段，或 force-push 覆盖共享 ref，来「清除」已推送 / 已共享历史中的私密信息（密钥、密码、Token、Cookie、证书、连接串等）。

原因（为什么重写历史不是正确的处置）：

1. **历史一旦推送即不可删除**：git filter-repo 只能改写本地对象，但已分发的副本——各协作者 clone、CI 缓存、gitService 服务端对象、备份、issue/PR 引用——**全部继续持有旧内容**。清历史无法达成「保密」目的。
2. **重写破坏全部基线**：历史重写使所有协作者、CI、部署记录与 `git log` 检索（含 auto-commit 检查点、`git log --grep="auto-commit"` 恢复安全网）全部失配，成本远大于收益。
3. **泄露的凭据在重写后依然有效**：凭据是否作废取决于服务端吊销，与 git 历史无关。清历史不等于止血。

## 硬约束（一级，禁止忽略）

**私密信息一旦进入任何已被推送 / 已离开本机的历史，一律视为已泄露**，必须走下方「标准处置流程」——**废弃泄露凭据 + 重新生成新凭据**。禁止以「清历史」代替凭据轮换。

### 禁止

- `git filter-repo` / `git filter-branch` / `git replace` 重写已推送历史
- force-push / `--force-with-lease` 之外的强制覆盖共享 ref 来掩盖历史内容
- `git rebase --onto`、交互式 rebase 删除已推送提交以「抹掉」敏感内容
- 删除远端分支 + 重建同名分支来绕开历史（旧对象仍留存于服务端直到 GC，且无法控制各克隆副本）
- Agent 收到「清历史 / 用 filter-repo 删掉提交中的密钥」类请求时**不得执行**，须说明本项目元规则并引导走标准处置流程

### 允许

- **本地未推送提交**（历史从未离开本机、从未 push）的 rebase / squash / amend 整理——这不影响任何共享基线
- `git filter-repo --analyze` 等**只读**分析（不写引用、不重写对象）
- 删除整个分支 / 整个仓库（删除而非重写，且须确认无未归档副本）
- 凭据轮换后对**新历史**的常规维护

## 私密信息泄露标准处置流程

发现私密信息已进入历史（或任何渠道泄露）时，按序执行：

1. **立即废弃（止血）**：在全部相关系统吊销 / 失效泄露凭据——taskAuth 账号、SSO / OIDC client secret、数据库账号、Redis / Kafka 口令、registry 登录、外部服务 API key、微信支付 APIv3 密钥（证书序列号 + APIv3 密钥须在商户平台重签）。不可轮换的泄露（如个人隐私数据）按第 4 步评估。
2. **生成新凭据并部署替换**：生成全新密钥 / 密码 / Token，按既有配置链路（`conf/<area>/<app>/config.yaml` SSOT → 对应服务）部署生效；密钥修改先备份（`backup-keys.sh` 约定）。
3. **落库与审计**：更新受影响的密钥存储 / secret 引用；记录泄露时间、范围与处置动作。
4. **不可轮换数据的评估**：若泄露内容为无法轮换的个人隐私数据，评估可见性（仓库是否公开 / 克隆面有多大），必要时将仓库迁移到私有化存储并重建（新仓库重新初始化，**不**携带旧历史），而非重写旧历史。
5. **防再犯**：补齐 `gitignore` / 密钥文件排除规则、引入 pre-commit secret 扫描（CI 与本地）、对涉事提交做事后分析并更新 `.learnings/` 经验库。

## 强制机制（pre-commit 兜底检测）

`git filter-repo` 运行后必然在 git 公共目录留下 `.git/filter-repo/` 标记；`git filter-branch` 留下 `refs/original/*` 备份 ref；`git replace` 留下 `refs/replace/*` ref。`.githooks/pre-commit`（v1.4.0+）检测到上述任一标记即**阻断提交**，提示按本规则处置：

- 处置正确路径：凭据废弃 + 重新生成（无需清理标记，历史保持原样即可正常提交——标记仅提示曾经发生过重写）
- 唯一允许清理标记的情形：历史重写确已发生（违反本规则或本规则生效前遗留），且**全员已接受当前基线**，由人工确认后清理标记再重试提交（清理命令见 pre-commit 输出）

## 触发

- 任何「清理 git 历史」「删除提交中的密钥」「filter-repo / filter-branch」相关请求
- 密钥、Token、Cookie、证书、连接串泄露事件响应与安全审计
- pre-commit 报出 `.git/filter-repo/`、`refs/original/*`、`refs/replace/*` 标记
- 新增 CI secret 扫描、密钥轮换脚本、敏感文件 gitignore 规则

## 验收标准

1. 新建临时仓库，运行 `git filter-repo` 后执行 pre-commit → 提交被阻断并提示「硬约束 61」。
2. 清理 `.git/filter-repo/` 与 `refs/original/*` 后，同内容提交可通过。
3. 本仓库（meta 与全部子仓）当前无 `.git/filter-repo/`、`refs/original/*`、`refs/replace/*` 遗留标记（`bash scripts/lib/check_git_history_rewrite.sh --scan` 可复查，见下）。
4. Agent 收到 filter-repo 类请求时拒绝执行并说明正确处置，无违规提交产生。

## 实现位置

| 组件 | 路径 |
|------|------|
| 约束专文 | `.ai/01_project_constraints/61_no_git_history_rewrite.md` |
| Cursor 元规则 | `.cursor/rules/no-git-history-rewrite.mdc` |
| 门禁 | `.githooks/pre-commit`（v1.4.0「git 历史重写禁止检测」段） |
| 复查脚本 | `scripts/lib/check_git_history_rewrite.sh` |
| 根摘要 | `CLAUDE.md`「硬约束 61」、`.ai.md`、`.ai/01_project_constraints/00_project_constraints.md` 第 56 条 |

## 关联

- 规则 6（Git 提交规范，禁止 `--no-verify`）：本规则门禁同样不可用 `--no-verify` 绕过
- 规则 32（子仓库优先提交）：历史重写会破坏全部子仓指针基线，加重本规则后果
- `gitService` 平台 GitLab：本规则适用于其承载的全部仓库；服务端对象一旦入库无法以 filter-repo 清除
- 恢复安全网（`git log --grep="auto-commit"`）：重写历史会破坏跨会话恢复能力，与 auto-commit 体系冲突
