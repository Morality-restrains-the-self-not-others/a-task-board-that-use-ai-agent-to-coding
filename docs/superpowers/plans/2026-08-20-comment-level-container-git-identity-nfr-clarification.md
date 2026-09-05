# NFR 澄清：容器拉取评论级 Git 提交者身份

- **日期**: 2026-08-20
- **前置**: 价值流

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| POST .../comment/{cid}/.../task-detail | tenant + task + comment | 已有 comment_id，适合评论级读 | 传入 FetchRepoIdentities |
| POST .../repo-clone-credentials | 同上 | 已有 token.CommentID | 传入 |
| POST .../layer-oauth-tokens | 同上 | 已有 token.CommentID | 传入 |
| GET /api/internal/tasks/{id}/container-snapshot?comment_id= | task + comment | 合适 | snapshot 补 name/email |

L0 升级触发：单任务评论数 > 10万（当前远低于）。

## 幂等性强制审视

| 路径 | 副作用 | 级别 | 理由 |
|------|--------|------|------|
| FetchRepoIdentities / task-detail | 无 | L0 | 只读查询 |
| container-snapshot | 无 | L0 | 只读 |
| 发评写 JSON | 有（既有） | 本增量不改 | — |

资金/云资源：不涉及。
