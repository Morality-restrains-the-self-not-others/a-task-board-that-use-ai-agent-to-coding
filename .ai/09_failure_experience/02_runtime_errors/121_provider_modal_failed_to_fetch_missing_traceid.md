# [运行时] 镜像市场弹层 Failed to fetch 无 data-traceId

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-28
- 最后修改：2026-08-28
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/`（标题「AI 容器镜像市场」）
- 选择器：`div.modal-bg > div.card.modal > p.err`
- 可见文案：`Failed to fetch`
- HTML：`<p class="err">Failed to fetch</p>`，**无** `data-traceId`
- 与 [33_provider_frontend_trace_id_only_400.md](./33_provider_frontend_trace_id_only_400.md) 不同：此处是浏览器 `fetch` **抛错**（未拿到 HTTP 响应），不是 tracelog 400。

## 根因

1. `taskAiProvider/frontend/src/api.js` 在出站前已用 `buildOutboundTraceHeaders` 生成 `X-Trace-Id`。
2. `const res = await fetch(...)` 在 CORS/断网/DNS 失败时抛 `TypeError("Failed to fetch")`，**不会**走到后面 `err.traceId = responseTraceId`。
3. 各弹层 `catch` 写 `e.message` + `e.traceId || ""`；`traceId` 为空则 Vue 省略 `data-traceId`（符合「确无 id 则省略」，但本端其实已有请求级 id）。
4. 同类：`taskChromePlugin/lib/api.js` 的 `request` / `requestUnauthenticated` / `uploadPluginScreenshot`。`taskFE` `apiFetch` 已在 catch 里 `attachTraceIdToError(..., requestTraceId)`。

## 解决方案

- `attachClientTraceId(err, requestTraceId)`：网络失败把本端已发的 `X-Trace-Id` 挂到 Error。
- `api()` 包裹 `fetch`；弹层仍展示 `Failed to fetch`，DOM 带 `data-traceId`。
- Chrome 插件 `fetchWithClientTrace` 同样挂 id。
- 单测：`frontend/tests/api.unit.test.js`、`traceId.unit.test.js`；插件 `test/api-endpoints.test.js` T10。

## 预防

- 凡封装 `fetch` 且事先生成 `X-Trace-Id`，**throw 路径必须带上该 id**；禁止只在 `!res.ok` 分支赋值。
- 禁止用 `"unknown"` 占位；纯前端校验不得伪造 `data-traceId`。

## 验证

```bash
cd taskAiProvider/frontend && npm run test:unit -- api.unit.test.js
cd taskChromePlugin && node --test ./test/api-endpoints.test.js
```

部署：`bash taskAiProvider/run.sh build` 或 9999「精准编译重启」`ai-provider` 后硬刷新 `https://provider.daydaymoney.com/`。
