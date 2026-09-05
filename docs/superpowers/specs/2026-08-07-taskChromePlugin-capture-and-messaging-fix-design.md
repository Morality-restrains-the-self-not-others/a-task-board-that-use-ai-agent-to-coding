# taskChromePlugin 网络请求列表为空 + getAuthStatus 消息超时 — 修复设计

> **迭代版本**: 纯 Bug 修复（基线 v13 enterprise-landscape / v65 application-integration ✅ current，本次无架构视图变更）
> **设计日期**: 2026-08-07
> **作者**: claude
> **需求**:
> 1. devtools 的 taskPlugin「网络」请求列表为空，无法捕获请求，请分析原因
> 2. 控制台输出 `[taskChromePlugin] getAuthStatus 回退 Storage: 消息超时: getAuthStatus`（content.js:1139）
> **关联**: `taskChromePlugin/` 子仓（Chrome MV3 扩展，v1.8.0）、`docs/architecture/` v13/v65 基线

---

## 1. 背景与问题

taskChromePlugin 扩展的功能链路：**devtools network 捕获 → devtools 页 → panel 页（请求列表）→ SW（addRecentRequest 持久化兜底）→ content script（登录态/悬浮球）**。

用户报告两个现象：

1. **devtools 的 TaskPlugin 面板「网络」请求列表恒为空**，devtools 明明在捕获请求（`chrome.devtools.network.onRequestFinished`），但面板列表不显示任何条目
2. **content script 获取登录态失败**：`getAuthStatus` 消息 5s 超时 → 回退到 Storage 读取（登录态可能不准，账号过期检测等依赖 auth status 的链路受影响），调用栈 `checkLoginStatus → refreshAuthAndWorkspaces → init`

## 2. 现状勘察（真实扩展实验证据）

在真实 Chromium（xvfb-run + `--load-extension` 加载完整扩展）下做了四轮实验，每轮结论如下：

### 实验 1（SW 全局状态）

| 探针 | 结果 |
|------|------|
| `chrome.alarms` | **undefined**（manifest 无 "alarms" 权限，`hasAlarms: false`） |
| `chrome.runtime.onMessage` 主 listener | 已注册（`hasMainListener: true`） |
| SW 自 ping（SW 内部发消息） | 超时 |
| SW console | 仅一条 `Initialized | baseUrl=https://daydaymoney.com | hasToken=false ...`，**无任何错误** |

### 实验 2（panel 页 CSP 与消息链路）

| 探针 | 结果 |
|------|------|
| `window.__tcpRequestMsgEarly`（head 内联脚本产物） | **不存在**（`hasEarlyBuffer: false`） |
| 页面 console CSP 报错 | **2 条**（`Content Security Policy` 拒绝内联脚本） |
| panel → SW `getAuthStatus` | 8s 超时 |
| 注入 `postMessage({action:'initRequests',...})` | 列表无变化（relay 不存在，消息无人消费） |

### 实验 3（消息通道全面探测）

| 探针 | 结果 |
|------|------|
| 页面 → SW `ping` | 5005ms 超时 |
| 页面 → SW `getFloatBallConfig` | 5004ms 超时 |
| 页面 → SW `getAuthStatus` | 8008ms 超时 |
| SW 直接读 `chrome.storage.local` | **正常**（`authExpiryMigratedV161` 等 1 个键） |
| SW 事件循环心跳（setTimeout 300ms） | **alive** |
| SW 进程 | 无 close 事件，console 无错误 |

### 实验 4（probe listener 二分消息投递层）

| 探针 | 结果 |
|------|------|
| SW 中**新注册** probe listener → 页面发 `__probe` | **不触发**（hits=0），消息根本没到达 SW |
| SW → 页面反向 `sendMessage` | 同样超时 |
| 结论 | **双向 runtime 消息通道整体中断**，与 listener 注册、handleMessage 逻辑无关 |

### 实验 5（最小补丁验证 — 决定性）

复制扩展到临时目录，**唯一改动**：manifest `permissions` 增加 `"alarms"`。重跑实验 4：

