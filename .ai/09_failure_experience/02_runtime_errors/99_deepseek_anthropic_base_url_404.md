# [运行时] DeepSeek provider 配 Anthropic base_url → 永久 Error code: 404

## 现象

任务详情评论执行日志（启动 TraceId 可关联 SSE）：

```text
DeepSeek API call failed: Error code: 404. Will sleep for 1.2 seconds and will retry.
step 1: 错误: Error code: 404
```

示例 TraceId：`9257e326bcadc9405df4b150`  
租户功能参数：`provider=deepseek` + `base_url=https://api.deepseek.com/anthropic`

## 根因

1. DeepSeek 官方同时提供 **OpenAI 兼容**（`/v1/chat/completions`）与 **Anthropic 兼容**（`/anthropic/v1/messages`）端点。
2. Trae `DeepSeekClient` 走 OpenAI SDK `chat.completions.create` → 请求 `{base_url}/chat/completions`。
3. 当 `base_url` 误配为 `https://api.deepseek.com/anthropic` 时，实际命中  
   `https://api.deepseek.com/anthropic/chat/completions` → **HTTP 404**（Anthropic 路径不存在该路由）。
4. 同窗口常伴随 `npx @playwright/mcp` 拉取 `registry.npmjs.org` **ETIMEDOUT**（默认 MCP 配置；与 404 独立，但污染启动日志）。

复现（密钥省略）：

```bash
# 404
curl -s -o /dev/null -w '%{http_code}\n' -X POST \
  'https://api.deepseek.com/anthropic/chat/completions' ...
# 200
curl -s -o /dev/null -w '%{http_code}\n' -X POST \
  'https://api.deepseek.com/v1/chat/completions' ...
```

## 现网策略（2026-08-18）

运营商端点写法多样，**写路径不再把 `/anthropic` 改写成官方 `/v1`**。设置页按用户填写原样保存；仅修复两个 `https://` 粘连的脏数据。若官方 DeepSeek + OpenAI 客户端仍配 `/anthropic`，运行时 404 由用户自行改端点。

## 修复（历史；已部分撤销）

1. **数据**：当时可将租户 `cloud_tenant_feature_params.providers[].base_url` 改为 `https://api.deepseek.com/v1`（仅当确认走 OpenAI 兼容客户端）。
2. **写路径**：不再按 provider 改写端点。`normalizeProviderBaseURL` / `coerceProvidersBaseURLsInEnv` 只拆粘连的绝对 URL。
3. **容器 YAML**：`featureParamsEnvToYaml` 同步保留用户路径；默认关闭 `@playwright/mcp`（`TASK_ENABLE_PLAYWRIGHT_MCP=1` 才启用）。
4. **代理**：`taskAIEndPoint.ensureUpstreamV1Base` 避免上游 `.../v1` 再拼 `/v1` 成 `.../v1/v1/...`（不改写用户保存的 path）。

## 验证

```bash
cd taskCloudService && go test ./src/ -run 'TestNormalizeProvider|TestCoerceProviders'
cd taskAIEndPoint && go test ./src/ -run 'TestEnsureUpstream'
cd trae-agent/onlineServiceJS && node --test src/featureParamsEnvToYaml.test.mjs
```

现网：精准编译重启 `task-cloud-service` / `task-ai-endpoint`；推送 trae online 镜像后重建任务容器并重新跑 job。
