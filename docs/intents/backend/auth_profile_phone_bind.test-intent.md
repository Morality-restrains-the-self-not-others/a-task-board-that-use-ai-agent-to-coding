# 资料页手机绑定真实写入且支持占用转移 — 测试意图

## 单元测试

1. `taskAuth/src/handlers_test.go`：`TestProfileBindPhoneRouteNotSwallowedByUpsertProfile`
   - bind-phone / replace-phone 匹配专用 pattern
   - profile POST 仍为精确 upsert（`{$}`）
2. `taskAuth/src/auth_phone_bind_test.go`
   - `TestBindPhone_SuccessAndTakenConflict`：409 含 `code=phone_taken` 与 `reclaim_available`
   - `TestBindPhone_TakenKeepsCodeForReclaim`：先 409 再同码 reclaim → 200，原绑定作废
3. `taskFE/app/src/utils/phoneBindingApi.test.js`：假成功不回跳、占用识别、reclaim 请求体
4. `UserProfilePhoneBindingPanel.navigate-back.test.js`：假成功不回跳；409 展示转移按钮并带 `reclaim`

## 手工/E2E

1. 邮箱账号绑定已被其他账号占用的号码 → 明确占用文案 + 确认转移 → 资料页显示已绑定，不再被手机门禁打回
