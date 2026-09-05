# Navbar 无仓库跳价格页 — 实施计划

- **日期**: 2026-08-29

## 任务

- [x] T1 更新意图：`navbar_git_service_region_nav` + `navbar-git-service-regions`（空列表 → `/pricing/`；失败 fail-open）
- [x] T2 Red：`Navbar.ui.test.js` — ready 空列表 href=`/pricing/?accessCode=`；error/unknown 当前页；VIP1 空列表仍价格页+角标
- [x] T3 Green：`NavbarGitServiceNav` + `Navbar.ui` 传入 `pricingHref` 与 `gitResourcesStatus`
- [x] T4 `Navbar.logic.fetchGitResources` 写 `ready`/`error`；无租户 `ready`；失败 `console.warn`
- [x] T5 逻辑测例透传 `gitResourcesStatus`（成功 ready / 失败 error）
- [x] T6 更新 `docs/flows/value-stream-test-integration.wsd` NAVGIT 测试点
- [x] T7 跑 Vitest 相关文件；gofmt/py 不适用

事件任务：无（纯前端）。
