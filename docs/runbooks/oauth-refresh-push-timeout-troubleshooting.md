# OAuth Refresh Push Timeout Troubleshooting

## 1) 现象与典型报错

- 容器侧调用 `POST /api/layers/<layer_id>/git/oauth-refresh-push` 失败，接口返回 `502`。
- 常见响应体：
  - `{"ok":false,"detail":"拉取任务详情失败：fetch failed"}`
  - 或链路中出现 `This operation was aborted`（通常对应上游调用超时被中断）。
- 关联日志常见形态：
  - `gitPush.log` 出现 `oauth-refresh-push begin ...` 与 `oauth-refresh-push fail ...`
  - `reqLogs/outbound.log` 出现 `layer-github-oauth-access-tokens -> error This operation was aborted 6000xms`

## 2) 三层排查顺序（必须按顺序）

### 第一层：`gitPush.log`（入口与阶段）

先确认请求是否进入容器端处理逻辑，以及失败发生在哪个阶段：

```bash
rg "oauth-refresh-push" trae-agent/onlineProject_state/logs/gitPush.log
```

重点看是否包含：

- `oauth-refresh-push begin layer_id=...`
- `oauth-refresh-push token-fetch ... timeout_sec=...`
- `oauth-refresh-push done ...` 或 `oauth-refresh-push fail ...`

### 第二层：`reqLogs/outbound.log`（出站请求耗时与中断）

定位容器发往 task2app 的出站请求是否超时/中断：

```bash
rg "layer-github-oauth-access-tokens|task-detail|aborted|fetch failed|timeout" trae-agent/onlineProject_state/reqLogs/outbound.log
```

重点判定：

- `task-detail` 是否成功（HTTP 200）
- `layer-github-oauth-access-tokens` 是否出现 `aborted` / `fetch failed`
- 耗时是否接近超时阈值（如 `60004ms`）

### 第三层：task2app logs（绑定与换票业务）

确认 task2app 侧是否存在绑定缺失、换票失败或上游超时。常用检索关键词：

- `LayerGithubOauthTokens`
- `container-layer-github-oauth-access-tokens`
- `task binding`
- `timed out` / `timeout`

默认先自动定位 `task2app` 最新日志文件（可直接执行）：

```bash
TASK2APP_LOG_FILE="${TASK2APP_LOG_FILE:-$(ls -t task2app/Saas_project/logs/*.log 2>/dev/null | head -n 1)}"
if [ -z "$TASK2APP_LOG_FILE" ]; then
  TASK2APP_LOG_FILE="$(ls -t task2app/Saas_project/*.log 2>/dev/null | head -n 1)"
fi
echo "$TASK2APP_LOG_FILE"
```

然后检索（也可手动替换 `<TASK2APP_LOG_FILE>`）：

```bash
rg "LayerGithubOauthTokens|container-layer-github-oauth-access-tokens|task binding|timed out|timeout" <TASK2APP_LOG_FILE>
```

## 3) 常见原因与处理

### 原因 A：上游超时（最常见）

特征：

- `reqLogs/outbound.log` 中 `layer-github-oauth-access-tokens` 或相关接口耗时接近阈值后 `aborted`
- 容器接口最终返回 `502`

处理：

1. 优先确认 task2app 后端和依赖（DB/GitHub API）响应时间。
2. 检查并适当提高容器侧 OAuth 拉取超时配置（确保在允许范围内）。
3. 复测并对比 `outbound.log` 耗时是否明显下降或不再中断。

### 原因 B：网络异常

特征：

- `fetch failed`、连接重置、DNS/路由异常等
- 出站请求在很短时间内失败（非稳定超时）

处理：

1. 先验证容器到 task2app 的网络连通性与 DNS。
2. 复查代理/防火墙配置（尤其是容器网络策略）。
3. 重试后确认错误是否从网络类转为业务类（说明链路恢复）。

### 原因 C：任务绑定缺失（仓库 OAuth 绑定不存在）

特征：

- task2app 侧日志出现任务未绑定/仓库无授权信息
- 容器侧可能表现为 `no github_auth_by_repo` 或业务性失败 detail

