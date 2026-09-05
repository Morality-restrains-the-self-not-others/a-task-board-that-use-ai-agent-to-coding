# 权限分析：auto_run 首指令与自动交付

- **日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- **功能意图**: `task2app/docs/intents/engineering/cloud/006_auto_run_first_instruction_and_delivery.intent.md`

## 变更面

| 改动点 | 变更 | 鉴权 / 边界 |
|--------|------|-------------|
| Go `taskCredentialService` `FetchTaskDetail` | 增 `task.auto_run`、`repo_git_identities[]` | 容器 access token（machine container token）校验后只读；不暴露 SaaS 会话 Cookie |
| Node `onlineServiceJS` bootstrap 后 `createJob` | 内部编排，无新 HTTP 入站契约 | 仅本容器进程；依赖 bootstrap 已换票后的 task-detail |
| Node `jobsRuntime` close 钩子交付 | identities sync → commit → oauth-refresh-push | 内部模块调用；不经 Django；复用既有 layer OAuth |
| `machine_container.md` §4.4 | 契约文档同步 | 无新公网 API |

**结论：无新公网 Django API；无新浏览器可调 REST 端点。**

## 角色定义

| 角色 | 说明 | 本迭代权限 |
|------|------|------------|
| **任务创建者** | 在 SaaS 创建/编辑任务、设置 `auto_run=true`、绑定仓库 Git 身份 | 通过既有 SaaS 会话 + 租户/工作区 scope 写任务与身份绑定；**不**直接触达容器内编排 |
| **容器进程（token）** | onlineServiceJS 持 container access token 调 taskCredentialService | **仅**读 task-detail（含 `auto_run`、已解析 `repo_git_identities`）；执行 bootstrap、建 job、交付；**不得**写 SaaS 身份表或任意 email |
| **租户成员** | 工作区内可访问任务的成员 | 与现网一致：须通过 Gateway/tcg 鉴权后才能获得容器 token；跨租户/跨任务 IDOR 仍由既有链路拦截 |

无新角色类型；不引入 task 级新 RBAC 粒度。

## 权限影响表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET task-detail（容器 token） | 容器进程 | Task | read | access token scope + task 归属 | ✅ | 保持；扩展字段仍为同一鉴权门 |
| `repo_git_identities` 解析 | Go 服务（代表 token） | Task + Repo 绑定 | read | `task_repo_identities` + `accounts_user_company_git_identity` | ✅ | **禁止**容器侧自行拼装 email；必须由 Go 从身份表解析后下发 |
| bootstrap 后 `createJob` | 容器进程 | Layer/Job | write（本地） | 无跨租户面 | ✅ | 仅当 `auto_run=true` 且 bootstrap 成功；写 `runtime/auto_run_first_job.json` |
| 交付 sync/commit/push/PR | 容器进程 | Layer/Git 远端 | write | layer OAuth + 已解析身份 | ✅ | 身份未绑定时跳过 sync、记 WARN；失败记 `AUTO_RUN_DELIVERY_FAILED`，不阻断 HTTP 服务 |
| SaaS 身份表 | — | Tenant Identity | write | Django/既有 API | — | **本期容器路径不得写入** |

## 身份与 Git 邮箱边界

| 规则 | 说明 |
|------|------|
| 身份来源 SSOT | `accounts_user_company_git_identity`（经 Go `FetchRepoIdentities` 同源查询） |
| 容器禁止绕过 | onlineServiceJS **不得**接受外部传入任意 `user_email` 写 git config；仅使用 task-detail 下发的 `repo_git_identities` |
| 缺绑定 | 数组为空或缺项 → 交付阶段跳过 sync，日志明确；push 仍可走 layer OAuth（可能沿用已有 local config） |
| 日志脱敏 | 禁止输出 access_token、refresh_token、完整密钥；邮箱可按既有脱敏策略 |

## IDOR / 越权风险

| 风险 | 缓解 |
|------|------|
| 伪造 task-detail 拉取他人任务 | container access token 绑定 task/tenant scope；Go 校验 token 与 task_id 一致 |
| 容器写任意 Git 身份 | 无写身份 HTTP 路径；sync 仅应用 Go 下发的 VO |
| 重复首指令/交付造成越权 push | `runtime/auto_run_first_job.json`、`runtime/auto_run_delivery.done` 幂等 |
| token 泄露于日志 | 结构化日志禁止 token；交付失败不 dump 环境变量 |

## 与现有权限模型关系

- **任务创建者** 在 SaaS 侧设置 `auto_run` 与仓库身份绑定 → 属既有 workspace 编辑权限，无变更。
- **容器 token** 为短期、scoped 凭证；扩展 task-detail 字段不改变 token 发放门（tcg / credential SSOT）。
- **租户成员** 须先通过平台身份获得容器访问权；auto_run 闭环在 VM 内完成，不降低「须有效 token 才能读 task-detail」门槛。

## 结论

- **无需新增权限模型或 Django 路由。**
- 权限边界核心：**容器仅凭 container access token 只读 task-detail**；Git 身份由 **Go 解析平台身份表后下发**，容器不得绕过写任意 email。
- 交付失败、身份缺失均为**可观测跳过/降级**，不得阻断 onlineServiceJS 主服务或其它 job 路径。
