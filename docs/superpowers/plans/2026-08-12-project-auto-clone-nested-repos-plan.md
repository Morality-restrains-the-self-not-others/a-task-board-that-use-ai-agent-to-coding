# Plan：项目自动克隆子仓库开关

- **日期**: 2026-08-12
- **设计**: `docs/superpowers/specs/2026-08-12-project-auto-clone-nested-repos-design.md`

## 切片（TDD）

- [ ] **T1** dataMigrate `012_auto_clone_nested_repos.sql` + schema 断言
- [ ] **T2** taskProjectService create/update/get 读写字段（缺省 true）
- [ ] **T3** taskTaskService container-snapshot 透传
- [ ] **T4** taskCredentialService Merge 门控 + 单测
- [ ] **T5** CreateProject.vue checkbox + 单测/提交字段
- [ ] **T6** ProjectDetailGitReposSection 开关 + 空态文案
- [ ] **T7** intents + INDEX 更新
- [ ] **T8** Playwright CreateProject 勾选可见

## 意图对照

| 意图 | 事件 | 备注 |
|------|------|------|
| project_auto_clone_nested_repos | — | 配置 CRUD 例外 |

## 回滚

- 关 FE 勾选隐藏 + 忽略字段；DB 列可保留（DEFAULT 1 无害）。
- 或 feature 上强制 enrich 忽略字段（热修）。
