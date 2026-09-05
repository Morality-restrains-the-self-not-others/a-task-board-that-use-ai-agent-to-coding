# 登录页「记住我」回填 — 测试意图

## 单元测试
1. `remembered_login_credentials_store.test.js`：save/load/clear；phoneCode/accessToken 不存高敏字段；坏 JSON 安全
2. `applyRememberedLoginCredentials.test.js`：邮箱/手机回填 DOM 与 refs，勾选记住我
3. `useLoginSubmit.test.js`：成功登录时 checked→save、unchecked→clear

## 手工/E2E（可选）
1. 公网 `/auth/login/` 勾选记住我登录 → 退出 → 再进登录页见回填
2. 取消勾选登录后回填消失
