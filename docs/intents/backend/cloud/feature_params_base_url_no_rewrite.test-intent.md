# 测试意图：feature-params API 端点按用户填写原样保存

## 测试目标

验证设置页/写路径/env coerce 不按运营商改写 `base_url`；仅拆开两个绝对 URL 粘连。

## 测试分层

- 单元：`taskCloudService/src/feature_params_public_handlers_test.go`、`feature_params_base_url_test.go`、`feature_params_proxy_rewrite_test.go`
- 前端：`taskFE/app/src/utils/featureParamsBaseUrl.test.js`、`FeatureParamsProvidersEditor.deepSeekBaseUrl.test.js`
- 适配：`trae-agent/onlineServiceJS/src/featureParamsEnvToYaml.test.mjs`
- 无 MQ 断言（同库配置写，见功能意图例外）

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | deepseek + `/anthropic` | normalize / POST 读回 | 仍为 `/anthropic` |
| T2 | 自定义网关 URL | normalize | 原样 |
| T3 | `…/v1https://…` 粘连 | normalize / coerce / 失焦 | 保留最后一段绝对 URL |
| T4 | 空 base_url，供应商改为 deepseek | change | 仍为空，不预填 |
| T5 | YAML env 含 `/anthropic` | featureParamsEnvToYaml | yaml 仍含 `/anthropic` |
| T6 | 库内 `…/v1https://…` | GET company settings | 响应 `base_url` 为最后一段 |
| T7 | POST `https://gateway.example.com/anthropic` | 保存 | 落库原样 |

## 数据与环境

- Go：`setupBudgetTestDB` 不需要；纯函数测即可
- 前端：vitest + vue-test-utils

## 通过标准

```bash
cd taskCloudService && go test ./src/ -count=1 -run 'TestNormalizeProviderBaseURL|TestNormalizeProviderEntryKeeps|TestCoerceProvidersBaseURLsInEnv|TestProvidersFromJSONString|TestCompanyFeatureParamsGetRepairs|TestCompanyFeatureParamsPostKeeps'
cd trae-agent/onlineServiceJS && node --test src/featureParamsEnvToYaml.test.mjs
```
