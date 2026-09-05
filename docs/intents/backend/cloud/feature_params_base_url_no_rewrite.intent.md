# 意图：feature-params API 端点按用户填写原样保存

## 背景与目标

租户设置页「API 端点(base_url)」保存后出现 URL 被覆写。现网租户 `877397588196749312` 落库值为  
`https://api.deepseek.com/v1https://api.deepseek.com`（默认预填与粘贴叠成两个绝对 URL）。

运营商多种多样，**不得**按 provider 名把用户填写的端点改写成官方 DeepSeek `/v1` 或其它规范路径。

目标：保存/读回/下发均保留用户填写的 `base_url`；仅修复两个 `https://` 粘在一起的脏数据。

## 范围与边界

- 范围内：`normalizeProviderBaseURL` 写/读/coerce；设置页编辑器；trae YAML 适配。
- 范围外：不限制路径写法（含 `/anthropic`、自定义网关）；不预填官方 URL。

## 约束与风险

- 用户填写 `/anthropic` 时，OpenAI 兼容客户端仍可能 404；由用户自行选择端点，产品不改写。
- 两个绝对 URL 粘连视为输入事故，保留最后一段。

## 验收标准

1. `provider=deepseek` + `base_url=https://api.deepseek.com/anthropic` 保存后原样读回。
2. 自定义网关 URL 保存后原样读回。
3. `…/v1https://…` 粘连在 GET/POST/env coerce 后变为最后一段绝对 URL。
4. 改供应商名不预填、不覆盖已填 `base_url`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 保存 feature-params API 端点 | — | — | — | — | 同服务配置写库，无跨边界副作用 |

## 实施计划

1. 去掉按 DeepSeek 官方路径改写；保留粘连修复。
2. 去掉前端预填与 `/anthropic` 告警改写文案。
3. YAML 适配同步：不改写运营商路径。

## 变更记录

- 2026-08-18：不限制端点写法；只修粘连脏数据。
