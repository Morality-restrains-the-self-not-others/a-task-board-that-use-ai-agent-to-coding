# 公网 SPA 白屏：Vite build 后未 collectstatic 导致 /static/main-*.js 404

> **历史文档（Django STATIC_ROOT 时代）**。Django saas-backend 已于 2026-07-30 退役；当前公网 taskFE 由 Vite `public/releases/` + Docker nginx（:4000）托管，`npm run build` **不再**调用 collectstatic。现网白屏请先核对 `public/html` hash 与网关静态路径，勿再跑 Django collectstatic。

## 基本信息

- 版本：1.4.0
- 创建日期：2026-07-11
- 最后修改：2026-08-10
- 维护者：Trae AI 团队

## 现象

- `https://www.daydaymoney.com/tenant/<id>/…/task-detail/…/`（及其它公网 SPA 路由）HTML 200，但页面空白/打不开。
- 浏览器 Network：`/static/main-<hash>.js` → **404**；同 hash 的 `/static/assets/main-<hash>.js` 可能仍为 **200**。

## 根因

1. 公网入口由 Django 渲染 `frontend/spa.html`，`{% vite_asset 'main.js' %}` 读 Vite **manifest** 生成静态 URL。
2. Vite `outDir` 为 `front_project/.../static/assets`（`assetsDir: ''`），manifest `file` 为 basename（如 `main-<hash>.js`）。
3. 自 `STATICFILES_DIRS` 改为 `("assets", <vite assets dir>)` 后，产物落在 **`STATIC_ROOT/assets/`**，公网路径为 **`/static/assets/<file>`**。
4. 若 `vite_asset` 仍拼 `/static/<file>`（缺 `assets/` 前缀），HTML 指向 404 → `#app` 空、黑屏/白屏。
5. 另一历史路径：跳过 `collectstatic` 时 HTML 已指向新 hash，但 `STATIC_ROOT` 无对应文件（见旧版说明）。DEBUG 下 `urls.py` 的 `static()` 托管的是 **`STATIC_ROOT`**，不是 Vite 源目录。
6. 自 2026-07-22：`npm run build` 已内置 `scripts/collectstatic-after-vite.sh`；易复发路径变为只跑 `npm run build:vite`，或改 STATIC 前缀后未同步改 `vite_tags`。

## 修复与预防

1. **立即（路径前缀）**：确认 `{% vite_asset %}` 输出 `/static/assets/main-<hash>.js`（见 `frontend_app/templatetags/vite_tags.py` 的 `_vite_static_rel_from_manifest_file`），改后 **HUP/重启 gunicorn**。
2. **立即（缺文件）**：`cd Saas_project && activate_env.sh run -- python3 manage.py collectstatic --noinput`（或 `front_project/app/scripts/collectstatic-after-vite.sh`），再访问 `/static/assets/main-*.js` 应为 `text/javascript` 200。
3. **启动脚本**：`run.sh` 第 8 步必须真实执行 collectstatic；build --watch 写出 assets 后须再次 collectstatic（watch 用 `npm run build:vite -- --watch`）。
4. **日常构建**：`cd front_project/app && npm run build`（或 `bash scripts/runall-lifecycle.sh build`）已含 collectstatic + `.vite` manifest 同步。`task2app_root` 须从 `scripts/` 上溯 **三级**（`app/` → `front_project/` → `task2app/`），误用 `../..` 会解析到 `front_project/` 并报 `activate_env.sh: 没有那个文件或目录`（exit 127）。
5. **回归**：`tests/test_vite_tags_cache_bust.py::test_vite_asset_main_resolves_to_collectable_file`（断言 URL 以 `/static/assets/` 开头）。

## 关联

- 元规则：`.ai/01_project_constraints/18_static_resource_cache_bust_query.md`
- 前端执行规则：`.ai/04_frontend_development/00_frontend_development.md`（公网 SPA 必须 build + collectstatic）
- 目录 companion（Agent 改 front_project 时伴读）：`taskFE/ai.md`、`taskFE/app/ai.md`
- 检查清单：`.claude/skills/simplify-and-harden/SKILL.md` Pass 4、`.claude/skills/10-ship/SKILL.md`（旧 `/7-ship` 已重定向至 `10-ship`）
- Runbook：`docs/runbooks/add-public-entry-origin.md`（build + collectstatic）
- 脚本：`taskFE/app/scripts/collectstatic-after-vite.sh`、`runall-lifecycle.sh`
