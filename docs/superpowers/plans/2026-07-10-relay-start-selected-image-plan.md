# 实施计划：relay 启动所选镜像

## Task 1 — go_relayToTrae：docker 镜像启动路径（TDD）

- [ ] 单测：start body 含 `image` 时走 pull/run（mock docker exe），不找 run.sh
- [ ] 实现 `startSelectedImageContainer`；state 增加 ContainerID/Name/Image
- [ ] stop 时 docker stop 命名容器
- [ ] 有 image 时跳过 host token 文件等待

## Task 2 — Gateway：resolve + 转发

- [ ] 单测：start 带 installed_image_id → resolve-image → payload.image
- [ ] 单测：resolve 失败 → 4xx，不 forwardToRelay
- [ ] 实现 handleRelayStart 集成

## Task 3 — Django 兜底对齐

- [ ] relay_to_trae_start 解析 installed_image_id（请求或任务快照）写入 payload

## Task 4 — 前端

- [ ] startRelayToTrae body 传 installed_image_id；未选镜像拦截

## Task 5 — 文档与架构

- [ ] intents 011（已写）
- [ ] v15 architecture 三件套 + VERSION_HISTORY
- [ ] 价值流测试点（已写）

## Task 6 — 验证

- [ ] go test go_relayToTrae / taskContainerGateway 相关包
- [ ] 前端相关单测（若有）
