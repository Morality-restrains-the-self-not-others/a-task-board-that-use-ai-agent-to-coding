# MCP 浏览器扩展调试能力 — 摘除 --disable-extensions + 原生扩展工具

> **迭代**: mcp-browser-extension-debugging
> **作者**: claude
> **日期**: 2026-08-08
> **状态**: 🎯 已设计（待审批）
> **架构版本**: 无架构变更（用户确认 — 纯工具链配置变更）
> **关联**: `.mcp.json`（meta 仓）、`taskChromePlugin/` 子仓（Chrome MV3 扩展 v1.8.0）、chrome-devtools-mcp@1.6.0

---

## 1. 背景与目标

### 1.1 业务诉求（用户原话）

> 如何解决 `MCP 浏览器带 --disable-extensions，无法直接复现扩展问题。`

### 1.2 需求澄清结论（用户已确认）

| # | 决策点 | 结论 |
|---|--------|------|
| 1 | 需复现/调试的扩展范围 | **全四部分**：内容脚本+页面交互、Service Worker 消息链路、Popup 弹窗 UI、DevTools 面板 |
| 2 | MCP 调试浏览器 profile 策略 | **专用持久 profile**（`--user-data-dir` 固定目录）：扩展装一次长期在、登录态可保留；定期清理 |
| 3 | 是否更新架构设计稿 | **不更新** — 视为纯工具链配置变更（`.mcp.json` + 辅助脚本），无新增组件/数据流 |
| 4 | `--enable-extensions` 替代方案 | **否决** — Chrome 对 `disable-extensions` 是 HasSwitch 存在性判断，追加对抗 flag 无效（详见 §2.3） |

### 1.3 目标态

MCP 驱动的浏览器（chrome-devtools-mcp 启动的实例）可：
- 安装并持久加载 taskChromePlugin（unpacked 路径）
- 在任意页面复现内容脚本行为（元素选择器、浮窗、DOM trace）
- 直接选中扩展 service worker 调试消息链路（evaluate / console）
- 触发扩展默认 action（弹窗）并快照驱动 Popup UI
- 兜底方案覆盖 DevTools 面板（panel.html 直连导航 + 既有 Playwright e2e）

---

## 2. 现状分析

### 2.1 现状与问题

`.mcp.json` 中 chrome-devtools 服务为**无参数启动**：

```json
"chrome-devtools": { "command": "npx", "args": ["-y", "chrome-devtools-mcp@1.6.0"] }
```

MCP 自行启动 Chrome（Puppeteer 内嵌）时，默认参数中包含 `--disable-extensions`（`build/src/third_party/index.js:72685` 实证）。后果：taskChromePlugin 的 content scripts / service worker / popup / devtools panel **全部不加载** → 扩展问题无法在 MCP 浏览器内直接复现。

现状的替代手段：真实 Chromium + `--load-extension`（xvfb + Playwright）做实验（见 `2026-08-07-taskChromePlugin-capture-and-messaging-fix-design.md`）——批处理回归可行，但无 MCP 交互式调试（快照/点击/evaluate/断点式排查）能力，排查周期长。

### 2.2 关键探索证据（chrome-devtools-mcp@1.6.0 源码实证）

