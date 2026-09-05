# NFR: taskGateway Ship Increment

| 类别 | 级别 | 场景 |
|------|------|------|
| 安全 | L3 | internal phone API 仅 secret；换绑须验证码 |
| 可用性 | L2 | taskAuth 503 时换绑失败明确 |
| 可测试性 | L2 | pytest + go test 覆盖 phone upsert |
| 可运维性 | L2 | smoke 无 Docker 时 skip exit 0 |
