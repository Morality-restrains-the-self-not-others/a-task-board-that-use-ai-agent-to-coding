# NFR 澄清：auto_run 首指令与自动交付

- **日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- **默认等级**: **L2（Standard）**；鉴权 / token 相关 **L3**

## 质量属性矩阵

| 属性 | 等级 | 场景 / 说明 | 度量 |
|------|------|-------------|------|
| **正确性** | L2 | `auto_run=true` 时 bootstrap 成功后恰好触发一次首指令；completed 后触发一次交付 | T4/T6/T8 单测绿 |
| **正确性（否定）** | L2 | `auto_run=false`、job 失败、无克隆层时不误触发 | T5/T7 单测绿 |
| **安全 / 鉴权** | **L3** | 容器仅持 scoped access token 读 task-detail；不得写 SaaS 身份表；不得日志输出 token | Go token 校验单测 + 日志审计 |
| **安全 / 身份** | **L3** | `repo_git_identities` 仅由 Go 从平台身份表解析下发；容器不得接受任意 email 写入 | 代码审查 + T2 |
| **幂等** | L2 | 同一容器生命周期：`auto_run_first_job.json`、`auto_run_delivery.done` 防止重复 job/交付 | T8 单测 |
| **可用性** | L2 | 交付失败（push/PR/身份缺失）**不阻断** onlineServiceJS HTTP 与其它 job | `AUTO_RUN_DELIVERY_FAILED` 后服务仍响应 |
| **可用性** | L2 | Agent 长时间运行：交付仅挂 `proc.on('close')`，不阻塞 bootstrap/HTTP | 设计约束 |
| **可观测** | L2 | 结构化日志：`AUTO_RUN_DELIVERY_COMPLETE` / `AUTO_RUN_DELIVERY_FAILED`；bootstrap 首指令 WARN | 日志关键字检索 |
| **可观测 / 脱敏** | L2 | 禁止日志输出 access_token、refresh_token、完整密钥；邮箱按既有策略 | logging 审计清单 |
| **性能** | L2 | 首指令与交付在 bootstrap/close 异步路径；不增加 task-detail 热路径往返 | 无额外 Django  hop |
| **兼容** | L2 | `auto_run=false` 或字段缺失时行为与现网一致（仅 bootstrap，无自动 job） | T5 |
| **兼容** | L2 | 无新 Python/Django 公网 API；契约扩展向后兼容（新 JSON 字段可选） | machine_container.md §4.4 |

## L3 项展开（auth / token）

| 项 | 要求 |
|----|------|
| Token scope | task-detail 请求须校验 container access token 与 task 归属一致 |
| 身份 SSOT | Git user.name / user.email 仅来自 Go 查询 `accounts_user_company_git_identity` |
| 无特权 escalation | 容器内编排不得换取 SaaS 会话或写身份绑定 API |
| 失败 closed（读路径） | token 无效 → task-detail 401/403，不返回部分快照 |

## L2 项展开（幂等 / 脱敏 / 失败隔离）

| 项 | 要求 |
|----|------|
| 幂等标志 | `runtime/auto_run_first_job.json`（含 job_id）；`runtime/auto_run_delivery.done` |
| 重启语义 | 同容器进程重启读标志，不重复；新 VM 标志清空（与 force_auto_run 一致） |
| 脱敏 | jsonLog / reqLogs 不含 token；错误栈不 dump env |
| 失败不阻断 | sync 跳过、commit 失败（nothing to commit）、push/PR 失败均记日志并 return，不 throw 至 HTTP 层 |
| 身份未绑定 | 跳过 sync + WARN，仍尝试 commit/push（降级可观测） |

## 领域模型影响

- 不新增持久化聚合；运行时标志为容器本地文件（Infrastructure）。
- `AutoRunTask.auto_run` 为 Task 快照属性；`RepoGitIdentity` 为值对象，由 Credential 上下文组装。

## 非目标（本期 NFR 不提升）

- 不保证 Agent 完成时限 SLA（L1）。
- 不自动 merge PR 到 merge_target。
- 不依赖浏览器详情页常开（L3 可用性不要求前端 SSE watch）。
