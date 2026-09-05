# NFR：项目自动克隆子仓库开关

- **日期**: 2026-08-12
- **级别**: L2 Standard（非支付/鉴权核心）

| 类别 | 级别 | 要求 |
|------|------|------|
| 兼容性 | L2 | 缺字段默认 true；旧客户端不传不影响 |
| 性能 | L2 | false 时跳过 nested HTTP 发现，降低 bootstrap 延迟 |
| 可观测 | L2 | enrich 日志带 project_id + auto_clone_nested_repos |
| 安全 | L2 | 无密钥；沿用既有授权 |
| 可用性 | L2 | UI 文案明确「容器启动时」以免误解为创建瞬间克隆 |

无 L3 项。
