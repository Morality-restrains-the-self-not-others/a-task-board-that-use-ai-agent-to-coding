# 实施计划 — 推荐资格个人介绍

- **Date:** 2026-08-20
- **DDD:** `docs/superpowers/specs/2026-08-20-referral-personal-intro-ddd.md`

## 事件契约

无新 Kafka 事件（见 DDD 例外）。意图对照更新：`docs/intents/backend/task-referral.intent.md`、`docs/intents/frontend/user_referral_access_code.intent.md`。

成功路径日志：`referral_application_submitted`（`personal_intro_len`）。

## 任务

- [x] **T1** 红：`referral_code_test.go` — 空/短/超长介绍失败；合法介绍 pending 且库中原文；列表含 intro
- [x] **T2** 绿：DDL `006_referral_personal_intro.sql` + `normalizePersonalIntro` + apply/list/status 读写
- [x] **T3** Handler：POST body `personal_intro`；400 `invalid_intro`；成功日志禁正文
- [x] **T4** FE：申请表 textarea + 名额说明 + clickGuard；pending 只读展示；超管列表/拒绝弹层展示
- [x] **T5** 意图/测试意图更新；`gofmt` / `go test` / vitest；登记精准重启 taskReferral + taskFE
