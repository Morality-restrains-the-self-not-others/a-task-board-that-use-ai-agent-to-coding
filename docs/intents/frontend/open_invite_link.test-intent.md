# 测试意图 — 前端开放邀请链接

- **对应意图:** `open_invite_link.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 默认单次 | generate body 无 open 或 link_kind=single |
| T2 | 选开放且不填人数 | body `link_kind=open`, `max_uses=0` |
| T3 | 选开放且填 5 | `max_uses=5` |
| T4 | 开放可不填成员名即可点生成 | 不 toast「请填写成员名称」 |
| T5 | 待处理列表开放行 | 展示「已用 x / 上限」或「不限」 |