| 探针 | 补丁前（原扩展） | 补丁后（仅加 alarms） |
|------|------------------|----------------------|
| `hasAlarms` | `false` | `true` |
| 页面 → SW probe | 超时，hits=0 | **立即响应 hits=1** |
| SW → 页面 反向 | 超时 | **立即响应** |
| 账号过期检测启动日志 | 无 | `账号过期检测已启动（每30分钟）` |

**结论：manifest 缺 `"alarms"` 权限是问题 2 的直接根因，双向消息通道整体恢复。** 两条独立根因均已实验证实。

## 3. 根因分析

### 3.1 问题 1：panel.html head 内联脚本被 MV3 扩展默认 CSP 阻止

`panel/panel.html` 的 `<head>` 中有内联 `<script>`（commit c142530 引入）：

```html
<script>
  window.__tcpRequestMsgEarly = window.__tcpRequestMsgEarly || [];
  window.addEventListener('message', function (event) { ... 转发到 __tcpRequestMsgSink ... });
</script>
```

- MV3 扩展页面默认 CSP 为 `script-src 'self'`，**内联脚本一律被阻止**（实验 2 证实：2 条 CSP 报错、`__tcpRequestMsgEarly` 不存在）
- `panel/panel.js` 的 `bindRequestMessagePipeline()` 在 bootstrap 分支下**只设置 `window.__tcpRequestMsgSink`，自己不注册 `message` listener** —— 消息接收完全依赖被 CSP 阻止的 head 内联脚本
- 于是 devtools 页 `postMessage({action:'newRequest'|'initRequests'|'requestUpdated'})` 无人消费 → 请求列表恒空
- 唯一兜底是 panel 1.5s 后向 SW 拉 `getRecentRequests`，但 SW 消息通道已断（见 3.2）→ 双路径同时失效
- 现有 e2e（`e2e/panel-request-refresh.playwright.test.js`）通过 `file://` 加载 panel.html + mock chrome —— file:// 页面无扩展 CSP，**内联脚本能执行**，因此测试全绿但生产必挂

### 3.2 问题 2：SW 缺 `alarms` 权限 → 顶层 TypeError → SW 启动失败 → 消息通道全断

`background/service-worker.js` 中 `chrome.alarms` 三处使用，manifest `permissions` 无 `"alarms"`（权限缺失时 `chrome.alarms` 为 `undefined`）：

| 行号 | 代码 | 后果 |
|------|------|------|
| 1253 | `chrome.alarms.get(...)`（`startAccountExpiryCheck()` 内，init IIFE **try/catch 之外**、第 1240 行调用） | TypeError → async IIFE 的 promise 变为 rejected → **unhandled rejection** |
| 1255 | `chrome.alarms.create(...)`（同函数内） | 同上（未执行到） |
| **1264** | **`chrome.alarms.onAlarm.addListener(...)`（SW 顶层代码）** | **TypeError 在脚本顶层求值阶段抛出 → SW 启动失败** |

关键机制（实验 4/5 证实）：

- 第 1264 行在脚本顶层执行。`chrome.alarms` 为 undefined → **顶层求值抛 TypeError** → SW 启动失败（进程虽未销毁、CDP 可 eval、storage 可读、init 日志已打印，但浏览器侧认为该 SW 未成功启动）
- 浏览器消息分发器不再向该扩展 SW 投递任何 runtime 消息 → **双向通道全断**：页面 → SW（`ping`/`getFloatBallConfig`/`getAuthStatus`/`addRecentRequest` 全超时）、SW → 页面全部失败
- 影响面是**全局性的**：不只 getAuthStatus —— content script 的所有后台通信、devtools 的 `addRecentRequest` 兜底、悬浮球配置、账号过期检测全部失效
- 为什么线上报的是「回退 Storage」而非硬错误：content.js `sendMessageWithTimeout` 5s 超时后 resolve `{error:'消息超时...'}`，调用方降级读 Storage，**SW 无任何错误日志**，排查极难

### 3.3 两条根因的耦合

