# NFR 澄清: runAll Web UI 按钮交互色彩

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-31-runall-ui-color-system-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-31-runall-ui-color-system-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L0 | 不适用 — 纯 CSS/160ms 内 JS class 切换 |
| 可伸缩性 | L0 | 不适用 |
| 可用性 | L0 | 不适用 — 不影响服务可用性 |
| 安全性 | L0 | 不适用 — 无新数据流 |
| 数据一致性 | L0 | 不适用 |
| 容错机制 | L0 | 不适用 |
| 可观测性 | L0 | 不适用 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L2 | 色彩令牌集中 `:root`；片段测试防回归 |
| **可访问性 (补充)** | **L2** | `:focus-visible` 保留；按下态对比度可辨 |

## 逐增量 NFR 分析

### Increment 1–3: UI 色彩与反馈

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 所有按钮色彩来自 `:root` 变量；禁止新增裸 hex 于 `:active`/`.is-clicked`
- **质量场景**: QS-01

#### NFR 类别: 可访问性（交互感知）
- **等级**: L2 - 标准
- **量化目标**: 点击反馈可见时长 ≥280ms；键盘 `:focus-visible` outline 保留
- **质量场景**: QS-02

## 质量场景

### QS-01: 色彩规则片段可被 CI 锁定
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者修改 `status.html` |
| 刺激 | 删除 `--ra-btn-bg-clicked` 或 `.is-clicked` accent 规则 |
| 制品 | `runAll/src/ui_test.go` |
| 环境 | 本地 `go test` |
| 响应 | 测试失败，提示缺失片段 |
| 响应度量 | `go test ./...` 在 runAll/src 全绿 |

### QS-02: 点击反馈时长
| 要素 | 内容 |
|------|------|
| 类别 | 可访问性 |
| 等级 | L2 |
| 刺激源 | 用户点击服务行按钮 |
| 刺激 | 单次 click |
| 制品 | `pulseClickFeedback` + `.is-clicked` |
| 环境 | 浏览器手动 / Playwright（可选） |
| 响应 | accent 背景+边框+光晕持续 ≥280ms |
| 响应度量 | `clickFeedbackDurationMs = 300` 在 HTML 中可 grep |

## 领域模型影响

**无。** 纯 presentation 层 CSS/JS，跳过 DDD 建模（见 `/5-ddd` 跳过条件）。

## 权衡与边界

### 取舍
- 用背景色+边框替代 `filter: brightness()`，换取深色主题下更高对比度。

### 明确不做什么
- 不做浅色主题、不做 WCAG AAA 全量审计、不做 Playwright 视觉回归（本迭代仅片段测试）。

### 升级触发条件
- 若 runAll UI 拆分为多页面/组件库，将 `:root` 令牌提取为独立 CSS 文件。

## 跳过声明

性能、可伸缩性、可用性、安全性、数据一致性、容错、可观测性、合规：**L0 跳过** — 纯静态 HTML 样式，无后端/API/数据变更。
