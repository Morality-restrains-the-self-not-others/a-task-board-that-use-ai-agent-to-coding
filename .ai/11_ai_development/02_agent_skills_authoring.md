# Agent Skills 编写要点（仓库内技能）

## 基本信息

- 版本：1.1.2
- 创建日期：2026-05-11
- 最后修改：2026-07-20
- 维护者：与 `00_ai_development.md` 同步维护

## 定位

本文件补充 **`.claude/skills/<name>/SKILL.md`** 的写法约定，与 [pskoett/pskoett-ai-skills](https://github.com/pskoett/pskoett-ai-skills) 及 `.ai/project_rules.md` 中「Matt Pocock Skills 实践吸收」表的 `write-a-skill` 一行配合使用。

内容吸收自公开实践（含 [anthropics/skills](https://github.com/anthropics/skills) 示例中的 **skill-creator**、**mcp-builder** 思路，[vercel-labs/skills](https://github.com/vercel-labs/skills) 的 **Skills CLI** 与 **find-skills** 甄别流程）与 [Agent Skills 规范索引](https://agentskills.io/specification)，经本项目裁剪；**不要求**安装任何上游插件或复制示例仓库到本仓库；执行约束仍以 `.ai/`、`.ai/project_rules.md`、`.claude/skills/` 为准。

## 元数据与触发

- **frontmatter**：至少包含 `name`（小写字母/数字/连字符 `[a-z0-9-]`，且与父目录名一致）与 `description`；与跨工具互操作时对齐 [agentskills.io](https://agentskills.io/specification) 的元字段约定。**禁止**中文、空格、下划线写入 `name`/目录名（Cursor 会跳过不合规技能）；中文说明放标题或正文。
- **`description` 是主要触发信号**：须同时写清「做什么」与「何时用」，并列出**典型用户措辞或场景**（多写几条同义词/上下游任务名），减轻模型 **under-trigger**（该用技能却未加载）的概率。
- **正文**：用祈使句写可执行步骤；与平台设计、安全、Git 等**强约束**仍以 `.ai/`、`docs/intents/` 为准，技能内不得弱化或绕过。

## 渐进披露（控制上下文体积）

1. **元数据**：`name` + `description` — 常驻或低成本加载。
2. **`SKILL.md` 正文**：触发后加载；保持**短而可执行**，超长时拆层。
3. **捆绑资源**（按需再读或黑盒执行）：
   - `scripts/`：可重复、确定性步骤（优先 **命令行 + `--help`**，由执行方调用而非整文件读入上下文）。
   - `references/`：长说明、分环境/分栈附录；正文内写明「何种情况下读哪一份」。
   - `assets/`：模板、静态资源等产出用文件。

**体量建议**：单份 `SKILL.md` 正文以可在一屏策略内扫完为宜；若逼近数百行，应拆出 `references/` 并在正文保留目录式索引。

## 可选：可验证技能的迭代

对输出可客观判定对错的技能（固定步骤、文件变换、代码生成契约），可维护少量 **golden prompts** 或脚本化检查；迭代时根据失败样例改 `description`（触发）或正文（步骤）。主观类（文风、审美）不必强行量化。

## MCP 类技能 / 工具设计（摘录）

编写或评审「通过 MCP 暴露能力」时，宜满足：

- **工具命名**：一致前缀 + 动作语义，便于发现与组合（如 `github_list_issues`）。
- **返回体**：默认可分页或过滤；默认返回小而全链路可继续的字段，避免单次塞满上下文。
- **错误信息**：包含可操作的下一步（缺哪个参数、权限不足时如何补救），而非仅状态码。
- **输入 schema**：字段含义与约束写清（便于生成合法调用）。

## 外部技能包与 Skills CLI（可选）

[vercel-labs/skills](https://github.com/vercel-labs/skills) 仓库实现的是 **开放 Agent Skills 生态的包管理器**（`npx skills add` / `find` / `list` / `update` / `init` 等），面向多 Agent（含 Cursor、Claude Code 等）的安装路径与 symlink/copy 策略；**技能正文本体**多在其它仓库（例如 README 中常引用的 `vercel-labs/agent-skills`，或与 [skills.sh](https://skills.sh/) 上收录的包）。

与本项目的关系建议如下：

- **默认**：能力以仓库内 **`.claude/skills/`** 与 `.ai` 专文为准（与 `project_rules.md`、Matt Pocock 实践吸收等一致）；不必为日常开发强依赖 `npx skills`。
- **需要引入上游技能包时**：可将 CLI 当作**发现与安装**工具；安装前宜做快速甄别（与上游 `find-skills` 技能一致的精神）：
  - 来源可信度（官方/知名组织优先）、维护活跃度；
  - 许可证与是否允许纳入版本库或仅个人 `-g` 安装；
  - 与现有 **Vue / Django / Playwright / 支付合规** 等栈是否匹配，避免与 `.ai/04_frontend_development/` 等平台设计规范冲突。
- **发现入口**：`npx skills find <关键词>`、[skills.sh](https://skills.sh/) 榜单；新技能骨架可用 `npx skills init <name>` 再按本文档迁入 `.claude/skills/` 并做项目化改写。

## 输出格式强制要求

### 选项总结清单

任何 skill 在给出建议、方案、设计、审查意见时，若涉及 **多项 × 多方向可选方案**（例如同时讨论 A/B/C 多个模块/问题各自有多种策略），在讨论完毕后**必须**在最后附上总结清单，格式：

```
总结清单：
- 项A: 方案1（简述）、方案2（简述）、方案3（简述）
- 项B: 方案1（简述）、方案2（简述）
- 项C: 方案1（简述）、方案2（简述）、方案3（简述）
```

此项要求适用于所有 skill（含 brainstorm、review、plan、design-guidelines 等），不可跳过。

## 排障与 data-traceId（编写排障类技能时）

凡技能正文涉及 **debug / diagnose / 运行时报错 / 前端错误 UI**：

- 须声明或交叉引用：**有 `data-traceId`（或可提取 traceId）时优先日志检索、重建全路径，再源码猜测**
- 权威短规范：`.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`
- 完整 Loki 步骤：同目录 `SKILL.md`「TraceId 驱动的 Grafana 日志分析」
- 勿在新技能中复制整段 LogQL；用链接引用，避免漂移

## 规则冲突处理

与 `.ai/01_project_constraints/`、`00_project_constraints.md` 等核心约束冲突时，以核心约束为准。

## 变更日志

- 2026-07-20：版本 1.1.2 - 增补「排障与 data-traceId」：新/改排障类技能须引用日志优先短规范
- 2026-07-17：版本 1.1.1 - 明确 Cursor `name`/目录名仅 `[a-z0-9-]` 且须一致（禁止中文），避免技能无法被发现
- 2026-05-11：版本 1.1.0 - 增补「外部技能包与 Skills CLI（可选）」：对齐 [vercel-labs/skills](https://github.com/vercel-labs/skills) 定位及 `find-skills` 类甄别要点
- 2026-05-11：版本 1.0.0 - 初版，吸收公开 Agent Skills 编写与 MCP 工具设计要点并项目化
