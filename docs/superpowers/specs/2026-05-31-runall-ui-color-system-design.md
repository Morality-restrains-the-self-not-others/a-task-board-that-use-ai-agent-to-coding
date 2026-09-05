# runAll Web UI 前端色彩与交互反馈规则

**日期：** 2026-05-31  
**状态：** 已批准（用户要求制定规则并应用）  
**范围：** `runAll/src/status.html`（`:9999` 状态页）

---

## 1. 背景与问题

### 1.1 现状

runAll 状态页为深色主题（`#1a1a2e` 背景），按钮使用 `.action-btn` / `.logs-panel-btn`。点击反馈依赖：

- `:active` — `filter: brightness(1.12)` + 轻微位移
- `.is-clicked` — `brightness(1.18)` + 灰色外发光，持续 **160ms**

在深色背景上，亮度滤镜变化约 12–18%，肉眼难以区分「已点击」与 hover。部分语义按钮（重启/日志/清空）仅有边框/文字色差异，按下时无对应 accent 反馈。

另：`obs-clear-all` 按钮调用未定义的 `flashButtonClick`，点击无反馈。

### 1.2 目标

- 建立 **可文档化的 CSS 设计令牌**（`:root` 变量），作为 runAll UI 色彩单一来源。
- 点击/按下时 **背景色 + 边框 accent + 外发光** 三重反馈，各 variant 使用语义色。
- 点击反馈时长 **≥280ms**，便于在 2s 自动刷新前被感知。
- 修复 `flashButtonClick` 引用错误。

### 1.3 非目标

- 不改变页面布局、API 或 runner 逻辑。
- 不引入外部 CSS 框架或构建步骤。
- 不做浅色主题。

---

## 2. 方案对比

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| **A. 设计令牌 + 强化 pressed（推荐）** | `:root` CSS 变量；`:active` / `.is-clicked` 改背景+边框+发光 | 改动集中、可维护、与现有单文件 HTML 契合 | 需手工维护变量表 |
| **B. 仅加长动画** | 保留 brightness，延长到 400ms | 改动最小 | 根因未解决，仍不明显 |
| **C. Material ripple** | JS 绘制水波纹 | 视觉强 | 过度设计；与运维工具风格不符 |

**决策：方案 A。**

---

## 3. 色彩规则（Design Tokens）

### 3.1 页面与表面

| Token | 值 | 用途 |
|-------|-----|------|
| `--ra-bg-page` | `#1a1a2e` | 页面背景 |
| `--ra-bg-card` | `#16213e` | 服务行背景 |
| `--ra-bg-card-hover` | `#1c2a4a` | 服务行 hover |
| `--ra-bg-panel` | `#0f172a` | 日志面板 |
| `--ra-text-primary` | `#e0e0e0` | 主文字 |
| `--ra-text-muted` | `#94a3b8` | 次要文字 |

### 3.2 状态指示（已有，保持不变）

| 状态 | 色值 | 类名 |
|------|------|------|
| healthy | `#4ade80` | `.dot.green` |
| starting/retrying | `#facc15` | `.dot.yellow` |
| failed | `#ef4444` | `.dot.red` |
| stopped/pending | `#6b7280` | `.dot.gray` |

### 3.3 按钮交互状态（核心规则）

所有可点击按钮（`.action-btn`、`.logs-panel-btn`）共享同一状态机：

| 状态 | 背景 | 边框 | 文字 | 附加效果 |
|------|------|------|------|----------|
| **default** | `--ra-btn-bg` `#111827` | `--ra-btn-border` `#555` | `--ra-btn-text` `#ccc` | 浅阴影 |
| **hover** | `--ra-btn-bg-hover` `#273449` | `#9ca3af` | `#fff` | 加深阴影 |
| **active（按下中）** | `--ra-btn-bg-active` `#334155` | accent 色 | `#fff` | `translateY(1px) scale(0.98)` + accent 内发光 |
| **clicked（点击脉冲）** | `--ra-btn-bg-clicked` `#3d4f66` | accent 色 2px 等效 | `#fff` | accent 外发光 `0 0 12px` |
| **disabled** | 同 default | `#374151` | `#6b7280` | `opacity: 0.45`，无 transform |
| **focus-visible** | — | — | — | `outline: 2px solid #60a5fa` |

**规则：** 交互反馈 **禁止仅依赖 `filter: brightness()`**；`:active` 与 `.is-clicked` 必须同时改变 **background-color** 与 **border-color**。

### 3.4 按钮语义 Variant（accent）

| Variant | 类名 | Accent 边框/发光 | 按下背景 tint |
|---------|------|------------------|---------------|
| 默认/启动/关闭/编译 | `.action-btn` / `.build-btn` | `#94a3b8` 灰蓝 | `#334155` |
| 重启 | `.restart-btn` | `#60a5fa` 蓝 | `#1e3a8a` |
| 日志 | `.logs-btn` | `#a78bfa` 紫 | `#4c1d95` |
| 清空/警告 | `.clear-logs-btn` / `.obs-clear-all-btn` | `#fb923c` 橙 | `#7c2d12` |
| 面板工具 | `.logs-panel-btn` | `#94a3b8` | `#334155` |

### 3.5 动效令牌

| Token | 值 | 用途 |
|-------|-----|------|
| `--ra-motion-fast` | `90ms` | transform |
| `--ra-motion-color` | `120ms` | 背景/边框/文字 |
| `--ra-motion-glow` | `140ms` | box-shadow |
| `--ra-click-feedback-ms` | `300`（JS） | `.is-clicked` 保持时长 |

---

## 4. 实现要点

1. 在 `status.html` `<style>` 顶部定义 `:root { ... }` 变量块，并附注释「runAll UI Color System」。
2. `.action-btn`、`.logs-panel-btn` 引用变量；删除 `:active` / `.is-clicked` 上的纯 brightness 方案。
3. 为各 variant 增加 `:active`、`.is-clicked` 的 accent 背景/边框/发光规则。
4. `clickFeedbackDurationMs` 改为 `300`；`flashButtonClick` → `pulseClickFeedback`。
5. disabled 状态下 suppress active/clicked 样式（已有，保留）。

---

## 5. 价值流影响

纯前端 UX 改进，**不触及** `value-stream.yaml` 中任何 stream/step/field。无需新增价值流条目。

---

## 6. 验收标准

- [x] 点击任意服务行按钮，可见 **背景变亮 + 彩色边框/光晕**，持续约 300ms。
- [x] 重启/日志/清空按钮按下时分别呈现 **蓝/紫/橙** accent，与 default 按钮可区分。
- [x] 「清空 Grafana 可观测数据」按钮点击有反馈（无 JS 报错）。
- [x] `:focus-visible` 键盘焦点仍可见。
- [x] `go test -run TestUIHomePage` 中 status.html 片段测试通过。

---

## 7. 领域概念清单（供后续 DDD 参考）

本变更为 presentation 层样式，无新业务实体。相关 bounded context：**runAll 运维控制台**（只读状态展示 + 生命周期命令触发）。
