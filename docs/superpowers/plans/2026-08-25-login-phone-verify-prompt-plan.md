# 实施计划：登录后未验证手机号弹窗引导

- 日期：2026-08-25
- 关联设计：2026-08-25-login-phone-verify-prompt-design.md

## 垂直切片

### 切片 1 — 领域谓词与深链（taskFE）

- [ ] `phone_verified_status.js` + 单测
- [ ] `phoneBindingDeepLink.js` + 单测（对齐 emailBindingDeepLink）
- 验证：vitest 上述文件
- 意图对照：无新事件（书面例外已记）

### 切片 2 — 弹窗决策服务 + 登录挂钩

- [ ] `post_login_phone_verify_prompt_service.js` + 单测
- [ ] `useLoginSubmit.js`：落盘后、跳转前调用；admin/phonePassword 跳过
- [ ] 扩展 `useLoginSubmit.test.js`
- 验证：vitest useLoginSubmit + prompt service

### 切片 3 — 微信回调 + 资料页锚点

- [ ] `resolveWechatSessionIdentity` 成功时把 profile（含 `has_phone`）交给调用方（保持 boolean 兼容或返回 `{ ok, profile }` 且更新现有测例）
- [ ] `Login.vue` 微信 token 分支：未绑手机则 confirm
- [ ] `UserProfile.vue`：手机面板 `data-rg-key`；加载后若 hash 为 phone_binding 则滚动
- [ ] `UserProfilePhoneBindingPanel` 外包一层或组件根节点带 rg-key
- 验证：vitest sessionUserIdUtils.wechat + UserProfile 锚点测例

### 切片 4 — 收尾

- [ ] 更新 `docs/flows/value-stream-test-integration.wsd` 测试点
- [ ] 登记精准编译重启 `taskFE`
- [ ] 相关 vitest 全绿
