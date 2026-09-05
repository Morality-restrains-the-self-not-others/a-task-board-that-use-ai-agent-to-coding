# 静态资源引用须带内容 Hash 查询参数（缓存击穿）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-05-17
- 最后修改：2026-05-17
- 维护者：Trae AI 团队

## 背景（为何是元规则）

2026-05 联调中，Vite 开发态下浏览器请求稳定路径 `/utils/workPanelBranchHelpers.js`，反向代理对该 URL 设置了较长 `Cache-Control`（如 `max-age=3600`）。组件已发布并 `import` 新的命名导出，但客户端仍命中**旧版无该导出的 JS**，触发：

`SyntaxError: The requested module '...' does not provide an export named 'WORK_BRANCH_PRESET_OPTIONS'`

根因是**资源 URL 未随内容变化而失效**，而非业务逻辑错误。此类问题在「开发域经 CDN/网关缓存」「Django 模板手写 `<script src>`」「动态 `import()` 使用绝对路径」等场景均可复现。

## 规则分类

### 核心规则

#### 资源 URL 必须可随内容失效

- **描述**：凡通过 **URL 字符串**（非打包器模块图）引用、且可能被浏览器或中间层缓存的前端静态资源（`.js`、`.css`、`.wasm`、字体、部分 JSON 配置等），**必须**在 URL 上附带与**文件内容**绑定的查询参数（下称 **hash query**），使内容变更后 URL 必然变化，避免读到过期副本。
- **适用场景**：
  - HTML / Django 模板 / SPA 壳中的 `<script src>`、`<link href>`
  - 运行时 `import('http(s)://.../path/file.js')` 或 `new URL(..., import.meta.url)` 拼出的**公网可缓存**路径
  - 网关、CDN、nginx 对 `/static/`、`/utils/` 等目录配置了 `max-age` 或 ETag 长缓存
  - 异域部署下前端域与构建产物域分离时的手工资源拼接
- **不适用**（由打包器保证文件名 hash 即可）：
  - Vite/Webpack 构建产物中 **文件名已含 contenthash** 且通过 `manifest` 引用
  - 同源 `import` 相对路径模块，且开发服务器对源码禁用长期缓存（仍须避免在模板中重复暴露同路径裸 URL）
- **优先级**：高

##### hash query 约定

| 项 | 要求 |
| --- | --- |
| 参数名 | 推荐 `h` 或 `v`；全项目择一并在工具函数中统一 |
| 参数值 | **内容摘要**（如 SHA-256 前 8～12 位十六进制）或**构建 ID**（须随该文件内容或整次构建变化） |
| 禁止 | 仅用固定版本号、日期字符串、手动递增整数且未与文件内容绑定 |
| 示例 | `/utils/workPanelBranchHelpers.js?h=3f2a9c1b` |

##### 实现与评审检查要点

1. **优先模块图**：业务代码中共享逻辑优先 `import ... from '../utils/xxx.js'`，由 Vite 处理依赖与 HMR；避免为可打包模块单独暴露长期缓存的裸 `/utils/*.js` 给浏览器直连。
2. **必须裸 URL 时**：通过构建脚本、Django 上下文或统一 helper（如 `assetUrl(path, contentHash)`）注入 hash query；禁止在模板/配置中写死无 query 的静态路径。
3. **仓库落地**：
   - Vite：`front_project/app/vite-plugin-asset-cache-bust.js`（`src` 模块 resolve 自动加 `?h=` / `&h=`）
   - 前端工具：`front_project/app/src/utils/assetUrl.js`（`appendAssetHashQuery` / `importAppModule`）
   - Django 模板：`{% vite_dev_asset %}`、`{% static_asset %}`（`frontend_app/templatetags/vite_tags.py`）
3. **反向代理**：对「源码目录映射」路径（开发态 `/src/`、`/utils/` 等）应使用 `Cache-Control: no-cache` 或极短 `max-age`；**不能**假设代理已禁用缓存——仍须在规则层面要求 hash query 作为双保险。
4. **变更联动**：修改被裸引用的 JS/CSS 时，同步更新 hash 生成逻辑或重新构建；PR 评审检查新增 URL 是否带 hash query。
5. **测试**：Playwright / E2E 可对关键页采集 `pageerror` 与 `SyntaxError ... does not provide an export` 类错误；必要时断言关键脚本 URL 含 `?h=` 或 `?v=`。

### 最佳实践

- 生产构建继续依赖 Vite `manifest` 与文件名 hash；本规则补齐**开发态、模板注入、跨域静态**等打包器未覆盖的缝隙。
- 新增共享常量或小模块时，优先独立文件并走 `import`，减少「只改 helpers 大文件、URL 不变」的缓存风险（参见 `workBranchPresetOptions.js` 拆分实践）。

## 关联

- 项目约束索引：[00_project_constraints.md](./00_project_constraints.md) 第 19 条
- 前端 JS 加载：[../04_frontend_development/00_frontend_development.md](../04_frontend_development/00_frontend_development.md)
- 失败经验（导出/模块）：[../09_failure_experience/01_compilation_errors/01_export_syntax_error.md](../09_failure_experience/01_compilation_errors/01_export_syntax_error.md)

## 变更日志

- 2026-05-17：1.0.0 初版；由任务详情 `WORK_BRANCH_PRESET_OPTIONS` 与 nginx 缓存导致的模块导出错误归纳。
