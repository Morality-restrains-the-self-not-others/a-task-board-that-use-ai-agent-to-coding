# 功能意图：厂商为镜像组设置必填图标

- **日期**: 2026-08-28
- **状态**: 实施中
- **接口**: `POST/PUT /api/vendor/image-groups/`；`POST .../icon-upload-url/`；`GET /api/public/image-groups/{id}/icon`
- **页面**: 厂商门户创建/编辑镜像组弹窗；主站镜像市场组卡

## 背景与目标

镜像组缺少图标，市场卡片仅有文字。要求创建/编辑时图标必填，并在市场展示。

## 范围与边界

- 范围内：图标上传（PNG/JPEG/WEBP ≤512KB）、组 CRUD 必填 `icon_file_key`、公开流式读取、门户与市场展示。
- 范围外：图标审核流、SVG、按图标搜索。

## 约束与风险

- 禁止 SVG；file_key 必须归属当前厂商；公开 GET 不接受客户端 file_key。
- 存量空图标：目录仍展示占位；下次编辑必补。

## 验收标准

1. 创建无 `icon_file_key` → 400「镜像组图标必填」。
2. 合法上传后创建 → 201，响应含 `icon_url`。
3. 公开 GET 返回对应 Content-Type 图像字节。
4. 弹窗无文件且无已有 key 时保存按钮被拦截，文案「镜像组图标必填」。
5. 市场组卡在 `icon_url` 存在时渲染 `<img>`。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|----------|--------|---------|
| 上传图标对象确认 | ImageGroupIconUploaded | — |
| 创建镜像组 | ImageGroupCreated | LogEventBus（服务无 Kafka 配置，与证照上传同构） |
| 更新镜像组 | ImageGroupUpdated | 同上 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-28 | 初稿：必填图标 + 公开 icon_url |