| 发现 | 证据 | 意义 |
|------|------|------|
| **原生扩展工具** `--categoryExtensions`（默认关） | `chrome-devtools-mcp-cli-options.js:228` | 开启 5 个工具：`install_extension`（unpacked 路径）/ `list_extensions` / `reload_extension` / `uninstall_extension` / `trigger_extension_action`（触发默认 action） |
| **list_pages 含扩展 SW** | `tools/pages.js:13`（`including extension service workers`） | 扩展 service worker 作为页面目标出现，可 select + evaluate/console 调试 |
| **install_extension = CDP `Extensions.loadUnpacked`** | `third_party/index.js:65297` | 原生域安装 unpacked 扩展，非 hack |
| **仅支持 pipe 连接** | `cli-options.js:228` 注 | 即 MCP 自启动 Chrome 模式（`browser.js:172 pipe: true` 实证，本就是默认）；`--browserUrl`/`--wsEndpoint`/`--auto-connect` 不支持 |
| **Chrome 149+ 门槛** | 同上 | 本机 **Chrome 153.0.7979.3 dev**（`/usr/bin/google-chrome`）✓ |
| **摘除默认参数** `--ignore-default-chrome-arg='--disable-extensions'` | `cli-options.js:373`（官方 example） | 从 Puppeteer 默认参数中移除，非追加对抗 |
| **isolated 默认 false + 默认持久 profile** | `cli-options.js:93-100` | 默认 profile 已持久（`$HOME/.cache/chrome-devtools-mcp/chrome-profile*`）；显式 `--user-data-dir` 与 isolated 互斥，指定后即专用持久 |

### 2.3 `--enable-extensions` 替代方案（A/B 实证修正）

设计阶段初判「追加 `--enable-extensions` 无效」（依据 `HasSwitch("disable-extensions")` 存在性判断），**实证推翻该判断**：

**V1-A 实证结果（Chrome 153.0.7979.3 dev）**：在默认参数（含 `--disable-extensions`）基础上追加 `--chrome-arg=--enable-extensions` + `--categoryExtensions` → `install_extension` 成功、`list_extensions` 显示 **Enabled**、内容脚本正常注入（38 个 `taskplugin-*` 元素）、SW 存活且出现在 `list_pages`。

即：Chrome 153 对自动化场景存在 `--enable-extensions` **重开覆盖逻辑**（`--disable-extensions` 与 `--enable-extensions` 并存时后者生效），`--enable-extensions` 是**可用**的替代路径。

**选型结论**：仍采用 `--ignore-default-chrome-arg=--disable-extensions`（官方文档示例路径，语义为「移除而非对抗」，不依赖 153 才有的覆盖逻辑，跨版本更稳）。两条路径均经实证可用，用户环境 Chrome 153 无兼容性风险。

---

## 3. 方案设计

### 3.1 方案选型

| 方案 | 做法 | 优点 | 缺点 | 结论 |
|------|------|------|------|------|
| **A. MCP 原生扩展工具** | `.mcp.json` 加 `--ignore-default-chrome-arg=--disable-extensions` + `--categoryExtensions` + `--user-data-dir` 固定目录；辅助脚本管理 profile | 零新增基础设施；扩展四部分基本全覆盖；SW 可直接 evaluate；官方支持路径 | DevTools 面板无法覆盖（需真实 DevTools UI）；Chrome ≥149 门槛；持久 profile 状态残留需清理 | ⭐ **推荐** |
| B. 外挂浏览器 + `--browserUrl` | 手动启动 Chrome（`--load-extension` + `--remote-debugging-port`），MCP 以 `--browserUrl` 接入 | 可带真实用户 profile；扩展工具上线前的老路 | **扩展工具不支持 browserUrl**（官方注）；手动编排生命周期；自动化与真实浏览状态混用风险 | ✖ 否决（MCP 版本已支持原生方案） |
| C. 仅 Playwright e2e | 维持现有 `e2e/*.playwright.test.js`（真实扩展 + xvfb） | 已有基础设施；面板捕获流水线可回归 | 无 MCP 交互式调试；排查周期长 | 保留为 **fallback**（面板流水线回归） |

### 3.2 方案 A 具体设计

#### A1. `.mcp.json` 参数变更

```json
"chrome-devtools": {
  "command": "/tmp/ram-work/scripts/mcp-chrome-debug.sh",
  "args": []
}
```

辅助脚本 `scripts/mcp-chrome-debug.sh`（meta 仓，单点持有配置，避免 .mcp.json 硬编码绝对路径）:

