# onlineServiceJS 克隆日志滚动回弹与复制失败 — 设计文档

**日期:** 2026-07-05  
**范围:** `trae-agent/onlineServiceJS/static/index.html` — 「克隆仓库」区块日志区（`#cloneOut`）  
**页面:** `http://183.250.1.132:8765/ui/tok_*`（容器内 Trae Online Service 控制台）  
**状态:** 已实现（2026-07-05）

---

## 问题描述

用户在容器 Online Service 控制台使用「克隆仓库」时遇到两个问题：

1. **日志区滚动自动回弹到顶部**：克隆进行中日志持续更新时，用户向下滚动查看历史输出，滚动位置会被强制重置到最上方，无法稳定向下阅读。
2. **点击「复制日志」报错**：点击 `#btnCopyCloneLog` 后弹出 `复制失败：…` 告警（典型为 Clipboard API 不可用）。

---

## 当前架构理解（基线 v1，无变更）

根据 `docs/architecture/` 当前 **current** 架构：

| 维度 | 内容 |
|------|------|
| 架构视图 | `v1-enterprise-landscape`（企业全景）、`v1-application-integration`（39 服务组件） |
| 本次相关组件 | **Container Stack** 内 `go-run-container` 启动的 Agent 容器；容器内 **onlineServiceJS**（Node/Express，:8765）提供 `/ui/:token` 静态控制台 |
| 数据流 | UI → `POST /api/repos/clone` 异步入队 → git clone stderr 经 exec-stream 分片 → SSE `exec_stream_segment` + 轮询 manifest 补片 → iframe 渲染 |
| 版本历史 | v1 ✅ current；v2/v3 🎯 target（与本 bug 无关） |

本次为 **纯前端交互缺陷**，不涉及服务拓扑、数据流或组件增删，**无需新建架构版本文件**。

---

## 根因调查

### 问题 1：日志滚动回弹

#### 现象机制

克隆日志渲染在 `#cloneOutFrame`（sandbox iframe，`srcdoc` 富文本/纯文本）内。日志更新时会**整页替换** `iframe.srcdoc`，导致 iframe 内文档 scrollTop 归零。

代码中已有「贴底跟随」意图（`setExecRichIframeSrcdocSticky`），但实现不完整：

```949:983:trae-agent/onlineServiceJS/static/index.html
    /** 原始控制台 / 克隆输出 iframe：仅在用户原本贴在底部时追加内容后才自动滚到底，避免打断向上翻阅 */
    function setExecRichIframeSrcdocSticky(fr, srcdocHtml) {
      let stickToBottom = true;
      try {
        stickToBottom = iframeLogNearBottom(fr.contentDocument);
      } catch (_) {}
      fr.srcdoc = srcdocHtml;  // ← 无条件重置 scrollTop 为 0
      try {
        if (stickToBottom) iframeLogScrollToBottom(fr.contentDocument);
        // ← 非贴底时未恢复 scrollTop，用户视角即「弹回顶部」
      } catch (_) {}
    }
```

#### 触发路径（至少三条）

| 路径 | 频率 | 是否走 sticky | 问题 |
|------|------|---------------|------|
| SSE / 轮询 `rebuildCloneOutFrameFromParts` → `setExecRichIframeSrcdocSticky` | ~900ms（`ensureCloneLogPoll`） | 是 | 非贴底时未恢复 scrollTop |
| 引导克隆 bootstrap 轮询 `setCloneFrameFromPlainLog` | ~400ms（`layersBoot` while 循环） | **否** | 每次直接 `fr.srcdoc = …`，必回顶 |
| 克隆结束 `fetchCloneLogOnce` → `rebuildCloneOutFrameFromParts` | 一次 | 是 | 同上 sticky 缺陷 |

现有 E2E `e2e/job-raw-log-scroll.spec.mjs` 仅断言「用户在**顶部**时分片更新仍 near top」，**未覆盖**「用户滚到中间/上方非顶部分区」——因此未捕获真实用户场景。

#### 次要因素（建议一并修复）

