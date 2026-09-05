# NFR 澄清：容器镜像 @ 模式

**日期**: 2026-07-15  
**价值流**: `docs/superpowers/plans/2026-07-15-container-image-at-mention-value-stream.md`  
**默认等级**: L2 Standard（goal-mode）；鉴权相关抬升 L3

## 类别决策

| NFR | 等级 | 决策 |
|-----|------|------|
| 安全 / 鉴权 | **L3** | 开关硬闸、镜像租户校验、容器 token 绑定回写；ContextPack 脱敏 |
| 性能 | L2 | ContextPack 默认全量；超 256KB 截断最早评论；不保证超大线程实时 |
| 可用性 | L2 | 开机器失败 → Agent 评论 `failed` + Feed 可见；不阻塞人类评论落库 |
| 一致性 | L2 | 评论先提交再异步编排（最终一致）；同一 parent 仅一进行中 run |
| 可观测性 | L2 | 结构化日志：mention、run_id、start-vm reuse/new、stream chunk/done |
| 可扩展性 | L1 | MVP 单镜像 mention；不并行多机 |

## 质量场景

1. **刺激**: 开关关闭时 POST mentions → **响应**: 400，无 Kafka，无 start-vm  
2. **刺激**: 合法 `@` + 闲置机存在 → **响应**: reuse=true 路径，Agent run `starting→running`  
3. **刺激**: 容器分片 stream 30s → **响应**: 浏览器 SSE 持续更新，complete 后落库可刷新  
4. **刺激**: ContextPack >256KB → **响应**: truncated=true，仍含触发评论  

## 领域模型影响

- `AtMentionRun` 需状态机与幂等键（parent_comment_id）  
- ContextPack 为值对象快照，非实时订阅  
- 安全边界在应用服务校验，不进纯领域实体

## 边界（不做）

- 多镜像并行、跨租户镜像、保证冷启动 <30s SLA
