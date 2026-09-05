# 实施计划：登录页电话验证码登录可用

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-15-login-phone-otp-enable-design.md`

## Task 1: 意图文档

- [ ] `docs/intents/frontend/login_phone_otp_enable.intent.md` + `.test-intent.md`

## Task 2: Data migration（TDD）

- [ ] Red/Green：migration 后 global policy `enable_phone_login is True`
- [ ] 文件：`projects/migrations/0060_enable_phone_login_global.py`（序号以现网最后为准）

## Task 3: Login 回跳 + 多账号槽（TDD）

- [ ] Red：Vitest — `next=/tenant/.../recharge/` 解析为回跳；OIDC next 仍优先
- [ ] Red：登录成功调用 `persistLoginAccountSlot`
- [ ] Green：改 `Login.vue`（及必要时抽出纯函数便于测）

## Task 4: value-stream + flows

- [ ] 更新 `conf/value-stream.yaml`
- [ ] 如需补 `docs/flows` 测试点

## Task 5: 回归

- [ ] Django migration / policy 相关测
- [ ] Vitest login redirect / slot
- [ ] 前端 build（ship 前 collectstatic）

## Task 6: Review + Ship

- [ ] simplify-and-harden / review
- [ ] PR（不直接 merge）
