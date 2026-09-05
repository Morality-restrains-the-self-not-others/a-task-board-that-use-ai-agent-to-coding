# 测试意图 — 开放式邀请链接

- **对应意图:** `open_invite_link.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 默认 link invite | 201；`link_kind=single`；`max_uses=1` |
| T2 | open + max_uses=0 | 201；两名不同用户 join 均 201 |
| T3 | open + max_uses=2 | 第三人 join 400 |
| T4 | 同用户再 join | 400 已在公司 |
| T5 | email + link_kind=open | 400 |
| T6 | validate 耗尽后 | valid=false |
| T7 | pending 含 use_count | 开放未耗尽仍列出 |
| T8 | 非 admin 创建 | 403 |
| T9 | MEMBER_JOINED 含 invitation_id | 每次成功 join 发布 |
