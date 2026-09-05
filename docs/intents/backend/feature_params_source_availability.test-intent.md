# 测试意图：智能体资源配置来源可用性含 LLM 配置

## 对应意图
`feature_params_source_availability.intent.md`

## 测试目标
防止「设置页已配公司 LLM 智能体、任务详情仍报暂无可用」回归。

## 测试分层
- 领域/单元：Go `featureParamsSourceConfigured` 与 serialize 标志
- 适配器：公司 GET summary HTTP
- 前端单元：flag 解析与 composable 可用性

## 用例矩阵

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | extra key / named provider / agent model 分别使来源可用；空占位不可用 | 单元 | `feature_params_source_availability_test.go` |
| T2 | 默认夹具 openai+模型、extra=[] 的公司 GET summary → company=true | HTTP | 同上 |
| T3 | 空 providers+空 extra → company=false | HTTP | `feature_params_public_handlers_test.go` |
| T4 | 工作空间 GET serialize：公司 LLM → company=true；本级 LLM → workspace=true | 单元 | `feature_params_source_availability_test.go` |
| T5 | 前端读取 `data.env_var_sources_available` | 单元 | `envParamsSourceSelection.test.js` |
| T6 | 创建任务 composable 消费嵌套 flag 全 false → 不可用 | 单元 | `useCreateTaskFeatureParams.sourcesAvailable.test.js` |

## 数据与环境
Go 单测用 `setupFeatureParamsPublicTest` 内存夹具；不连生产库、不打印 API key。

## 通过标准
T1–T6 全绿。
