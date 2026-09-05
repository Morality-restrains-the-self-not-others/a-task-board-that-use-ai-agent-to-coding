# 临时配置展开须露出「服务器硬件配置」面板

- **Status:** accepted
- **Date:** 2026-08-13
- **Iteration:** hardware-temp-config-reveal-panel
- **Based on:** `2026-08-13-hardware-panel-hide-lifecycle-controls-design.md`（accepted）
- **Architecture impact:** **否** — 纯 taskFE 展示，不增删服务/数据流/接口
- **ADR:** No-ADR: trivial tech choice, no architectural impact
- **python_api_approval:** not_applicable（无新接口）
- **Decision:** 方案 C — 放弃硬件 Teleport，composer 直接渲染真源面板

---

## 0. 问题

页面：任务详情「添加评论」区（`hardware-config-comment-bar`）。

已 `@镜像`（可见「将使用镜像 trae-agent 运行」），摘要条为「临时配置 / 收起配置」，但 **看不见「服务器硬件配置」面板**（无 `server-hardware-config-panel`、无「环境与硬件」卡）。

产品期望：**点开临时配置时，必须出现服务器硬件配置面板**（云平台 / 区域 / 实例表单）。

无 `data-traceId`（纯前端展示缺口，不查 Loki）。

## 1. 架构理解（本需求不改架构）

根据 current 架构设计稿：

- 视图：`application-integration` current **v75**；`enterprise-landscape` current **v13**
- 应用层：taskFE → taskTaskService → Kafka `TASK_COMMENT_IMAGE_MENTIONED` → taskEvents → taskCloudService start-vm（029 已扩展可选 `server_run_template`）
- 积压 target：v74 / v76 / v77，本缺口不叠加

本次只修评论区 DOM / Teleport，**不创建 v78**。

## 2. 根因

真源面板是 `ServerConfig.logic.vue` 里的 `ServerConfigHardwarePanel`，经

```html
<Teleport defer to="[data-testid=comment-composer-env-hardware-slot]">
```

送进评论 composer。029 把该槽放进 `v-if="showRunConfig"`（仅 `@镜像` 后才进 DOM）。

Vue 3.5 `Teleport defer` **只推迟到首屏兄弟树挂载结束**，不会在之后 `v-if` 从 false→true 时重绑。结果：

| 时刻 | 槽 | Teleport | 用户看到 |
|------|----|----------|----------|
| 进页（未 @） | 不存在 | 绑定失败 | 无卡 |
| 用户 @镜像 | 空槽出现 | **不会重试** | 有「将使用镜像…」，槽空 |
| 点摘要条「临时调节」 | `HardwareConfigCommentBar` 设 `showInlinePanel` | 内联第二套面板依赖 `inject('serverConfigHardwareContext')` | 条变成「收起配置」，**面板仍没有** |

内联面板失败的第二原因：`provide` 在 `ServerConfig.logic`（Runtime 区），评论区是**兄弟**不是子孙，`inject` 恒为 `null`，`v-if="showInlinePanel && hardwareContext"` 永不渲染。

证据：用户可见文案有摘要条 + 「将使用镜像」+ 架构 hints + 「执行依赖」，**没有**「环境与硬件」/「服务器硬件配置」。

智能体资源配置槽 `comment-composer-feature-params-slot` 同样被 `v-if="showRunConfig"` 包住，同类 Teleport 时序风险，一并修。

## 3. 决策（已批准：方案 C）

1. **放弃硬件卡 Teleport。** `@镜像` 后由 `CommentComposerHardwareCard` 在 composer 内直接渲染「环境与硬件」+ `ServerConfigHardwarePanel`。
2. **真源只有一套面板。** 摘要条 `HardwareConfigCommentBar` 不再 mount 第二套内联面板；「临时调节」调用本卡 `openTemporaryConfig()`。
3. **智能体资源配置仍 Teleport**，但槽改为常驻 DOM + `v-show="showRunConfig"`，避免同类首屏绑定失败。
4. **@镜像 后卡自动可见**；点「临时配置 / 临时调节」后 `showHardwareConfigForm === true`。

| 状态 | 摘要条 | 硬件卡 | 硬件表单 |
|------|--------|--------|----------|
| 未 @镜像 | 不展示 | 不挂载 | 不可见 |
| 已 @镜像、项目模版 | 「项目模版」+「临时调节」 | 可见「环境与硬件」+ 模版 banner | 未展开 |
| 已 @镜像、点了临时配置 | 「临时配置」+「收起配置」 | 同上 | **完整「服务器硬件配置」表单** |

摘要条已删除：与「环境与硬件」卡内项目模版 banner /「临时配置」按钮信息重叠。临时配置改由 `HardwarePanelHeader` 按钮展开。

## 4. 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| C. 放弃硬件 Teleport，composer 直接渲染面板 | 无时序问题；点临时配置必见真源 | 硬件状态从 ServerConfig.logic 迁到 composer | **采用（已批准）** |
| A. 槽改 `v-show`，Bar 只驱动真源 `openTemporaryConfig` | 修时序；仍用 Teleport | Teleport 对后出现目标仍脆弱 | 仅用于智能体配置槽 |
| B. 把 provide 提到 TaskDetail，复活 Bar 内联第二套面板 | 条下立刻有面板 | 双实例状态分叉 | 废弃 |

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 展开临时硬件配置（纯 UI） | — | — | — | 只读展示/本地状态，无服务端事实变更 |

无新 Python 接口。无新 Kafka 事件。

## 6. Domain Concept Inventory

无新实体/聚合。仍用既有「项目运行模版 / 本次临时配置」UI 状态。

## 7. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：Nodes 0 / Edges 0 / Files 0；Last updated: never |
| 关键发现 | CRG unavailable: 图未建（空库），以源码检索为准 |
| 决策影响 | 爆炸半径：`TaskDetailCommentComposer.vue`、`HardwareConfigCommentBar.vue`、`taskDetailSectionBindings.js`、composer/hardware 单测、Playwright 硬件槽断言 |
| skip 理由 | 图空，不阻断 |

## 8. 价值流影响

- 展示侧：任务详情评论提交（`task-detail` / 硬件卡 Teleport）
- 测点：`TaskDetailCommentComposer.imageSelect.test.js`、`ServerConfig.logic.hardwareInImage.test.js`、`TaskDetail.server-config-hardware-tab.playwright.test.js`
- 不改 `<service>.<table>.<field>`；不新增 stream

## 9. 🏛️ 架构变更影响

不创建新架构 target。current 仍为 v75 / v13。老 `.puml` 不修改。

## 10. 实施计划（批准后）

1. 新增 `CommentComposerHardwareCard`：摘要条 + 真源 `ServerConfigHardwarePanel`；登记 `registerCommentHardwarePanelReader`
2. 从 `ServerConfig.logic` 移除硬件 Teleport；lifecycle 对空 ref 保持 optional
3. composer：`@镜像` 后挂载该卡，位于智能体资源配置槽**下方**、提交按钮之前；feature-params 槽 `v-show` 常驻
4. Bar 去掉内联第二套面板；临时调节 / 收起 / 恢复项目默认驱动真源
5. 单测 + Playwright：@镜像 后可见面板；点临时调节调用 `openTemporaryConfig`
6. 登记精准编译重启：`taskFE`
