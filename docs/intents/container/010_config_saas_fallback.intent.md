# 容器配置拉取本地 miss 时回源 SaaS

## 意图

onlineServiceJS 控制台「拉取当前配置」时：若本地尚无 `service_config.yaml`，应回源 SaaS `feature-params-env` 落盘后再返回；仍失败时 `#cfgErr` 引导按 `traceId` / `data-traceId` 排查，不再提示「请先上传配置」。

## 验收要点

- [x] `GET /api/config` 本地命中 → `source: local`
- [x] 本地缺失 → 调用 `…/server-container-token/feature-params-env/` 并 `persistFeatureParamsEnv` → `source: saas`
- [x] 回源不可用/失败 → 404/502，响应带 `X-Trace-Id`
- [x] `#cfgErr` 文案含「请根据traceId 寻找原因」，且挂载 `data-traceId`

## 业务意图 → 事件对照

**无对应事件**：配置回源为容器内同步读路径，不新增业务领域事件；沿用既有出站 `reqLogs` 与 trace 透传。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 配置本地 miss 回源 SaaS | — | — | — | 同步读路径，无新增 MQ 事件 |
