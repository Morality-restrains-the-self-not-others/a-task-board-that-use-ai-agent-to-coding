# 意图：任务详情「智能体资源配置」改名并移到评论输入框下方

## 摘要
任务详情评论区原「环境变量参数」选择器改名为「智能体资源配置」，从「环境与硬件」卡内移到评论输入框（`#comment-content`）下方。空状态提示中「功能参数」同步改名，且「公司 / 工作空间 / 个人环境变量」分别为真实 `<a href>` 链到对应设置页。

## 用户价值
- 与侧栏、设置页已统一的「智能体资源配置」称谓一致，避免同页两套术语
- 写评论后再选资源配置，贴近「提交并运行」心智
- 无可用来源时一键跳到公司 / 工作空间 / 个人配置页

## 行为约定
1. `data-testid="feature-params-block"` 的 label 与未选占位为「智能体资源配置」，不再出现「环境变量参数」
2. 评论区 `data-testid="comment-composer-feature-params-slot"` 与镜像说明同处 `comment-composer-run-config-row`（镜像左、本槽右），位于评论输入之后、**硬件卡之前**、提交按钮之前；槽内 **composer 就地挂载** `ServerConfigFeatureParamsBlock`（状态经 `taskDetailFeatureParamsBridge` 与 ServerConfig 共享）。禁止 Teleport 到该槽。
3. 「环境与硬件」卡由 composer 在 `@镜像` 后直接渲染（`CommentComposerHardwareCard`），位于智能体资源配置**下方**、提交按钮之前；卡内仅保留硬件面板；不经 Teleport
4. 空状态文案使用「智能体资源配置」；「公司」「工作空间」「个人环境变量」为原生 `<a href>`（禁止 click.prevent）：
   - 公司 → `/tenant/{tenantId}/settings/feature-params/`
   - 工作空间 → `/tenant/{tenantId}/settings/task-panel/`
   - 个人 → `/profile/feature-params/`
5. 创建任务表单共用同一组件，文案与链接规则一致（布局仍在镜像字段旁，不 Teleport）

## 非目标
- 不改变 `feature_params_source` API 字段名与解析契约
- 不把工作空间链接改到 `/settings/workspace/:id/feature-params/`（产品指定 task-panel 入口）

## 变更记录
- 2026-08-13：拆掉 ServerConfig → composer 槽 Teleport；功能卡就地挂载 + feature-params bridge
- 2026-08-13：镜像说明迁到本槽左侧同行（`comment-composer-run-config-row`）
- 2026-08-13：公网仍见硬件在镜像 `mb-3` 内、智能体配置之上；回归断言 DOM 顺序并发布 SPA
- 2026-08-13：硬件卡挪到本槽下方（先智能体资源配置，再硬件）
- 2026-08-13：硬件卡改为 composer 直挂；本槽改为常驻 DOM + v-show，避免 Teleport 首屏绑定失败
- 2026-08-16：空状态「暂无可用」与设置页 LLM 智能体配置对齐，见 `docs/intents/backend/feature_params_source_availability.intent.md`
