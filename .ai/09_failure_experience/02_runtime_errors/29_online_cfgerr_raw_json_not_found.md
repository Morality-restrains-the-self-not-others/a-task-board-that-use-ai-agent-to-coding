# [运行时] onlineServiceJS 控制台 #cfgErr 展示原始 JSON `{"detail":"not found"}`

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- Trae Online Service 控制台「推送配置」区 `#cfgErr` 在点击「拉取当前配置」后显示 `{"detail":"not found"}`，或误导为「请先上传配置」。
- 用户无法立刻理解应如何按链路排查；且本地无文件时未回源 SaaS。

## 根因

1. `GET /api/config` 在 `service_config.yaml` 不存在时直接 `404 { detail: "not found" }`，未尝试 SaaS `feature-params-env`。
2. 前端将 `await r.text()` **原样**写入 `#cfgErr`，未解析 `detail`、未加「拉取配置失败」前缀；后续虽有可读映射，但文案指向「请先上传」且未挂 `data-traceId`。

## 解决方案

- `GET /api/config`：本地缺失时经 `readOrPullServiceConfig` 回源 SaaS 并落盘；仍失败则 404/502，响应头带 `X-Trace-Id`。
- 控制台用 `formatConfigErrMsg`（`src/formatConfigErrMsg.mjs` + `static/index.html` 内联同逻辑）将拉取失败映射为：
  `拉取配置失败，请根据traceId 寻找原因`（或附带 detail 的变体）。
- `#cfgErr` 写入失败文案时同步设置 `data-traceId`（与 FE-31 / 元规则 24 一致）。

## 预防

- 凡 `#*Err` / `.err-msg` 展示 API 错误体时，优先解析 `detail`/`message` 并加操作语义前缀；禁止直接 dump 原始 JSON。
- 配置读取路径：本地 miss → SaaS 回源 → 再对用户报错。
- 回归：`node --test test/formatConfigErrMsg.test.mjs test/ensureServiceConfig.test.mjs`。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test test/formatConfigErrMsg.test.mjs test/ensureServiceConfig.test.mjs
# 无本地配置时点「拉取当前配置」：应先回源 SaaS；仍失败则 #cfgErr 含 traceId 指引且带 data-traceId
```
