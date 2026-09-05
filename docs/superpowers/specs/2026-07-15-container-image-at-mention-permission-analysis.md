# 角色权限分析：容器镜像 @ 模式 + 统一评论区

**日期**: 2026-07-15  
**设计文档**: `docs/superpowers/specs/2026-07-14-container-image-at-mention-design.md`  
**状态**: 完成（goal-mode 自动）

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有/拟定检查 | 是否缺失 | 建议 |
|--------|------|----------|------|---------------|----------|------|
| PATCH workspaces `container_image_at_mode_enabled` | workspace_admin / tenant_admin | Workspace | write | 与 `task_archive_tier` 同级：已登录 + 工作空间管理权限 | ✅ | 禁止 viewer 改开关 |
| GET workspaces 含开关字段 | workspace_member | Workspace | read | 既有 workspace 读权限 | ✅ | — |
| POST comments + `mentions` | workspace_member（可评论） | Task | write | 既有评论写权限 + **开关硬闸** + 镜像属本租户 | ⚠️ 须补 | 开关关 → 400；镜像跨租户 → 400；多 mention → 400 |
| 伪造 mentions（开关关） | 攻击者 | Task | write | 后端拒绝，不发 Kafka | ✅ 设计已含 | 单测覆盖 |
| 创建/读 `container_agent_comments` | 同任务成员读；系统写 | Task | RW | 任务访问权；写仅编排/容器 token | ⚠️ 须补 | 列表接口校验 task 归属 |
| POST container-agent stream/complete | **容器 token** | Task（绑定） | write | token → task/workspace 匹配 + agent_comment_id 归属 | ⚠️ 关键 | 禁止用户会话冒充；禁止跨 task 回写 |
| SSE `container_agent_stream` | 打开任务详情的成员 | Task | read | 既有任务 SSE 订阅鉴权 | ✅ | 仅推送本 task 事件 |
| start-vm 编排（内部） | 服务账号 / 事件消费者 | Cloud | write | 既有 compute 权限链 + idle policy | ✅ | 不新增公网开机器 API |
| ContextPack 含评论线程 | 容器（持 token） | Task | read | 仅本 task；含敏感字段须脱敏 | ⚠️ | token/secret 不入 pack |

## IDOR / 串租户风险

| 风险 | 缓解 |
|------|------|
| 评论 mention 指向他租户 image id | 解析时校验 `tenant_installed_images.company_id` |
| 容器 token 回写他任务 Agent 评论 | stream 路径校验 token.task_id == comment.task_id 且 id 匹配 |
| 开关仅前端隐藏 | 后端硬闸（已设计） |
| SSE 订阅串任务 | 既有 SSE 按 task_id 过滤，保持 |

## 新角色

无。不引入新 RBAC 角色。

## 结论

无阻塞；实现时须落地「开关硬闸」「镜像租户校验」「容器 token 回写绑定」三处检查。可进入价值流。
