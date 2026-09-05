# 意图：任务详情镜像选择迁入评论提交区并与 @镜像 同步

## 摘要
任务详情的「镜像」下拉从上方 ServerConfig 卡迁入「添加评论」提交区；评论中 `@` 已安装镜像时，下拉选中项随之变更，并同步任务级 `container_image_id`（经 bridge + PATCH）。「环境与硬件」卡挂在该镜像选择下方（见 `026_hardware_config_in_image_card`）。

## 用户价值
- 「提交并运行」与镜像选择同区，降低来回滚动
- @镜像 与启动所用镜像一致，避免选错
- 评论输入框下先选智能体资源配置（镜像说明在其左侧同行），再调本次运行硬件（见 028 / 026）

## 行为约定
1. 评论区展示 `data-testid="comment-composer-image-select"`（含 `@镜像` 后的「将使用镜像 …」说明）
2. `@镜像` 后镜像说明与智能体资源配置同一行（`comment-composer-run-config-row`）：镜像说明在左，智能体配置在右
3. 评论区 `@镜像` 后由 `CommentComposerHardwareCard` 直接渲染环境与硬件卡（`data-testid="comment-composer-env-hardware-slot"`），位于该行下方；卡内不再另挂与面板重叠的摘要条
4. `@镜像` → `pendingImageMention` → `selectedImageId` / `bridgedSelectedImageId`
5. 手动改下拉同样经 bridge 更新 ServerConfig（硬件架构过滤随之更新）
6. 看板 TaskCard 评论区不展示镜像下拉（`show-image-select=false`），亦不提供 env-hardware 槽
7. `comment-composer-mentioned-image` 展示已安装镜像的具体版本：有 `version`（或 `tag`）时为「将使用镜像 {name}:{version} 运行」；无版本时省略冒号与版本号。评论正文 `@镜像` 仍只用名称，不改 mentions API 契约

## 非目标
- 不改变评论 mentions API 契约

## 变更记录
- 2026-08-18：镜像说明补具体版本号（`name:version`），数据来自已安装镜像列表，不改 @mention 正文
- 2026-08-13：镜像说明从输入框上方迁到智能体资源配置左侧同行；去掉硬件摘要条
- 2026-08-13：硬件卡挪到智能体资源配置下方（不再紧跟镜像说明）
- 2026-08-13：环境与硬件改为 composer 直挂 `CommentComposerHardwareCard`（废止对该槽的 Teleport）
- 2026-08-13：未 @镜像 不展示独立镜像下拉与硬件卡（见 029）；@ 本身即选镜像
- 2026-08-13：智能体资源配置迁出镜像块、挂到评论输入框下方（028）
- 2026-08-12：环境与硬件迁入镜像下（与 026 对齐）；原「仅摘要当前镜像、硬件仍在上方」约定废止
