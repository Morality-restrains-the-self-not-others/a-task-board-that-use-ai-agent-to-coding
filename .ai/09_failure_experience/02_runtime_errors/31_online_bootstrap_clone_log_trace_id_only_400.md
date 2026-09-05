# [运行时] onlineServiceJS `GET /api/repos/bootstrap-clone-log` 仅带 X-Trace-Id 返回 400

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 容器控制台 UI 轮询 `GET /api/repos/bootstrap-clone-log` 时 HTTP **400 Bad Request**（约数十 ms）。
- 响应体：
  `{"detail":"trace propagation incomplete: require X-Parent-Span-Id or traceparent with X-Trace-Id"}`
- 请求头含 `X-Trace-Id`（UUID），**无** `X-Parent-Span-Id` / `traceparent`。
- Referer 为 `/ui/tenant/.../task/.../{access_token}`（onlineServiceJS 内置控制台）。

## 根因

1. `src/server.mjs` 的 `traceMiddleware` 对「只带 `X-Trace-Id`、不带 parent span / traceparent」的请求一律 400（`isTraceIdOnlyRequest`）。
2. `static/index.html` 的 `api()` 只设置了 `X-Trace-Id`，未生成并发送 16-hex 的 `X-Parent-Span-Id`。
3. 业务接口本身（bootstrap-clone-log）在补全传播头后可正常返回 200（如 `{"layer_id":null,"text":""}`）。

## 解决方案

- 控制台 `api()`：每次请求生成 `generateRequestSpanId()`（8 字节 → 16 hex），一并发送 `X-Parent-Span-Id`。
- 单测：`src/traceId.test.mjs` 覆盖 `isTraceIdOnlyRequest` 的 bare / +Parent / +traceparent 分支。
- 文档：`skill.md` 注明该接口（及所有经 `api()` 的调用）须完整传播头。

## 预防

- 浏览器/前端凡主动写 `X-Trace-Id`，必须同时写 `X-Parent-Span-Id`（16 hex）或合法 `traceparent`。
- 新增出站 HTTP 客户端时优先复用 `traceHeadersForOutbound` / `traceHeadersFromRequest`。
- 调试 400 时先看响应 `detail` 是否为 trace propagation，再查业务参数。

## 验证

```bash
# 单测
cd trae-agent/onlineServiceJS && node --test src/traceId.test.mjs

# 本地复现（需服务已起）
curl -sS -D- 'http://127.0.0.1:PORT/api/repos/bootstrap-clone-log?access_token=TOKEN' \
  -H 'X-Access-Token: TOKEN' \
  -H 'X-Trace-Id: 71c09823-3d45-4777-88a9-9f57610805a1'
# → 400

curl -sS -D- 'http://127.0.0.1:PORT/api/repos/bootstrap-clone-log?access_token=TOKEN' \
  -H 'X-Access-Token: TOKEN' \
  -H 'X-Trace-Id: 71c09823-3d45-4777-88a9-9f57610805a1' \
  -H 'X-Parent-Span-Id: b1b2c3d4e5f67890'
# → 200
```

部署：源码修复后需重建并推送 onlineServiceJS 镜像（`DOCKER_PUSH=1 ./buildDocker.sh`，见目录 `ai.md`），任务容器拉新镜像后 UI 轮询即恢复。