```bash
#!/usr/bin/env bash
# MCP 调试浏览器：摘除 --disable-extensions + 开启扩展工具 + 专用持久 profile
set -euo pipefail
PROFILE_DIR="${MCP_CHROME_PROFILE:-$HOME/.cache/chrome-devtools-mcp/ram-work}"
# Chrome ≥149 检查（扩展工具门槛）
"$(command -v google-chrome || command -v chromium)" --version | grep -qE 'Chrome (1[5-9][0-9]|[2-9][0-9]{2})' || {
  echo "[mcp-chrome-debug] 警告: Chrome < 149, 扩展工具不可用" >&2; }
exec npx -y chrome-devtools-mcp@1.6.0 \
  "--ignore-default-chrome-arg=--disable-extensions" \
  --categoryExtensions \
  "--user-data-dir=${PROFILE_DIR}" \
  "$@"
```

要点：
- **两个参数缺一不可**：`--ignore-default-chrome-arg=--disable-extensions`（扩展能加载）+ `--categoryExtensions`（暴露扩展工具）
- **`--user-data-dir`** 指定专用持久 profile（独立于 MCP 默认共享 profile，登录态/扩展安装跨会话保留）；`$MCP_CHROME_PROFILE` 环境变量可覆盖
- 保持 pipe 连接（默认）→ 满足扩展工具连接约束；不引入 `--browserUrl`
- 不启用 `--isolated`（其与 `--user-data-dir` 互斥，且隔离 profile 与「持久」目标冲突）

#### A2. 覆盖矩阵（扩展四部分）

| 扩展面 | MCP 工具序列 | 说明 |
|--------|-------------|------|
| **内容脚本 + 页面交互** | `install_extension` → `navigate_page`（任务页/任意页）→ `take_snapshot` / `click` / `evaluate_script` | content.js 注入后即可复现元素选择器、浮窗、DOM trace |
| **SW 消息链路** | `list_pages`（含 SW target）→ `select_page`（SW）→ `evaluate_script` / `list_console_messages` | 直接调试 service-worker.js：消息分支、captured-buffer、账号槽位 |
| **Popup 弹窗 UI** | `trigger_extension_action` → `take_snapshot` → `fill_form` / `click` | 弹窗登录、快捷键配置、使用说明折叠区 |
| **DevTools 面板** | （原生工具覆盖不到）→ `navigate_page` 到 `chrome-extension://<id>/panel/panel.html` | 面板 UI 可复现（扩展页 chrome.runtime 可用）；**捕获数据链路为空属预期**（需 devtools.js 数据源）；devtools.html 需真实 DevTools 上下文不可导航 |

#### A3. DevTools 面板兜底策略

| 问题类型 | 复现方式 |
|----------|----------|
| 面板 UI / 布局 / 按钮交互 / 登录态展示 | MCP 直接导航 `chrome-extension://<id>/panel/panel.html`（面板页不依赖 chrome.devtools API，可直接开在标签页） |
| 捕获流水线（devtools.js 监听 → panel 列表 → SW 持久化） | 既有 Playwright e2e（`e2e/real-extension-messaging.playwright.test.js` 等，真实扩展 + xvfb）— 已覆盖消息通道与 panel 流水线回归 |
| devtools.html 自身逻辑 | 需真实 DevTools UI 上下文，MCP 与 Playwright 均不可直接驱动；走人工步骤 + 既有设计文档实验方法（`2026-08-07-...capture-and-messaging-fix-design.md`） |

#### A4. 辅助脚本命令（profile 管理）

| 命令 | 行为 |
|------|------|
| `scripts/mcp-chrome-debug.sh`（无参） | 以调试参数启动 MCP 服务器（被 .mcp.json 调用） |
| `scripts/mcp-chrome-debug.sh --version-check` | 仅输出 Chrome 版本与工具可用性预检（供调试前确认） |
| `scripts/mcp-chrome-debug.sh --cleanup` | 删除专用 profile 目录（登录态/扩展重装） |

