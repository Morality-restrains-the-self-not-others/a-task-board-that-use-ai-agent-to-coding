# task2app 出站调用治理落地切片（/3-value-stream 输入）

## 受影响 value stream
- `internal-api-timeout-governance`（新增）
- `oauth-token-fetch-timeout-governance`（保持 OAuth async 为 planned）
- `cloud-integration`（新增 `github-pr-create-async` active）
- `task-detail-runtime-relay`（后续长任务异步化依赖）
- `relay-status-push-timeout-go-relay`（后续 ACK/超时耦合梳理）

## 迁移顺序（identify-migration-order）
1. **OAuth 内网桥接调用收口**  
   - 范围：`accounts/github_app_tokens.py`  
   - 动作：接入统一 `internal_sync` 网关，强制 1 秒超时与统一异常映射。
2. **外网异步样板（GitHub PR）**  
   - 范围：`cloud/services/github_pull_request_after_push.py`  
   - 动作：将直接 GitHub API 调用改为 `external_async` 入队执行，结果通过 SSE 事件推送。
3. **cloud-integration 其余外网接口**  
   - 范围：云平台认证、VPC 查询、镜像列表等外网调用点  
   - 动作：逐步改造为异步任务执行 + 事件推送；保留查询兜底。
4. **横向治理与收尾**  
   - 动作：扫描并下沉散落 `requests.*` 直连调用，默认纳管到 `internal_sync` / `external_async` / `streaming` 三策略。

## 测试策略（draft-test-strategy）
1. **同步超时契约测试**  
   - 文件：`tests/core/test_outbound_gateway.py`  
   - 目标：验证内网同步调用超时上限强制为 1 秒，超时映射为统一异常码。
2. **异步回传测试**  
   - 文件：`tests/test_github_pr_after_layer_push_async.py`  
   - 目标：验证外网 PR 创建改为入队返回 `job_id`，而非阻塞同步请求。
3. **流式豁免与回归**  
   - 文件：`tests/test_ai_task_comment.py`、`tests/test_relay_to_trae_status.py`（既有）  
   - 目标：确保流式/状态推送链路不被同步 1 秒规则误杀。

## 说明
- 当前版本先完成“统一网关骨架 + 两条关键链路改造（gitoauth 内网同步、GitHub PR 外网异步）”。
- OAuth 外网 async（`oauth-async-job-future`）保持 planned，下一增量承接。