问题 1 的主路径（devtools→panel postMessage）与问题 2 无关，**单独成立**；但问题 1 的兜底路径（panel→SW `getRecentRequests`）依赖问题 2 的消息通道，因此线上表现是「列表空 + 兜底也拉不到」。**修复必须两条根因同时处理**。

## 4. 修复方案设计

### 4.1 问题 1 修复：relay 逻辑并入 panel.js（单注册点）

**方案**：删除 panel.html head 内联 `<script>`，消息 relay 统一在 panel.js `bindRequestMessagePipeline()` 内注册（唯一注册点），`sink` 就绪时直接转发、未就绪时 push 到 `__tcpRequestMsgEarly` 缓冲。

```
[迁移前] panel.html head 内联 <script>（被 CSP 阻止）← 唯一消息入口（失效）
[实施后] panel.js bindRequestMessagePipeline() ← 注册 window message listener（唯一注册点）
          head 内联脚本删除（消除 2 条 CSP 报错）
```

> ⚠️ **实施调整说明**（相对初版设计）：初版设计拟新建 `panel/msg-relay.js` 外部脚本 + panel.js 防御性双注册。实施评审发现**双注册会导致同一消息被消费两次**（relay 与 panel.js 都向 sink 转发 → `requestMsgBuffer.push` 两次 → 请求列表重复）。故收敛为**单注册点**：relay 逻辑并入 panel.js（panel.js 是外部脚本，不受 CSP 影响，与迁移到独立外部文件效果等价），head 不再需要任何脚本。

要点：

1. **`bindRequestMessagePipeline()`** 统一注册 `window` `message` listener：`sink` 已设置（bootstrap/直接模式）→ 转发；未设置 → push 到 `__tcpRequestMsgEarly`（保持既有缓冲语义，作时序兜底）
2. **删除** head 内联 `<script>`（顺带消除 2 条 CSP 报错，避免误导后续维护）
3. **时序安全**：panel.js 在页面解析时同步执行完毕，先于 devtools `onShown` 的 `postMessage`（panel 可见回调必然晚于 panel 页面加载完成），早到消息落入 `__tcpRequestMsgEarly` 缓冲，由 `bindRequestMessagePipeline` 末尾消费
4. panel.html 保留注释警告：勿改回内联 `<script>`（CSP 会阻止）

### 4.2 问题 2 修复：manifest 加 `alarms` 权限 + SW 防御性 guard

**主修复**：`manifest.json` `permissions` 增加 `"alarms"`（账号过期检测本来就是该权限的用途）。

**防御修复**（防止未来权限变更再次造成 SW 启动失败、消息通道全断的灾难性故障）：

```js
// startAccountExpiryCheck() 入口增加权限守卫
function startAccountExpiryCheck() {
  if (typeof chrome.alarms === 'undefined') {
    console.warn('[taskChromePlugin] chrome.alarms 不可用（缺 alarms 权限），跳过账号过期检测');
    return;
  }
  ...原逻辑...
}

// 顶层 onAlarm listener 同样守卫（这是 SW 启动失败的直接元凶）
if (typeof chrome.alarms !== 'undefined') {
  chrome.alarms.onAlarm.addListener((alarm) => { ... });
}
```

效果：

- 权限正常时：账号过期检测按原设计每 30 分钟运行
- 权限缺失时：SW 正常启动、消息通道正常，仅日志告警 + 账号过期检测降级（功能降级而非全盘瘫痪）

### 4.3 不采纳的方案

| 方案 | 不采纳理由 |
|------|-----------|
| 只在 manifest 加 alarms，不动 SW | 无法防未来回归；权限缺失的故障模式（SW 静默启动失败、无任何错误日志）过于隐蔽，必须让 SW 对缺失权限有弹性 |
| 移除账号过期检测功能 | 功能本身有效（迁移自 v161 的 token 过期处理），保留 |
| e2e 沿用 file:// 加载方式 | 绕过 CSP 无法捕获问题 1，必须新增真实扩展加载测试 |

## 5. 🕸️ Code Review Graph 分析

