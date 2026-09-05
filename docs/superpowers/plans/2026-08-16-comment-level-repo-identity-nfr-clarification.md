# NFR 澄清：评论级仓库身份

> 设计：`docs/superpowers/specs/2026-08-16-comment-level-repo-identity-design.md`  
> 价值流：`docs/superpowers/plans/2026-08-16-comment-level-repo-identity-value-stream.md`

默认支撑等级：**L2 Standard**（auto-flow）。非支付/鉴权核心写路径，不升 L3。

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| POST `/api/tasks/{taskId}/comments/tenant_id/{tid}` | tenant_id + task_id | tenant_id 合适 | L2 | 写评论带租户；JSON 随评论行 |
| GET 评论列表同一 path | tenant_id + task_id | 合适 | L2 | 列表按 task 已分区访问 |
| GET `/api/internal/tasks/{id}/container-snapshot?comment_id=` | task_id（internal） | task_id 非租户分片主键，但 internal 单任务读 | L1 | 须 comment 属于该 task；升级触发：跨租户内部网关时补 tenant_id |
| 前端 `/tenant/:tenantId/workspace/:ws/task-detail/:taskId/` | tenantId | 合适 | L2 | 不变 |
| Kafka `TASK_COMMENT_IMAGE_MENTIONED` | 现有 key（task_id） | 与既有评论启动一致 | L2 | payload 增补字段，不改分区键 |
| github-credential-status（既有） | tenant + workspace + task | 合适 | L2 | 只读连接列表 |

无「完全无分片 ID」的新公网路径。internal snapshot 标 L1：单任务读、低频；升级触发为内部网关按租户分片时补 `tenant_id`。

## 类别定级

| 类别 | 等级 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 评论 JSON 随 task_comments；不新表爆炸 |
| 一致性 | L2 | 评论写入与事件同一成功路径；snapshot 读评论 |
| 安全 | L2 | 只存 ID；校验 github_user_id 属于当前用户连接 |
| 可用性 | L2 | 旧评论回退任务级表 |
| 可观测性 | L2 | comment_create / snapshot 日志带 comment_id、task_id、trace_id；禁止打 token |
| 性能 | L2 | JSON 反序列化 O(repos)；repos 通常 < 20 |

## 质量场景

1. **刺激**：两评论并行提交不同 github_user_id。**响应**：各容器按各自 JSON 克隆，任务级表不被后写覆盖。
2. **刺激**：运行评论缺 git_identity_id。**响应**：400，无 start-vm。
3. **刺激**：旧评论 snapshot 无 JSON。**响应**：回退 task_repo_identities，5s 内返回。

## 领域模型影响

- Comment 聚合包含 `RepoIdentitySelection` 值对象集合。
- Task 聚合不再是运行身份一致性边界。
- 不引入新限界上下文。
