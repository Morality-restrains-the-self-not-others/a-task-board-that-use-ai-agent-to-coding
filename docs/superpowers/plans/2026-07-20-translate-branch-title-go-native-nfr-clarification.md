# NFR: translate-branch-title Go native

| 类别 | 等级 | 说明 |
|------|------|------|
| 可用性 | L2 | 上游 AI 失败返回 502，前端已有红字降级 |
| 延迟 | L2 | 出站 timeout 20s（对齐 Django） |
| 安全 | L2 | 密钥不进前端；无 env proxy |
| 可观测 | L2 | info/error 日志 + 既有 X-Trace-Id |
| 一致性 | L2 | sanitize 规则对齐 Django |
