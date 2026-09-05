# Architecture Decision Records (ADR)

本目录包含本项目的**架构决策记录**（Architecture Decision Records），用于记录所有重要的架构决策及其上下文、权衡和后果。

## 什么是 ADR

ADR 是一种轻量级文档，记录架构决策的背景、决策内容和影响。每条 ADR 描述：

- 我们在某个上下文下做出了什么架构决策
- 为什么做出这个决策（替代方案和权衡）
- 这个决策带来什么后果（正面和负面）

## ADR 索引

| 编号 | 标题 | 状态 | 日期 |
|------|------|------|------|
| [0001](0001-ddl-must-reside-in-datamigrate.md) | 所有 DDL 必须放置在 dataMigrate 目录中 | accepted | 2026-08-02 |
| [0039](0039-ztree-step-full-cos-archive.md) | ztree 执行全文以腾讯云 COS 归档 step_full.json | accepted | 2026-08-23 |
| [0006](0006-tencent-cos-vendor-documents.md) | 厂商证照对象存储选用腾讯云 COS | accepted | 2026-08-14 |
| [0008](0008-workspace-task-display-seq.md) | 任务帖人读序号与技术主键分离 | accepted | 2026-08-14 |
| [0009](0009-comment-level-repo-identity.md) | 仓库 Git 身份与授权随评论持久化 | accepted | 2026-08-16 |
| [0010](0010-comment-id-path-kv.md) | comment_id 作为 compute 转发 path kv | accepted | 2026-08-16 |
| [0011](0011-no-service-internal-poll-loop.md) | 业务服务禁止进程内轮询 / 循环 | accepted | 2026-08-16 |
| [0012](0012-runall-health-port-ssot.md) | runAll 健康检查端口必须等于进程监听 SSOT | accepted | 2026-08-16 |
| [0013](0013-remove-idle-machine-reuse.md) | 去掉跨任务闲置机器复用 | accepted | 2026-08-16 |
| [0014](0014-pluggable-multi-region-gitlab.md) | 可插拔多区域 GitLab（独立 CE 实例 + 区域注册表） | accepted | 2026-08-18 |
| [0015](0015-event-consumer-idempotency.md) | 领域事件消费采用 at-least-once + 幂等处理 | accepted | 2026-08-18 |
| [0016](0016-gitlab-sso-only-no-self-signup.md) | GitLab 禁止自行注册、仅允许 SSO | accepted | 2026-08-18 |
| [0017](0017-globally-unique-resource-order-number.md) | 资源订单号由 Snowflake 派生、全局唯一 | accepted | 2026-08-19 |
| [0018](0018-order-id-tenant-shard-routing.md) | 资源订单展示号嵌入 tenant_id，主键仍为订单 Snowflake | accepted | 2026-08-19 |
| [0019](0019-agent-ship-direct-merge-main.md) | Agent 交付默认直接合入 main 并推送 origin | accepted | 2026-08-19 |
| [0020](0020-frontend-button-anti-replay.md) | 前端副作用按钮必须具备防重放设计 | accepted | 2026-08-19 |
| [0021](0021-logic-rollback-requires-approval.md) | 逻辑回退须经审批，禁止无用旧逻辑残留 | accepted | 2026-08-19 |
| [0022](0022-taskfe-nginx-static-resident.md) | 公网 taskFE 由 Docker nginx 静态常驻，Vite 只负责构建 | accepted | 2026-08-20 |
| [0023](0023-ccb-logs-workspace-hash-shards.md) | 评论容器启动日志按 workspace_id 哈希分表 | accepted | 2026-08-20 |
| [0024](0024-saas-inbound-skill-version.md) | 容器→SaaS 接口契约版本与镜像声明 | accepted | 2026-08-20 |
| [0025](0025-referral-multi-channel-codes.md) | 多渠道推荐码与绑边资格快照（分账门禁见 ADR-0033） | accepted | 2026-08-20 |
| [0026](0026-image-container-skill-list.md) | 镜像容器技能列表文件与上传时绑定 | accepted | 2026-08-21 |
| [0027](0027-restart-compile-separation.md) | 进程重启与源码编译分离；编译失败保留 last-good | accepted | 2026-08-22 |
| [0028](0028-pr-reply-one-click-merge-audit.md) | PR 回复与一键合并审计 | accepted | 2026-08-22 |
| [0029](0029-referral-qualification-revoke-audit.md) | 推荐分账资格可取消且操作必须带理由审计 | accepted | 2026-08-22 |
| [0030](0030-admin-only-order-profit-sharing.md) | 订单分账明细仅管理员可见 | accepted | 2026-08-22 |
| [0031](0031-instruction-idle-recycle-no-sts-in-comments.md) | 指令闲置回收：平台优先，STS 不进评论 | proposed | 2026-08-22 |
| [0032](0032-wechatpay-go-sdk.md) | 微信支付必须调用仓内 sdk/wechatpay-go | accepted | 2026-08-22 |
| [0033](0033-paytime-referral-profit-sharing.md) | 微信分账资格以支付时刻为准 | accepted | 2026-08-22 |
| [0034](0034-order-wechat-fapiao.md) | 订单电子发票走微信支付普通商户开票 API | accepted | 2026-08-23 |
| [0035](0035-runall-orchestrator-independence.md) | runAll 编排器与托管服务生命周期解耦 | accepted | 2026-08-23 |
| [0036](0036-gitlab-traffic-quota-gitaccess-gate.md) | GitLab 出站流量配额在 GitAccess 闸门强制执行 | accepted | 2026-08-23 |
| [0037](0037-admin-user-impersonation.md) | 平台用户模拟登录使用独立会话与 user:impersonate | accepted | 2026-08-23 |
| [0038](0038-impersonation-audit-label-and-inbox.md) | 模拟登录日志标识、必填理由与用户收信箱 | accepted | 2026-08-23 |
| [0039](0039-ztree-step-full-cos-archive.md) | ztree 执行全文以腾讯云 COS 归档 step_full.json | accepted | 2026-08-23 |
| [0040](0040-always-flag-wechat-profit-sharing.md) | 租户微信订单一律打分账标识 | accepted | 2026-08-23 |
| [0041](0041-tester-role-gitlab-region-access-mode.md) | 测试角色与 GitLab 区域开发/发布模式 | accepted | 2026-08-23 |
| [0042](0042-gitlab-traffic-meter-workhorse-logs.md) | GitLab 出站流量以 workhorse 日志为计量 SSOT | accepted | 2026-08-25 |
| [0043](0043-tenant-selfhosted-gitlab-oidc-sso.md) | 租户自建 GitLab 使用独立 OIDC client 与成员闸门做平台 SSO | accepted | 2026-08-25 |
| [0044](0044-tenant-scoped-oidc-protocol-paths.md) | 租户自建 GitLab OIDC 协议端点路径携带 tenantId | accepted | 2026-08-25 |
| [0046](0046-no-hardcoded-secrets.md) | 密钥禁止硬编码在源码中 | accepted | 2026-08-27 |
| [0047](0047-local-gitlab-runall-start-opt-in.md) | 本机 GitLab 须 conf 显式打开后才能经 runAll 面板启动 | accepted | 2026-08-27 |
| [0048](0048-gitlab-not-runall-start-dependency.md) | 其它 runAll 服务不得 depends_on GitLab | accepted | 2026-08-27 |
| [0049](0049-git-oauth-resource-grant-marker.md) | Git OAuth L1 凭据与项目/评论 L2 使用标记分离 | proposed | 2026-08-29 |
| [0051](0051-task-project-entity-revisions.md) | 任务与项目主要属性不可变历史版本 | accepted | 2026-08-30 |
| [0052](0052-binary-deploy-config-repo.md) | 二进制部署；运行时配置与源码仓隔离 | accepted | 2026-08-30 |
| [0053](0053-host-secrets-catalog.md) | 主机密钥必须登记在 HOST_SECRETS 册 | superseded | 2026-08-31 |
| [0054](0054-conf-local-secrets-only.md) | 机密参数仅放在 conf-local | accepted | 2026-08-31 |

