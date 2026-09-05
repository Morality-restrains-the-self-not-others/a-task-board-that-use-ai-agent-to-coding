# 登录后未验证手机号硬门禁 — 测试意图

## 单元测试

1. `phoneVerifiedStatus.test.js`：`userHasVerifiedPhone`（既有谓词不变）
2. `phoneBindingDeepLink.test.js`：RG key、URL、绑定成功回跳暂存
3. `postLoginPhoneVerifyPrompt.test.js`（领域服务）
   - 已验证 → `{ action: 'redirect', href }`
   - 未验证且用户确认 → `{ action: 'verify', href: profile deep link }` 并暂存原落点
   - 未验证且用户关闭/拒绝弹窗 → 仍 `{ action: 'verify' }`（不可跳过）
   - 无 acknowledge 适配器 → 仍 verify（fail-closed）
   - 判定抛错 → fail-closed verify
   - `skipPrompt`（admin / phonePassword / impersonation）→ 直接 redirect
4. `useLoginSubmit.test.js`：未绑手机跳转资料绑定深链（含弹窗 reject）；已绑定不弹窗
5. `phone_verify_access_gate.test.js`：豁免路径/角色/未知会话不阻断；业务路径未验证则阻断
6. `PhoneVerifyAccessGate`：工作面板未验证展示阻断层；点「去验证」整页跳资料绑定
7. `UserProfilePhoneBindingPanel`：绑定成功回跳；假成功不回跳；占用 409 可确认转移

## 手工/E2E（可选）

1. 公网无手机号邮箱账号登录 → 仅「去验证」→ 资料页手机绑定区；页面无「稍后再说」
2. 未验证状态下打开工作面板 URL → 阻断层 → 去验证
3. 已绑手机账号登录 → 无弹窗、无阻断
