# 功能意图：任务详情关联仓库地址可跳转

## 背景与目标

任务详情「关联项目」只读/编辑区把 Git 仓库 URL 渲染为纯文本 `span`，用户无法一键打开 GitLab/GitHub。目标改为：http(s) 地址渲染为真实 `<a href>` 外链。

## 范围与边界

- **范围内**：`TaskDetailLinkedProjectsViewMode`、`TaskDetailLinkedProjectsEditMode` 的仓库 URL 展示。
- **范围外**：不改项目详情页已有仓库链接；不把 `git@` / `javascript:` 等非 http(s) 引用变成链接。

## 约束与风险

- 导航必须是真实 `<a href>`（禁止 click.prevent + router.push）。
- `target=_blank` + `rel=noopener noreferrer`。
- 仅 `https?://` 前缀可作 href，避免 `javascript:` XSS。

## 验收标准

1. 只读态仓库 URL（如 `https://gitlab-…/org/repo`）为可点击外链，新标签打开。
2. 编辑态同样可点击。
3. 非 http(s) 地址保持纯文本。

## 业务意图 → 事件对照

**无对应事件（纯前端导航例外）**：不改变任务或项目状态。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 关联仓库 URL 可跳转 | — | — | — | 纯展示/导航，无服务端状态变更 |

## 变更记录

- 2026-08-20：初版。任务详情关联仓库 URL 从纯文本改为 http(s) 外链。
