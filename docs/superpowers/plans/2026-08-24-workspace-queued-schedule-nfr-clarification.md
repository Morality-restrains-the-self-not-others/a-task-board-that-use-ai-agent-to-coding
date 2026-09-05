# 工作空间级排队调度 — NFR 澄清

- 日期：2026-08-24
- 级别：L2（Standard），无认证/金融域提升点

| 类别 | 要求 | 验收 |
|---|---|---|
| 性能 | 分发每周期 O(工作空间数 + 成员数)，DB 查询参数化；无 N+1 放大（成员 title 用 JOIN 一次取回） | 单次 dispatch-once < 2s（既有量级） |
| 可靠性 | 分发幂等：starting 跳过、槽位去重（ON DUPLICATE KEY 既有）；失败不中断后续工作空间 | 单空间失败仅记日志继续 |
| 可观测性 | 保存/分发/启动/关闭沿用 tracelog.LogForwardStage + domain events；日志含 workspace_id；无密钥/PII | 日志字段断言测试 |
| 安全 | requireTenantMember + hasWorkspaceAccess 双层门禁；SQL 参数化；输入校验（HH:MM、交叠、并发数≥0） | 越权 403 测试 |
| 兼容 | 旧端点与 PATCH schedule_rhythm 不破坏；未配置工作空间节奏时回退 task 级分发 | 回退单测 |
| 前端 | 页面加载/保存错误展示（showRequestError 模式 + data-traceId）；移动端可用（Tailwind 响应式） | vitest + Playwright |
