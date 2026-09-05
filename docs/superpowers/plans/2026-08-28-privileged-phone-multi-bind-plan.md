# 实施计划：特权角色一号最多 5 绑定

- **日期:** 2026-08-28
- **设计 / 权限 / 价值流 / NFR / DDD:** 同主题 2026-08-28 文档

## Task 1 — 领域策略（Red→Green）

- [ ] `taskAuth/domain/phone_share.go` + `phone_share_test.go`
- [ ] `EvaluatePhoneShareBind` / `MaxSharedPhoneBindings=5` / `IsPhoneLoginAmbiguous`
- **验证:** `go test ./domain -count=1 -run PhoneShare`

## Task 2 — 列出活跃手机绑定 + 占用判定

- [ ] `listLiveLoginMethodsByPhone`（lookup）
- [ ] `phoneLoginMethodTaken` 改为策略：对 excludeUser 计算 Allow/Taken/Limit
- [ ] 新增 `errPhoneBindLimit`
- **验证:** 先红后绿：staff 第二绑定不再 Taken

## Task 3 — upsert / reclaim / bind handler

- [ ] Limit → 409 `phone_bind_limit`，**不消耗**验证码
- [ ] Taken → 既有 409 `phone_taken`
- [ ] Allow → upsert（同人一行）
- [ ] reclaim 作废该号**全部**其他活跃绑定
- [ ] `phone_bound` 日志 `shared`/`holder_count`
- **验证:** `go test ./src -count=1 -run BindPhone`

## Task 4 — 内部 API

- [ ] `handlePhoneTaken` / `handleUpsertPhoneLoginMethod` 走同一策略
- **验证:** `auth_phone_profile_internal_test.go`

## Task 5 — 登录消歧

- [ ] `handleLogin` 手机路径：list + 密码匹配
- [ ] 0 命中既有文案；1 命中 finalize；≥2 → 400 `phone_ambiguous`
- **验证:** 新建 `auth_phone_share_login_test.go`

## Task 6 — 重置 + 注册

- [ ] 共享号 send/reset by phone → 400 `phone_ambiguous`（send 不发短信）
- [ ] phone_register 仍拒绝任意活跃占用
- **验证:** register + reset 测例

## Task 7 — 解档冲突

- [ ] phone 共享特权 ≤5 不列入冲突
- **验证:** identifier conflict 测例（若已有则扩展）

## Task 8 — 前端

- [ ] `phoneBindingApi.js`：`PHONE_BIND_LIMIT_ERROR_CODE`，limit 不是 reclaim
- [ ] 绑定面板展示上限文案
- [ ] 登录错误展示 `detail`/`error`（phone_ambiguous）
- **验证:** `phoneBindingApi.test.js` + 面板测例

## Task 9 — 意图索引 / 价值流 YAML / WSD

- [ ] INDEX B-019e、`conf/value-stream.yaml`、`docs/flows/value-stream-test-integration.wsd`

## 事件任务

- [ ] 对照表已写豁免 `auth-privileged-phone-multi-bind-no-new-event`；登录不改 USER_LOGGED_IN 契约。无新 publish/consumer。
