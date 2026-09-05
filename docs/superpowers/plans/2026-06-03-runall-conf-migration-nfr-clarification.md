# NFR 澄清: runAll + conf/ 配置统一

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-03-runall-conf-migration-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-03-runall-conf-migration-value-stream.md`

## 跳过声明

**本次变更跳过 NFR 澄清。**

理由：
- 这是一次**配置迁移**任务：将 `runAll.yaml` 迁移到 `conf/runAll.yaml`，修改 runAll Go 程序读取 conf/ 目录
- 不引入新的数据流、外部依赖或用户面功能
- 不改变任何运行时行为（健康检查 URL 的 IP/端口拼接结果与原来硬编码值相同）
- 无性能、安全、可用性、一致性等非功能需求的变更

| 类别 | 等级 | 理由 |
|------|------|------|
| 所有 NFR 类别 | L0 - 不适用 | 纯配置重构，无运行时行为变更 |

如果后续 runAll 的配置加载失败导致启动时间延长，属于 bug 而非 NFR 不足。
