# NFR Clarification: runAll 链式关闭混合所有权

> 输入: 设计 `2026-05-27-runall-cascade-mixed-ownership-design.md`  
> 价值流: `2026-05-27-runall-cascade-mixed-ownership-value-stream.md`

## NFR 概览

| 类别 | 等级 | 量化 |
|------|------|------|
| 安全性 | L2 | 单点 `cascade=false` 仍 strict ownership；级联委托仅限计划内服务 |
| 可用性 | L2 | bootstrap+UI 混用场景关闭 git-oauth 成功率 100%（Go 集成测试覆盖） |
| 可维护性 | L2 | 解析规则纯函数单测；不修改 takeover CLI |
| 性能/容错/伸缩 | L0-L1 | 无新增 I/O；解析 O(1) per step |

## 质量属性场景

### QS-01: 混合所有权关闭 git-oauth

| 维度 | 内容 |
|------|------|
| 刺激 | taskFE/saas-backend 属 bootstrap，git-oauth 属 UI session，点关 git-oauth |
| 响应 | 三服务均 stopped，无 takeover 错误 |
| 度量 | `TestRunner_StopServiceCascade_DelegatesOwnershipPerStep` PASS |

### QS-02: 单点关闭仍拒绝 foreign owner

| 维度 | 内容 |
|------|------|
| 刺激 | 非 owner 对单服务 `StopServiceWithActor` |
| 响应 | requires explicit takeover |
| 度量 | `TestStopService_RejectsNonOwnerSession` PASS |
