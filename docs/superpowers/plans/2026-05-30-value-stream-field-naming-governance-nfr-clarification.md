# NFR Clarification: value-stream 字段命名治理

> Design: `docs/superpowers/specs/2026-05-30-value-stream-field-naming-governance-design.md`

| 质量属性 | 级别 | 说明 |
|----------|------|------|
| 可维护性 | L2 | 规则集中在 `fields.go` + `value-stream.yaml.ai.md` |
| 可靠性 | L2 | CI + pre-commit 双门禁 |
| 性能 | L1 | Go 测试 <5s，不扫描全仓库 markdown |
| 安全 | L1 | 无运行时面 |
| 可观测性 | L1 | 测试失败输出违规文件与行号 |

**默认 L2**；无 auth/financial 域，不升 L3。
