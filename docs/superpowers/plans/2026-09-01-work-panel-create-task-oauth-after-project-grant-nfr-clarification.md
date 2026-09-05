# NFR 澄清 — 项目 L2 种下创建任务自动运行

- **Date:** 2026-09-01
- **Design:** ADR-0055 / 2026-09-01 work-panel create-task OAuth spec

## 路径分片键审视

| 路径 | 分片 ID | 适配性 | 级别 | 动作 |
|------|---------|--------|------|------|
| GET `/api/internal/projects/git-oauth-grant/?project_id=&user_id=&gitsite=` | `project_id` + `user_id` | 与表唯一键一致；grant 按项目/用户/站点点查 | L1 | 查询必须三键齐全；禁止无 project_id 列表 |
| POST `/api/projects/validate-git-repos/tenant_id/{tid}/`（存量） | `tenant_id` + body `project_id` | 租户 + 项目 | L1 | 创建弹窗 `probe_access:false` |
| 创建任务 POST（存量） | `tenant_id` / `workspace_id` | 已有 | L1 | 不改路径 |
| Kafka `COMMENT_GIT_OAUTH_GRANTED` | key `grant:comment:{commentID}:{userID}:{gitsite}` | 评论级，非 tenant 粗键 | L1 | seed 必须带 comment_id，禁止只用 user_id |
| 工作面板路由 `/tenant/{tid}/work-panel` | `tenantId` | 前端壳 | L0 | 无新路由 |

## 幂等性审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----------|--------------|--------|----------|
| GET internal grant | 无 | — | — | — | L0 只读 |
| FE validate-git-repos | 无 | 弹窗重开 | — | — | L0 |
| POST mark project grant（存量） | 写项目 L2 | OAuth 回调重放 | (project, user, gitsite) 一行 | DB UNIQUE | 更新 remote_user_id |
| `applyGrantTicketToIdentities` | 消费 ticket + 写评论 JSON + 事件 | 创建重试 | ticket 一次性 | 现网 consume | 已消费则不再 stamp |
| `applyProjectL2SeedToIdentities` | 写评论 JSON + `COMMENT_GIT_OAUTH_GRANTED` | 复用 auto-run 评论 / 重复 ensure | 该 comment × user × gitsite | `grant:comment:{commentID}:{userID}:{gitsite}` | 已有 `oauth_gitsite` 则跳过；事件同 key 可重复投递，消费侧 Ack 空操作 |
| 创建任务本身 | 写 task / comments | 双击提交 | 前端 click guard + 服务端任务 id | 现网 | 不改资金路径 |

资金/配额：本增量不涉及，非 L3。

## 其它 NFR

- **安全：** internal secret；种下只用操作者 id。
- **延迟：** validate 不 probe Git API；GET grant 为点查。
- **日志：** seed 成功/查找失败打 `event=`；不打 token。
