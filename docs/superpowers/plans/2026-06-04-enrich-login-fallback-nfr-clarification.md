# NFR 澄清：enrich-login 兜底修复

| 类别 | 等级 | 场景 |
|------|------|------|
| 安全 | L3 | enrich 400 不得返回 200；条款仅无 consent 时强制 body id |
| 可靠性 | L2 | Django 不可用时 taskAuth 502，不伪造 user |
| 性能 | L1 | 登录路径无额外 IO；复用已有 consent 查询 |
| 可维护性 | L2 | 条款逻辑单点 `login_policy_consent` 服务 |

**领域影响**：合规同意为应用服务，不新增聚合。