图构建：`taskChromePlugin` 子仓首次构建，61 文件 / 1021 节点 / 8210 边 / 119 执行流 / 10 社区。

| 查询 | 结果 | 对设计的影响 |
|------|------|-------------|
| `callers_of bindRequestMessagePipeline` | 仅 panel.js `init` 调用 | 修复点收敛：改 bindRequestMessagePipeline + panel.html，不影响其他调用方 |
| `callers_of handleMessage` | 仅 SW 顶层 onMessage listener（L319-327）调用 | 消息处理逻辑本身无需改动；问题在投递层（SW 启动失败）而非 handler |
| `callers_of pushRequest`（devtools） | `ingestHarEntry`、`backfillFromHar` | HAR 采集 → panel 推送链路完整，唯一断点确认为 relay 消费侧（CSP） |
| `callers_of startAccountExpiryCheck` | 仅 SW 文件顶层 init IIFE（L1240，try/catch 外） | 修复需同时覆盖调用点（移入 try/catch 或函数内守卫）与顶层 onAlarm listener |

爆炸半径结论：修复涉及 3 个文件（`manifest.json`、`panel/panel.html` + 新增 `panel/msg-relay.js`、`panel/panel.js`、`background/service-worker.js`），均无跨社区耦合；content/devtools 侧无逻辑改动。

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| （无新增业务意图） | — | — | — | **纯前端 Chrome 扩展缺陷修复**：修复消息投递与 CSP 拦截，恢复既有能力，不引入任何新的服务端状态变更/跨边界副作用，无对应事件 |

（依据技能硬门禁：纯前端、无服务端状态变更 → 设计文档写明「无对应事件」及理由即可。）

## 7. 架构变更影响

**无需更新架构视图。** 依据架构交付物规则：「纯 Bug 修复…不需要更新架构」——本次为两条缺陷修复：

1. 无新增组件/节点/关系（不改变 devtools → panel → SW → content 拓扑）
2. 无新增接口、无数据模型变更、无外部系统交互变化
3. 基线 v13 enterprise-landscape / v65 application-integration 保持不变，无 target 版本、无 VERSION_HISTORY 变更、无 `.puml`/`.archimate`/`.mermaid.md` 交付

## 8. 测试计划

### 8.1 真实扩展回归测试（新增，防 CSP/权限类回归）

新增 e2e：`e2e/real-extension-messaging.playwright.test.js`，`launchPersistentContext` + `--load-extension` 加载真实扩展：

| # | 断言 | 捕获的回归 |
|---|------|-----------|
| 1 | manifest `permissions` 含 `"alarms"` | 问题 2 权限回退 |
| 2 | 页面 → SW `ping` 往返 < 1s | 消息通道健康（SW 启动失败回归） |
| 3 | SW → 页面反向消息往返 < 1s | 同上（双向） |
| 4 | panel.html 加载无 CSP 报错，`__tcpRequestMsgEarly` 就绪 | 问题 1 回归 |
| 5 | 注入 `postMessage({action:'initRequests'})` 后列表出现条目 | relay 消费链路 |
| 6 | SW console 无 `chrome.alarms` 相关 TypeError | 顶层守卫生效 |

### 8.2 现有测试

- `e2e/panel-request-refresh.playwright.test.js`（file:// + mock chrome）继续保留：快速、无浏览器环境依赖，覆盖 bootstrap 缓冲语义
- `test/` 单元测试（har-request、panel-request-bootstrap 等）不受影响

## 9. 验证清单

- [ ] 真实扩展下 panel 列表能显示 devtools 捕获的请求（含页面刷新后的 backfill）
- [ ] 真实扩展下 `ping` / `getAuthStatus` / `getFloatBallConfig` 全部快速响应，无超时
- [ ] content.js 不再输出「getAuthStatus 回退 Storage: 消息超时」
- [ ] SW console 无 TypeError；账号过期检测日志「每30分钟」正常打印
- [ ] 面板 head 无 CSP 报错
- [ ] 新增 e2e 全绿；既有 e2e 全绿
