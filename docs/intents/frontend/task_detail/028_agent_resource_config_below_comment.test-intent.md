# 测试意图：智能体资源配置改名并移到评论输入框下方

## 对应意图
`028_agent_resource_config_below_comment.intent.md`

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | label / 占位 / 必选提示为「智能体资源配置」 | 单元 | `ServerConfigFeatureParamsBlock.test.js` |
| T2 | 空状态含三级真实 `<a href>` | 单元 | 同上；`envParamsSourceSelection.test.js` `featureParamsSettingsHrefs` |
| T3 | Composer 槽与镜像说明同行（右），在评论输入后、硬件卡前、提交前；硬件卡不在 `comment-composer-image-select` 内 | 单元 | `TaskDetailCommentComposer.imageSelect.test.js` |
| T4 | FeatureParamsBlock 由 composer 就地挂载；ServerConfig.logic 无 Teleport | 单元 | `ServerConfig.logic.hardwareInImage.test.js`；`vueTeleportReview.test.js` |
| T5 | 创建任务表单同步新文案 | 单元 | `CreateTaskModal.test.js` |
| T6 | 公网任务详情评论区可见新文案且位于输入框下 | Playwright | `TaskDetail.feature-params-source-switching.playwright.test.js` |
| T7 | 公司已配 LLM 时不展示暂无可用 | 单元 | 见 `docs/intents/backend/feature_params_source_availability.test-intent.md` |

## 成功标准
T1–T5 vitest 全绿；T6 需登录 Cookie / 公网硬刷新后人工或 Playwright 验收。
