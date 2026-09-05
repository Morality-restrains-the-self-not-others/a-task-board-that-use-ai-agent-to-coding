# `.ai/` — 仓库规则唯一细则目录（SSOT）

## 权威位置

- **唯一正文目录**：仓库根目录 **`.ai/`**（本目录）。
- **`task2app/.ai`**：必须是指向本目录的**符号链接**（`task2app/.ai` → `../.ai`），禁止再维护第二份实体副本。
- **金字塔总索引**：[`project_rules.md`](./project_rules.md)（版本沿革摘要见文末；详细归档见 [`project_rules_CHANGELOG_ARCHIVE.md`](./project_rules_CHANGELOG_ARCHIVE.md)）。

## 与 Cursor / Trae 元规则的关系

| 层级 | 路径 | 职责 |
|------|------|------|
| 会话加载器 | `.cursor/rules/ai-rules-loader.mdc` | 按任务类型加载本目录下专文；Companion 伴读 |
| 常驻短规则 | `.cursor/rules/*.mdc` | 薄封装 / alwaysApply 提示；细则仍以本目录为准 |
| 细则与索引 | **本目录** | `00`～`11` 专文 + `project_rules.md` |

冲突时：**核心规则 > 最佳实践 > 风格指南**；本目录专文与 Cursor 短规则冲突时，以本目录**更新更具体**的专文为准（支付/KYC、Git 禁止 `--no-verify` 等专项除外，仍以专章优先）。

## 维护约定

1. **只改本目录**；不要在 `task2app/` 下再创建实体 `.ai/` 文件树。
2. 新增约束：写入对应 `0x_*/` 专文，并在 `project_rules.md` 与（如需要）`00_project_constraints.md` / `ai-rules-loader.mdc` 增加入口。
3. 修改后自检：`test -L task2app/.ai && readlink task2app/.ai` 应为 `../.ai`。

## 目录一览

| 目录 | 主题 |
|------|------|
| `00_start/` | 会话起点与续接 |
| `01_project_constraints/` | 项目硬约束 |
| `02_documentation/` | 文档与流程 |
| `03_technical_implementation/` | 后端与技术实现 |
| `04_frontend_development/` | 前端 |
| `05_testing_quality/` | 测试质量 |
| `06_execution_monitoring/` | 执行与监控 |
| `07_togaf_project_management/` | TOGAF / 架构管理 |
| `08_prompt_management/` | 提示词管理 |
| `09_failure_experience/` | 失败经验库 |
| `10_software_design_philosophy/` | 软件设计哲学 |
| `11_ai_development/` | AI 辅助开发工作流 |

## 相关：Agent 技能

技能包 SSOT 为仓库根 **[`.claude/skills/`](../.claude/README.md)**（`task2app/.agents` / `task2app/.claude` 均为指向 `.claude` 的符号链接）。
