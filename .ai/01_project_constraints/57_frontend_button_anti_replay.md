# 前端按钮点击须有防重放设计（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-19
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 52 条
- Cursor：`.cursor/rules/frontend-button-anti-replay.mdc`（alwaysApply）
- ADR：[ADR-0020](../../docs/adr/0020-frontend-button-anti-replay.md)
- 参考实现：`taskFE/app/src/utils/clickGuard.js`
- 门禁：`db/scripts/ci/check_frontend_button_anti_replay.py`

## 背景与动机

用户连点、Vue `disabled` 要等到下一帧才生效、HTTP 超时后客户端重试，都会把**同一用户意图**打成两次写请求。仅靠 debounce **不是**防重放：debounce 挡误触，挡不住超时重试与进行中的第二次点击。

本条约束前端点击侧必须同时具备：

1. **交互锁**（同步门闩 + 禁用态 + 可感知 pending）
2. **写请求身份**（`Idempotency-Key`，同一意图内重试回传同一键）

服务端幂等仍由元规则 48 / 49 负责；前端锁是 UX 与降低重复流量，**不能**替代服务端去重。

## 核心规则

### 1. 适用范围

凡会触发**副作用**的可点击控件（`<button>`、`type="submit"`、`role="button"`、带 `@click`/`onClick` 的卡片操作），包括：

- 创建 / 更新 / 删除 / 支付 / 充值 / 退款 / 提交表单
- 启停云资源、授权、邀请、发放配额
- 其它会发出 `POST` / `PUT` / `PATCH` / `DELETE`（或等价 RPC）的点击

纯 UI 切换（Tab、下拉展开、关闭弹层、本地折叠）与真实导航 `<a href>`（元规则 44）不适用本条的 Idempotency-Key 要求；读刷新按钮仍须交互锁，避免请求风暴。

### 2. 三层设计（缺一不可，写路径）

| 层 | 作用 | 最低要求 |
|---|---|---|
| L1 同步门闩 | 首帧重绘前挡住连点 | `createClickGuard().run` 或等价 `busyRef`；**禁止**只设 `creating.value = true` 而无同步锁 |
| L1 可感知态 | 用户知道进行中 | `disabled` + `aria-busy` + 文案（如「提交中…」） |
| L1 短窗 debounce | 误触 | 默认 ~300ms（可按控件调整）；**单独 debounce 不够** |
| L2 请求身份 | 超时重试 / 二次点击仍是同一笔 | 点击被接受时生成 `Idempotency-Key`；同一次 `run` 内所有 HTTP 重试回传**同一键**；下一次被接受的新点击才换新键 |
| L3 服务端 | 真正防双写 | 后端按元规则 48 兑现该键；资金/配额/云资源 ≥ L3 |

推荐使用 `taskFE/app/src/utils/clickGuard.js` 的 `createClickGuard` + `mergeIdempotencyHeaders`。其它前端栈须提供等价物。

### 3. 禁止

- 只把按钮 `:disabled` 绑到异步 `ref`，没有同步门闩
- 每次 HTTP 重试都 `crypto.randomUUID()` 新键（把重试变成新业务）
- 声称「用户不会连点」或「debounce 已足够」作为资金/建单路径的防重放
- 用 `company_id` / `tenant_id` / `user_id` 当点击幂等键（过粗；与元规则 48 一致）

### 4. 例外（须在控件旁注释）

| 注释 | 适用 |
|------|------|
| `Anti-Replay-OK: ui-only` | 无网络写：展开/关闭/Tab |
| `Anti-Replay-OK: navigation` | 真实 `<a href>`，非按钮冒充链接 |
| `Anti-Replay-OK: read-refresh` | 只读刷新；仍须 L1 锁，可无 Idempotency-Key |

无注释的写点击一律按 §2 执行。

## 与既有规则的关系

| 规则 | 管什么 | 与本条关系 |
|---|---|---|
| 禁止链接点击拦截（元规则 44） | 导航必须用真实 `href` | 本条管**按钮写操作**；导航不走 Idempotency-Key |
| NFR 幂等性审视（元规则 48） | 设计文档里的键与重放语义 | 用户双击是重复触发源；NFR 表须写前端锁 + 键如何产生 |
| 事件消费者幂等（元规则 49） | Kafka 消费去重 | 本条在点击入口减流量；49 在消费侧兜底 |
| API 幂等（`02_api_specifications.md`） | 接口接受 `Idempotency-Key` | 前端必须回传；后端必须兑现 |
| `.cursor/rules/frontend-button-interaction.mdc` | 打开 Vue/TSX 时的按钮提醒 | 指向本专文 |

## 验收

```bash
python3 db/scripts/ci/test_check_frontend_button_anti_replay.py
python3 db/scripts/ci/check_frontend_button_anti_replay.py
# 新增/改写的写按钮 Vue：--strict --files
# python3 db/scripts/ci/check_frontend_button_anti_replay.py --strict --files 'taskFE/app/src/views/OrderCreate.vue'
cd taskFE/app && npx vitest run src/utils/clickGuard.test.js
```

## 变更日志

- 2026-08-19：版本 1.0.0 - 初版；按钮点击强制交互锁 + 写路径 Idempotency-Key
