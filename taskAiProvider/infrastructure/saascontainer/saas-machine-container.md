# SaaS Machine Container Skill（容器 → SaaS）

> **接口版本**: **v1**（当前）。厂商登记容器镜像时必须选择本版本号。已发布目录见 [versions.yaml](./versions.yaml)。

容器 `onlineServiceJS` 出站调用的 SaaS 接口清单。实现以 `taskCredentialService` / `taskCloudService` 为准；本文取代已删除的 `task2app/Saas_project/skillList/machine_container.md`。§5.2 另列**浏览器**在层图推送生成 PR 后调用的评论与一键合并 API（会话用户，不是 `access_token`）。

反向（SaaS/浏览器 → 容器）见 [`trae-agent/onlineServiceJS/skill.md`](../../../trae-agent/onlineServiceJS/skill.md)（容器 `GET /skill.md`）。浏览器 compute 转发 path 约定见 [ADR-0010](../../adr/0010-comment-id-path-kv.md)。PR 回复与远端合并见 [ADR-0028](../../adr/0028-pr-reply-one-click-merge-audit.md)。

## 1. 前缀与鉴权

UserData / 环境注入：

| 变量 | 说明 |
|------|------|
| `TaskApiEndPoint` / `TASK_API_ENDPOINT` | 必须 `https://<api>/api/tenant/{tid}/workspace/{wid}/task/{taskId}/comment/{cid}/cloud`。禁止无 cid 的旧值 `…/task/{taskId}/cloud` |
| `ACCESS_TOKEN` | 容器访问令牌；出站 JSON 字段 `access_token` |
| `COMMENT_ID` / `CONTAINER_NAME` | 评论级 CSC；UserData 写入 path 的 `{cid}`，并合并进 body（`saasInboundScope.mjs`） |
| `tenantId` / `workspaceId` / `taskId` | 可从 `TaskApiEndPoint` 路径解析；`commentId` 从 `/comment/{cid}/` 或 `COMMENT_ID` 解析 |

所有下表接口均为 **POST JSON**，body 必填 `access_token`。令牌与 URL 中的 tenant/workspace/task 必须一致，否则 403。

`comment_id`：容器 → SaaS **必须**出现在 **path**（`/comment/{cid}/`，位置段）。无该段的 `/task/{id}/cloud/...` 网关与 AgentSupport / Credential **不匹配（404）**。JSON body 的 `comment_id` 仅作双重校验，不能替代 path。缺省时 Cloud `resolveInboundCommentCSC` 在两评论场景会失败。

## 2. taskCredentialService（`server-container-token` 凭证面）

路径：`POST {TaskApiEndPoint}/server-container-token/{action}/`

| action | 用途 | 主要请求字段 | 成功要点 |
|--------|------|--------------|----------|
| `exchange-refresh` | 启动换 refresh | `access_token` | 清空 access，写入 refresh |
| `refresh-access` | 用 refresh 换新 access | `refresh_token` 或当前 token | 新 `access_token` |
| `task-detail` | 引导克隆输入 | `access_token` | 含 `project_repos` / 分支计划；`idle_recycle_minutes` 与 `instruction_idle.{enabled,minutes}`（无策略行默认 5）；可选 `machine_release_sts`（无 RAM Role 则省略，禁止写入评论） |
| `repo-clone-credentials` | 每仓克隆凭证 | `access_token` | `repo_clone_credentials`；缺身份 409 |
| `layer-github-oauth-access-tokens` | 层内 GitHub 推送票 | `access_token`、`layer_id` | 按仓 OAuth token |

路由实现：`taskCredentialService/interfaces/handlers.go` `handleContainerAPI`。

## 3. taskCloudService（`server-container-token` 运行面）

同一路径前缀，由 Cloud `handleContainerInboundToken` 分发：

| action | 用途 |
|--------|------|
| `register-reachability` | 写入 `server_url` / `business_api_endpoint`（须在长克隆前） |
| `heartbeat` | 容器存活 → 任务详情 SSE `container_heartbeat` |
| `git-clone-progress` | 引导/单仓克隆进度 |
| `boot-progress` | 云主机/容器启动进度 |
| `runtime-event` | 运行时事件（软失败） |
| `layer-graph-push` | 层图快照上报 |
| `layer-changes-push` | 层文件变动 |
| `feature-params-env` | 拉取功能参数 env（写入 `process.env` + yaml） |
| `request-machine-release` | 容器请求释放机器 |

## 4. 非 `server-container-token` 的 scoped inbound

仍在任务 cloud 前缀下，鉴权同令牌：

| 路径后缀 | 用途 |
|----------|------|
| `relay-to-trae/status-push/` | relay 状态推送（高频，Cloud 收敛后 SSE） |
| `model-budget-usage/` | 模型预算用量 |

## 5. 浏览器侧（不走容器 inbound）

### 5.1 浏览器 → 容器 compute

任务详情拉层图/日志/文件树走：

```
GET|POST /api/cloud/compute/{container-*}/tenant_id/{t}/workspace_id/{w}/task_id/{task}/comment_id/{cid}/
```

Cloud `isContainerOutboundComputeSub` 原样代理 `taskContainerGateway`；网关按 CSC `(task, comment)` 选实例再调容器 `skill.md` 所列 API。`layer_id` / `job_id` 留 query。

容器 `git-clone-progress` 由 Cloud 转为任务 SSE `container_git_clone_progress`（`repo_url` + 规范化 `segment`）。

