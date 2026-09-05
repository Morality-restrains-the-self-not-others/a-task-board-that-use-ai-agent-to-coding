# SaaS ↔ 容器对接 Skill 索引

Django `task2app/Saas_project/skillList/machine_container.md` 已随 Django 退役删除。对接契约拆成两份 **方向相反** 的 skill，互为交叉引用：

| 方向 | 调用方 → 被调方 | 权威文档 | 运行时入口 |
|------|-----------------|----------|------------|
| **容器 inbound** | 浏览器 / SaaS 网关 → 容器 `onlineServiceJS` | [`trae-agent/onlineServiceJS/skill.md`](../../../trae-agent/onlineServiceJS/skill.md) | 容器 `GET /skill.md` |
| **SaaS inbound** | 容器 → SaaS（凭证 / 云 / 心跳 / 进度） | [saas-machine-container.md](./saas-machine-container.md) | 厂商门户 `GET /saas-machine-container.md`（如 `https://provider.daydaymoney.com/saas-machine-container.md`） |
| **浏览器 compute 转发** | 任务详情 → 网关 → 容器 | [ADR-0010](../../adr/0010-comment-id-path-kv.md) | `/api/cloud/compute/{action}/…/comment_id/{cid}/` |
| **浏览器 PR 审查** | 任务详情 → taskTaskService / taskGitOauth | [saas-machine-container.md §5.2](./saas-machine-container.md) / [ADR-0028](../../adr/0028-pr-reply-one-click-merge-audit.md) | 评论 `git_pr`、`/api/git-oauth/merge-request-status/`、`/api/git-oauth/merge-request-merge/` |

约定：

- CSC 键是 `(task_id, comment_id)`。path 必须保留 `task_id`，`comment_id` 是追加 kv，不是替换。
- 容器调 SaaS：`TaskApiEndPoint` 必须 `…/task/{taskId}/comment/{cid}/cloud`（位置段 `/comment/{cid}/`）；HTTP inbound 无该段则 404。`comment_id` 同时写入 **JSON body**（UserData `COMMENT_ID`）。禁止无 cid 的旧 `…/task/{taskId}/cloud`。
- 浏览器调容器 compute：`comment_id` 在 **path kv** `/comment_id/{cid}/`（ADR-0010）；query 仅兼容。
