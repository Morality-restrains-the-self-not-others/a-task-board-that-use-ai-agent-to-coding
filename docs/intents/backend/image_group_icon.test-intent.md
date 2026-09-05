# 测试意图：镜像组图标必填

- **日期**: 2026-08-28
- **对应功能意图**: `docs/intents/backend/image_group_icon.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | POST 组仅 name+description | 400 镜像组图标必填 |
| T2 | 上传 png 后 POST 组 | 201 + icon_url |
| T3 | PUT 组清空 icon_file_key | 400 |
| T4 | PUT 保留已有 key | 200 |
| T5 | GET public icon 无 key | 404 |
| T6 | GET public icon 有文件 | 200 image/* |
| T7 | 他厂商 file_key | 400 归属失败 |
| T8 | 前端无文件保存 | 文案镜像组图标必填、不发请求 |
| T9 | 市场卡片有 icon_url | 渲染 img[src] |

## 实现落点

- Go: `src/handlers_test.go` / `domain/image_group_icon_test.go`
- JS: `frontend/tests/imageGroupIcon.unit.test.js`
- taskFE: `ImageMarket.card-fields.test.js` 增断言
