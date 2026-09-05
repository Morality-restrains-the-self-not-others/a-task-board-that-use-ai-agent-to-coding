# 测试意图：任务详情硬件配置并入环境与硬件卡（评论区镜像下）

## 对应意图
`026_hardware_config_in_image_card.intent.md`

## 摘要
任务详情页「服务器硬件配置」嵌在「环境与硬件」卡内，由评论 composer 在 @镜像 后直接渲染；服务器信息区无独立硬件 Tab。

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | 环境与硬件卡由 composer 直挂，硬件面板在卡内且不在 server-section-body / logic Teleport | 单元 | `ServerConfig.logic.hardwareInImage.test.js` + `CommentComposerHardwareCard.test.js` |
| T2 | SectionTabs 无硬件 Tab / 默认 runtime | 单元 | `useServerConfigSections.hardwareNest.test.js` |
| T3 | expandHardwareForComment 不改 Tab，会 scroll | 单元 | `useServerConfigHardwareContext.expandNest.test.js` |
| T4 | @镜像 后硬件面板云平台下拉可用；点临时配置展开表单 | Playwright | `TaskDetail.server-config-hardware-tab.playwright.test.js` |
| T5 | 折叠时点「服务器内容」自动展开 | Playwright | `TaskDetail.server-section-collapse.playwright.test.js` |
| T6 | Composer @镜像 后挂载 CommentComposerHardwareCard（位于智能体资源配置槽之后） | 单元 | `TaskDetailCommentComposer.imageSelect.test.js` |
| T7 | 智能体资源配置槽在评论输入后、硬件卡前（028）；与镜像说明同行，行常驻 DOM + v-show | 单元 | 同上 |
| T8 | 硬件卡不含 `hardware-config-comment-bar`，直接渲染环境与硬件面板并登记 reader | 单元 | `CommentComposerHardwareCard.test.js` |
| T9 | 临时硬件配不齐：横幅提示无法运行；submit 仍 POST 且不带残缺 template | 单元 | `CommentComposerHardwareCard.test.js` + `taskDetailFetchFns.imageMention.test.js` |

## 成功标准
- T1–T3、T6、T8–T9 vitest 全绿
- T4–T5 在 mock 环境下通过（@镜像 后 `comment-composer-env-hardware-slot` 内可见 `server-image-config-card` / 硬件面板；无 `server-config-hardware-tab`）
