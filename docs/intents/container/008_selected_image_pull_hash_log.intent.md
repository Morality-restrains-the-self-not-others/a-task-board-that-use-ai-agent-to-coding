# 意图：selected_image 启动日志记录镜像 hash

- **日期**: 2026-07-10
- **关联页面**: `http://183.250.1.132:4000/tenant/.../task-detail/task_12590983282794675865/?relayToTrae=true`
- **状态**: 已实现

## 问题

任务详情「启动」后，侧车/容器日志仅有镜像 tag（如 `...:x86_64-latest`），无法核对实际拉取到的镜像内容是否与预期一致。

## 验收标准

- [x] `docker pull` 成功后执行 `docker image inspect`，日志输出 `id=sha256:…` 与（若有）`digest=…@sha256:…`
- [x] `docker run` 日志行附带 `image_id` / `image_digest`
- [x] start 结果 JSON 含 `image_id` / `image_digest`（有则返回）
- [x] Playwright（CDP）启动后在 go_relay 日志中可见 `sha256:` 镜像 hash



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：启动日志增强（可观测性），无业务领域事件投递。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 意图：selected_image 启动日志记录镜像 hash | — | — | — | — | 启动日志增强（可观测性），无业务领域事件投递 |
## 变更要点

| 组件 | 变更 |
|------|------|
| go_relayToTrae | `inspectImageIdentity` + pull/run 日志与 start 结果字段 |