`.exec-rich-frame` 在 flex 容器 `.exec-out-host`（`max-height: 220px; overflow: hidden`）内缺少 `min-height: 0`（runAll `#logs-content` 同类问题见 `2026-06-02-runall-logs-content-scroll-clipping-design.md`）。可能导致 iframe 高度计算不稳定；主因仍是 srcdoc 替换丢 scrollTop。

---

### 问题 2：复制日志报错

#### 代码路径

```2453:2471:trae-agent/onlineServiceJS/static/index.html
    document.getElementById('btnCopyCloneLog').onclick = debounceClick(async () => {
      const text = getCloneSnapshotPlain();
      ...
      try {
        await navigator.clipboard.writeText(text);  // ← 无 fallback
      } catch (e) {
        alert('复制失败：' + (e && e.message ? e.message : String(e)));
      }
    });
```

#### 根因

用户访问地址为 **`http://183.250.1.132:8765`**（HTTP + 非 localhost IP）。  
[Clipboard API](https://developer.mozilla.org/en-US/docs/Web/API/Clipboard/writeText) 要求 [**Secure Context**](https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts)（HTTPS 或 `localhost` / `127.0.0.1`）。在此环境下 `navigator.clipboard.writeText` 抛出 `NotAllowedError` 或 API 不可用。

`127.0.0.1` 本地开发通常正常；**远程 HTTP 部署必现**，与业务逻辑无关。

同文件内任务卡片「复制日志」（`.copy-job-log`）及 JSON 复制（~3684 行）同样仅依赖 Clipboard API，存在相同隐患；本次一并纳入修复范围以保持行为一致。

`getCloneSnapshotPlain()` 本身可正常读取 iframe `contentDocument.body.innerText`（sandbox 含 `allow-same-origin`），**不是**复制失败主因。

---

## 目标与非目标

### 目标

- 克隆日志流式更新时：用户**不在底部**则保持当前阅读位置（scrollTop）；用户在底部则继续自动跟随最新输出。
- 「复制日志」在 HTTP 非安全上下文下可用（execCommand / textarea 回退）。
- 引导 bootstrap 克隆轮询与 exec-stream 路径行为一致。
- 补充 Playwright 回归：中间滚动位置 + 贴底跟随。

### 非目标

- 不改克隆后端 API、exec-stream 协议或 SSE 事件格式。
- 不改为 `<pre>` 增量 DOM append（可后续优化；本次最小修复 sticky + fallback）。
- 不强制容器 UI 走 HTTPS（运维层另议）；前端须在无 Secure Context 下可用。
- 不更新 Archimate 架构文件（无组件/拓扑变更）。

---

## 方案设计

### 方案 A（推荐）：修复 sticky 滚动 + 统一 copy 工具函数

#### A1. 增强 `setExecRichIframeSrcdocSticky`

```javascript
function setExecRichIframeSrcdocSticky(fr, srcdocHtml) {
  let stickToBottom = true;
  let savedScrollTop = 0;
  try {
    const doc = fr.contentDocument;
    const el = iframeLogScrollingEl(doc);
    stickToBottom = iframeLogNearBottom(doc);
    if (!stickToBottom && el) savedScrollTop = el.scrollTop;
  } catch (_) {}
  fr.srcdoc = srcdocHtml;
  try {
    const doc = fr.contentDocument;
    const el = iframeLogScrollingEl(doc);
    if (!el) return;
    if (stickToBottom) {
      iframeLogScrollToBottom(doc);
    } else {
      el.scrollTop = savedScrollTop;
    }
  } catch (_) {}
}
```

日志追加在底部时，保持 `savedScrollTop` 即可稳定视口；无需比例换算。

#### A2. `setCloneFrameFromPlainLog` 改走 sticky

```javascript
function setCloneFrameFromPlainLog(text) {
  const fr = document.getElementById('cloneOutFrame');
  if (!fr) return;
  setExecRichIframeSrcdocSticky(
    fr,
    buildExecRichSrcdoc([{ contentType: 'text/plain', text: normalizeCloneLogText(text || '') }]),
  );
}
```

修复 bootstrap 400ms 轮询路径。

#### A3. CSS 微调（可选但建议）

```css
.exec-rich-frame {
  flex: 1 1 auto;
  min-height: 0;   /* 新增：flex 子项可收缩，iframe 高度稳定 */
  ...
}
```

srcdoc 内 `html, body` 增加 `height: 100%; overflow: auto;` 使滚动容器明确（减少跨浏览器差异）。

#### A4. 抽取 `copyTextToClipboard(text)`

参考 `task2app/.../useTaskDetail.js` 已有模式：

1. 优先 `navigator.clipboard?.writeText(text)`（Secure Context）
2. 失败或不可用时：`textarea` + `select()` + `document.execCommand('copy')`
3. 仍失败时再 `alert` 并可选展示可选手动复制对话框

替换调用点：

- `#btnCopyCloneLog`
- `.copy-job-log`
- 其它 `navigator.clipboard.writeText` 单点（JSON 复制）

#### A5. 测试

| 用例 | 文件 | 断言 |
|------|------|------|
| 中间 scroll 保留 | 扩展 `e2e/job-raw-log-scroll.spec.mjs` 或新建 `clone-log-scroll.spec.mjs` | `scrollTop` 在中间时分片更新后仍 > 阈值 |
| 贴底跟随 | 同上 | 贴底更新后 near bottom |
| 复制 fallback | 新建 API/单元测试或在 Playwright 中 mock 非 secure context | `copyTextToClipboard` 走 textarea 路径成功 |

---

### 方案 B（备选）：改为 `<pre>` 增量 append，避免整页 srcdoc 替换

- 优点：天然保留 scroll；性能更好。
- 缺点：与 exec-stream 富文本 HTML 分片、现有 `buildExecRichSrcdoc` 架构耦合深；改动面大。
- **结论：** 不作为本次首选；若 A 验证后仍有边缘 case 再评估。

---

## 价值流影响

纯前端 UI 修复，无 `value-stream.yaml` 字段变更，无新增价值流步骤。

受影响的用户旅程：**容器内 Agent 控制台 → 克隆仓库 → 查看/复制日志**（开发运维路径，非 SaaS 主站价值流）。

---

## 领域概念（轻量清单，供后续 DDD 跳过）

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | Container Runtime UI | onlineServiceJS 静态控制台 |
| 值对象 | ExecStreamSegment | kind=clone, seq, text, content_type |
| 聚合根 | WritableLayer | layer_id 关联克隆日志 exec-stream |

---

## 验收标准

- [ ] 克隆进行中（含 bootstrap 与手动克隆），用户滚到日志中间后等待 ≥3 次轮询更新，scrollTop 不回跳到 0。
- [ ] 用户在日志底部时，新输出到达后仍自动滚到底。
- [ ] 在 `http://<非localhost IP>:8765` 点击「复制日志」，无报错，剪贴板含 banner + iframe 文本。
- [ ] 本地 `127.0.0.1:8765` 复制仍正常（Clipboard API 路径）。
- [ ] Playwright 回归通过。

---

## 实施计划（审批后 `/8-build`）

1. 修改 `setExecRichIframeSrcdocSticky` + `setCloneFrameFromPlainLog` + CSS（~30 行）
2. 新增 `copyTextToClipboard`，替换 3 处复制按钮（~40 行）
3. 扩展 E2E 滚动测试 + 复制 smoke（mock 或 document 环境）
4. 手动在远程 HTTP 页面验证（用户环境 `183.250.1.132:8765`）

**预估工作量：** 小（单文件为主，0.5 天内可交付含测试）

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| `execCommand('copy')` 在部分移动端/旧浏览器失效 | 保留 alert + 可选 modal 展示全文 |
| 日志暴增时 scrollHeight 变化导致轻微跳动 | 仅追加底部位移时 `savedScrollTop` 仍稳定；极端情况可记录 scroll 比例 |
| iframe sandbox 限制 | 已含 `allow-same-origin`；不增加 `allow-scripts` |

---

## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-05 | 初稿：滚动 sticky 缺陷 + Clipboard Secure Context 根因与方案 A |
