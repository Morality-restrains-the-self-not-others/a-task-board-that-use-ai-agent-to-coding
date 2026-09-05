# NFR Clarification: onlineServiceJS DEBUG_AGENT 调试链路

> 输入：`docs/superpowers/plans/2026-05-25-online-service-debug-agent-value-stream.md`
>  
> 目标：明确本次增量在质量属性上的支持等级，并给出对领域模型的约束。

## NFR 支持等级（本期）

- 可观测性（Observability）：**L3**
  - `DEBUG_AGENT=True` 时必须记录完整 `method/url/headers/body` 与响应上下文。
- 安全性（Security）：**L1（调试模式例外）**
  - 常规模式维持现状；调试模式按需求允许敏感信息原样记录。
- 性能（Performance）：**L2**
  - 仅在开关打开时承担额外日志成本；默认关闭避免常态性能回退。
- 可靠性（Reliability）：**L2**
  - 调试日志异常不能影响业务主流程；日志写入失败需降级为 no-op。
- 可测试性（Testability）：**L2**
  - 至少覆盖直启默认注入、开关透传、领域对象约束校验。
- 兼容性（Compatibility）：**L2**
  - 关闭开关时行为与原摘要日志完全兼容。

## 质量属性场景

### 场景 1：调试模式全量证据
- Given：任务详情以 `relayToTrae=true` 进入并直接启动
- When：`DEBUG_AGENT=True` 且 onlineServiceJS 处理入站并发起出站请求
- Then：日志中可完整看到请求与响应上下文，且不做脱敏、截断

### 场景 2：默认模式零影响
- Given：`DEBUG_AGENT` 未设置或为 false
- When：服务处理同样请求
- Then：只保留原有摘要日志，新增调试能力不影响业务结果

### 场景 3：日志异常降级
- Given：日志文件不可写或序列化异常
- When：请求进入调试记录链路
- Then：业务请求继续完成，日志失败不传播为业务失败

## 对领域模型的影响

| NFR | 决策 | 领域模型影响 |
|---|---|---|
| 可观测性 L3 | 记录入站/出站完整上下文 | 增加 `OnlineServiceDebugLogEntry` 聚合根，完整保存 method/url/headers/body |
| 性能 L2 | 开关门控日志路径 | 增加 `DebugAgentFlag` 值对象，服务层先判定再记录 |
| 可靠性 L2 | 记录失败不可中断主流程 | 领域服务只返回事件/实体，不抛基础设施异常契约 |
| 可测试性 L2 | 领域约束可单测验证 | 增加领域单测覆盖合法性与不变式 |

## 范围声明

- 本轮不做日志持久化后端实现（数据库/ES/对象存储），仅定义领域层契约。
- 本轮不引入数据治理策略（脱敏模板、采样策略）；由后续增量演进。
