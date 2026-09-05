# NFR Clarification: relay bootstrap GitLab clone auth

| 类别 | 级别 | 说明 |
|------|------|------|
| 可靠性 | L2 | clone 失败需明确日志；凭证字段向后兼容 |
| 安全 | L2 | 日志不输出 token；HTTP Basic 仅容器内使用 |
| 性能 | L2 | 无额外网络往返 |
| 可维护性 | L2 | clone/push 用户名策略统一 |
