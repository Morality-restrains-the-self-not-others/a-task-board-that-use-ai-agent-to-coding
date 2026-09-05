# 测试意图 — 意见与建议链接（后端）

- **对应意图:** `tenant_feedback_links_by_consumption.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 空阈值启用组 | 任意租户 GET 含该组 |
| T2 | 组要求帖 ≥ 100，租户累计 99 | 不含该组 |
| T3 | 组要求帖 ≥ 100，租户累计 100 | 含该组 |
| T4 | 组要求帖 ≥ 100 且流量 ≥ 10，仅帖达标 | 不含该组 |
| T5 | 高消耗租户 | 同时看到低阈值组与高阈值组 |
| T6 | 禁用组 / 禁用链接 | 租户响应中不出现 |
| T7 | POST `http://` URL | 400 |
| T8 | 非超管 PUT | 403 |
| T9 | 无 `nav.feedback.main` region | 403 |
| T10 | POST 成功 | 发布 FEEDBACK_LINK_GROUP_CREATED |
| T11 | 未登记 resource_kind | 400 |
| T12 | 租户响应 | 无 thresholds 字段 |
