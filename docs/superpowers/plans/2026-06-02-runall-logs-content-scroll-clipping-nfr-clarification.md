# NFR 澄清: runAll 日志内容区滚动裁切修复

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-02-runall-logs-content-scroll-clipping-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-02-runall-logs-content-scroll-clipping-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L0 | 不适用 — 纯 CSS |
| 可伸缩性 | L0 | 不适用 |
| 可用性 | L0 | 不适用 — 不影响服务可用性 |
| 安全性 | L0 | 不适用 |
| 数据一致性 | L0 | 不适用 |
| 容错机制 | L0 | 不适用 |
| 可观测性 | L0 | 不适用 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L2 | CSS flex 规则 + Playwright/Go 片段测试防回归 |
| 可访问性（补充） | L2 | 键盘/滚轮可到达全部日志行 |

## 逐增量 NFR 分析

### Increment 1–2: 日志滚动修复

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `.logs-content` 使用 `min-height: 0`；禁止恢复 `min-height: 280px`
- **质量场景**: QS-01

#### NFR 类别: 可访问性
- **等级**: L2 - 标准
- **量化目标**: 滚到 max 时末行 `getBoundingClientRect().bottom ≤ panel.bottom + 1px`
- **质量场景**: QS-02

## 质量场景

### QS-01: CSS 片段 CI 锁定
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者修改 `status.html` |
| 刺激 | 恢复 `min-height: 280px` 或删除 `min-height: 0` |
| 制品 | `runAll/src/ui_test.go` |
| 环境 | 本地 `go test` |
| 响应 | 测试失败 |
| 响应度量 | `go test -run TestUIHomePage_LogsContentScroll ./...` 全绿 |

### QS-02: 末行可见
| 要素 | 内容 |
|------|------|
| 类别 | 可访问性 |
| 等级 | L2 |
| 刺激源 | 用户打开日志并滚到底 |
| 刺激 | `scrollTop = scrollHeight - clientHeight` |
| 制品 | `#logs-content` / `#logs-panel` |
| 环境 | Playwright `1200×682` |
| 响应 | 最后一行在面板内完全可见 |
| 响应度量 | Playwright 断言 `lastLineBottom ≤ panelBottom + 1` |

## 领域模型影响

**无。** 纯 presentation 层 CSS，跳过 DDD 建模。

## 权衡与边界

### 取舍
- 移除 `min-height: 280px` 以换取正确 flex 收缩；极矮视口下面板可能较矮，仍可通过滚动阅读全部内容。

### 明确不做什么
- 不做 auto-refresh 吸底、不做 header 单行布局、不做 fixed 面板重构。

### 升级触发条件
- 若 runAll UI 拆分为组件库，将 flex 布局规则提取为共享 layout 模块。

## 跳过声明

性能、安全、一致性等类别均 L0 — 无新数据流或后端变更。
