# 意图：translate-branch-title 上游失败用户可读

## 业务意图 → 事件对照

**无对应事件**：与设计 `2026-07-20-translate-branch-title-go-native-design.md` 一致，纯查询/变换。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 中文标题翻译 | — | — | — | 不改变任务/项目状态 |

## 功能意图

1. `translateTitleWithFanyiAgent` 不得忽略响应 body 读取错误。
2. 空 body / 超时不得包装为 `json.Unmarshal` 的 `unexpected end of JSON input`。
3. `handleTranslateBranchTitle` 对上游失败返回**分类后的用户短句** 502（说明为什么失败：超时 / 无可用译文 / 响应异常 / 暂未就绪 / 暂时不可用）；`fanyi_agent`、status/bytes、`finish_reason` 等细节只打日志。
4. 标题翻译 `max_tokens` 封顶 128，显式 `stream=false`，出站超时 16s（覆盖实测成功 ~11.5s，避免旧 20s 截断 JSON）。
5. 截断/非法 JSON 报 `响应无效`（含 status/bytes/preview），不得包装 `json.Unmarshal` 的 `unexpected end of JSON input`。
6. 对 DeepSeek v4（默认开启 thinking）标题翻译请求必须带 `thinking.type=disabled`，避免思考占用全部 `max_tokens` 导致 HTTP 200 + 空 `content`。
