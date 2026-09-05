# 镜像组图标必填 — 设计文档

- **日期**: 2026-08-28
- **状态**: accepted（/goal 自动采用）
- **迭代**: image-group-icon
- **作者**: cursor

## 背景

厂商门户「创建/编辑镜像组」弹窗仅有名称、描述。镜像市场卡片无法用图标区分镜像组。产品要求：用户可为镜像组添加图标，且为必填。

页面：`https://provider.daydaymoney.com/` → 编辑镜像组 `div.card.modal`。

## 成功标准

1. 创建镜像组：未选图标 → 前端拦截 + 后端 400「镜像组图标必填」，不落库。
2. 编辑镜像组：存量无图标时必须补传；已有图标可不改文件直接保存。
3. 接受 PNG / JPEG / WEBP，最大 512KB；拒绝 SVG（XSS）与 PDF。
4. 厂商列表、公开目录、主站镜像市场卡片展示图标；无图标（存量）用占位，不隐藏组。
5. 图标 URL 带内容绑定查询参数 `?h=`（缓存击穿）。

## 方案（选定）

复用既有 `VendorDocStore`（local / COS 预签名 PUT），新增 kind `image_group_icon`。

| 步骤 | 接口 |
|------|------|
| 签发上传 | `POST /api/vendor/image-groups/icon-upload-url/` |
| 直传 | local PUT `/api/vendor/image-groups/icon-local-put/?file_key=`；COS 用预签名 URL |
| 确认 | `POST /api/vendor/image-groups/icon-upload-complete/` |
| 创建/更新 | 既有 POST/PUT `/api/vendor/image-groups/`，body 增 `icon_file_key`（必填） |
| 公开读取 | `GET /api/public/image-groups/{id}/icon?h=`（经 store.Open 流式输出，对象仍私有） |

表 `ai_provider_containerimagegroup` 增列 `icon_file_key VARCHAR(512) NOT NULL DEFAULT ''`。JSON 增 `icon_file_key`、`icon_url`。公开目录 `image_group` 与顶层 `icon_url` 同步。

## 拒绝方案

| 方案 | 原因 |
|------|------|
| 图标 URL 由厂商粘贴外链 | 不稳定、SSRF、无法强制格式 |
| JSON 内嵌 base64 | 撑大 API、不便 COS、难缓存 |
| COS public-read | 与证照桶 ACL 不一致；经本服务流式即可公开读 |
| 新建对象存储服务 | 无架构必要，复用 VendorDocStore |

## 架构制品

**不更新** `docs/architecture/` 四件套：无新 Application_Component / 新库 / 新基础设施；仅扩展既有 ImageGroup 聚合字段并复用证照存储端口。

Python 新 API 门禁：not_applicable（Go taskAiProvider）。

## 兼容

- 存量组 `icon_file_key=''`：列表/目录仍返回；下次编辑必填。
- 创建/更新 API 对客户端为破坏性必填；厂商门户同步上线。