> **V7 实证结论**：`Extensions.loadUnpacked` 为**会话级注册，不跨 MCP/Chrome 重启持久**（同一 `--user-data-dir` profile，新实例 `list_extensions` 为空）。标准流程定为「**每次会话首个动作 `install_extension`**」——幂等，且天然热更新（重载即最新代码，利于迭代调试）。

#### A5. 调试工作流示例（会话内）

```
0. install_extension /tmp/ram-work/taskChromePlugin   # 每次会话首个动作（V7: 不跨会话持久；幂等+热更新）
1. list_extensions          # 确认扩展在位 Enabled
2. navigate_page 任务页     # 复现内容脚本问题（快照/点击/选择器）
3. list_pages → 选中 SW     # 排查消息链路（evaluate_script 直接探 SW 状态）
4. trigger_extension_action # 打开弹窗 → 快照驱动登录/快捷键配置
5. navigate chrome-extension://<id>/panel/panel.html  # 面板 UI 复现
```

### 3.3 影响面

| 变更 | 位置 | 说明 |
|------|------|------|
| 🟡 修改 | `.mcp.json` chrome-devtools 服务 | command 指向包装脚本 |
| 🆕 新增 | `scripts/mcp-chrome-debug.sh` | 启动/预检/清理 |
| 🆕 新增 | 本文档 | 调试工作流 SSOT（含 A/B 验证记录） |
| 不变 | taskChromePlugin 源码 / Playwright e2e | 完全兼容，e2e 保留为面板流水线回归 |

---

## 4. 验证计划（实现后验收清单）

| # | 验证项 | 通过判据 |
|---|--------|----------|
| V1 | A/B 实证 `--enable-extensions` vs 摘除参数 | ✅ 双路径均有效（Chrome 153）；主路径用官方 `--ignore-default-chrome-arg`（见 §2.3 修正记录） |
| V2 | 扩展安装 | ✅ `install_extension` 返回扩展 id `ogicebjhkhmjlacnmibkkgphibgbojom` |
| V3 | 内容脚本注入 | ✅ 导航 https://example.com 后 `#taskplugin-daydaymoney-status` 存在、`[id^="taskplugin-"]` 38 个 |
| V4 | Popup 驱动 | ✅ `trigger_extension_action` 后 `list_pages` 出现 `.../popup/popup.html` 扩展页 target（可选中驱动） |
| V5 | SW 可调试 | ✅ `list_pages` 含 `sw-1: chrome-extension://…/background/service-worker.js`（可选中 evaluate） |
| V6 | 面板 UI 兜底 | ✅ 导航 panel.html 快照完整渲染：5 个 tab（单请求/批量/错误列表/历史/使用说明）+ 请求列表 + 搜索 + 方法过滤 |
| V7 | 持久化 | ❌ **不持久** — loadUnpacked 为会话级；标准流程改为「每次会话首动作 install_extension」（幂等+热更新） |
| V8 | 回归 | ✅ 未触碰 taskChromePlugin 源码与 Playwright e2e，零回归面 |

---

## 5. 横切分析

### 🕸️ Code Review Graph 分析

```
CRG unavailable: 图过期（built 2026-08-06, head 2026-08-08 不匹配; 仅索引 12 文件 93 节点，
不含 taskChromePlugin / chrome-devtools-mcp 代码）
```

理由：本次变更范围是 MCP 配置与辅助脚本（`scripts/mcp-chrome-debug.sh` 为新建），无既有符号面；对扩展代码零修改。CRG 不提供增量信息，重建成本高于收益，设计基于 chrome-devtools-mcp 源码实证完成。

### Domain Concept Inventory

**n/a（skipped_non_code）** — 纯前端/工具链配置变更：无服务端 bounded context、实体、聚合、领域事件；不产生任何服务端状态变更。`taskChromePlugin` 的域模型（captured request / account slot 等）已在其既有设计文档中登记，本次不新增概念。

### 业务意图 → 事件对照

**无对应事件** — 本设计不引入任何业务意图（纯调试工具链）；服务端无状态变更，按硬门禁「纯查询/只读/纯工具链」例外豁免。

