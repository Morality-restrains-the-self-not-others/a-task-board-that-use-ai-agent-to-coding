# 实施计划：项目详情页三项点击即编辑

日期：2026-07-12

## 任务

- [x] 1. 工具函数 `projectDetailInlineEditUtils.js`：构建 tags/image/auto_run 的 PATCH body；`canEnableDefaultAutoRun`
- [x] 2. Vitest 先写红测（utils + 组件）
- [x] 3. 组件 `ProjectDetailInlineEditableFields.vue`
- [x] 4. `ProjectDetail.vue` 接入，替换只读块；保留镜像 display testid
- [x] 5. 更新意图索引；跑 vitest
- [x] 6. Playwright 冒烟（`ProjectDetail.inline-edit-smoke`）
- [x] 7. 拆分 OAuth/分支预览：`ProjectDetailGitReposSection` + `useProjectDetailGitRepos`（详情页 ≤500 行）

## 验证命令

```bash
cd task2app/front_project/app && npx vitest run src/utils/projectDetailInlineEditUtils.test.js src/components/ProjectDetailInlineEditableFields.test.js src/views/ProjectDetail.test.js
bash task2app/playwright/front_project/tests/ProjectDetail.inline-edit-smoke.playwright.test.sh
```

验证结果（2026-07-12）：
- Vitest：20 passed（utils 6 + inline fields 5 + ProjectDetail 9）
- Playwright smoke：3 passed
- `ProjectDetail.vue`：410 行
