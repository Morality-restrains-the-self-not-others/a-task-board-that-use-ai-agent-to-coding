# 容器配置拉取本地 miss 时回源 SaaS — 测试意图

| 场景 | 测例 | 期望 |
|------|------|------|
| 本地已有配置 | `test/ensureServiceConfig.test.mjs` | 不调用 SaaS，`source=local` |
| 本地缺失回源 | 同上 | 调用 feature-params-env 并 persist，`source=saas` |
| 无 TaskApi 前缀 | 同上 | `SAAS_CONFIG_UNAVAILABLE` |
| SaaS 缺 env | 同上 | 失败，不静默成功 |
| not found 文案 | `test/formatConfigErrMsg.test.mjs` | `拉取配置失败，请根据traceId 寻找原因` |

```bash
cd trae-agent/onlineServiceJS && node --test test/formatConfigErrMsg.test.mjs test/ensureServiceConfig.test.mjs
```