处理：

1. 在 task2app 确认任务与仓库 OAuth 绑定关系完整。
2. 修复绑定后重新执行 `oauth-refresh-push`。
3. 观察 `gitPush.log` 是否从 `fail` 变为 `done`。

## 4) 任务页显示 `onlineServiceJS` 未启动但 `8765` 可访问

当任务页显示 `onlineServiceJS` 未启动，但你手工访问 `127.0.0.1:8765` 仍可联通时，通常是 relay 状态推送目标地址选择不一致或本地前置服务未监听。

### A. 环境前置检查：确认 `127.0.0.1:8001` 正在监听

先确认 Django 本地回环端口存在监听（否则 relay 推送状态到本地 API 会直接超时）：

```bash
lsof -nP -iTCP:8001 -sTCP:LISTEN
```

若无输出，先启动/修复本地 Django 服务，再继续后续排查。

### B. relay status 快照命令

抓取 relay 当前状态快照，确认 relay 自身是否健康、以及状态面是否可读：

```bash
curl -sS -H "X-Relay-To-Trae-Secret: dev-secret" http://127.0.0.1:8797/v1/status
```

### C. 出现 `status push timeout` 时检查目标地址选择

若日志中连续出现 `status push failed ... timeout`，需要检查 `get_relay_task_api_base_url()` 的实际选择值是否符合部署拓扑（可在运行配置和相关日志中核对）：

1. 确认当前配置中的 `django.internalApiBase`、`vue.apiBaseUrl`。
2. 对照 `get_relay_task_api_base_url()` 的选择逻辑，判断当前推送目标是否被选成了错误 origin。
3. 重点排除“relay 在推送本地回环地址，但前端任务页实际读取公网 API 网关”的错配。

### D. 选择规则说明（本地 Django 回环 + 前端公网 API 网关）

当部署形态是“本地 Django 仅回环可达（如 `127.0.0.1`）+ 前端通过公网 API 网关访问后端”时，relay 状态上报应优先使用 `vue.apiBaseUrl`，避免写入到仅本机可见的回环地址导致前端侧状态不同步。

## 5) 验证命令（验收/回归）

### A. 手工 API 验证（curl）

```bash
curl -i -X POST "http://127.0.0.1:8765/api/layers/<layer_id>/git/oauth-refresh-push" \
  -H "X-Access-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{}'
```

期望（同时覆盖正例/负例）：

- 正例：`X-Access-Token` 为有效 token 时，返回 `2xx` 且 `ok=true`
- 负例：`X-Access-Token` 为无效/缺失 token 时，返回 `401/403` 且错误信息可解释（如 `Invalid or missing access token`）

### B. 日志链路校验（rg）

复用第 2 节“三层排查顺序”中的同一组 `rg` 命令作为验收，不再重复维护两套检索指令。

### C. 最小回归测试（pytest + node test）

```bash
(cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py -q)
(cd trae-agent/onlineServiceJS && node --test src/layerGitOauthRefreshPush.test.mjs)
```

## 6) 本次执行证据（2026-05-21）

### A. 手工接口验证

- 负例请求：
  - `POST /api/layers/layer-x/git/oauth-refresh-push`
  - Header: `X-Access-Token: invalid-token`
- 结果：
  - HTTP `401`
  - 响应体：`{"detail":"Invalid or missing access token"}`

### B. 日志链路验证

- `logs/gitPush.log` 命中示例：
  - `oauth-refresh-push begin layer_id=layer-x`
  - `oauth-refresh-push fail layer_id=layer-x detail=缺少容器 ACCESS_TOKEN`
- `reqLogs/outbound.log` 命中示例：
  - `layer-github-oauth-access-tokens/ -> error This operation was aborted 60004ms`
  - `task-detail/ -> error fetch failed 1ms`

### C. 回归测试结果

- `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py -q`
  - 结果：`3 passed`
- `cd trae-agent/onlineServiceJS && node --test src/layerGitOauthRefreshPush.test.mjs`
  - 结果：`5 passed`
