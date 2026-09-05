# 测试意图：镜像选择迁入评论区

## 对应意图
`027_image_select_in_comment_composer.intent.md`

## 测点

| ID | 测点 | 类型 | 位置 |
|---|---|---|---|
| T1 | Composer 含 detail-image；ServerConfig 不含 | 单元 | `TaskDetailCommentComposer.imageSelect.test.js` |
| T2 | @mention 写入 bridgedSelectedImageId | 单元 | 同上 |
| T3 | bridge 与 ServerConfig selectedImageId 双向同步 | 单元 | `taskDetailImageSelectionBridge.test.js` |
| T4 | @镜像 后挂载 CommentComposerHardwareCard（位于智能体资源配置槽后） | 单元 | `TaskDetailCommentComposer.imageSelect.test.js` |
| T5 | 镜像说明与智能体配置同一行（左/右），位于评论输入之后 | 单元 | 同上 |
| T6 | @镜像 后说明展示 name:version；无 version 时仅名称；无 version 有 tag 时用 tag | 单元 | 同上 |

## 成功标准
T1–T6 vitest 全绿；公网硬刷新后评论区 @镜像 后顺序为：输入 → 镜像说明（左，含版本号）+ 智能体资源配置（右）→ 环境与硬件 → 提交。