> **新决策**: 使用 `template.md` 模板创建新 ADR，按递增序号命名。

## ADR 生命周期

```
proposed → accepted → (deprecated | superseded)
```

1. **proposed** — 提案，正在讨论中
2. **accepted** — 已接受，当前正在实施的决策
3. **deprecated** — 已废弃，不再适用
4. **superseded** — 被新的 ADR 取代（需注明取代它的 ADR 编号）

当 ADR 状态变更为 `superseded` 时，必须在其文档中注明：
```markdown
**Superseded by:** [ADR-0002: xxx](0002-xxx.md)
```

## 何时需要写 ADR

以下情况**必须**创建 ADR：

- 选择一种新的技术栈、框架或第三方服务
- 引入新的架构模式（如事件驱动、CQRS、微服务拆分）
- 制定重要的设计约束或编码规范（如所有 ID 使用 Snowflake）
- 决定不采用某种显而易见的方案
- 基础设施重大变更（如消息队列从 Redis 迁移到 Kafka）
- 跨服务的协议或数据格式约定
- 安全策略或合规架构决策

以下情况**不需要** ADR：

- 常规功能开发（无架构层面影响）
- Bug 修复
- 小规模重构（不改变架构边界）
- 已在现有规范/元规则中明确定义的模式

## 格式规范

所有 ADR 必须遵循 [template.md](template.md) 的结构：

1. **Title** — 简短名词短语，概括决策
2. **Status** — proposed / accepted / deprecated / superseded
3. **Context** — 描述当前面对的架构问题，以及驱动力（技术、业务、约束）
4. **Decision** — 清晰陈述我们做出的决策（用 "We will..." 句式）
5. **Alternatives Considered** — 列出我们考虑过的方案及拒绝原因
6. **Consequences** — 正面/负面后果，列出因这个决策变得更简单或更困难的事情
7. **References** — 相关文档、讨论链接

## 文件命名

```
NNNN-title-in-kebab-case.md
```

- `NNNN` — 4 位递增序号（如 `0001`）
- `title-in-kebab-case` — 简短的英文标题，全小写，连字符分隔

示例：
- `0001-use-snowflake-for-primary-keys.md`
- `0002-migrate-message-queue-redis-to-kafka.md`
- `0003-adopt-dependency-inversion-for-infrastructure.md`

## 参考

- Michael Nygard, "[Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions)"
- Joel Parker Henderson, "[Architecture Decision Record (ADR)](https://github.com/joelparkerhenderson/architecture-decision-record)"
