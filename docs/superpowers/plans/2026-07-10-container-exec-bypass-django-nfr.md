# NFR 澄清：容器执行热路径零 Django

- **日期**: 2026-07-10
- **默认等级**: L2（Standard）；鉴权相关 L3

| 属性 | 等级 | 场景 | 度量 |
|------|------|------|------|
| 可用性 | L3 | Django :8001 停止时，已注册容器的 layer-command / job 轮询仍成功 | 成功率 100%（冒烟） |
| 延迟 | L2 | validate+resolve 两跳 Go internal | p99 额外开销 < 50ms（相对现 Django） |
| 安全 | L3 | Internal secret；无 token 入日志；跨租户 403 | 单测覆盖 |
| 可观测 | L2 | forward_stage=auth_validate\|cloud_resolve\|upstream_forward | Loki/本地日志可查 |
| 一致性 | L2 | 401/403/409 语义对齐旧 Django | 契约测 |

## 领域模型影响

- 不新增聚合；复用 User/Membership（taskAuth）、CloudServerConfig（taskCloudService）、AccessToken（Credential）。
- tcg 为应用服务编排，不持有业务真源。
