# 平台设计规范技能包（Platform Design Skills）

## 基本信息
- 版本：1.0.0
- 创建日期：2026-03-19
- 最后修改：2026-03-19
- 维护者：Trae AI 团队
- 外部来源：https://github.com/ehmo/platform-design-skills

## 规则概述

本项目采用 [ehmo/platform-design-skills](https://github.com/ehmo/platform-design-skills) 作为平台设计规范的补充参考。该技能包包含 450+ 条规则，涵盖 Apple HIG、Material Design 3、WCAG 2.2，适用于 iOS、iPadOS、macOS、watchOS、visionOS、tvOS、Android 及 Web 平台。

### 安装

```bash
npx skills add ehmo/platform-design-skills
```

## 规则分类

### 核心规则
> 在执行 UI/UX 设计与评审时，应参考对应的平台设计规范

#### 平台设计技能包使用规则
- 描述：在进行界面设计、设计评审或可访问性审计时，应加载并使用 platform-design-skills 中对应平台的规范
- 适用场景：
  - Web 界面设计与开发：使用 `web` 技能（响应式设计、WCAG 可访问性、性能、现代 CSS/HTML 模式）
  - iOS/iPadOS 应用：使用 `ios`、`ipados` 技能（Apple HIG）
  - Android 应用：使用 `android` 技能（Material Design 3）
  - macOS/watchOS/visionOS/tvOS：使用对应平台技能
- 优先级：中
- 规则：
  - 进行 Web 前端 UI 设计或代码评审时，参考 platform-design-skills 的 web 技能，确保符合 WCAG 2.2 可访问性及响应式最佳实践
  - 进行设计合规性检查时，根据目标平台选择对应的技能文件
  - 技能会在检测到平台相关任务时自动激活，也可在提示中显式请求（如「按 WCAG 审计此页面」「按 Material Design 检查此界面」）

### 可用技能一览

| 平台 | 技能 | 主要覆盖 |
|------|------|----------|
| Web | web | 响应式设计、WCAG、性能、渐进增强、现代 CSS/HTML |
| iOS | ios | 导航、布局、可访问性、手势、Tab Bar、Sheet、Dynamic Island |
| iPadOS | ipados | 多任务、指针支持、侧边栏、键盘快捷键、Stage Manager |
| Android | android | Material You、动态色彩、导航模式、组件 |
| macOS | macos | 菜单栏、窗口管理、工具栏、键盘交互 |
| watchOS | watchos | 一览式界面、Digital Crown、表盘复杂功能 |
| visionOS | visionos | 空间 UI、眼动与手势输入、沉浸式空间 |
| tvOS | tvos | 焦点导航、Siri Remote、Top Shelf |

### 规范性来源

- Apple Human Interface Guidelines (2025) — developer.apple.com/design/human-interface-guidelines
- Material Design 3 — m3.material.io
- Web Content Accessibility Guidelines (WCAG) 2.2 — w3.org/WAI/WCAG22/quickref

## 变更日志
- 2026-03-19：版本 1.0.0 - 初版，引入 platform-design-skills 作为项目设计规范补充参考
