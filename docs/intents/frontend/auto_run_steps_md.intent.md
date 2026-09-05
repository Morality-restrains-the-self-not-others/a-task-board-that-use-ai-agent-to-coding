# 意图：autoRunStep.md 镜像说明

## 目标

镜像内 `/app/autoRunStep.md` 为自动运行说明真源；注册（含**开发中**草稿）时 OCI 抽取缓存；创建任务 / 任务详情 / 镜像市场（可用 + **开发中**）按所选镜像展示。

## 范围

- onlineServiceJS：`GET /api/auto-run-steps`
- Go：internal 抽取 API；taskAiProvider 保存镜像与 catalog GET 触发抽取并写缓存
- ai-provider / taskCloud：字段扩展与安装拷贝
- Vue：ImageMarket、CreateTask、TaskDetail

## 非目标

- 厂商手填 Markdown 作为真源
- 已安装快照自动跟厂商每次改镜像同步（本期快照语义）


## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 厂商保存/更新镜像 URL 后抽取 autoRunStep.md | ContainerImageAutoRunStepsExtracted | taskAiProvider `runAutoRunExtract` | 写 `auto_run_steps_md` / extract_status / digest；无独立 Kafka 消费者 | — |
| 公开 catalog / 开发中 catalog 对空缓存回填抽取 | ContainerImageAutoRunStepsExtracted | 同上（GET 触发 schedule） | 同上 | — |
## 变更记录

- 2026-08-22：第 4 步说明补 OAuth 绑定前提；GitLab 交付须把 match-key token 写入 oauth_auth_by_repo
- 2026-08-20：厂商保存与 catalog GET 接上 OCI 抽取；空缓存回填 pending→ok/not_found
- 2026-08-20：镜像市场空自动运行不占位；catalog payload 与卡片补齐 `updated_at`
- 2026-07-14：初版；明确开发中镜像与上架镜像同一管道
