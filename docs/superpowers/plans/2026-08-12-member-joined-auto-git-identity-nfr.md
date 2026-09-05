# Step 5 — NFR：MEMBER_JOINED 自动 Git 身份

| 类别 | 级别 | 说明 |
|------|------|------|
| Security / AuthZ | **L3** | 管理 API：self / `member:manage` / `group-members:manage`；内部 ensure-default 仅服务间；禁止伪造 member 写异公司身份 |
| Reliability | L2 | 消费者永久失败 → DLT；ensure-default 幂等；发布失败不阻断成员创建（记日志） |
| Performance | L2 | 每成员一次 ensure；哈希邮箱 O(1)；PeopleManage 弹窗按成员拉列表 |
| Consistency | L2 | 最终一致：成员可见略早于默认身份；重复事件跳过插入 |
| Observability | L2 | 消费/ensure 日志带 `trace_id` + member_id/company_id；DLT 可人工排查 |
| Usability | L2 | 弹窗失败展示 traceId；自助页 UserGitIdentities 行为不变 |

存量成员无自动回填（后续批处理 OPT）。
