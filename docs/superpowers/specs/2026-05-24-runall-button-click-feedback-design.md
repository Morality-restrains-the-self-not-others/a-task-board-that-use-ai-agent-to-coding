# runAll 按钮点击反馈增强设计

## 背景与目标

`http://localhost:9999/` 页面当前按钮点击反馈较弱，用户会产生“点了和没点差不多”的感受。  
本设计目标是在不改变业务行为和接口契约的前提下，增强“按下感 + 状态确认感”，并保持统一风格。

## 用户确认结论

- 反馈目标：按下感与状态确认感并重
- 视觉强度：可接受明显颜色变化
- 策略范围：全按钮统一反馈强度
- 方案选择：方案 B（CSS + 短暂确认态）

## 方案对比

### 方案 A：纯 CSS 按压态

- 内容：补齐 `:active` / `:focus-visible` / 强化 `:hover`
- 优点：改动最小、风险最低
- 缺点：确认感持续时间短

### 方案 B：CSS + 短暂确认态（选定）

- 内容：在方案 A 基础上，点击后追加 `is-clicked`（约 160ms）
- 优点：按下感与确认感均明显，改动仍轻量
- 缺点：需要少量 JS 管理 class 与计时器

### 方案 C：完整动作反馈

- 内容：在 B 基础上增加请求中禁用态、过程文案、完成提示
- 优点：反馈最完整
- 缺点：改动范围大，超出当前最小目标

## 设计明细

## 1) 交互行为定义

- 作用范围：`status.html` 中 `.action-btn` 与 `.logs-panel-btn`
- 状态层级：`default -> hover -> active -> clicked(160ms)`
- 统一策略：
  - `hover`：背景提亮、边框增强、阴影增强
  - `active`：`translateY(1px) + scale(0.98)` + 阴影收缩
  - `clicked`：短暂高亮/外发光以确认触发
  - `focus-visible`：提供清晰键盘焦点轮廓

## 2) 实现结构

- CSS 增量：
  - 为 `.action-btn`、`.logs-panel-btn` 增加过渡属性
  - 新增 `:active`、`:focus-visible`、`.is-clicked` 状态样式
- JS 增量：
  - 新增 `pulseClickFeedback(buttonEl)`：
    - 点击时添加 `is-clicked`
    - 160ms 后移除
    - 连点时先清除旧计时器再重置，避免状态粘连
- 接入点：
  - 服务列表事件委托中，在执行业务动作前触发 `pulseClickFeedback(btn)`
  - 日志面板 `Refresh` / `Close` 按钮点击时同样触发

## 3) 数据流、错误处理与验证

- 数据流：保持现有动作函数与 API 调用路径不变
- 错误处理：沿用既有 `alert` 与刷新策略；视觉反馈不依赖请求成功
- 验收点：
  - 手工验证四态（`hover/active/clicked/focus-visible`）均可感知
  - 连点时按钮不会卡在 `is-clicked`
  - 启动/关闭/编译/重启/日志/清空日志功能行为不变

## Value Stream 影响分析

本需求为纯前端视觉交互优化，不涉及后端字段、接口契约或业务流程调整。  
对 `value-stream.yaml` 的影响如下：

- 受影响 streams：无
- 是否新增 stream：否
- fields 影响：无
- test_file 影响：无（可选补充前端 e2e，但非本次必要）
- status 变更：无
- 跨 stream 依赖变化：无

## Domain Concept Inventory（后端任务适用性）

本需求不属于后端/领域建模变更，暂不引入新的 Bounded Context、Entity、Aggregate 或 Domain Event。

## 实施边界

- 做：
  - 按钮交互反馈增强（样式 + 短暂点击确认）
- 不做：
  - 业务接口调整
  - 请求生命周期 UI 重构（loading/success toast 体系化）
  - 大范围组件抽象改造
