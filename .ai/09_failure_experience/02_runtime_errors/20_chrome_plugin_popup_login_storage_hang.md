# [运行时] Chrome 插件 Popup 点击登录无响应 / 无法登录

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-16
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 扩展 Popup 填写服务器地址、账号、`at_` 访问令牌后点击「登录」，按钮无反馈或一直停在可点状态，无法进入已登录态。
- 或短暂显示「登录中...」后仍回到登录表单，错误信息不清晰。
- **典型**：Network 已见 `POST .../login-with-access-token/` **200**（含 `token`/`user`，如 traceId `17346d83d6656defa923845e`），但 Popup 仍停在登录页。

## 根因

1. **`handleTokenLogin` 在发起登录前无超时地 `await Storage.saveBaseUrl()`**。与 Popup 启动态同类：当 `chrome.storage` 挂起时，点击登录的 Promise 永久卡住，UI 不进入「登录中」、也不发 SW 消息（参见 `3e3a104` spinner 修复，但当时未覆盖登录点击路径）。
2. **登录请求复用通用 `request()`，会附带 SW 内存中的旧 `Authorization`**。`API.init(baseUrl, undefined)` 不会清空 token；登出只清 storage 不清内存。脏 Token 在部分公开/鉴权链路上会干扰（对照失败经验 `09_allowany_public_api_invalid_token_403.md`）。
3. **SW 在响应无 `token` 时仍 `success: true`**，Popup 以为成功再 `loadState`，表现成「登不上」。
4. **SW 在 HTTP 登录成功后仍 `await broadcastAuthStateChanged()`**。对部分 discarded/frozen 标签页 `tabs.sendMessage` 永不 resolve，导致 `sendResponse` 迟迟不返回；Popup 收不到 `success`，界面卡在登录页（API 已 200）。

## 解决方案

- Popup：先切「登录中」；`saveBaseUrl` / 登录后 `loadState` 均 `withTimeout`；storage 失败仍继续 `loginWithAccessToken`；**收到 `success` 后立即 `showLoggedInUI`**，再异步刷新态。
- `API.requestUnauthenticated`：密码/访问令牌登录不带 Authorization；`init('', …)` / `clearSession()` 可清空内存会话。
- SW：登录前 `API.init(baseUrl, '', …)`；无 session token 返回 `success: false`；新增 `logout` 清 storage + `clearSession`。
- SW：`finalizeLoginSuccess` / `scheduleAuthBroadcast` — **持久化带超时，广播 fire-and-forget**；单标签广播亦 `withTimeout`。
- 错误体解析：优先展示 `error` / `detail` / `message` / `non_field_errors`。

## 预防

- 凡 Popup 内 `await chrome.storage.*` 须带超时或不得阻塞用户主路径（登录/登出/提交）。
- **登录/登出响应路径禁止 `await` 全量标签页广播**。
- 登录类公开 API 禁止附带旧 Authorization。
- 回归：`taskChromePlugin/test/api-login.test.js`、`test/login-finalize.test.js`、`e2e/popup-token-login.playwright.test.js`（pre-commit 在改登录链路时强制跑）。

## 验证

```bash
cd taskChromePlugin && npm test && npm run test:e2e:popup-login
# 手动：chrome://extensions 重新加载扩展 → Popup 用 at_ 令牌登录 → 应显示已登录 / DevTools 引导
```

## 关联

- Popup spinner 卡死：`3e3a104` / `test/async-timeout.test.js`
- 脏 Token 打挂公开接口：`09_allowany_public_api_invalid_token_403.md`
