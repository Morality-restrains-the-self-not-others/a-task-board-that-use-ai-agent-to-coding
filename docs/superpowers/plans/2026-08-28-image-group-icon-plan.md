# 镜像组图标 — 实施计划

- **日期**: 2026-08-28

## 任务

- [x] 领域：`image_group_icon` kind + ValidateImageGroupIconMeta + 单测
- [x] DDL：`dataMigrate/taskAiProvider/010_image_group_icon.sql`
- [x] Store：ImageGroup CRUD 读写 `icon_file_key`；catalog 附加 `icon_url`
- [x] Handler：upload-url / local-put / complete；POST/PUT 必填；GET public icon
- [x] 事件：ImageGroupIconUploaded / Created / Updated
- [x] 前端：`imageGroupIcon.js` + 弹窗必填 + 列表缩略图 + clickGuard
- [x] taskFE ImageMarket 组卡 `<img :src="icon_url">`
- [x] OpenAPI + route ownership 前缀
- [x] 意图文档 `docs/intents/`
- [x] 跑 Go 相关单测 + 前端 unit test

## 意图 → 事件

| 意图 | 事件 | 发布点 |
|------|------|--------|
| 上传镜像组图标 | ImageGroupIconUploaded | upload-complete |
| 创建镜像组 | ImageGroupCreated | POST 成功 |
| 更新镜像组 | ImageGroupUpdated | PUT 成功 |
