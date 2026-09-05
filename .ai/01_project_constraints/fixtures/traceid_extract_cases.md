# traceId 提取 Golden Cases（大小写不敏感）

供 brainstorming「检测 traceId」与元规则 24 防回归。期望：从载体抽出规范化 ID 字符串（保留原值大小写，匹配键名忽略大小写）。

| # | 载体 | 输入摘录 | 期望抽出 |
|---|------|----------|----------|
| 1 | JSON body | `{"trace_id":"abc-123"}` | `abc-123` |
| 2 | JSON body | `{"traceId":"ABC-456"}` | `ABC-456` |
| 3 | JSON body | `{"TraceId":"x9"}` | `x9` |
| 4 | DOM | `<p class="text-danger" data-traceId="dom-1">err</p>` | `dom-1` |
| 5 | DOM | `<div data-traceid="dom-2">`（HTML 小写化） | `dom-2` |
| 6 | DOM | 文案含 `traceid=t3` / `TraceId: t3` | `t3` |
| 7 | HTTP 头 | `X-Trace-Id: hdr-7` | `hdr-7` |
| 8 | HTTP 头 | `x-trace-id: hdr-8` | `hdr-8` |

规则：写入前端属性名仍只用字面 `data-traceId`；Agent **读取**时忽略大小写。
