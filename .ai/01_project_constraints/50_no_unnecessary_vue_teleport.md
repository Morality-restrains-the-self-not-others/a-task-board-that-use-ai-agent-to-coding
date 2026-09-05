# 前端无必要禁止 Teleport（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-13
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 45 条
- Cursor：`.cursor/rules/no-unnecessary-vue-teleport.mdc`（globs：Vue / React 前端源码）

## 背景与动机

Vue `<Teleport>`（及 React `createPortal`）把节点挂到组件树之外的 DOM。视觉父与逻辑父分离后会出现：

- **时序空洞**：`Teleport defer` 只等到首屏兄弟树挂载结束，不会在之后 `v-if` 从 false→true 时重绑；目标槽晚出现则内容永远不挂上
- **双实例分叉**：一处 Teleport、一处内联，同一面板两套状态
- **跨树耦合**：用 `[data-testid=…]` 当 Teleport 目标，等于用 CSS 选择器当跨组件接口
- **可访问性与测试变难**：焦点、事件冒泡、ARIA 归属与查询根都不再是视觉父

本仓库曾用 Teleport 把整块功能卡（如硬件配置）从所属区域「送」到评论输入框槽位，属于**用传送代替在目标父组件内直接渲染**。正确做法是：UI 出现在哪棵子树，就在那棵子树挂载组件。

**默认：不要使用 Teleport / createPortal。只有溢出裁剪、层叠上下文、固定浮层等无法在原树解决时才允许。**

## 核心规则

### 1. 默认就地渲染

- 组件的视觉位置由其父组件的模板决定：在目标父里 `import` 并挂载，或用普通 slot / 具名插槽。
- **禁止**为了「这块 UI 想出现在别的区域」而从远亲组件 Teleport 过去。
- **禁止**用 Teleport 代替 props、provide/inject、composable 或事件来跨区域搬 DOM。

### 2. 仅在「无 Teleport 则功能不可用」时允许

同时满足以下全部条件才算「有必要」：

1. 浮层/下拉/提示必须画在触发器附近或盖住全页；
2. 祖先存在 `overflow: hidden|auto|scroll`、`transform` / `filter` / `will-change` 形成的包含块或层叠上下文，**就地 `position: absolute|fixed` 会被裁剪或错位**；
3. 无法通过调整祖先 overflow、把下拉提为触发器的直接子节点、或把组件改挂到本就无裁剪的父级来解决。

允许的典型场景（仍须在代码旁标注理由，见第 3 节）：

| 场景 | 说明 |
|------|------|
| 表格/滚动容器内的下拉 | 祖先 `overflow-x: auto` 会裁掉绝对定位菜单，可 Teleport 到 `body` + `position: fixed` 跟随 `getBoundingClientRect` |
| 全页模态遮罩 | 必须盖住整个视口且不受父级 `z-index` / overflow 限制时，可挂到 `body`（优先走项目已有 `modalService` / 独立 `*Modal.vue`，不要为普通卡片开 Teleport） |
| 被 transform 祖先困住的 `position: fixed` 浮层 | 例如下拉被 `.app-modal-overlay` 盖住且无法改祖先时 |

### 3. 使用处必须标注 `Teleport-OK`

允许使用时，**紧邻** `<Teleport>` / `createPortal(` 写明理由，格式固定：

```html
<!-- Teleport-OK: overflow-clip — 表头搜索下拉在 overflow-x:auto 表格内会被裁剪 -->
<Teleport to="body">
```

类别取其一：`overflow-clip` | `stacking` | `fixed-overlay`。缺注释视为不合规，评审应要求删 Teleport 或补理由。

React 等价：

```jsx
// Teleport-OK: overflow-clip — ...
createPortal(menu, document.body)
```

### 4. 明确禁止的模式

| 禁止 | 替代 |
|------|------|
| 把功能卡/面板从 A 区 Teleport 到 B 区槽（如 `[data-testid=comment-composer-*-slot]`） | 在 B 的父组件内直接渲染该卡；状态用 composable / provide 共享 |
| `Teleport defer` 等待另一区域 `v-if` 槽出现 | 目标父自己挂载；不要用 defer 当跨树生命周期同步 |
| 同一面板 Teleport 一套 + 内联再挂一套 | 只保留视觉父那一套 |
| 新增 Teleport 却无 `Teleport-OK` 注释 | 先证明第 2 节三条件，再写注释 |

存量 Teleport：**触及该文件时**须按本规则重审；若已能就地渲染，顺手拆掉 Teleport，不要以「历史如此」继续复制。

## 验收

```bash
# 列出全部 Teleport，评审时核对是否有 Teleport-OK 且符合「无 Teleport 则浮层不可用」
rg -n '<Teleport' --glob '*.vue' taskFE task2app trae-agent

# 功能卡搬到远亲 data-testid 槽：新增禁止；存量触及须改为就地渲染
rg -n 'Teleport[^>]*to="\[data-testid' --glob '*.vue'

# React 等价
rg -n 'createPortal\(' --glob '*.{js,jsx,ts,tsx}' taskFE task2app trae-agent
```

Code Review：看到 `<Teleport>` / `createPortal` 时第一问「能否在视觉父里直接挂？」；答不出第 2 节三条，则退回。新增该用法必须带 `Teleport-OK` 注释。

## 与前端规范关系

与 [`.ai/04_frontend_development/00_frontend_development.md`](../04_frontend_development/00_frontend_development.md) 互补：该文管组件拆分、模态框独立文件、下拉单击/双击；**本条管 DOM 传送**。模态框仍须独立 `*Modal.vue` + `modalService`，不因此条改回 `alert`/`confirm`。
