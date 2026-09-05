# Navbar 无仓库跳价格页 — 价值流

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-navbar-git-empty-to-pricing-design.md`

## 影响的既有流

`docs/flows/value-stream-test-integration.wsd` 中 **租户设置 · GitLab 资源购买** → `(Navbar 代码仓库按区域跳转) as NAVGIT`。

本增量只改 NAVGIT 空列表出口：从「留在当前页」改为「去价格页（转化）」；有资源出口不变。

## 增量（单切片）

1. 已登录用户打开带顶栏的页面 → 拉取 gitlab-resources。
2. 空列表且 ready → 点击「代码仓库」到达 `/pricing/`（可购 GitLab/会员）。
3. 有资源 → 仍下拉进 GitLab。

不新增 YAML value-stream 条目（无新表字段、无新 pytest 文件）；更新 wsd 测试点即可。
