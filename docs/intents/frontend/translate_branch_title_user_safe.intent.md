# 意图：创建任务标题翻译失败对用户可读

## 业务意图 → 事件对照

**无对应事件**：标题翻译是创建任务弹窗的分支命名辅助变换，失败不改变任务/项目状态。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 中文标题译为分支名片段 | — | — | — | 纯工具变换；失败走本地 sanitize |

## 功能意图

1. 创建任务弹窗对中文标题调用 `POST .../projects/translate-branch-title/`。
2. fanyi 上游超时、空 body、非法 JSON 时，HTTP 502 的 `error` 为用户可读短句（含失败原因分类），不包含 `fanyi_agent` / `unexpected end of JSON input` / `finish_reason`。
3. 详细失败原因（status、bytes、超时分类、finish_reason）只写入带 `trace_id` 的服务日志。
4. 前端在已用本地规则回填分支名后，红字提示须包含**为什么无法自动翻译**，并附加「已使用本地规则生成分支名」，保留 `data-traceId`。
5. 标题翻译请求 `max_tokens` 上限为短短语预算，禁止沿用通用 chat 的大 token 配置；服务端须对 DeepSeek v4 关闭 thinking。
6. 标题连续变更时 abort 上一次 translate 请求；AbortError 不写红字、不回退已填分支名。
