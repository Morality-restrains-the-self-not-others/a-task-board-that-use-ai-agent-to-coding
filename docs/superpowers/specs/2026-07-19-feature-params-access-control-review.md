# Feature-Params 访问控制 — Review

## 对照计划

| 任务 | 状态 |
|------|------|
| Audit 模型 + migration | ✅ |
| feature_params_access 服务 | ✅ |
| 公司/工作空间 API 门禁 | ✅ |
| Django + 前端测试 | ✅ 31 + 5 passed |
| intents（审计无 MQ 例外） | ✅ |
| 行数门禁 | ✅ feature_views 拆分为 294/245 |

## Intent→Event

书面例外：审计落库 SSOT，无 MQ。

## Log Audit

- denied → `logger.warning` + DB
- full/summary → `logger.info` + DB
- 审计失败 exception 不阻断主响应

## 残留风险

伪造 `X-Feature-Params-Access-Context` 仍可取 full（成员本可在设置页看见密钥）；靠审计追责。后续可收紧为仅 admin full（见 OPT）。
