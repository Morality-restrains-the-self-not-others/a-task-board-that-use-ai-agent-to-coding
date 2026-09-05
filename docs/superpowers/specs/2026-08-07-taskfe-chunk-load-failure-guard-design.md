# taskFE 懒加载 chunk 获取失败「重试 + reload 兜底」设计

- **日期**: 2026-08-07
- **作者**: claude
- **状态**: 🎯 target（待批准）
- **关联服务**: taskFE/app（Vue 3 + Vite 前端）
- **OPT 编号**: OPT-20260807-056
- **决策记录**: 头脑风暴阶段用户已选定「保留动态导入 + 重试兜底」（否决仅 UserProfile 静态导入方案）

## 1. 问题现象

线上 www.daydaymoney.com 前端控制台报错（导航到用户资料页时）：

```
index-6ovTL2oo.js:34 TypeError: Failed to fetch dynamically imported module:
https://www.daydaymoney.com/static/assets/UserProfile-COTeeBTX.js
T @ index-6ovTL2oo.js:34 → (anonymous) → Promise.catch → v → g → navigate @ ...
```

- 错误 URL **无 `?h=` 缓存破坏参数** → 生产构建产物（dev 态 assetCacheBustQueryPlugin 才追加 hash query）。
- 堆栈 `navigate` 链 = vue-router 导航解析懒加载路由组件时 `import()` 拉 chunk 失败。

## 2. 根因分析

### 2.1 机制

浏览器已缓存旧 `index-*.js`（或旧 HTML），其中引用旧哈希 chunk `UserProfile-COTeeBTX.js`；发布后该文件已不存在（哈希变更）→ 404 / 或 CDN 边缘节点缓存不一致 / 或瞬时网络失败。生产构建为默认 Vite 分块（`manualChunks: undefined`），全部 ~40+ 路由 `() => import(...)` 懒加载，**无任何 chunk 加载失败兜底**（[main.js](../../taskFE/app/src/main.js) 仅 49 行，无 errorHandler / 重试 / reload）。

### 2.2 为什么不做静态导入（头脑风暴结论）

| 维度 | 静态导入 | 保留动态导入 + 兜底 |
|------|---------|-------------------|
| 本组件报错 | 消除 | 兜底后自动恢复 |
| 首包/LCP | ❌ 依赖树全量进首包，全站用户买单，逆转 [router.js:9-16](../../taskFE/app/src/router.js#L9-L16) 既有 LCP 优化决策 | 不变 |
| 缓存粒度 | ❌ 改 UserProfile → 主包 hash 变 → 全站重下 | 不变 |
| 覆盖面 | ❌ 只救 3 条 profile 路由，其余 ~40 路由同模式无兜底 | ✅ 全部懒加载路由 |
| 部署竞态 | ❌ 把失败窗口从「小 chunk 404」移到「主包 404」（更宽） | ✅ 重试 + 刷新拿新 index.html |

## 3. 方案设计

### 3.1 代码层 — taskFE/app/src

新建 `src/utils/chunkLoadGuard.js`（纯函数 + 安装器，可单测）：

| 导出 | 职责 |
|------|------|
| `isChunkLoadError(error)` | 跨浏览器错误签名识别（正则，大小写不敏感） |
| `installRouterChunkGuard(router)` | 挂 `router.onError`：首次 chunk 错 → 重试导航 1 次（`router.replace({ path: to.fullPath, force: true })`，重跑 lazy import）→ 重试仍失败 → `window.location.reload()` 兜底（每页面生命周期仅 1 次 reload，防循环） |

**错误签名覆盖**（跨浏览器 + MIME 变体）：

```
Failed to fetch dynamically imported module
error loading dynamically imported module      (Firefox)
Importing a module script failed               (Safari)
Loading chunk N failed                         (webpack 类)
Unable to preload CSS                          (Vite preload)
```

**重试语义**：vue-router 4 导航失败时路由记录的 lazy getter 未被标记成功，`force: true` 重导航会重新执行 `import()`；瞬时网络失败即可恢复。重试失败后 reload 一次 → 新页面加载新 index.html → 新 chunk 映射 → 收敛（不会死循环：每页 1 次导航重试 + 1 次 reload 封顶）。

### 3.2 配套检查（非本次代码范围，部署时核对）

- 边缘 nginx：确认 `/static/assets/*.js` 返回**真 404**（非 SPA fallback HTML 200）——若 chunk 收到 text/html，兜底重试/reload 也无法恢复（MIME 校验必然失败），需在 nginx 侧修复。
- 发布流程：可考虑保留前 N 版 assets（grace period），治本项，单独评估。

### 3.3 测试（vitest，随实现提交）

`src/utils/chunkLoadGuard.test.js`：

1. `isChunkLoadError` 命中 5 类签名 + MIME 变体；普通 TypeError/业务错误返回 false。
2. 首次 chunk 错误 → 触发 1 次导航重试。
3. 重试导航失败 → 触发 1 次 reload。
4. 同会话再次 chunk 错误 → 不再无限重试（≤1 次 reload 封顶断言）。

## 4. 🕸️ Code Review Graph 分析

`CRG unavailable: taskFE graph.db 存在但 0 nodes（从未构建），无法提供调用链/爆炸半径`。影响面由人工核查：`router.js` 全部懒加载路由 + `main.js` 入口均经 `router.onError` 单点兜底，无其他 `import()` 使用方（grep 确认无内联动态 import）。

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 前端懒加载 chunk 获取失败自动恢复 | — | 纯前端加载健壮性，无服务端状态变更、无跨边界副作用 → 「无对应事件」（硬门禁例外条款） |

## 6. 🏛️ 架构变更影响

- **无架构文件变更**：前端实现级修复（错误兜底），不涉及组件/服务/数据流/基础设施增删改，不符合架构更新触发场景（纯 Bug 修复/实现级）。当前架构基线 v14 current 不动。
- 不新增 Python 接口（纯前端）→ Python 接口专项审批门**未触发**。

## 7. Value Stream 影响

无现有流受影响：不触碰任何 `<service>.<table>.<field>`，无新流、无测试文件归属变更（仅新增 taskFE 单测）。仅前端加载健壮性提升，对所有流的最终用户导航体验有隐性改善。

## 8. 交付物

- 🆕 `taskFE/app/src/utils/chunkLoadGuard.js`（~40 行）
- 🆕 `taskFE/app/src/utils/chunkLoadGuard.test.js`（~4 用例）
- 🟡 `taskFE/app/src/main.js`（或 router 出口）接入安装器（~2 行）
- 📋 OPT-20260807-056 登记 `.learnings/OPTIMIZATION_TODOS.md`
- 部署后复验：生产导航到 `/profile/` 无报错；可临时用 devtools offline 模拟 chunk 失败观察重试/刷新行为
