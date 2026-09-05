# NFR 澄清：Git OAuth 资源使用标记

- **日期**: 2026-08-29
- **价值流**: `docs/superpowers/plans/2026-08-29-git-oauth-resource-grant-marker-value-stream.md`
- **默认等级**: L2 Standard；换票/打标安全 L3

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| `GET /api/git-oauth/{github\|gitlab}-start-from-gateway/` | 无租户；有 `X-User-Id` | user 非租户分片键 | **L0** | OAuth start 低频；升级触发：单实例 QPS>100 再按 company 前置 |
| OAuth IdP callback | 无 | — | **L0** | 浏览器回流，不可加 tenant 到 GitHub URL |
| Internal MarkGrant project | `company_id`（项目行）+ `project_id` | company_id 合适 | L1 | 表含 `company_id`；查询必须带项目归属 |
| Internal MarkGrant comment | `tenant_id`（任务）+ `comment_id` | tenant 合适 | L1 | 经 task 行解析 tenant |
| ConsumeGrantTicket | ticket id + uid | 全局短 TTL 表 | **L0** | 票寿命分钟级；行数上限 10 万；触发：改 Redis+tenant 前缀 |
| `access-for-user` | uid（现网） | 与 L1 凭据 PK 对齐 | L1 维持 | 不改换票分片；L2 查 owner 服务时带 project/comment 的 tenant/company |
| Kafka `PROJECT_GIT_OAUTH_GRANTED` | key=`grant:{project}:{user}:{gitsite}` | 非 tenant | L0 审计 | 无自动消费者；升级：按 company_id 作 key |
| Kafka `COMMENT_GIT_OAUTH_GRANTED` | key=`grant:{comment}:{user}:{gitsite}` | 同 | L0 | 同上 |
| FE 项目详情 | 路由含 tenant/workspace/project | 是 | L1 | 保持 |
| FE 创建任务 | tenant/workspace | 是 | L1 | 保持 |

无分片 ID 的 OAuth start/callback/**L0 理由**：流量随用户点击而非租户膨胀；升级触发为网关 QPS 或 ticket 表>10 万行。

## 幂等性强制审视

| 路径 | 副作用 | 重复源 | 业务边界 | 幂等键 | 重放 | 持久化 | 等级 |
|------|--------|--------|----------|--------|------|--------|------|
| GET start-from-gateway | 无（重定向） | — | — | — | — | — | L0 只读跳转 |
| OAuth callback upsert L1 | 有 | IdP 重放 code | provider×user×remote_user | 现网 UNIQUE | 复用行 | DB | L2 已有 |
| MarkGrant project | 有 | 双击/回调重放 | project×user×gitsite | UNIQUE 三元组 | upsert 覆盖 remote_user_id | DB | **L3** |
| MarkGrant comment JSON | 有 | 同上 | comment×user×gitsite | JSON 字段 upsert | 覆盖 | 评论行 | **L3** |
| ConsumeGrantTicket | 有 | 双 POST 创建任务 | ticket id | PK + consumed_at | 第二次 409/空操作 | DB | **L3** |
| access-for-user | 有（审计+出站 refresh） | 克隆重试 | 现网凭据行 | 现网 | 短票 | 现网 | L2 维持；**无 L2 则拒绝不换票** |
| 发布 GRANTED 事件 | 有 | 回调重放 | 与 MarkGrant 同 | `idempotency_key=grant:...` | 消费者无则至少一次审计 | Kafka | L2；禁止用 tenant 作消费键 |
| 排队 start-vm | 有 | timer 重试 | membership×task | 现网 slot | 现网 | 现网 | 不改资金；只改 UserID |
| FE 授权 `<a href>` | 无写（导航） | — | Anti-Replay-OK: 真实链接 | — | — | — | L0 |

前端写操作：创建任务提交已有 createClickGuard；OAuth 入口为真实 `<a href>`（元规则 52 豁免导航）。

资金/云资源：clone 会起容器，换票闸门 L3（无 L2 不得换票）。

## 类别定级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L3 | L2 闸门 + uid 绑定 ticket |
| 数据一致性 | L2 | upsert；无分布式事务；先写 L2 再 302 |
| 容错 | L2 | MarkGrant 失败则不视为已授权（回调仍可完成 L1） |
| 可伸缩性 | L0–L1 | 见表 |
| 性能 | L2 | 换票前一次 grant 点查，禁止扫表 |
| 可观测性 | L2 | 结构化日志 grant_kind/id 指纹，无 token |

## 质量场景

1. **刺激**: 用户在项目 B 打开详情，L1 已在项目 A 授权。**响应**: 徽章「需要授权」；validate 不因 L1 返回 token_available。
2. **刺激**: 同一项目同一用户重复 OAuth 回流。**响应**: grant 一行；事件幂等键相同。
3. **刺激**: 排队开火且 Owner≠启用者。**响应**: clone 用评论 `created_by_id`。

## 领域模型影响

- Grant 不是凭据聚合的一部分（跨 BC）
- Ticket 短寿命实体，消费即死
- 评论聚合扩展 VO（oauth_gitsite / oauth_remote_user_id / oauth_granted_at）

## 权衡

- 不回填 L2（破坏性 UX，换正确语义）
- 不做跨评论继承
- 项目 Kafka 新建最小 publisher（现网无）