层图「合并到目标分支」（`container-layer-git-merge`）是**容器本地 git merge**，不是合并 GitHub/GitLab 上的 PR/MR。

### 5.2 浏览器 → SaaS：PR 回复与一键合并

层图 `layer-ztree-pr-btn` 拿到 PR/MR `html_url` 后，任务详情用**当前登录用户**调用下列接口（ADR-0028）。鉴权：会话 + 租户成员。`html_url` 仅接受 GitHub `/pull/N`、GitLab `/-/merge_requests/N`，且 host 须为已配置 provider 或 `github.com`。

| 方法 | 路径 | Owner | 请求 | 成功要点 |
|------|------|-------|------|----------|
| POST | `/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/` | taskTaskService | `content`（PR URL）、`git_pr: {html_url, provider}`、`execution_mode=independent`。**parent 在 path**，不要靠 body `parent_comment_id` | 同 `task_id`+`git_pr_html_url` 已存在则 200 返回已有评论；首次插入发布 `TASK_GIT_PULL_REQUEST_RECORDED`。workspace 与任务不符 404；父评不存在或不属于该任务 400 |
| POST | `/api/git-oauth/merge-request-status/tenant_id/{tid}/` | taskGitOauth | `{html_urls: string[]}`，最多 20 | `{results:[{html_url, state, title?}]}`；`state` 为 `open` / `merged` / `closed` / `unknown` |
| POST | `/api/git-oauth/merge-request-merge/tenant_id/{tid}/` | taskGitOauth | `{html_url, task_id, comment_id}`；前端须带 `Idempotency-Key` | `{ok, merged, noop, state: "merged"}`；已 merged 为成功 noop。审计 `action=merge_request_merge` 写入 `git_oauth_taskcredentialaudit` 与 `git_oauth_appaccesstokenuseaudit`（`user_id` + url，不写 token 明文）。发布 `GIT_MERGE_REQUEST_MERGED` |

未绑定 Git OAuth：状态查询项为 `unknown`；一键合并 **409**。实现：`taskTaskService` 评论创建；`taskGitOauth` `handleMergeRequestStatus` / `handleMergeRequestMerge`。

## 6. 访问日志

Django `container_machine_api_access_middleware` 已退役。Go 服务用 `tracelog`（禁止记录 `access_token` / `refresh_token`）。原则见 `.ai/01_project_constraints/15_container_machine_api_access_logging.md`。

## 7. 镜像内技能列表（`/app/imageSkills.yaml`）

本文件与 **SaaS HTTP inbound 版本**（ADR-0024 `saas_inbound_skill_version`）正交。本节约定镜像内 **Agent 技能目录**：上传到 https://provider.daydaymoney.com/ 时由平台从 OCI 层抽取并绑定到该镜像版本。

### 7.1 文件位置与格式

- **路径（强制）**：`/app/imageSkills.yaml`（与 `/app/autoRunStep.md` 同层）
- **编码**：UTF-8 YAML，`version: 1`
- **顺序**：`skills` 为有序列表；**第一项为默认技能**（不要另写 `default:` 字段）
- **上限**：最多 32 项；`name` 必须匹配 `^[a-z0-9][a-z0-9-]{0,62}$`；`description` ≤ 512 字符
- **缺文件**：镜像仍可上架，`extract_status=not_found`，市场不展示技能列表
- **非法 YAML / 重名 / 非法 name**：抽取 `failed`，不绑定垃圾数据

```yaml
version: 1
skills:
  - name: general-coding
    description: 本镜像默认软件开发流程
  - name: k8s-debug
    description: Kubernetes 排障
```

Dockerfile 示例：`COPY imageSkills.yaml /app/imageSkills.yaml`

### 7.2 运行时选用

| 入口 | 行为 |
|------|------|
| 镜像市场 | 查看绑定后的技能列表；第一项标「默认」 |
| 创建任务 | 选择镜像后，任务描述中的 `/技能名` 高亮；可点芯片插入 |
| 评论 `@镜像` | 可继续输入 `/{技能}`；未指定则使用列表第一项 |
| 容器进程 | 环境变量 `IMAGE_SKILL=<name>`（所选或默认） |

评论示例：`@trae-agent /k8s-debug 请查 CrashLoop`

未知技能：描述中不高亮；评论提交返回 400。

实现见 ADR-0026。

## 8. 变更清单

| 日期 | 变更 |
|------|------|
| 2026-08-22 | §5.2 回复路径改为 `/api/tenant_id/{tid}/workspaceId/{wid}/tasks/{taskId}/comments/{parent_comment_id}/`；顶层发评旧路径仍保留 |
| 2026-08-22 | §5.2 列出推送生成 PR 后的评论落库、合并状态查询与一键合并（ADR-0028）；不改变容器 inbound 版本 |
| 2026-08-21 | 镜像内 `/app/imageSkills.yaml` 技能列表：第一项为默认；上传抽取绑定；`IMAGE_SKILL`；ADR-0026 |
| 2026-08-16 | SaaS HTTP inbound 必须 `/comment/{cid}/`；无 comment 段的 `/task/{id}/cloud/...` 404 |
| 2026-08-16 | `TaskApiEndPoint` **必须**含 `/comment/{cid}/`；不再兼容旧 `…/task/{taskId}/cloud` |
| 2026-08-16 | 从退役 Django skill 按 Go 实现重建；交叉引用容器 `skill.md` 与 ADR-0010 |
| 2026-07-30 | Django `task2app` 退役，原 `machine_container.md` 从仓库消失 |
