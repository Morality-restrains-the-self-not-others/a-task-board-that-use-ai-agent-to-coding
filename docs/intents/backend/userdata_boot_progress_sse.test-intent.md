# 测试意图：UserData boot-progress SSE

| ID | 场景 | 期望 |
|----|------|------|
| T1 | Linux generateTemplateContent | 含 report_progress、boot-progress、CLOUD_PREFIX |
| T2 | Windows generateTemplateContent | 含 Report-Progress、boot-progress |
| T3 | handleBootProgress progress=100 | 响应 progress=99 |
| T4 | handleBootProgress status=success | SSE status 改写为 processing（不关闭启动态） |
| T5 | taskAgentSupport forwardsToCloudService(boot-progress) | true |
| T6 | Django internal_dispatch boot-progress | 410 ACTION_MIGRATED |
