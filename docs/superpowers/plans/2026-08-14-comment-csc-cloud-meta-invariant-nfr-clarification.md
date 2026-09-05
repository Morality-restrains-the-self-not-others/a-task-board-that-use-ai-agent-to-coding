# NFR Clarification: 评论 CSC 云平台元数据不变量

> 价值流：`docs/superpowers/plans/2026-08-14-comment-csc-cloud-meta-invariant-value-stream.md`  
> 默认等级：L2（0-auto-flow）

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 可伸缩性 | 动作 |
|------|---------|------|----------|------|
| ensure / resolve 评论 CSC | tenant + workspace + task + comment | tenant 合适 | L1 | 无新 path |
| GET workbench-link | 同上 | tenant 合适 | L1 | 补齐后拼 URL |
| POST ensure-client-ingress `?comment_id` | 同上 | tenant 合适 | L1 | 有 comment 走评论行 |
| 020 UPDATE JOIN | 同库 | 非跨服务 | L0 | 幂等 CASE WHEN |

**Hard Gate：通过。**

## 类别定级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | 复用工作区成员门禁；internal lookup 仍要 internal secret |
| 完整性 | L2 | 按字段补齐；不覆盖合法三元组；不拷 instance |
| 可用性 | L2 | 模板非法则新建失败，前端走既有未配置提示 |
| 性能 | L2 | 每请求最多 1 次模板读 + 1 次 upsert |
| 可伸缩性 | L1 | 路径带 tenant |
| 可观测性 | L2 | `comment_csc_healed_from_base` / `comment_csc_ensured` |

## 质量场景

1. 模板非法 → ensure 不 INSERT。
2. 评论杭州+cpa-user → resolve 不改成青岛模板。
3. 评论 mock+空字段 → 只补三元组，instance 不变。
4. `mock-` instance → Workbench 仍 400。

## 领域模型影响

- 任务模板 = Default-on-Create，不是读时 SSOT。
- CommentCloudServer 聚合不变量：真实 Platform + 非空 Region + 非空 AuthorizationId。
