# [运行时] AI Provider 前端仅发 X-Trace-Id 被 tracelog 400，且 p.msg 无 data-traceId

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 页面：`https://provider.daydaymoney.com/`（标题「AI 容器镜像市场」）
- 可见错误文案：`trace propagation incomplete: require X-Parent-Span-Id or traceparent with X-Trace-Id`
- 承载元素为 `p.msg`，**无** `data-traceId` 属性，无法从 DOM 一键追到后端日志。

## 根因

1. `shareLib/tracelog.Middleware` 经 `RejectTraceIdOnlyHTTP` 拒绝「只带 `X-Trace-Id`、不带 `X-Parent-Span-Id` / `traceparent`」的请求（HTTP 400）。
2. `taskAiProvider/frontend/src/api.js` 主动写入 `X-Trace-Id`，但未生成并发送 16-hex 的 `X-Parent-Span-Id`。
3. `VendorPortal` / `AdminPortal` / `VendorCloudCredentialsTab` 的 `p.msg` 在展示请求错误时未绑定 `data-traceId`（与元规则 24 不一致）。

## 解决方案

- `traceId.js`：`newRequestSpanId` + `buildOutboundTraceHeaders`；`api.js` 每次请求附带 `X-Parent-Span-Id`。
- 各门户 `p.msg`：请求失败时写入 `msgTraceId` 并 `v-bind` `data-traceId`。
- `RejectTraceIdOnlyHTTP`：拒绝时回显入站 `X-Trace-Id`，便于响应头与前端兜底对齐。
- 前端单测覆盖 span id 与 outbound headers；`npm run build` 产出新 dist。

## 预防

- 浏览器凡主动写 `X-Trace-Id`，必须同时写 `X-Parent-Span-Id`（16 hex）或合法 `traceparent`（与 onlineServiceJS / FE-31 同模式）。
- 新增请求错误 UI 时同步挂 `data-traceId`；禁止只写文案。
- 调试 Provider 400 时先读 `detail` 是否为 trace propagation，再查业务参数。

## 验证

```bash
# 前端单测
cd taskAiProvider/frontend && npm run test:unit

# 生产 API（修复前）
curl -sS -D- 'https://provider.daydaymoney.com/api/public/catalog/' \
  -H 'X-Trace-Id: 71c09823-3d45-4777-88a9-9f57610805a1'
# → 400 + detail=trace propagation incomplete

# 修复后客户端行为（须带 Parent-Span）
curl -sS -D- 'https://provider.daydaymoney.com/api/public/catalog/' \
  -H 'X-Trace-Id: 71c09823-3d45-4777-88a9-9f57610805a1' \
  -H 'X-Parent-Span-Id: b1b2c3d4e5f67890'
# → 200
```

部署：提交后需将 `taskAiProvider/frontend/dist`（及若需重启的 Go 二进制）发布到 `provider.daydaymoney.com`，否则线上仍跑旧 bundle。
