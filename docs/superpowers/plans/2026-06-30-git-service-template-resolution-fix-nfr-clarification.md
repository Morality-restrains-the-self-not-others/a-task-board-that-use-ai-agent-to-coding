# NFR 澄清: 修复 git-service GitLab 容器 crash loop（模板变量未解析）

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-git-service-template-resolution-fix-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 理由 |
|------|------|------|
| 全部类别 | L0 | 不适用 — 3 行 Python 防御逻辑，无新数据流/端点/外部依赖 |

## 跳过声明

本变更为基础设施 bug 修复：`gitService/run.sh` Python 内联脚本中增加 3 行防御逻辑。不涉及新业务概念、数据流或外部依赖。

## 领域模型影响

无。

## 质量场景

无。
