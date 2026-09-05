# NFR 澄清：工作面板过滤选项持久化

- 日期：2026-07-13
- 默认级别：L2 Standard

| 类别 | 级别 | 场景 | 领域影响 |
|------|------|------|----------|
| 可用性 | L2 | GET 失败时前端落默认，不阻塞看板 | 客户端降级 |
| 性能 | L2 | PUT debounce 400ms；payload &lt; 64KB；最多 20 栏 | 规范化校验 |
| 安全 | L3 | 本人偏好 + 租户归属；auth 头强制 | 权限矩阵 |
| 一致性 | L2 | 单行 upsert；最后写覆盖 | 无冲突合并 |
| 可观测 | L2 | info 日志含 bars_count / ok|error | logging |
| 可维护 | L2 | version 字段便于演进 | payload.version |

无 L4 金融/支付相关要求。
