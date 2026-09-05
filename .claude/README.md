# `.claude/` — Agent 技能唯一目录（SSOT）

## 权威位置

- **唯一技能正文**：仓库根 **`.claude/skills/`**
- **流水线编号 SSOT**：[`.ai/11_ai_development/03_superpowers_workflow.md`](../.ai/11_ai_development/03_superpowers_workflow.md)（权威**十步**）
- **技能目录表**：[`.claude/skills/README.md`](skills/README.md)
- **兼容入口（符号链接，禁止再维护实体副本）**：
  - `task2app/.claude` → `../.claude`
  - `task2app/.agents` → `../.claude`（历史路径 `.agents/skills/...` 仍可用，实际解析到本目录 `skills/`）

## 与规则目录的关系

| 目录 | 职责 |
|------|------|
| `.ai/` | 项目细则与 `project_rules.md` 总索引 |
| `.claude/skills/` | 可执行 Agent Skills（`SKILL.md`） |
| `.cursor/rules/` | Cursor 会话元规则（薄封装） |

## 维护约定

1. 新增/修改技能只改 **本目录** `skills/<name>/`。
2. 自检：`test -L task2app/.claude && test -L task2app/.agents && readlink task2app/.claude`。
3. 短别名（如 `1-brainstorming`）与完整名（如 `1-brainstorming-design-docs`）可并存；别名 `SKILL.md` 应指向完整技能。
4. **编号一对一**：步号 N 仅对应权威目录（见 skills README）。旧同号目录必须是薄重定向，禁止再写独立「Step N」流程。
5. **Cursor 技能名约束**（不合规会被跳过）：目录名与 frontmatter `name` 须一致，且仅允许 `[a-z0-9-]`（禁止中文/空格/下划线）；中文说明放标题或正文。
