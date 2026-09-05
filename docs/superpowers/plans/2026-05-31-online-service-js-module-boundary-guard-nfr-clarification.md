# NFR Clarification: onlineServiceJS 模块边界回归防护

> 设计：`docs/superpowers/specs/2026-05-31-online-service-js-module-boundary-guard-design.md`

| 类别 | 级别 | 说明 |
|------|------|------|
| 可靠性 | L2 | 关键 API 路径须有 unit 烟雾；删层失败不得因裸 ReferenceError |
| 可维护性 | L2 | import 契约写入 spec；不引入新 lint 工具链 |
| 性能 | L1 | 单测仅临时目录，无网络依赖 |
| 安全 | — | 不适用（无鉴权/数据面变更） |
| 可用性 | — | 不适用 |

**领域模型影响：** 无聚合一致性变更；测试验证容器运行时边界行为。