### Value Stream Impact

**无 value-stream.yaml**（仓根不存在），且为纯工具链变更，无业务流/字段/测试文件影响。跳过价值流映射（符合 skill 跳过规则）。

### 🐍 Python 新增接口门禁

**not_applicable** — 无任何新增接口（前后端均无）。

---

## 6. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 扩展工具需 Chrome ≥149 | 旧环境无扩展工具 | 本机 153 ✓；包装脚本做版本预检，低于门槛时给出明确告警 |
| 持久 profile 状态残留（登录态/多账号槽位/捕获缓存） | 复现场景与真实用户偏差 | `--cleanup` 命令一键重置；文档注明「复现登录类问题前先清理」 |
| ~~loadUnpacked 跨重启持久~~ | — | ✅ **已实证不持久**（V7），已固化标准流程：每次会话首动作 `install_extension`（幂等 + 天然热更新） |
| `--ignore-default-chrome-arg` 与其它默认参数（如 `--disable-component-extensions-with-background-pages`）交互 | 个别扩展功能仍被禁 | ✅ V3/V4/V5 全链路验证通过，无残留禁用项 |
| 面板捕获流水线仍不可 MCP 驱动 | 面板数据类问题排查受限 | 既有 Playwright e2e 兜底（已覆盖消息通道与 panel 流水线） |
| 包装脚本引入的启动延迟/失败 | MCP 服务不可用 | 脚本只做 exec npx 转发 + 轻量预检；异常路径 echo 到 stderr 后仍启动 |

---

## 7. 待实现清单

- [x] `scripts/mcp-chrome-debug.sh` 新建（启动/预检/清理三态）— 2026-08-08
- [x] `.mcp.json` chrome-devtools 服务指向包装脚本 — 2026-08-08（**下次会话重启生效**）
- [x] §4 V1–V7 全项验证 — 2026-08-08（结果见 §8）
- [x] V7 持久化失败 → 登记「会话首动作 install_extension」为标准流程（§3.2 A4/A5）
- [x] 文档化调试工作流（本文档 §3.2 A5 为 SSOT）

---

## 8. 验证记录（2026-08-08 实证回填）

| 验证项 | 结果 | 备注 |
|--------|------|------|
| V1 A/B 实证 | ✅ | 双路径均有效：`--chrome-arg=--enable-extensions` 与 `--ignore-default-chrome-arg=--disable-extensions` 都成功加载扩展（修正 §2.3 初判）；主路径选官方摘除参数 |
| V2 扩展安装 | ✅ | `install_extension` → id `ogicebjhkhmjlacnmibkkgphibgbojom`；`list_extensions` 显示 "Task Chrome Plugin v1.8.0 Enabled" |
| V3 内容脚本 | ✅ | https://example.com 导航后：`#taskplugin-daydaymoney-status` 存在、`[id^="taskplugin-"]` 计 38 |
| V4 Popup | ✅ | `trigger_extension_action` → `list_pages` 出现 `Extension Pages: …/popup/popup.html` target |
| V5 SW 调试 | ✅ | `list_pages` 含 `sw-1: …/background/service-worker.js` |
| V6 面板兜底 | ✅ | panel.html 快照完整：🔧 TaskPlugin / ⚠️ 未登录 / 5 tab / 请求列表 / 搜索框 / 方法过滤（请求列表为空属预期 — 无 devtools.js 数据源） |
| V7 持久化 | ❌ 不持久 | loadUnpacked 会话级；同 profile 新实例 `list_extensions` 为空 → 标准流程：每会话首动作 install_extension |
| V8 回归 | ✅ | 未触碰 taskChromePlugin 源码/e2e，零回归面；MCP 其余工具（导航/快照/evaluate）在调试实例上全部正常 |

> 注：本会话正在运行的 MCP chrome-devtools 仍为旧配置（默认 profile）；新配置在**下次会话启动时生效**。
