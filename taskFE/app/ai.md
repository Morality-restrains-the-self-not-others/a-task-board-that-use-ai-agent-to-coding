# 规则文件

## 基本信息
- 版本：1.5.0
- 创建日期：2026-07-15
- 最后修改：2026-08-20
- 维护者：Trae AI 团队

## 规则分类

### 核心规则（一级分类）
> 影响代码质量和安全性的关键规则，必须严格遵守

#### 构建发布（二级分类）
##### 改 `src/` 后执行完整 `npm run build`（原子切 html symlink）
- 描述：本目录为 Vue/Vite 应用根。凡修改 `src/`（或本目录 `vite.config.js` / 影响产物的插件）并需要公网 SPA 生效时，Agent **必须在同一任务收尾**执行完整构建。`npm run build` = `scripts/atomic-vite-build.sh`：先编译到 `public/.next`，校验 `index.html` 后再 `mv` 进 `public/releases/<id>` 并 `ln -sfn` 切换 `public/html`；失败时**不切** symlink，Docker nginx 继续旧 release。Django 已退役，**不再**串联 `collectstatic`。
- 适用场景：编辑 `app/src/**`、调整 Vite 构建配置后
- 优先级：高
- 规则：
  1. 工作目录：`taskFE/app/`
  2. 执行：`npm run build`（或 `bash scripts/runall-lifecycle.sh build`，二者均委托 `scripts/atomic-vite-build.sh`）
  3. **禁止**构建入口先 `rm -rf dist` / `rm -rf public/html` 再 `vite build`（失败会打挂仍在跑的公网）
  4. 成功判据：命令 exit 0；`public/html` 为 symlink 且含新 hash 入口 JS；本地 Docker nginx :4000 / 公网静态路径抽查 200
  5. 启动日志不得再出现 Django / `collectstatic` / `task2app/` 退役提示（`collectstatic-after-vite.sh` 已为无声 no-op，且不得挂在 `build` 脚本上）
  6. 门禁自测：`bash scripts/test_build_clean_dist.sh`、`bash scripts/test_nginx_static_resident.sh`、`bash scripts/test_build_npm_diagnostics.sh`

##### 禁止只跑未产出 release 的路径收尾公网任务
- 描述：公网与 `runall-lifecycle.sh start`（Docker nginx）读的是 `public/html`。未执行 `npm run build` / `runall-lifecycle.sh build` = 旧 hash 或缺失资源。
- 适用场景：任何声称「已部署/已让公网生效」的前端改动
- 优先级：高
- 规则：必须使用 `npm run build` / `runall-lifecycle.sh build`（纯 Vite；原子 staging→releases + symlink）。构建**不得** docker stop / recreate `taskfe-nginx`。

##### FAQ 联系邮箱改 conf 不改 markdown
- 描述：`/faq/` 展示的联系邮箱 SSOT 为 `conf/frontend/vue/config.yaml` 的 `contactEmail`。markdown 只写 `{{contactEmail}}`。
- 适用场景：更换公网 FAQ 联系邮箱
- 优先级：高
- 规则：只改 conf 该键，然后在本目录 `npm run build`。禁止把真实邮箱写回 `src/faq/*.md`。

#### 组件 DOM（二级分类）
##### 副作用按钮须防重放
- 描述：写操作按钮必须同步门闩 + pending 态 + 点击意图级 `Idempotency-Key`（`createClickGuard`）。
- 适用场景：编辑 `app/src/**/*.vue` 中会 POST/PUT/PATCH/DELETE 的点击
- 优先级：高
- 规则：详见仓库根 `.ai/01_project_constraints/57_frontend_button_anti_replay.md`；实现 `app/src/utils/clickGuard.js`

##### 无必要禁止 Teleport / createPortal
- 描述：本目录 Vue 源码默认就地渲染；禁止用 `<Teleport>` 把功能卡送到远亲槽位。仅当祖先 overflow/transform 裁剪或错位浮层且无法就地解决时才允许，并须 `Teleport-OK` 注释。
- 适用场景：编辑 `app/src/**/*.vue`、新增浮层/下拉/跨区域 DOM
- 优先级：高
- 规则：详见仓库根 `.ai/01_project_constraints/50_no_unnecessary_vue_teleport.md`

### 最佳实践（一级分类）
> 提升开发效率和代码可维护性的建议

#### 本地开发（二级分类）
##### Vite dev 与公网产物分离
- 描述：`npm run dev` 不替代生产构建；用户验证的是 preview/公网入口时仍须走 `npm run build` 链路。
- 适用场景：本地 HMR 调试 vs 公网验收
- 优先级：中

## 规则冲突处理
- 当规则冲突时，遵循以下优先级：
  1. 核心规则 > 最佳实践 > 风格指南
  2. 文件级规则 > 目录级规则 > 全局规则
  3. 新版本规则覆盖旧版本规则

## 变更日志
- 2026-08-20：版本 1.5.0 - 公网托管改为 Docker nginx + `public/html` symlink；Vite 只 build
- 2026-08-13：版本 1.3.0 - 增补无必要禁止 Teleport（交叉引用 `50_no_unnecessary_vue_teleport.md`）
- 2026-08-10：版本 1.1.0 - Django 退役后移除 build 链 collectstatic；成功判据改为 dist/preview
- 2026-07-22：版本 1.0.1 - `npm run build` 内置 collectstatic；禁止仅用 `build:vite` 收尾公网
- 2026-07-15：版本 1.0.0 - 初始创建：约束 Agent 在改 src 后执行 runall-lifecycle.sh build
